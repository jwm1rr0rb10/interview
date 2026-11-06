// Пакет httpapi — REST API задач на net/http без фреймворков.
package httpapi

import (
	"cmp"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

var ErrNotFound = errors.New("todo not found")

type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// Store — хранилище в памяти. В модуле 14 его заменит база данных
// с тем же набором методов.
type Store struct {
	mu     sync.Mutex
	nextID int64
	todos  map[int64]Todo
}

func NewStore() *Store { return &Store{todos: make(map[int64]Todo)} }

func (s *Store) Create(title string) Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	t := Todo{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC()}
	s.todos[t.ID] = t
	return t
}

func (s *Store) Get(id int64) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return Todo{}, ErrNotFound
	}
	return t, nil
}

func (s *Store) List() []Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := slices.Collect(maps.Values(s.todos))
	slices.SortFunc(list, func(a, b Todo) int { return cmp.Compare(a.ID, b.ID) })
	return list
}

// Update применяет изменение под блокировкой: прочитать-изменить-записать атомарно.
func (s *Store) Update(id int64, apply func(*Todo)) (Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.todos[id]
	if !ok {
		return Todo{}, ErrNotFound
	}
	apply(&t)
	s.todos[id] = t
	return t, nil
}

func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.todos[id]; !ok {
		return ErrNotFound
	}
	delete(s.todos, id)
	return nil
}
