package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	db, err := initDB(filepath.Join(t.TempDir(), "notes.db"))
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &App{db: db}
}

func lockedNote(t *testing.T, a *App, content, pass string) int64 {
	t.Helper()
	n, err := a.CreateNote()
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := a.UpdateNote(n.ID, "Secret", content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := a.LockNote(n.ID, pass); err != nil {
		t.Fatalf("lock note: %v", err)
	}
	return n.ID
}

func TestLockEncryptsContentAtRest(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	raw, err := dbGetNote(a.db, id)
	if err != nil {
		t.Fatalf("read raw note: %v", err)
	}
	if strings.Contains(raw.Content, "launch codes") {
		t.Fatalf("plaintext still on disk: %q", raw.Content)
	}
	if !isEncrypted(raw.Content) {
		t.Fatalf("content is not encrypted: %q", raw.Content)
	}
}

func TestLockedContentHiddenUntilRevealed(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	got, err := a.GetNote(id)
	if err != nil {
		t.Fatalf("get locked note: %v", err)
	}
	if got.Content != "" {
		t.Fatalf("locked note returned content %q, want empty", got.Content)
	}

	notes, err := a.ListNotes()
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 1 || notes[0].Content != "" {
		t.Fatalf("list leaked locked content: %#v", notes)
	}

	plain, err := a.RevealNote(id, "hunter2")
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if plain != "launch codes" {
		t.Fatalf("revealed %q, want %q", plain, "launch codes")
	}
	if got, _ := a.GetNote(id); got.Content != "launch codes" {
		t.Fatalf("GetNote after reveal = %q", got.Content)
	}
}

func TestRevealRejectsWrongPassphrase(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if _, err := a.RevealNote(id, "wrong"); !errors.Is(err, errBadPassphrase) {
		t.Fatalf("reveal with wrong passphrase = %v, want %v", err, errBadPassphrase)
	}
	if a.IsRevealed(id) {
		t.Fatal("failed reveal left a live session")
	}
	if got, _ := a.GetNote(id); got.Content != "" {
		t.Fatalf("content leaked after failed reveal: %q", got.Content)
	}
}

func TestRelockDropsTheKey(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if _, err := a.RevealNote(id, "hunter2"); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	a.RelockNote()

	if a.IsRevealed(id) {
		t.Fatal("session survived RelockNote")
	}
	if got, _ := a.GetNote(id); got.Content != "" {
		t.Fatalf("content readable after relock: %q", got.Content)
	}
}

func TestEditsToRevealedNoteStayEncrypted(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if _, err := a.RevealNote(id, "hunter2"); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if err := a.UpdateNote(id, "Secret", "new codes"); err != nil {
		t.Fatalf("autosave revealed note: %v", err)
	}

	raw, _ := dbGetNote(a.db, id)
	if strings.Contains(raw.Content, "new codes") {
		t.Fatalf("edit written as plaintext: %q", raw.Content)
	}
	if !raw.Locked {
		t.Fatal("note lost its lock flag on save")
	}

	a.RelockNote()
	plain, err := a.RevealNote(id, "hunter2")
	if err != nil {
		t.Fatalf("re-reveal: %v", err)
	}
	if plain != "new codes" {
		t.Fatalf("round-tripped %q, want %q", plain, "new codes")
	}
}

func TestUpdateRejectedWhileLocked(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if err := a.UpdateNote(id, "Secret", "overwrite"); !errors.Is(err, errNoteLocked) {
		t.Fatalf("update without session = %v, want %v", err, errNoteLocked)
	}
	if _, err := a.RevealNote(id, "hunter2"); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if plain, _ := a.RevealNote(id, "hunter2"); plain != "launch codes" {
		t.Fatalf("content changed despite rejected write: %q", plain)
	}
}

func TestRemoveLockNeedsAReveal(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if err := a.RemoveLock(id); !errors.Is(err, errNoteLocked) {
		t.Fatalf("RemoveLock without session = %v, want %v", err, errNoteLocked)
	}
	if _, err := a.RevealNote(id, "hunter2"); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if err := a.RemoveLock(id); err != nil {
		t.Fatalf("RemoveLock: %v", err)
	}

	got, _ := a.GetNote(id)
	if got.Locked || got.Content != "launch codes" {
		t.Fatalf("after RemoveLock = %#v, want unlocked plaintext", got)
	}
}

func TestHighlightsHiddenWhileLocked(t *testing.T) {
	a := newTestApp(t)
	id := lockedNote(t, a, "launch codes", "hunter2")

	if _, err := a.AddHighlight(id, "launch", "yellow", "", 0); !errors.Is(err, errNoteLocked) {
		t.Fatalf("AddHighlight while locked = %v, want %v", err, errNoteLocked)
	}
	if _, err := a.RevealNote(id, "hunter2"); err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if _, err := a.AddHighlight(id, "launch", "yellow", "", 0); err != nil {
		t.Fatalf("AddHighlight while revealed: %v", err)
	}

	a.RelockNote()
	hls, err := a.GetHighlights(id)
	if err != nil {
		t.Fatalf("get highlights: %v", err)
	}
	if len(hls) != 0 {
		t.Fatalf("highlights leaked while locked: %#v", hls)
	}
}

func TestSealUsesAFreshNonceEachTime(t *testing.T) {
	salt, err := newSalt()
	if err != nil {
		t.Fatalf("newSalt: %v", err)
	}
	key, err := deriveKey("hunter2", salt)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	first, err := sealContent("same text", key, salt)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	second, err := sealContent("same text", key, salt)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if first == second {
		t.Fatal("identical ciphertext for repeated seals — nonce is being reused")
	}
	for _, c := range []string{first, second} {
		plain, err := openContent(c, key)
		if err != nil || plain != "same text" {
			t.Fatalf("open = %q, %v", plain, err)
		}
	}
}

func TestLegacyFlagLockIsCleared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")
	db, err := initDB(path)
	if err != nil {
		t.Fatalf("initDB: %v", err)
	}
	n, err := dbCreateNote(db)
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	// A note locked by the old passphrase-less scheme: flag set, plaintext body.
	if err := dbSetLockedContent(db, n.ID, "old plaintext", true); err != nil {
		t.Fatalf("simulate legacy lock: %v", err)
	}
	db.Close()

	migrated, err := initDB(path)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer migrated.Close()

	got, err := dbGetNote(migrated, n.ID)
	if err != nil {
		t.Fatalf("get note: %v", err)
	}
	if got.Locked {
		t.Fatal("legacy locked note still demands a passphrase that never existed")
	}
	if got.Content != "old plaintext" {
		t.Fatalf("content = %q, want it left intact", got.Content)
	}
}

func TestLockRejectsEmptyPassphrase(t *testing.T) {
	a := newTestApp(t)
	n, err := a.CreateNote()
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if err := a.LockNote(n.ID, ""); !errors.Is(err, errEmptyPass) {
		t.Fatalf("LockNote with empty passphrase = %v, want %v", err, errEmptyPass)
	}
}
