package httpapi

import (
	json "encoding/json/v2"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	return New(NewStore(), slog.New(slog.DiscardHandler)) // DiscardHandler: Go 1.24
}

// do выполняет запрос прямо через обработчик, без сети.
func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCreateGetUpdateDelete(t *testing.T) {
	h := newTestAPI(t)

	rec := do(t, h, "POST", "/todos", `{"title": "  выучить Go  "}`)
	if rec.Code != http.StatusCreated || rec.Header().Get("Location") != "/todos/1" {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	var created Todo
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Title != "выучить Go" || created.Done {
		t.Fatalf("created = %+v", created)
	}

	rec = do(t, h, "PATCH", "/todos/1", `{"done": true}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if rec = do(t, h, "DELETE", "/todos/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec = do(t, h, "GET", "/todos/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: %d", rec.Code)
	}
}

func TestBadRequests(t *testing.T) {
	h := newTestAPI(t)
	tests := []struct {
		name, method, target, body string
		want                       int
	}{
		{"пустой title", "POST", "/todos", `{"title": " "}`, http.StatusUnprocessableEntity},
		{"неизвестное поле", "POST", "/todos", `{"title": "x", "prio": 1}`, http.StatusBadRequest},
		{"битый JSON", "POST", "/todos", `{"title":`, http.StatusBadRequest},
		{"огромное тело", "POST", "/todos", `{"title": "` + strings.Repeat("a", maxBody) + `"}`, http.StatusRequestEntityTooLarge},
		{"кривой id", "GET", "/todos/abc", "", http.StatusBadRequest},
		{"нет маршрута", "GET", "/nope", "", http.StatusNotFound},
		{"метод не разрешён", "PUT", "/todos/1", "", http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(t, h, tt.method, tt.target, tt.body); rec.Code != tt.want {
				t.Errorf("%s %s = %d, want %d (%s)", tt.method, tt.target, rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestMethodNotAllowedSetsAllow(t *testing.T) {
	rec := do(t, newTestAPI(t), "PUT", "/todos/1", "")
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "PATCH") {
		t.Errorf("Allow = %q", allow)
	}
}

func TestRecoverMiddleware(t *testing.T) {
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	h := RequestID(Recover(slog.New(slog.DiscardHandler))(boom))
	rec := do(t, h, "GET", "/", "")
	if rec.Code != http.StatusInternalServerError || rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("code=%d headers=%v", rec.Code, rec.Header())
	}
}

// Сквозной тест через настоящий HTTP на локальном порту.
func TestEndToEnd(t *testing.T) {
	srv := httptest.NewServer(newTestAPI(t))
	defer srv.Close()

	resp, err := srv.Client().Post(srv.URL+"/todos", "application/json", strings.NewReader(`{"title":"e2e"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = srv.Client().Get(srv.URL + "/todos")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"title":"e2e"`) {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
}
