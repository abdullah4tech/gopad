package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails-bound backend. Every exported method becomes callable from
// the frontend as window.go.main.App.<Method>(), returning a Promise that
// resolves with the return value or rejects with the returned error.
type App struct {
	ctx context.Context
	db  *sql.DB
}

func NewApp() *App { return &App{} }

// startup opens the database. Wails calls it once the runtime is ready, so the
// context is valid for native dialogs and events.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	db, err := initDB(getDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "gopad: failed to open database: %v\n", err)
		os.Exit(1)
	}
	a.db = db
}

func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		a.db.Close()
	}
}

func getDBPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "gopad")
	if err := os.MkdirAll(dir, 0755); err != nil {
		dir = "."
	}
	return filepath.Join(dir, "notes.db")
}

// ── Notes ──────────────────────────────────────────────────────────────────

func (a *App) ListNotes() ([]Note, error) {
	return dbListNotes(a.db)
}

func (a *App) GetNote(id int64) (*Note, error) {
	return dbGetNote(a.db, id)
}

func (a *App) CreateNote() (*Note, error) {
	return dbCreateNote(a.db)
}

func (a *App) UpdateNote(id int64, title, content string) error {
	return dbUpdateNote(a.db, id, title, content)
}

func (a *App) DeleteNote(id int64) error {
	return dbDeleteNote(a.db, id)
}

// ── Settings ───────────────────────────────────────────────────────────────

func (a *App) GetSetting(key string) (string, error) {
	return dbGetSetting(a.db, key)
}

func (a *App) SetSetting(key, value string) error {
	return dbSetSetting(a.db, key, value)
}

// ── File import / export (native Wails dialogs) ────────────────────────────

// SaveToFile exports the note's content to a user-chosen path and remembers it.
// Returns the chosen path, or "" if the user cancelled.
func (a *App) SaveToFile(id int64) (string, error) {
	note, err := dbGetNote(a.db, id)
	if err != nil {
		return "", err
	}

	startName := note.FilePath
	if startName == "" {
		startName = note.Title
		if !strings.HasSuffix(startName, ".txt") && !strings.HasSuffix(startName, ".md") {
			startName += ".txt"
		}
	}

	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                "Save File",
		DefaultFilename:      filepath.Base(startName),
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{DisplayName: "Text & Code", Pattern: "*.txt;*.md;*.log;*.csv;*.json;*.yaml;*.toml;*.go;*.py;*.js;*.ts;*.html;*.css;*.sh;*.rs;*.c;*.cpp;*.h"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil || path == "" {
		return "", err // empty path == cancelled
	}

	if err := os.WriteFile(path, []byte(note.Content), 0644); err != nil {
		return "", err
	}
	if err := dbSetFilePath(a.db, id, path); err != nil {
		return "", err
	}
	return path, nil
}

// OpenFile imports a file from disk as a new note. Returns the created note, or
// nil if the user cancelled.
func (a *App) OpenFile() (*Note, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text & Code", Pattern: "*.txt;*.md;*.log;*.csv;*.json;*.yaml;*.toml;*.go;*.py;*.js;*.ts;*.html;*.css;*.sh;*.rs;*.c;*.cpp;*.h"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	note, err := dbCreateNote(a.db)
	if err != nil {
		return nil, err
	}

	title := filepath.Base(path)
	content := string(data)
	if err := dbUpdateNote(a.db, note.ID, title, content); err != nil {
		return nil, err
	}
	if err := dbSetFilePath(a.db, note.ID, path); err != nil {
		return nil, err
	}

	note.Title = title
	note.Content = content
	note.FilePath = path
	return note, nil
}
