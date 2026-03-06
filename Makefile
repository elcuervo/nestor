# Linux: no expert bundle in stable releases; extract from full browser tarball
# macOS/Windows: expert bundles last available in 13.5.29 (as .tar.gz)
TOR_LINUX_VERSION  ?= 15.0.7
TOR_BUNDLE_VERSION ?= 13.5.29
TOR_BASE_URL = https://dist.torproject.org/torbrowser
LDFLAGS = -ldflags="-s -w" -trimpath

.PHONY: all format lint test build build-all clean update download-tor \
        download-tor-linux-amd64 download-tor-darwin-amd64 \
        download-tor-darwin-arm64 download-tor-windows-amd64

all: test build-all

update:
	@go get -u
	@go mod tidy

format:
	@echo "==> Formatting project ..."
	@goimports -w .

lint:
	@echo "==> Linting project ..."
	@staticcheck ./...

build:
	@echo "==> Building ..."
	@go build $(LDFLAGS) -o bin/nestor .

clean:
	@rm -rf bin/

test:
	@echo "==> Testing nestor ..."
	go test ./...

# Download Tor binaries and compress them for embedding.
# Phony aliases — actual work is in the file targets below.

download-tor-linux-amd64:   tor_binaries/linux-amd64/tor.xz
download-tor-darwin-amd64:  tor_binaries/darwin-amd64/tor.xz
download-tor-darwin-arm64:  tor_binaries/darwin-arm64/tor.xz
download-tor-windows-amd64: tor_binaries/windows-amd64/tor.exe.xz

download-tor: download-tor-linux-amd64 download-tor-darwin-amd64 \
              download-tor-darwin-arm64 download-tor-windows-amd64

# Linux: tor lives inside the full browser tarball (no expert bundle in stable releases)
tor_binaries/linux-amd64/tor.xz:
	@mkdir -p tor_binaries/linux-amd64
	@echo "==> Downloading Tor $(TOR_LINUX_VERSION) for linux/amd64 ..."
	@tmpdir=$$(mktemp -d) && \
		curl -sSL $(TOR_BASE_URL)/$(TOR_LINUX_VERSION)/tor-browser-linux-x86_64-$(TOR_LINUX_VERSION).tar.xz | \
		tar -xJf - -C $$tmpdir --strip-components=4 \
			tor-browser/Browser/TorBrowser/Tor/tor \
			tor-browser/Browser/TorBrowser/Tor/libevent-2.1.so.7 && \
		mv $$tmpdir/tor tor_binaries/linux-amd64/tor && \
		mv $$tmpdir/libevent-2.1.so.7 tor_binaries/linux-amd64/libevent-2.1.so.7 && \
		rm -rf $$tmpdir
	@chmod +x tor_binaries/linux-amd64/tor
	@xz -9 -f tor_binaries/linux-amd64/tor
	@xz -9 -f tor_binaries/linux-amd64/libevent-2.1.so.7
	@echo "    tor_binaries/linux-amd64/tor.xz"
	@echo "    tor_binaries/linux-amd64/libevent-2.1.so.7.xz"

# macOS and Windows: expert bundles (last stable release: 13.5.29, .tar.gz format)
tor_binaries/darwin-amd64/tor.xz:
	@mkdir -p tor_binaries/darwin-amd64
	@echo "==> Downloading Tor $(TOR_BUNDLE_VERSION) for darwin/amd64 ..."
	@tmpdir=$$(mktemp -d) && \
		curl -sSL $(TOR_BASE_URL)/$(TOR_BUNDLE_VERSION)/tor-expert-bundle-macos-x86_64-$(TOR_BUNDLE_VERSION).tar.gz | \
		tar -xzf - -C $$tmpdir && \
		cp $$tmpdir/tor/tor tor_binaries/darwin-amd64/tor && \
		cp $$tmpdir/tor/libevent-2.1.7.dylib tor_binaries/darwin-amd64/libevent-2.1.7.dylib && \
		rm -rf $$tmpdir
	@chmod +x tor_binaries/darwin-amd64/tor
	@xz -9 -f tor_binaries/darwin-amd64/tor
	@xz -9 -f tor_binaries/darwin-amd64/libevent-2.1.7.dylib
	@echo "    tor_binaries/darwin-amd64/tor.xz"
	@echo "    tor_binaries/darwin-amd64/libevent-2.1.7.dylib.xz"

tor_binaries/darwin-arm64/tor.xz:
	@mkdir -p tor_binaries/darwin-arm64
	@echo "==> Downloading Tor $(TOR_BUNDLE_VERSION) for darwin/arm64 ..."
	@tmpdir=$$(mktemp -d) && \
		curl -sSL $(TOR_BASE_URL)/$(TOR_BUNDLE_VERSION)/tor-expert-bundle-macos-aarch64-$(TOR_BUNDLE_VERSION).tar.gz | \
		tar -xzf - -C $$tmpdir && \
		cp $$tmpdir/tor/tor tor_binaries/darwin-arm64/tor && \
		cp $$tmpdir/tor/libevent-2.1.7.dylib tor_binaries/darwin-arm64/libevent-2.1.7.dylib && \
		rm -rf $$tmpdir
	@chmod +x tor_binaries/darwin-arm64/tor
	@xz -9 -f tor_binaries/darwin-arm64/tor
	@xz -9 -f tor_binaries/darwin-arm64/libevent-2.1.7.dylib
	@echo "    tor_binaries/darwin-arm64/tor.xz"
	@echo "    tor_binaries/darwin-arm64/libevent-2.1.7.dylib.xz"

tor_binaries/windows-amd64/tor.exe.xz:
	@mkdir -p tor_binaries/windows-amd64
	@echo "==> Downloading Tor $(TOR_BUNDLE_VERSION) for windows/amd64 ..."
	@tmpdir=$$(mktemp -d) && \
		curl -sSL $(TOR_BASE_URL)/$(TOR_BUNDLE_VERSION)/tor-expert-bundle-windows-x86_64-$(TOR_BUNDLE_VERSION).tar.gz | \
		tar -xzf - -C $$tmpdir && \
		cp $$tmpdir/tor/tor.exe tor_binaries/windows-amd64/tor.exe && \
		rm -rf $$tmpdir
	@xz -9 -f tor_binaries/windows-amd64/tor.exe
	@echo "    tor_binaries/windows-amd64/tor.exe.xz"

# Cross-compile builds (CGO_ENABLED=0 = pure Go, no C toolchain needed)

build-linux-amd64: tor_binaries/linux-amd64/tor.xz
	@mkdir -p bin/
	@echo "==> Building for linux/amd64 ..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/nestor-linux-amd64 .

build-darwin-amd64: tor_binaries/darwin-amd64/tor.xz
	@mkdir -p bin/
	@echo "==> Building for darwin/amd64 ..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/nestor-darwin-amd64 .

build-darwin-arm64: tor_binaries/darwin-arm64/tor.xz
	@mkdir -p bin/
	@echo "==> Building for darwin/arm64 ..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/nestor-darwin-arm64 .

build-windows-amd64: tor_binaries/windows-amd64/tor.exe.xz
	@mkdir -p bin/
	@echo "==> Building for windows/amd64 ..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/nestor-windows-amd64.exe .

build-all: build-linux-amd64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
