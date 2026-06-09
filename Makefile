BIN := bin/dovin
BINARY_NAME := dovin
INSTALL_DIR := $(HOME)/.local/bin
PLIST_NAME := com.glow.dovin
PLIST_PATH := $(HOME)/Library/LaunchAgents/$(PLIST_NAME).plist
LOG_PATH := $(HOME)/Library/Logs/dovin.log

.PHONY: build test install uninstall

build:
	mkdir -p bin
	go build -o $(BIN) .

test:
	go test ./...

install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Writing launchd plist to $(PLIST_PATH)"
	@printf '%s\n' \
		'<?xml version="1.0" encoding="UTF-8"?>' \
		'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
		'<plist version="1.0">' \
		'<dict>' \
		'    <key>Label</key>' \
		'    <string>com.glow.dovin</string>' \
		'    <key>ProgramArguments</key>' \
		'    <array>' \
		"        <string>$(INSTALL_DIR)/$(BINARY_NAME)</string>" \
		'    </array>' \
		'    <key>RunAtLoad</key>' \
		'    <true/>' \
		'    <key>KeepAlive</key>' \
		'    <true/>' \
		'    <key>StandardOutPath</key>' \
		"    <string>$(LOG_PATH)</string>" \
		'    <key>StandardErrorPath</key>' \
		"    <string>$(LOG_PATH)</string>" \
		'</dict>' \
		'</plist>' \
		> $(PLIST_PATH)
	launchctl load $(PLIST_PATH)
	@echo "Dovin installed and running."

uninstall:
	-launchctl unload $(PLIST_PATH) 2>/dev/null
	-rm -f $(PLIST_PATH)
	-rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Dovin uninstalled."
