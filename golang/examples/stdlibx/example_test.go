// Пакет stdlibx — примеры time, encoding/json (v1 и v2) и uuid.
package stdlibx

import (
	jsonv1 "encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"strings"
	"time"
	"uuid"
)

// Формат времени задаётся эталонной датой Mon Jan 2 15:04:05 MST 2006 (1 2 3 4 5 6 7).
func Example_timeFormat() {
	msk := time.FixedZone("MSK", 3*60*60)
	t := time.Date(2026, time.November, 6, 14, 30, 0, 0, msk)
	fmt.Println(t.Format("2006-01-02 15:04"), t.Format(time.RFC3339))
	fmt.Println(t.UTC().Format(time.DateTime))

	p, err := time.Parse(time.DateOnly, "2026-12-31")
	fmt.Println(p.Weekday(), err)
	// Output:
	// 2026-11-06 14:30 2026-11-06T14:30:00+03:00
	// 2026-11-06 11:30:00
	// Thursday <nil>
}

func Example_durations() {
	d := 90 * time.Minute
	fmt.Println(d, d.Hours(), d.Truncate(time.Hour))
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0) // плюс месяц: с учётом календаря
	fmt.Println(end.Sub(start))
	t1 := start.Add(time.Hour)
	fmt.Println(t1.After(start), t1.Equal(start.Add(60*time.Minute)))
	// Output:
	// 1h30m0s 1.5 1h0m0s
	// 744h0m0s
	// true true
}

type Order struct {
	ID    int       `json:"id"`
	Items []string  `json:"items"`
	Note  string    `json:"note,omitempty"`
	Paid  time.Time `json:"paid_at,omitzero"` // omitzero (Go 1.24): пропустить нулевое время
}

// v1 и v2 по-разному кодируют nil-срез.
func Example_jsonNilSlice() {
	o := Order{ID: 1}
	b1, _ := jsonv1.Marshal(o)
	b2, _ := jsonv2.Marshal(o)
	fmt.Println(string(b1))
	fmt.Println(string(b2))
	// Output:
	// {"id":1,"items":null}
	// {"id":1,"items":[]}
}

// v1 сопоставляет имена без учёта регистра, v2 — точно.
func Example_jsonCase() {
	var a, b Order
	_ = jsonv1.Unmarshal([]byte(`{"ID": 7}`), &a)
	_ = jsonv2.Unmarshal([]byte(`{"ID": 7}`), &b)
	fmt.Println(a.ID, b.ID)
	// Output: 7 0
}

// v2 строже: дубликаты имён — ошибка; неизвестные поля можно запретить опцией.
func Example_jsonStrict() {
	dup := []byte(`{"id": 1, "id": 2}`)
	var a, b Order
	errV1 := jsonv1.Unmarshal(dup, &a)
	errV2 := jsonv2.Unmarshal(dup, &b)
	fmt.Println(a.ID, errV1 == nil, errV2 != nil)

	unknown := []byte(`{"id": 1, "colour": "red"}`)
	var c Order
	err := jsonv2.Unmarshal(unknown, &c, jsonv2.RejectUnknownMembers(true))
	fmt.Println(err != nil, strings.Contains(err.Error(), "colour"))
	// Output:
	// 2 true true
	// true true
}

// Пакет uuid появился в стандартной библиотеке в Go 1.27.
func Example_uuid() {
	id := uuid.NewV7() // v7: в начале метка времени — удобно для индексов БД
	time.Sleep(2 * time.Millisecond)
	later := uuid.NewV7()
	fmt.Println(len(id.String()), id.Compare(later)) // v7 упорядочены по времени

	_, err := uuid.Parse("не-uuid")
	fmt.Println(err != nil, uuid.Nil())
	// Output:
	// 36 -1
	// true 00000000-0000-0000-0000-000000000000
}
