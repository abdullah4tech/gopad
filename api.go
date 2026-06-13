package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	webview "github.com/webview/webview_go"
)

type API struct {
	db *sql.DB
	w  webview.WebView
}

type Resp struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func okResp(data interface{}) Resp { return Resp{OK: true, Data: data} }
func errResp(err error) Resp       { return Resp{OK: false, Error: err.Error()} }

func (a *API) bind() {
	a.w.Bind("apiListNotes", func() Resp {
		notes, err := dbListNotes(a.db)
		if err != nil {
			return errResp(err)
		}
		return okResp(notes)
	})

	a.w.Bind("apiGetNote", func(id int64) Resp {
		note, err := dbGetNote(a.db, id)
		if err != nil {
			return errResp(err)
		}
		return okResp(note)
	})

	a.w.Bind("apiCreateNote", func() Resp {
		note, err := dbCreateNote(a.db)
		if err != nil {
			return errResp(err)
		}
		return okResp(note)
	})

	a.w.Bind("apiUpdateNote", func(id int64, title, content string) Resp {
		if err := dbUpdateNote(a.db, id, title, content); err != nil {
			return errResp(err)
		}
		return okResp(nil)
	})

	a.w.Bind("apiDeleteNote", func(id int64) Resp {
		if err := dbDeleteNote(a.db, id); err != nil {
			return errResp(err)
		}
		return okResp(nil)
	})

	a.w.Bind("apiSaveToFile", func(id int64) Resp {
		note, err := dbGetNote(a.db, id)
		if err != nil {
			return errResp(err)
		}

		// If already linked to a file, default to that path
		startName := note.FilePath
		if startName == "" {
			startName = note.Title
			if !strings.HasSuffix(startName, ".txt") && !strings.HasSuffix(startName, ".md") {
				startName += ".txt"
			}
		}

		path, err := zenityFileSave(startName)
		if err != nil || path == "" {
			return okResp(nil) // user cancelled
		}

		if err := os.WriteFile(path, []byte(note.Content), 0644); err != nil {
			return errResp(err)
		}
		if err := dbSetFilePath(a.db, id, path); err != nil {
			return errResp(err)
		}
		return okResp(path)
	})

	a.w.Bind("apiOpenFile", func() Resp {
		path, err := zenityFileOpen()
		if err != nil || path == "" {
			return okResp(nil)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return errResp(err)
		}

		note, err := dbCreateNote(a.db)
		if err != nil {
			return errResp(err)
		}

		title := filepath.Base(path)
		content := string(data)

		if err := dbUpdateNote(a.db, note.ID, title, content); err != nil {
			return errResp(err)
		}
		if err := dbSetFilePath(a.db, note.ID, path); err != nil {
			return errResp(err)
		}

		note.Title = title
		note.Content = content
		note.FilePath = path
		return okResp(note)
	})

	a.w.Bind("apiGetSetting", func(key string) Resp {
		val, err := dbGetSetting(a.db, key)
		if err != nil {
			return errResp(err)
		}
		return okResp(val)
	})

	a.w.Bind("apiSetSetting", func(key, value string) Resp {
		if err := dbSetSetting(a.db, key, value); err != nil {
			return errResp(err)
		}
		return okResp(nil)
	})
}

func zenityFileOpen() (string, error) {
	out, err := exec.Command("zenity",
		"--file-selection",
		"--title=Open File",
		"--file-filter=Text & Code | *.txt *.md *.log *.csv *.json *.yaml *.toml *.go *.py *.js *.ts *.html *.css *.sh *.rs *.c *.cpp *.h",
		"--file-filter=All files | *",
	).Output()
	if err != nil {
		return "", nil // user cancelled or zenity not available
	}
	return strings.TrimSpace(string(out)), nil
}

func zenityFileSave(defaultName string) (string, error) {
	out, err := exec.Command("zenity",
		"--file-selection",
		"--save",
		"--confirm-overwrite",
		"--title=Save File",
		"--filename="+defaultName,
	).Output()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}
