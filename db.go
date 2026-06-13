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
