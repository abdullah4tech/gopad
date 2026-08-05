package main

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		remote, local string
		want          bool
	}{
		{"v1.3.0", "v1.2.0", true},
		{"v1.2.1", "v1.2.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.2.0", "v1.2.0", false},
		{"v1.2.0", "v1.3.0", false},
		{"v1.10.0", "v1.9.0", true}, // not string comparison
		{"v1.2", "v1.1.9", true},
		{"v1.2.0-beta", "v1.1.0", true},
		{"v1.3.0", "dev", false}, // unstamped source build never self-updates
		{"garbage", "v1.2.0", false},
	}
	for _, c := range cases {
		if got := isNewer(c.remote, c.local); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.remote, c.local, got, c.want)
		}
	}
}

func TestExpectedSumParsesChecksumFile(t *testing.T) {
	// sha256sum output uses two spaces, and "*" marks binary mode.
	body := "aaaa  gopad-linux-amd64\nbbbb  gopad-windows-amd64.exe\ncccc *gopad-linux-arm64\n"
	for _, c := range []struct{ name, want string }{
		{"gopad-linux-amd64", "aaaa"},
		{"gopad-windows-amd64.exe", "bbbb"},
		{"gopad-linux-arm64", "cccc"},
	} {
		got, ok := sumFor(body, c.name)
		if !ok || got != c.want {
			t.Errorf("sumFor(%q) = %q, %v, want %q", c.name, got, ok, c.want)
		}
	}
	if _, ok := sumFor(body, "gopad-darwin-arm64"); ok {
		t.Error("sumFor returned a hash for an asset that isn't listed")
	}
}
