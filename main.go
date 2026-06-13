package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	webview "github.com/webview/webview_go"
)

//go:embed ui/index.html
var indexHTML string

func main() {
	dbPath := getDBPath()
	db, err := initDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	w := webview.New(true)
	defer w.Destroy()
	w.SetTitle("GoPad")
	w.SetSize(1200, 800, webview.HintNone)

	api := &API{db: db, w: w}
	api.bind()

	w.SetHtml(indexHTML)
	w.Run()
}

func getDBPath() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "gopad")
	if err := os.MkdirAll(dir, 0755); err != nil {
		dir = "."
	}
	return filepath.Join(dir, "notes.db")
}
