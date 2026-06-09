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
	@mkdir -p $(HOME)/Library/LaunchAgents
	@printf '<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0">\n<dict>\n  <key>Label</key><string>$(PLIST_NAME)</string>\n  <key>ProgramArguments</key><array><string>$(INSTALL_DIR)/$(BINARY_NAME)</string></array>\n  <key>RunAtLoad</key><true/>\n  <key>KeepAlive</key><true/>\n  <key>StandardOutPath</key><string>$(LOG_PATH)</string>\n  <key>StandardErrorPath</key><string>$(LOG_PATH)</string>\n</dict>\n</plist>\n' > $(PLIST_PATH)
	-launchctl bootout gui/$(shell id -u) $(PLIST_PATH) 2>/dev/null; true
	launchctl bootstrap gui/$(shell id -u) $(PLIST_PATH)
	@echo "Dovin installed and running."

uninstall:
	-launchctl bootout gui/$(shell id -u) $(PLIST_PATH) 2>/dev/null
	-rm -f $(PLIST_PATH)
	-rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Dovin uninstalled."
