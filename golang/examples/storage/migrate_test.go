package storage

import (
	"slices"
	"testing"
	"testing/fstest"
)

func TestPending(t *testing.T) {
	fsys := fstest.MapFS{ // файловая система в памяти
		"migrations/002_b.sql": {Data: []byte("SELECT 2")},
		"migrations/001_a.sql": {Data: []byte("SELECT 1")},
		"migrations/003_c.sql": {Data: []byte("SELECT 3")},
		"migrations/readme.md": {Data: []byte("не миграция")},
	}
	got, err := Pending(fsys, map[string]bool{"001_a.sql": true})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"migrations/002_b.sql", "migrations/003_c.sql"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestEmbeddedMigrations(t *testing.T) {
	got, err := Pending(migrationsFS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "migrations/001_create_tasks.sql" {
		t.Fatalf("встроенные миграции: %v", got)
	}
}
