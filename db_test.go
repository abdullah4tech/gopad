package main

import (
	"database/sql"
	"errors"
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

func TestLockPreventsNoteUpdates(t *testing.T) {
	db, err := initDB(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	defer db.Close()

	note, err := dbCreateNote(db)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := dbSetLockedContent(db, note.ID, "GOPADENC1:ciphertext", true); err != nil {
		t.Fatalf("lock note: %v", err)
	}
	if err := dbUpdateNote(db, note.ID, "Locked title", "Locked content"); !errors.Is(err, errNoteLocked) {
		t.Fatalf("update locked note error = %v, want %v", err, errNoteLocked)
	}
	locked, err := dbGetNote(db, note.ID)
	if err != nil {
		t.Fatalf("get locked note: %v", err)
	}
	if locked.Title == "Locked title" || locked.Content == "Locked content" {
		t.Fatalf("locked note was updated: %#v", locked)
	}

	if err := dbSetLockedContent(db, note.ID, "", false); err != nil {
		t.Fatalf("unlock note: %v", err)
	}
	if err := dbUpdateNote(db, note.ID, "Unlocked title", "Unlocked content"); err != nil {
		t.Fatalf("update unlocked note: %v", err)
	}
}

func TestInitDBMigratesNoteStateColumns(t *testing.T) {
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
	if len(notes) != 1 || notes[0].Archived || notes[0].Locked {
		t.Fatalf("migrated notes = %#v, want one active unlocked note", notes)
	}
}
