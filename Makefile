BINARY     := getnote-cli
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -s -w"
BUILD_DIR  := dist

.PHONY: build build-all clean test lint install

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_amd64  .
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_darwin_arm64  .
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_amd64   .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_linux_arm64   .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)_windows_amd64.exe .

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

test:
	go test ./...

lint:
	go vet ./...

INSTALL_DIR ?= $(shell \
	if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then echo /usr/local/bin; \
	elif [ -d $(HOME)/go/bin ]; then echo $(HOME)/go/bin; \
	elif [ -d $(HOME)/.local/bin ]; then echo $(HOME)/.local/bin; \
	elif [ -d $(HOME)/bin ]; then echo $(HOME)/bin; \
	else echo /usr/local/bin; fi)

install: build
	@echo "Installing to $(INSTALL_DIR)/getnote"
	@mkdir -p $(INSTALL_DIR)
	install -m 755 $(BINARY) $(INSTALL_DIR)/getnote
	@echo "Done."
	@if ! $(INSTALL_DIR)/getnote auth status 2>/dev/null | grep -q "Authenticated"; then \
		echo ""; \
		$(INSTALL_DIR)/getnote auth login; \
	fi
