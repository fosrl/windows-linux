.PHONY: build build-debug build-linux package-linux package-linux-tarball check-release-version install-linux uninstall-linux clean rsrc frontend bindings test help

# Variables
BINARY_NAME=Pangolin
MANIFEST=pangolin.manifest
BUILD_DIR=build
RSRC_SYSO=rsrc.syso
FRONTEND_DIR=ui/frontend
GOOS=windows
GOARCH=amd64
LINUX_BINARY_NAME=pangolin-client
LINUX_GOARCH ?= $(shell go env GOHOSTARCH)
PREFIX ?= /usr/local
LINUX_PKG_DIR=packaging/linux
GNOME_EXTENSION_UUID=pangolin@pangolin.net
VERSION ?= dev
NFPM ?= nfpm
OLM_VERSION := $(shell go list -m -f '{{.Version}}' github.com/fosrl/olm | sed 's/^v//')
VERSION_LDFLAGS = -X github.com/fosrl/windows-linux/version.Number=$(VERSION) -X github.com/fosrl/windows-linux/version.OlmVersionOverride=$(OLM_VERSION)

# Default target
all: clean rsrc build

# Build the Windows executable (GUI mode - no console).
# The "production" tag turns off Wails' dev server and devtools.
build: rsrc frontend
	@echo "Building Windows executable (GUI mode)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -tags production -ldflags="-s -w -H windowsgui $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME).exe
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"

# Same as build, but keeps Wails devtools (right-click > Inspect) and debug logging
build-debug: rsrc frontend
	@mkdir -p $(BUILD_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-H windowsgui $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME).exe
	@echo "Debug build complete: $(BUILD_DIR)/$(BINARY_NAME).exe"

# Build the Linux binary. Needs cgo, gtk3 and webkit2gtk-4.1: the UI uses
# Wails' GTK3 stack (the gtk3 tag) because GTK4 can't position the tray popup.
build-linux: frontend
	@echo "Building Linux binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=$(LINUX_GOARCH) go build -tags "gtk3 production" -ldflags="-s -w $(VERSION_LDFLAGS)" -o $(BUILD_DIR)/$(LINUX_BINARY_NAME)
	@echo "Build complete: $(BUILD_DIR)/$(LINUX_BINARY_NAME)"

# Build the deb, rpm and Arch packages from the Linux binary with nfpm
# (https://nfpm.goreleaser.com), as build/pangolin-client-<arch>-<version>.*,
# and the tar.gz below. The in-app updater looks for the package names in the
# release manifest.
package-linux: check-release-version build-linux package-linux-tarball
	@for fmt in deb rpm archlinux; do \
		case $$fmt in deb) ext=deb;; rpm) ext=rpm;; archlinux) ext=pkg.tar.zst;; esac; \
		out=$(BUILD_DIR)/$(LINUX_BINARY_NAME)-$(LINUX_GOARCH)-$(VERSION).$$ext; \
		VERSION=$(VERSION) GOARCH=$(LINUX_GOARCH) BINARY=$(BUILD_DIR)/$(LINUX_BINARY_NAME) \
			$(NFPM) pkg -f $(LINUX_PKG_DIR)/nfpm.yaml -p $$fmt -t $$out || exit 1; \
	done

# Build build/pangolin-client-<arch>-<version>.tar.gz for distributions with
# no package (Gentoo, Void, ...): the binary and everything the packages
# install, laid out as in the packages, plus the OpenRC service and an
# install.sh that puts them in place (packaging/linux/tarball/).
package-linux-tarball: check-release-version build-linux
	@name=$(LINUX_BINARY_NAME)-$(LINUX_GOARCH)-$(VERSION); \
	root=$(BUILD_DIR)/tarball; dir=$$root/$$name; \
	rm -rf $$root && \
	install -Dm755 $(BUILD_DIR)/$(LINUX_BINARY_NAME) $$dir/bin/$(LINUX_BINARY_NAME) && \
	install -Dm755 -t $$dir $(LINUX_PKG_DIR)/tarball/install.sh && \
	install -Dm644 -t $$dir $(LINUX_PKG_DIR)/tarball/README.md LICENSE && \
	install -Dm644 -t $$dir/share/applications $(LINUX_PKG_DIR)/pangolin-client.desktop && \
	install -Dm644 -t $$dir/share/icons/hicolor/256x256/apps $(LINUX_PKG_DIR)/pangolin-client.png && \
	install -Dm644 -t $$dir/share/gnome-shell/extensions/$(GNOME_EXTENSION_UUID) $(LINUX_PKG_DIR)/gnome-shell-extension/$(GNOME_EXTENSION_UUID)/* && \
	install -Dm644 -t $$dir/share/polkit-1/actions $(LINUX_PKG_DIR)/net.pangolin.client.policy && \
	install -Dm644 -t $$dir/etc/xdg/autostart $(LINUX_PKG_DIR)/pangolin-client-autostart.desktop && \
	install -Dm644 -t $$dir/lib/systemd/system $(LINUX_PKG_DIR)/pangolin-client.service && \
	install -Dm755 -t $$dir/etc/init.d $(LINUX_PKG_DIR)/openrc/pangolin-client && \
	tar -C $$root --sort=name --owner=0 --group=0 --numeric-owner -czf $(BUILD_DIR)/$$name.tar.gz $$name && \
	rm -rf $$root && \
	echo "Archive: $(BUILD_DIR)/$$name.tar.gz"

# Release packages need a real X.Y.Z version, not "dev".
check-release-version:
	@if [ "$(VERSION)" = "dev" ]; then echo "Set a release version, e.g. make $(MAKECMDGOALS) VERSION=X.Y.Z" >&2; exit 1; fi

# Install a Linux build for testing: the binary, the systemd service, the app
# menu and autostart entries, and the GNOME Shell extension (enable it with
# "gnome-extensions enable pangolin@pangolin.net" after logging back in).
# Packages (deb, rpm, AUR) replace this.
# Run "make build-linux" first, then "sudo make install-linux" and
# "sudo systemctl enable --now pangolin-client".
install-linux:
	install -Dm755 $(BUILD_DIR)/$(LINUX_BINARY_NAME) $(DESTDIR)$(PREFIX)/bin/$(LINUX_BINARY_NAME)
	install -d $(DESTDIR)/etc/systemd/system
	sed 's|/usr/bin/$(LINUX_BINARY_NAME)|$(PREFIX)/bin/$(LINUX_BINARY_NAME)|' $(LINUX_PKG_DIR)/pangolin-client.service > $(DESTDIR)/etc/systemd/system/pangolin-client.service
	install -Dm644 $(LINUX_PKG_DIR)/pangolin-client.desktop $(DESTDIR)$(PREFIX)/share/applications/pangolin-client.desktop
	install -Dm644 $(LINUX_PKG_DIR)/pangolin-client-autostart.desktop $(DESTDIR)/etc/xdg/autostart/pangolin-client-autostart.desktop
	install -Dm644 $(LINUX_PKG_DIR)/pangolin-client.png $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/pangolin-client.png
	install -Dm644 $(LINUX_PKG_DIR)/net.pangolin.client.policy $(DESTDIR)/usr/share/polkit-1/actions/net.pangolin.client.policy
	install -Dm644 -t $(DESTDIR)$(PREFIX)/share/gnome-shell/extensions/$(GNOME_EXTENSION_UUID) $(LINUX_PKG_DIR)/gnome-shell-extension/$(GNOME_EXTENSION_UUID)/*
	-systemctl daemon-reload

uninstall-linux:
	-systemctl disable --now pangolin-client
	rm -f $(DESTDIR)$(PREFIX)/bin/$(LINUX_BINARY_NAME) $(DESTDIR)/etc/systemd/system/pangolin-client.service
	rm -f $(DESTDIR)$(PREFIX)/share/applications/pangolin-client.desktop $(DESTDIR)/etc/xdg/autostart/pangolin-client-autostart.desktop
	rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/256x256/apps/pangolin-client.png
	rm -f $(DESTDIR)/usr/share/polkit-1/actions/net.pangolin.client.policy
	rm -rf $(DESTDIR)$(PREFIX)/share/gnome-shell/extensions/$(GNOME_EXTENSION_UUID)
	-systemctl daemon-reload

# Build the webview frontend into ui/frontend/dist (embedded into the exe)
frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm ci && npm run build

# Regenerate the TypeScript bindings for the Go services in ui/
bindings:
	GOOS=$(GOOS) GOARCH=$(GOARCH) wails3 generate bindings -ts -i -d $(FRONTEND_DIR)/src/bindings .

# Run the tests for this platform (the gtk3 tag only matters on Linux)
test:
	go test -tags gtk3 ./...

# Compile the manifest and icon into a .syso. Wails loads the app icon for
# message boxes and toast notifications from icon group resource ID 3, which
# is where its own syso generator puts it (akavel/rsrc used ID 2).
rsrc:
	@echo "Compiling manifest and icon..."
	@go run github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0 generate syso -manifest $(MANIFEST) -icon icons/icon-orange.ico -arch $(GOARCH) -out $(RSRC_SYSO)
	@echo "Resources compiled: $(RSRC_SYSO)"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(RSRC_SYSO)
	@echo "Clean complete"

# Show help
help:
	@echo "Pass VERSION=X.Y.Z to stamp a release version (default: dev)"
	@echo "Available targets:"
	@echo "  make build       - Build the Windows executable to build/ (GUI mode, no console)"
	@echo "  make build-debug - Build with Wails devtools enabled"
	@echo "  make build-linux - Build the Linux binary to build/ (needs gtk3, webkit2gtk-4.1)"
	@echo "  make package-linux - Build the deb, rpm and Arch packages with nfpm, and the tar.gz"
	@echo "  make package-linux-tarball - Build only the tar.gz for distributions without a package"
	@echo "  make install-linux - Install the Linux build and its systemd service (run as root)"
	@echo "  make uninstall-linux - Remove what install-linux installed (run as root)"
	@echo "  make frontend    - Build the webview frontend"
	@echo "  make bindings    - Regenerate TypeScript bindings for the Go services"
	@echo "  make test        - Run unit tests"
	@echo "  make rsrc        - Compile the manifest and icon"
	@echo "  make clean       - Remove build/ directory"
	@echo "  make help        - Show this help message"
