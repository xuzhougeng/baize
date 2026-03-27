APP_NAME := myclaw
CMD_PATH := ./cmd/myclaw
DIST_DIR := dist
GO ?= go
CGO_ENABLED ?= 0

.PHONY: help test clean install-hooks build build-current build-linux build-linux-amd64 build-linux-arm64 build-windows build-windows-amd64 build-windows-arm64 build-macos build-macos-amd64 build-macos-arm64 release

help:
	@printf "Targets:\n"
	@printf "  make install-hooks\n"
	@printf "  make test\n"
	@printf "  make build-current\n"
	@printf "  make build-linux\n"
	@printf "  make build-windows\n"
	@printf "  make build-macos\n"
	@printf "  make release\n"
	@printf "  make clean\n"

install-hooks:
	sh ./scripts/install-hooks.sh

test:
	$(GO) test ./...

clean:
	rm -rf $(DIST_DIR)

build: build-current

build-current:
	mkdir -p $(DIST_DIR)
	$(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME) $(CMD_PATH)

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)

build-linux-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)

build-windows: build-windows-amd64 build-windows-arm64

build-windows-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)

build-windows-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=arm64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-windows-arm64.exe $(CMD_PATH)

build-macos: build-macos-amd64 build-macos-arm64

build-macos-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)

build-macos-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)

release: test build-linux build-windows build-macos
