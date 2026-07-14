package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Note struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	FilePath  string `json:"file_path"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Highlight struct {
	ID        int64  `json:"id"`
	NoteID    int64  `json:"note_id"`
	SelText   string `json:"sel_text"`
	Color     string `json:"color"`
	Comment   string `json:"comment"`
	OffStart  int64  `json:"off_start"`
	CreatedAt string `json:"created_at"`
}

func initDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      TEXT    NOT NULL DEFAULT 'Untitled',
			content    TEXT    NOT NULL DEFAULT '',
			file_path  TEXT    NOT NULL DEFAULT '',
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL
		);
		CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS highlights (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			note_id    INTEGER NOT NULL,
			sel_text   TEXT    NOT NULL DEFAULT '',
			color      TEXT    NOT NULL DEFAULT 'yellow',
			comment    TEXT    NOT NULL DEFAULT '',
			off_start  INTEGER NOT NULL DEFAULT 0,
			created_at TEXT    NOT NULL
		);
	`)
	return db, err
}

func dbListNotes(db *sql.DB) ([]Note, error) {
	rows, err := db.Query(
		`SELECT id, title, content, file_path, created_at, updated_at
		 FROM notes ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.FilePath, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

func dbGetNote(db *sql.DB, id int64) (*Note, error) {
	var n Note
	err := db.QueryRow(
		`SELECT id, title, content, file_path, created_at, updated_at FROM notes WHERE id = ?`, id,
	).Scan(&n.ID, &n.Title, &n.Content, &n.FilePath, &n.CreatedAt, &n.UpdatedAt)
	return &n, err
}

func dbCreateNote(db *sql.DB) (*Note, error) {
	now := time.Now().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO notes (title, content, file_path, created_at, updated_at) VALUES ('Untitled', '', '', ?, ?)`,
		now, now,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return dbGetNote(db, id)
}

func dbUpdateNote(db *sql.DB, id int64, title, content string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(
		`UPDATE notes SET title = ?, content = ?, updated_at = ? WHERE id = ?`,
		title, content, now, id,
	)
	return err
}

func dbSetFilePath(db *sql.DB, id int64, filePath string) error {
	_, err := db.Exec(`UPDATE notes SET file_path = ? WHERE id = ?`, filePath, id)
	return err
}

func dbDeleteNote(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM notes WHERE id = ?`, id)
	return err
}

func dbGetHighlights(db *sql.DB, noteId int64) ([]Highlight, error) {
	rows, err := db.Query(
		`SELECT id, note_id, sel_text, color, comment, off_start, created_at
		 FROM highlights WHERE note_id = ? ORDER BY off_start ASC`, noteId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hl []Highlight
	for rows.Next() {
		var h Highlight
		if err := rows.Scan(&h.ID, &h.NoteID, &h.SelText, &h.Color, &h.Comment, &h.OffStart, &h.CreatedAt); err != nil {
			return nil, err
		}
		hl = append(hl, h)
	}
	if hl == nil {
		hl = []Highlight{}
	}
	return hl, nil
}

func dbAddHighlight(db *sql.DB, noteId int64, selText, color, comment string, offStart int64) (*Highlight, error) {
	now := time.Now().Format(time.RFC3339)
	res, err := db.Exec(
		`INSERT INTO highlights (note_id, sel_text, color, comment, off_start, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		noteId, selText, color, comment, offStart, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	var h Highlight
	err = db.QueryRow(
		`SELECT id, note_id, sel_text, color, comment, off_start, created_at FROM highlights WHERE id = ?`, id,
	).Scan(&h.ID, &h.NoteID, &h.SelText, &h.Color, &h.Comment, &h.OffStart, &h.CreatedAt)
	return &h, err
}

func dbDeleteHighlight(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM highlights WHERE id = ?`, id)
	return err
}

func dbGetSetting(db *sql.DB, key string) (string, error) {
	var val string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

func dbSetSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}
