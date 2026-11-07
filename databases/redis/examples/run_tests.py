#!/usr/bin/env python3
"""Прогоняет примеры курса (*.redis) на сервере Redis и сверяет вывод с эталоном.

    python3 run_tests.py              — проверить все примеры
    python3 run_tests.py 05           — только модуль 05
    python3 run_tests.py --update     — перезаписать эталоны
    REDIS_HOST=... REDIS_PORT=...     — куда подключаться (по умолчанию 127.0.0.1:6379)

Формат сценария:
    # комментарий            печатается как есть
    SET k v                  команда клиента A, вывод как в redis-cli
    [B] GET k                команда клиента B (отдельное соединение)
    [B&] BLPOP q 0           отправить и не ждать ответа (клиент заблокирован)
    [B?]                     дочитать ответ клиента B (или сообщение Pub/Sub)
    ! HELLO 3                выполнить, но не печатать ответ (зависит от окружения)
    @sleep 0.2               пауза
    @lua name ... @end       сохранить многострочный скрипт; в командах подставляется как $name
Нужен только Python 3, без сторонних библиотек.
"""
import difflib, glob, os, shlex, socket, sys, time

HOST = os.environ.get("REDIS_HOST", "127.0.0.1")
PORT = int(os.environ.get("REDIS_PORT", "6379"))


class Conn:
    def __init__(self):
        self.sock = socket.create_connection((HOST, PORT), timeout=10)
        self.file = self.sock.makefile("rb")

    def send(self, args):
        parts = [b"*%d\r\n" % len(args)]
        for a in args:
            b = a if isinstance(a, bytes) else a.encode()
            parts.append(b"$%d\r\n%s\r\n" % (len(b), b))
        self.sock.sendall(b"".join(parts))

    def call(self, *args):
        self.send(list(args))
        return self.read()

    def read(self):
        line = self.file.readline()
        if not line:
            raise EOFError("соединение закрыто")
        t, rest = line[:1], line[1:-2]
        if t == b"+":
            return ("status", rest.decode())
        if t == b"-":
            return ("error", rest.decode())
        if t == b":":
            return ("int", int(rest))
        if t in (b"$", b"="):
            n = int(rest)
            if n < 0:
                return ("nil", None)
            data = self.file.read(n + 2)[:-2]
            if t == b"=":
                data = data[4:]          # verbatim: "txt:..."
            return ("bulk", data)
        if t in (b"*", b"~", b">"):
            n = int(rest)
            if n < 0:
                return ("nil", None)
            kind = {b"*": "array", b"~": "set", b">": "push"}[t]
            return (kind, [self.read() for _ in range(n)])
        if t == b"%":
            n = int(rest)
            return ("map", [(self.read(), self.read()) for _ in range(n)])
        if t == b"|":                     # атрибуты RESP3 пропускаем
            for _ in range(int(rest)):
                self.read(); self.read()
            return self.read()
        if t == b"_":
            return ("nil", None)
        if t == b",":
            return ("double", rest.decode())
        if t == b"#":
            return ("bool", rest == b"t")
        if t == b"(":
            return ("int", int(rest))
        raise ValueError("неизвестный тип ответа: %r" % line)

    def close(self):
        try:
            self.sock.close()
        except OSError:
            pass


def quote(b):
    """Экранирование строк как в redis-cli (sdscatrepr)."""
    out = ['"']
    for c in b:
        ch = chr(c)
        if ch in '\\"':
            out.append("\\" + ch)
        elif ch == "\n":
            out.append("\\n")
        elif ch == "\r":
            out.append("\\r")
        elif ch == "\t":
            out.append("\\t")
        elif ch == "\a":
            out.append("\\a")
        elif ch == "\b":
            out.append("\\b")
        elif 32 <= c < 127:
            out.append(ch)
        else:
            out.append("\\x%02x" % c)
    out.append('"')
    return "".join(out)


def fmt(reply):
    """Ответ в виде строк, как его печатает redis-cli."""
    t, v = reply
    if t == "status":
        return [v]
    if t == "error":
        return ["(error) " + v]
    if t == "int":
        return ["(integer) %d" % v]
    if t == "nil":
        return ["(nil)"]
    if t == "bulk":
        return [quote(v)]
    if t == "double":
        return ["(double) " + v]
    if t == "bool":
        return ["(true)" if v else "(false)"]
    if t in ("array", "set", "push"):
        if not v:
            return ["(empty set)" if t == "set" else "(empty array)"]
        mark = "~" if t == "set" else ")"
        w = len(str(len(v)))
        lines = []
        for i, e in enumerate(v):
            prefix = "%*d%s " % (w, i + 1, mark)
            sub = fmt(e)
            lines.append(prefix + sub[0])
            lines += [" " * len(prefix) + s for s in sub[1:]]
        return lines
    if t == "map":
        if not v:
            return ["(empty hash)"]
        w = len(str(len(v)))
        lines = []
        for i, (k, val) in enumerate(v):
            prefix = "%*d# %s => " % (w, i + 1, fmt(k)[0])
            sub = fmt(val)
            lines.append(prefix + sub[0])
            lines += [" " * len(prefix) + s for s in sub[1:]]
        return lines
    return [repr(reply)]


def reset(conn):
    """Чистое состояние сервера перед каждым сценарием."""
    conn.call("FLUSHALL")
    conn.call("SCRIPT", "FLUSH")
    conn.call("FUNCTION", "FLUSH")
    for k, v in [("maxmemory", "0"), ("maxmemory-policy", "noeviction"),
                 ("notify-keyspace-events", ""), ("slowlog-log-slower-than", "10000")]:
        conn.call("CONFIG", "SET", k, v)
    conn.call("SLOWLOG", "RESET")
    _, users = conn.call("ACL", "USERS")
    for u in users:
        name = u[1].decode()
        if name != "default":
            conn.call("ACL", "DELUSER", name)


def run_script(path):
    out = []
    conns = {}
    scripts = {}

    def conn(name):
        if name not in conns:
            conns[name] = Conn()
        return conns[name]

    reset(conn("A"))
    lines = open(path, encoding="utf-8").read().split("\n")
    i = 0
    while i < len(lines):
        raw = lines[i]
        i += 1
        line = raw.strip()
        if not line or line.startswith("#"):
            out.append(raw.rstrip())
            continue
        if line.startswith("@sleep"):
            time.sleep(float(line.split()[1]))
            out.append("# (пауза %s с)" % line.split()[1])
            continue
        if line.startswith("@lua"):
            name = line.split()[1]
            body = []
            while lines[i].strip() != "@end":
                body.append(lines[i])
                i += 1
            i += 1
            scripts[name] = "\n".join(body)
            out.append("# --- скрипт $%s (Lua) ---" % name)
            out += body
            out.append("# --- конец скрипта ---")
            continue
        client, mode = "A", ""
        if line.startswith("!"):
            mode, line = "!", line[1:].strip()
        if line.startswith("["):
            tag, line = line[1:].split("]", 1)
            line = line.strip()
            client = tag.rstrip("&?!")
            mode = tag[len(client):] or mode
        prompt = "redis> " if client == "A" else "redis[%s]> " % client
        c = conn(client)
        if mode == "?":
            out.append("(ответ клиенту %s)" % client)
            out += fmt(c.read())
            continue
        args = [scripts[a[1:]] if a.startswith("$") and a[1:] in scripts else a
                for a in shlex.split(line)]
        out.append(prompt + line)
        if mode == "&":
            c.send(args)
            out.append("(клиент %s ждёт ответа)" % client)
            continue
        reply = c.call(*args)
        if mode == "!":
            out.append("(ответ опущен: зависит от окружения)")
        else:
            out += fmt(reply)
    for c in conns.values():
        c.close()
    return "\n".join(out).rstrip("\n") + "\n"


def main():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    update = "--update" in sys.argv
    filters = [a for a in sys.argv[1:] if not a.startswith("--")]
    ok = fail = 0
    for path in sorted(glob.glob("[0-9][0-9]_*/*.redis")):
        if filters and not any(path.startswith(f) for f in filters):
            continue
        d, n = os.path.split(path)
        exp = os.path.join(d, "expected", n[:-6] + ".out")
        try:
            got = run_script(path)
        except Exception as e:                      # noqa: BLE001
            got = "СБОЙ СЦЕНАРИЯ: %r\n" % (e,)
        if update:
            os.makedirs(os.path.dirname(exp), exist_ok=True)
            open(exp, "w", encoding="utf-8").write(got)
            print("обновлён", exp)
            ok += 1
            continue
        want = open(exp, encoding="utf-8").read() if os.path.exists(exp) else ""
        if got == want:
            print("ok   ", path)
            ok += 1
        else:
            print("FAIL ", path)
            diff = difflib.unified_diff(want.splitlines(), got.splitlines(), "ожидалось", "получено", lineterm="")
            print("\n".join(list(diff)[:40]))
            fail += 1
    print("итого: %d ok, %d fail" % (ok, fail))
    sys.exit(1 if fail else 0)


if __name__ == "__main__":
    main()
