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

install: build
	mkdir -p /usr/local/bin
	mv $(BINARY) /usr/local/bin/getnote
