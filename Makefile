# GoPad — Makefile
# Requires: gcc, pkg-config, webkit2gtk (from Arch packages below)
#   sudo pacman -S webkit2gtk gtk3 pkg-config gcc zenity

BINARY     := gopad
# Arch ships webkit2gtk-4.1; we provide a 4.0 shim in .pkgconfig/
PKG_PATH   := $(CURDIR)/.pkgconfig:/usr/lib/pkgconfig
BUILD_FLAGS := CGO_ENABLED=1 PKG_CONFIG_PATH="$(PKG_PATH)"

.PHONY: build run install deps clean

deps:
	go get github.com/webview/webview_go@latest
	go get github.com/mattn/go-sqlite3@latest
	go mod tidy

build:
	$(BUILD_FLAGS) go build -o $(BINARY) .

run: build
	./$(BINARY)

install: build
	install -Dm755 $(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed to ~/.local/bin/$(BINARY)"

clean:
	rm -f $(BINARY)
