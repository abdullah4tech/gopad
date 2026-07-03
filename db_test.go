package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestArchiveFiltersNotes(t *testing.T) {
	db, err := initDB(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	defer db.Close()

	active, err := dbCreateNote(db)
	if err != nil {
		t.Fatalf("create active note: %v", err)
	}
	archived, err := dbCreateNote(db)
	if err != nil {
		t.Fatalf("create archived note: %v", err)
	}
	if err := dbSetArchived(db, archived.ID, true); err != nil {
		t.Fatalf("archive note: %v", err)
	}

	activeNotes, err := dbListNotes(db)
	if err != nil {
		t.Fatalf("list active notes: %v", err)
	}
	if len(activeNotes) != 1 || activeNotes[0].ID != active.ID || activeNotes[0].Archived {
		t.Fatalf("active notes = %#v, want only note %d", activeNotes, active.ID)
	}

	archivedNotes, err := dbListArchivedNotes(db)
	if err != nil {
		t.Fatalf("list archived notes: %v", err)
	}
	if len(archivedNotes) != 1 || archivedNotes[0].ID != archived.ID || !archivedNotes[0].Archived {
		t.Fatalf("archived notes = %#v, want only note %d", archivedNotes, archived.ID)
	}
}

func TestInitDBMigratesArchivedColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old-notes.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open old db: %v", err)
	}
	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`
		CREATE TABLE notes (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      TEXT    NOT NULL DEFAULT 'Untitled',
			content    TEXT    NOT NULL DEFAULT '',
			file_path  TEXT    NOT NULL DEFAULT '',
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL
		);
		CREATE TABLE settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE highlights (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			note_id    INTEGER NOT NULL,
			sel_text   TEXT    NOT NULL DEFAULT '',
			color      TEXT    NOT NULL DEFAULT 'yellow',
			comment    TEXT    NOT NULL DEFAULT '',
			off_start  INTEGER NOT NULL DEFAULT 0,
			created_at TEXT    NOT NULL
		);
		INSERT INTO notes (title, content, file_path, created_at, updated_at)
		VALUES ('Before archive', '', '', ?, ?);
	`, now, now); err != nil {
		db.Close()
		t.Fatalf("create old schema: %v", err)
	}
	db.Close()

	migrated, err := initDB(path)
	if err != nil {
		t.Fatalf("init migrated db: %v", err)
	}
	defer migrated.Close()

	notes, err := dbListNotes(migrated)
	if err != nil {
		t.Fatalf("list migrated notes: %v", err)
	}
	if len(notes) != 1 || notes[0].Archived {
		t.Fatalf("migrated notes = %#v, want one active note", notes)
	}
}
