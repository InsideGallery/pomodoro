VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := build
APPDIR    := $(BUILD_DIR)/AppDir
APPIMAGETOOL := $(BUILD_DIR)/appimagetool
PLUGIN_DIR := $(HOME)/.config/pomodoro/plugins
PLUGIN_SRC := plugins

LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build build-fingerprint build-android-aar build-android-apk test clean appimage icon install plugins lint coverage

all: build

## Build pomodoro timer
build:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/pomodoro ./services/pomodoro/cmd/pomodoro/

## Build fingerprint game
build-fingerprint:
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/fingerprint ./services/fingerprint/cmd/fingerprint/

## Build Android AAR (requires Android NDK)
build-android-aar:
	ebitenmobile bind -target android \
		-javapkg com.insidegallery.fingerprint \
		-o services/fingerprint/mobile/android/app/libs/fingerprint.aar \
		./services/fingerprint/mobile/

## Build Android APK (requires Android SDK + gradle)
build-android-apk: build-android-aar
	cd services/fingerprint/mobile/android && ./gradlew assembleDebug

## Build all products
build-all: build build-fingerprint

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
	go run ./services/pomodoro/cmd/genicon/ packaging/pomodoro.png

## Build .so plugins (Linux/macOS)
plugins:
	@mkdir -p $(PLUGIN_DIR)
	@if [ -d "$(PLUGIN_SRC)" ]; then \
		for dir in $(PLUGIN_SRC)/*/; do \
			if [ -f "$$dir/main.go" ]; then \
				name=$$(basename $$dir); \
				echo "==> Building plugin: $$name"; \
				go build -buildmode=plugin -tags plugin -o $(PLUGIN_DIR)/$$name.so ./$$dir; \
			fi; \
		done; \
		echo "==> Plugins installed to $(PLUGIN_DIR)"; \
	else \
		echo "No plugins/ directory found"; \
	fi

## Build AppImage (pomodoro)
appimage: build icon $(APPIMAGETOOL)
	@echo "==> Preparing AppDir..."
	rm -rf $(APPDIR)
	mkdir -p $(APPDIR)/usr/bin
	mkdir -p $(APPDIR)/usr/share/applications
	mkdir -p $(APPDIR)/usr/share/icons/hicolor/256x256/apps
	cp $(BUILD_DIR)/pomodoro $(APPDIR)/usr/bin/pomodoro
	cp packaging/pomodoro.desktop $(APPDIR)/pomodoro.desktop
	cp packaging/pomodoro.desktop $(APPDIR)/usr/share/applications/pomodoro.desktop
	cp packaging/pomodoro.png $(APPDIR)/pomodoro.png
	cp packaging/pomodoro.png $(APPDIR)/usr/share/icons/hicolor/256x256/apps/pomodoro.png
	ln -sf usr/bin/pomodoro $(APPDIR)/AppRun
	@echo "==> Building AppImage..."
	ARCH=x86_64 $(APPIMAGETOOL) $(APPDIR) $(BUILD_DIR)/pomodoro-$(VERSION)-linux-amd64.AppImage
	@echo "==> Done"

$(APPIMAGETOOL):
	@echo "==> Downloading appimagetool..."
	mkdir -p $(BUILD_DIR)
	curl -fsSL -o $(APPIMAGETOOL) \
		https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
	chmod +x $(APPIMAGETOOL)

## Install pomodoro to /usr/local
install: build
	install -Dm755 $(BUILD_DIR)/pomodoro /usr/local/bin/pomodoro
	install -Dm644 packaging/pomodoro.desktop /usr/local/share/applications/pomodoro.desktop
	install -Dm644 packaging/pomodoro.png /usr/local/share/icons/hicolor/256x256/apps/pomodoro.png

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out
