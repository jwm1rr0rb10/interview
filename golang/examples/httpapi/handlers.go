package httpapi

import (
	json "encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// API держит зависимости обработчиков: никаких глобальных переменных.
type API struct {
	store *Store
	log   *slog.Logger
}

// New собирает обработчик: маршруты плюс middleware.
func New(store *Store, log *slog.Logger) http.Handler {
	a := &API{store: store, log: log}
	mux := http.NewServeMux()
	// Шаблоны Go 1.22: метод, путь и параметры {id}. GET отвечает и на HEAD.
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /todos", a.list)
	mux.HandleFunc("POST /todos", a.create)
	mux.HandleFunc("GET /todos/{id}", a.get)
	mux.HandleFunc("PATCH /todos/{id}", a.update)
	mux.HandleFunc("DELETE /todos/{id}", a.remove)

	// Порядок: внешний middleware выполняется первым.
	return RequestID(Logging(log)(Recover(log)(mux)))
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.store.List())
}

type createRequest struct {
	Title string `json:"title"`
}

func (a *API) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	title, err := validateTitle(req.Title)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	t := a.store.Create(title)
	w.Header().Set("Location", fmt.Sprintf("/todos/%d", t.ID))
	writeJSON(w, http.StatusCreated, t)
}

func (a *API) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	t, err := a.store.Get(id)
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// updateRequest: указатели отличают «поле не передано» от нулевого значения.
type updateRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}

func (a *API) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req updateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var title string
	if req.Title != nil {
		var err error
		if title, err = validateTitle(*req.Title); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	t, err := a.store.Update(id, func(t *Todo) {
		if req.Title != nil {
			t.Title = title
		}
		if req.Done != nil {
			t.Done = *req.Done
		}
	})
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *API) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.store.Delete(id); err != nil {
		a.storeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "todo not found")
		return
	}
	a.log.Error("store failure", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error") // детали — в лог, не клиенту
}

func validateTitle(s string) (string, error) {
	s = strings.TrimSpace(s)
	switch n := utf8.RuneCountInString(s); {
	case n == 0:
		return "", errors.New("title is required")
	case n > 200:
		return "", errors.New("title is longer than 200 characters")
	}
	return s, nil
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

const maxBody = 1 << 20 // 1 МБ: без лимита клиент может прислать гигабайт

// decodeJSON читает тело строго: лимит размера, неизвестные поля — ошибка.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	err := json.UnmarshalRead(r.Body, dst, json.RejectUnknownMembers(true))
	if err == nil {
		return true
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return false
	}
	writeError(w, http.StatusBadRequest, "invalid JSON body")
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status) // заголовки — до первой записи тела
	_ = json.MarshalWrite(w, v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
