<div align="center">

```
  ██████╗  ██████╗ ██████╗  █████╗ ██████╗
 ██╔════╝ ██╔═══██╗██╔══██╗██╔══██╗██╔══██╗
 ██║  ███╗██║   ██║██████╔╝███████║██║  ██║
 ██║   ██║██║   ██║██╔═══╝ ██╔══██║██║  ██║
 ╚██████╔╝╚██████╔╝██║     ██║  ██║██████╔╝
  ╚═════╝  ╚═════╝ ╚═╝     ╚═╝  ╚═╝╚═════╝
```

**Crash-proof desktop notepad for Linux.**  
Built with Go + WebKitGTK. No Chromium. No Electron. No data loss.

![Go](https://img.shields.io/badge/Go-1.21+-00ACD7?style=flat-square&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-WAL-003B57?style=flat-square&logo=sqlite&logoColor=white)
![Platform](https://img.shields.io/badge/Linux-Arch-1793D1?style=flat-square&logo=arch-linux&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

</div>

---

## The Problem

Every native notepad app on Linux has the same silent flaw: unsaved text lives only in RAM. Suspend, kernel panic, dead battery — it's gone. GoPad treats the database as the primary store and the filesystem as an optional export target. Your keystrokes are durable before you ever think to hit Save.

---

## How It Works

```
Keystroke
    │
    ▼
900 ms debounce timer (resets on each keystroke)
    │
    ▼
SQLite WAL write  ──►  ~/.local/share/gopad/notes.db
    │
    ▼  (optional, user-triggered)
Filesystem export via zenity Save dialog
```

The UI is HTML/CSS/JS rendered by **WebKitGTK** (the same engine behind GNOME Web). Go functions are exposed to JavaScript via `webview.Bind()` — each bound call is a typed Go function invoked as an `async` JS Promise. There is no HTTP server, no IPC socket, no Electron — the UI and the backend share the same process.

---

## Features

| Category | Details |
|---|---|
| **Durability** | Every keystroke auto-saves to SQLite within 900 ms; WAL mode + 5 s busy-timeout prevents write contention |
| **Multi-note** | Unlimited notes; sidebar sorted by last-modified descending |
| **Search** | Live full-text sidebar search across title and content |
| **Find & Replace** | In-editor search with match counter (`3/17`), cycle prev/next, replace one or all |
| **File I/O** | Open any text/code file from disk; export/save back via native GTK file dialog (zenity) |
| **Line numbers** | Rendered in a synced sibling div — scroll-locked to the textarea at all times |
| **Editor** | Monospace font stack, `Tab` → 4 spaces, configurable font size (10–36 px), word wrap toggle |
| **Themes** | Dark (default) and light; selection persisted to the `settings` table |
| **Status bar** | Real-time line/column, word count, character count, linked filename |
| **No Chromium** | Rendering engine is WebKitGTK; binary depends only on system GTK/WebKit libraries |

---

## Requirements

### Runtime

| Package | Arch | Purpose |
|---|---|---|
| `webkit2gtk` | `sudo pacman -S webkit2gtk` | Rendering engine |
| `gtk3` | pulled as a dep | Window and widget toolkit |
| `zenity` | `sudo pacman -S zenity` | Native file open/save dialogs |

### Build

| Tool | Arch | Purpose |
|---|---|---|
| `gcc` | `sudo pacman -S gcc` | CGO compilation |
| `pkg-config` | `sudo pacman -S pkgconf` | Library flag resolution |
| Go ≥ 1.21 | [go.dev/dl](https://go.dev/dl/) | Compiler |

---

## Installation

### From source (recommended)

```bash
git clone <repo-url> gopad
cd gopad

# Fetch Go module dependencies
make deps

# Build
make build

# Optional: install to ~/.local/bin
make install
```

### Quick one-liner

```bash
make deps && make install
```

The `install` target copies the binary to `~/.local/bin/gopad`. Make sure that directory is in your `$PATH`.

---

## Building Manually

If you prefer not to use Make:

```bash
PKG_CONFIG_PATH="$(pwd)/.pkgconfig:/usr/lib/pkgconfig" \
  CGO_ENABLED=1 \
  go build -o gopad .
```

> **Why the custom `PKG_CONFIG_PATH`?**
> `webview_go` hardcodes the pkg-config name `webkit2gtk-4.0` in its CGO directives. Arch Linux ships `webkit2gtk-4.1`. The `.pkgconfig/webkit2gtk-4.0.pc` file in this repo is a shim that transparently redirects to the installed 4.1 library. No source patching required.

---

## Usage

```bash
./gopad
# or, after make install:
gopad
```

The app opens at 1200×800. All notes are loaded from the local database on startup; the most-recently-modified note is opened automatically.

---

## Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+N` | New note |
| `Ctrl+O` | Open file from disk |
| `Ctrl+S` | Flush pending save + export to file |
| `Ctrl+F` | Open Find & Replace bar |
| `Tab` | Insert 4 spaces (does not move focus) |
| `Enter` (in find bar) | Next match |
| `Shift+Enter` (in find bar) | Previous match |
| `Esc` | Close find bar / dismiss modal |

---

## Data & Storage

### Database location

```
~/.local/share/gopad/notes.db
```

SQLite database created automatically on first launch. Two tables:

```sql
-- Every note, whether ever saved to disk or not
CREATE TABLE notes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL DEFAULT 'Untitled',
    content    TEXT NOT NULL DEFAULT '',
    file_path  TEXT NOT NULL DEFAULT '',  -- empty if never exported
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Persisted UI preferences
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);
```

### Persisted settings keys

| Key | Values | Default |
|---|---|---|
| `theme` | `dark` \| `light` | `dark` |
| `fontSize` | `10`–`36` (px) | `14` |
| `wordWrap` | `1` \| `0` | `1` |

### Backup

The database is a single self-contained file. To back up all your notes:

```bash
cp ~/.local/share/gopad/notes.db ~/notes-backup-$(date +%Y%m%d).db
```

To inspect notes directly:

```bash
sqlite3 ~/.local/share/gopad/notes.db "SELECT title, updated_at FROM notes ORDER BY updated_at DESC;"
```

---

## Project Structure

```
gopad/
├── main.go              # Entry point — initializes DB, creates WebView, binds API
├── db.go                # SQLite schema, typed query helpers
├── api.go               # Go functions exposed to JS via webview.Bind()
├── ui/
│   └── index.html       # Complete frontend — HTML, CSS, JS; embedded at compile time
├── logo.svg             # Standalone 64 px SVG icon (for .desktop files / window managers)
├── .pkgconfig/
│   └── webkit2gtk-4.0.pc  # Shim: redirects webview_go's 4.0 lookup to the installed 4.1
├── Makefile
├── go.mod
└── go.sum
```

### Architecture

```
┌─────────────────────────────────────────────────┐
│                  gopad process                  │
│                                                 │
│  ┌──────────────┐     webview.Bind()            │
│  │  Go backend  │ ◄──────────────────────────┐  │
│  │              │                            │  │
│  │  db.go       │  JSON over IPC bridge      │  │
│  │  api.go      │ ──────────────────────────►│  │
│  │  main.go     │                            │  │
│  └──────────────┘                            │  │
│                                              │  │
│  ┌──────────────────────────────────────┐    │  │
│  │         WebKitGTK renderer           │    │  │
│  │                                      │    │  │
│  │  ui/index.html  (embedded via        │────┘  │
│  │  //go:embed at compile time)         │       │
│  │                                      │       │
│  └──────────────────────────────────────┘       │
└─────────────────────────────────────────────────┘
```

The entire frontend is embedded into the binary at compile time via `//go:embed ui/index.html`. There are no assets to ship alongside the executable.

---

## JS ↔ Go Bridge Reference

Each function is available in the browser context as a global `async` function returning `{ ok: boolean, data?: any, error?: string }`.

| JS Function | Go Handler | Description |
|---|---|---|
| `apiListNotes()` | `dbListNotes` | All notes, ordered by `updated_at DESC` |
| `apiGetNote(id)` | `dbGetNote` | Single note by ID |
| `apiCreateNote()` | `dbCreateNote` | Insert new blank note, return it |
| `apiUpdateNote(id, title, content)` | `dbUpdateNote` | Persist title + content, bump `updated_at` |
| `apiDeleteNote(id)` | `dbDeleteNote` | Hard delete |
| `apiSaveToFile(id)` | zenity → `os.WriteFile` | Export content to a user-chosen path |
| `apiOpenFile()` | zenity → `os.ReadFile` | Import file into a new note |
| `apiGetSetting(key)` | `dbGetSetting` | Read one setting value |
| `apiSetSetting(key, value)` | `dbSetSetting` | Upsert one setting value |

---

## Dependencies

| Module | Version | License | Role |
|---|---|---|---|
| [`github.com/webview/webview_go`](https://github.com/webview/webview_go) | `20240831` | MIT | WebKitGTK window + JS bridge |
| [`github.com/mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) | `v1.14.45` | MIT | SQLite driver (CGO) |

System libraries (not vendored): `libwebkit2gtk-4.1`, `libgtk-3`, `libsqlite3`.

---

## Troubleshooting

**`Package 'webkit2gtk-4.0' not found` during build**

You are on a distro that ships `webkit2gtk-4.1` (Arch, Fedora 37+, Ubuntu 22.10+). The `.pkgconfig/` shim handles this automatically when building via `make`. If building manually, ensure you include the `PKG_CONFIG_PATH` prefix shown in the [Building Manually](#building-manually) section.

**File dialogs do not appear**

`zenity` is required for file open/save dialogs. Install it with `sudo pacman -S zenity`. If `zenity` is absent, the open/save actions silently no-op — all other functionality (auto-save, note management) continues to work.

**App fails to start with `failed to init db`**

GoPad cannot create `~/.local/share/gopad/`. Check that your home directory is writable. As a fallback it will attempt to write `notes.db` in the current working directory.

**Notes from a previous session are missing**

The database persists at `~/.local/share/gopad/notes.db`. If you moved or deleted that file the notes are gone. Always back up with `cp` before doing filesystem operations in that directory.

---

## License

MIT — see [`LICENSE`](LICENSE).
