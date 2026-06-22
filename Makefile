# GoPad — Makefile
# Requires: gcc, pkg-config, webkit2gtk-4.1, gtk3
#   sudo pacman -S webkit2gtk-4.1 gtk3 pkg-config gcc
# Optional (live-reload dev mode): the Wails CLI
#   go install github.com/wailsapp/wails/v2/cmd/wails@latest

BINARY      := gopad
# Arch ships webkit2gtk-4.1; the webkit2_41 tag selects it over the legacy 4.0
# API. A plain `go build` (no Wails CLI) also needs the desktop+production tags,
# or the binary compiles but refuses to run.
BUILD_TAGS  := desktop,production,webkit2_41
# `wails dev` injects desktop+dev itself, so it only needs the webkit tag.
DEV_TAGS    := webkit2_41
BUILD_FLAGS := CGO_ENABLED=1

.PHONY: build run dev install deps clean

deps:
	go mod tidy

build:
	$(BUILD_FLAGS) go build -tags "$(BUILD_TAGS)" -ldflags "-s -w" -o $(BINARY) .

run: build
	./$(BINARY)

# Live-reload development (needs the Wails CLI, see above).
dev:
	wails dev -tags $(DEV_TAGS)

install: build
	install -Dm755 $(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed to ~/.local/bin/$(BINARY)"

clean:
	rm -f $(BINARY)
