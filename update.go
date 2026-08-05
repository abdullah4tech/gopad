package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	wails "github.com/wailsapp/wails/v2/pkg/runtime"
)

// version is stamped at build time with -ldflags "-X main.version=v1.2.3".
// A source build without the flag reports "dev" and never offers updates.
var version = "dev"

const (
	releaseAPI    = "https://api.github.com/repos/abdullah4tech/gopad/releases/latest"
	checkInterval = time.Hour
	firstCheck    = 20 * time.Second // let the window settle before the first poll
	checksumAsset = "SHA256SUMS"
)

// UpdateInfo is what the frontend renders in the update pill and dialog.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Body       string    `json:"body"`
	HTMLURL    string    `json:"html_url"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// assetName is the release asset this build knows how to install. The release
// workflow must publish assets under exactly these names.
func assetName() string {
	name := fmt.Sprintf("gopad-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// Version lets the About dialog show what's running.
func (a *App) Version() string { return version }

// CheckForUpdate asks GitHub for the latest release and reports whether it is
// newer than this build. Called hourly in the background and by Help → Check for
// updates.
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	rel, err := fetchLatestRelease()
	if err != nil {
		return nil, err
	}
	info := &UpdateInfo{Current: version, Version: rel.TagName, Notes: rel.Body, URL: rel.HTMLURL}
	if asset := findAsset(rel, assetName()); asset != nil {
		info.Size = asset.Size
	}
	info.Available = isNewer(rel.TagName, version) && findAsset(rel, assetName()) != nil
	return info, nil
}

// watchUpdates polls in the background and pushes a single event to the
// frontend when a newer release appears. Failures are silent: a laptop offline
// in a café should not produce error toasts.
func (a *App) watchUpdates() {
	if version == "dev" {
		return
	}
	notified := ""
	check := func() {
		info, err := a.CheckForUpdate()
		if err != nil || !info.Available || info.Version == notified {
			return
		}
		notified = info.Version
		wails.EventsEmit(a.ctx, "update:available", info)
	}

	// One goroutine owns `notified`; a separate first-check timer would race
	// with the ticker for it.
	go func() {
		time.Sleep(firstCheck)
		check()
		t := time.NewTicker(checkInterval)
		defer t.Stop()
		for range t.C {
			check()
		}
	}()
}

// DownloadAndInstall fetches the new binary, verifies it against the release's
// SHA256SUMS, and swaps it into place. The running process keeps executing the
// old image until RestartApp is called.
func (a *App) DownloadAndInstall() error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	if !isNewer(rel.TagName, version) {
		return errors.New("already up to date")
	}
	asset := findAsset(rel, assetName())
	if asset == nil {
		return fmt.Errorf("release %s has no build for %s/%s", rel.TagName, runtime.GOOS, runtime.GOARCH)
	}

	exePath, err := currentExe()
	if err != nil {
		return err
	}

	// Stage the download beside the current binary: os.Rename is only atomic
	// within a filesystem, and /tmp is often a different one.
	tmp, err := os.CreateTemp(filepath.Dir(exePath), ".gopad-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w", filepath.Dir(exePath), err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename succeeds

	sum, err := download(a, asset, tmp)
	tmp.Close()
	if err != nil {
		return err
	}

	want, err := expectedSum(rel, asset.Name)
	if err != nil {
		return err
	}
	if sum != want {
		return fmt.Errorf("checksum mismatch: refusing to install (got %s, want %s)", sum[:16], want[:16])
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}
	if err := swapBinary(tmpPath, exePath); err != nil {
		return err
	}
	wails.EventsEmit(a.ctx, "update:ready", rel.TagName)
	return nil
}

// RestartApp launches the freshly installed binary and quits this one.
func (a *App) RestartApp() error {
	exePath, err := currentExe()
	if err != nil {
		return err
	}
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	if err := cmd.Start(); err != nil {
		return err
	}
	// Don't wait on the child; let it outlive us.
	go func() {
		time.Sleep(300 * time.Millisecond)
		wails.Quit(a.ctx)
	}()
	return nil
}

// OpenReleasePage is the fallback when an in-place install isn't possible, e.g.
// a binary installed system-wide that the user can't overwrite.
func (a *App) OpenReleasePage(url string) {
	wails.BrowserOpenURL(a.ctx, url)
}

// ── internals ──────────────────────────────────────────────────────────────

func fetchLatestRelease() (*ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gopad-updater/"+version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s", resp.Status)
	}
	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.Draft || rel.Prerelease {
		return nil, errors.New("latest release is not a stable build")
	}
	return &rel, nil
}

func findAsset(rel *ghRelease, name string) *ghAsset {
	for i := range rel.Assets {
		if rel.Assets[i].Name == name {
			return &rel.Assets[i]
		}
	}
	return nil
}

// download streams the asset to dst, hashing as it goes and emitting progress.
func download(a *App, asset *ghAsset, dst io.Writer) (string, error) {
	req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "gopad-updater/"+version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	h := sha256.New()
	total := asset.Size
	if total <= 0 {
		total = resp.ContentLength
	}
	var done int64
	last := -1
	buf := make([]byte, 64<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return "", werr
			}
			h.Write(buf[:n])
			done += int64(n)
			if total > 0 {
				if pct := int(done * 100 / total); pct != last {
					last = pct
					wails.EventsEmit(a.ctx, "update:progress", pct)
				}
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return "", rerr
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// expectedSum pulls the hash for name out of the release's SHA256SUMS asset.
func expectedSum(rel *ghRelease, name string) (string, error) {
	sums := findAsset(rel, checksumAsset)
	if sums == nil {
		return "", errors.New("release has no SHA256SUMS; refusing to install unverified binary")
	}
	resp, err := httpClient.Get(sums.URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching SHA256SUMS: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	sum, ok := sumFor(string(body), name)
	if !ok {
		return "", fmt.Errorf("no checksum listed for %s", name)
	}
	return sum, nil
}

// sumFor reads one entry out of sha256sum-style output ("<hash>  <name>", with
// an optional "*" binary-mode marker on the name).
func sumFor(body, name string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

// currentExe resolves symlinks so an update replaces the real binary rather
// than the link pointing at it.
func currentExe() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// swapBinary moves the staged download over the running executable. Windows
// won't let a running image be replaced, so the old one is renamed aside first
// and cleaned up on the next launch.
func swapBinary(staged, exePath string) error {
	if runtime.GOOS != "windows" {
		if err := os.Rename(staged, exePath); err != nil {
			return fmt.Errorf("cannot replace %s: %w", exePath, err)
		}
		return nil
	}

	old := exePath + ".old"
	os.Remove(old)
	if err := os.Rename(exePath, old); err != nil {
		return fmt.Errorf("cannot move the running binary aside: %w", err)
	}
	if err := os.Rename(staged, exePath); err != nil {
		os.Rename(old, exePath) // put things back the way we found them
		return fmt.Errorf("cannot install the new binary: %w", err)
	}
	return nil
}

// cleanupOldBinary removes the previous Windows executable left behind by an
// update. Called at startup, once the new image is the one running.
func cleanupOldBinary() {
	if runtime.GOOS != "windows" {
		return
	}
	if exe, err := currentExe(); err == nil {
		os.Remove(exe + ".old")
	}
}

// isNewer compares dotted version numbers, ignoring a leading "v" and any
// pre-release suffix. Unparsable versions (notably "dev") are never upgraded.
func isNewer(remote, local string) bool {
	r, ok := parseVersion(remote)
	if !ok {
		return false
	}
	l, ok := parseVersion(local)
	if !ok {
		return false
	}
	for i := 0; i < 3; i++ {
		if r[i] != l[i] {
			return r[i] > l[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
