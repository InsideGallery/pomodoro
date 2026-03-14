APP_NAME := pomodoro
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build
APPDIR    := $(BUILD_DIR)/AppDir
BINARY    := $(BUILD_DIR)/$(APP_NAME)
APPIMAGE  := $(BUILD_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64.AppImage
APPIMAGETOOL := $(BUILD_DIR)/appimagetool
PLUGIN_DIR := $(HOME)/.config/pomodoro/plugins
PLUGIN_SRC := plugins

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build test clean appimage icon install plugins lint coverage

all: build

## Build the binary
build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/pomodoro/

## Run tests
test:
	go test ./... -v

## Run linter
lint:
	golangci-lint run ./...

## Check test coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go-test-coverage --config .testcoverage.yml

## Generate app icon
icon:
	go run ./cmd/genicon/ packaging/pomodoro.png

## Build and install plugins from plugins/ directory
## Each subdirectory with a main.go is compiled as a Go plugin (.so)
plugins:
	@mkdir -p $(PLUGIN_DIR)
	@if [ -d "$(PLUGIN_SRC)" ]; then \
		for dir in $(PLUGIN_SRC)/*/; do \
			if [ -f "$$dir/main.go" ]; then \
				name=$$(basename $$dir); \
				echo "==> Building plugin: $$name"; \
				go build -buildmode=plugin -o $(PLUGIN_DIR)/$$name.so $$dir; \
			fi; \
		done; \
		echo "==> Plugins installed to $(PLUGIN_DIR)"; \
	else \
		echo "No plugins/ directory found"; \
	fi

## Build AppImage
appimage: build icon $(APPIMAGETOOL)
	@echo "==> Preparing AppDir..."
	rm -rf $(APPDIR)
	mkdir -p $(APPDIR)/usr/bin
	mkdir -p $(APPDIR)/usr/share/applications
	mkdir -p $(APPDIR)/usr/share/icons/hicolor/256x256/apps
	cp $(BINARY) $(APPDIR)/usr/bin/$(APP_NAME)
	cp packaging/pomodoro.desktop $(APPDIR)/$(APP_NAME).desktop
	cp packaging/pomodoro.desktop $(APPDIR)/usr/share/applications/$(APP_NAME).desktop
	cp packaging/pomodoro.png $(APPDIR)/$(APP_NAME).png
	cp packaging/pomodoro.png $(APPDIR)/usr/share/icons/hicolor/256x256/apps/$(APP_NAME).png
	ln -sf usr/bin/$(APP_NAME) $(APPDIR)/AppRun
	@echo "==> Building AppImage..."
	ARCH=x86_64 $(APPIMAGETOOL) $(APPDIR) $(APPIMAGE)
	@echo "==> Done: $(APPIMAGE)"

## Download appimagetool if missing
$(APPIMAGETOOL):
	@echo "==> Downloading appimagetool..."
	mkdir -p $(BUILD_DIR)
	curl -fsSL -o $(APPIMAGETOOL) \
		https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
	chmod +x $(APPIMAGETOOL)

## Install to /usr/local
install: build
	install -Dm755 $(BINARY) /usr/local/bin/$(APP_NAME)
	install -Dm644 packaging/pomodoro.desktop /usr/local/share/applications/$(APP_NAME).desktop
	install -Dm644 packaging/pomodoro.png /usr/local/share/icons/hicolor/256x256/apps/$(APP_NAME).png

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out
