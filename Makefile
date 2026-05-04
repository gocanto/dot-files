SHELL := /bin/bash

ROOT_PATH := $(shell pwd)
UI_DIR := packages/ui
PORTLESS_APP_NAME := dot-files-ui
PORTLESS_DEFAULT_DEV_SERVER_URL := https://$(PORTLESS_APP_NAME).localhost
GO_FMT_COMPOSE_FILE := go-fmt.compose.yaml
GO_FMT_SERVICE := go-fmt
GO_FMT_COMPOSE := docker compose -f $(GO_FMT_COMPOSE_FILE)
GO_FMT_BIN := /usr/local/bin/go-fmt
GO_FMT_EXEC := $(GO_FMT_COMPOSE) exec -T $(GO_FMT_SERVICE) $(GO_FMT_BIN)

.PHONY: dev format format-start format-stop format-login

dev:
	@cd $(UI_DIR) && \
	set -euo pipefail; \
	pids=""; \
	cleanup() { \
		status=$$?; \
		trap - EXIT INT TERM; \
		if [ -n "$$pids" ]; then \
			kill $$pids >/dev/null 2>&1 || true; \
			wait $$pids >/dev/null 2>&1 || true; \
		fi; \
		exit $$status; \
	}; \
	trap cleanup EXIT INT TERM; \
	echo "Compiling Electron main/preload"; \
	pnpm exec tsc -p tsconfig.electron.json --pretty false; \
	pnpm exec vite build --config vite.electron.config.ts; \
	echo "Starting Vite via portless at $(PORTLESS_DEFAULT_DEV_SERVER_URL)"; \
	pnpm exec portless --name $(PORTLESS_APP_NAME) --force -- vite & \
	vite_pid=$$!; \
	pids="$$pids $$vite_pid"; \
	echo "Starting Electron TypeScript watcher"; \
	pnpm exec tsc -p tsconfig.electron.json --watch --preserveWatchOutput & \
	tsc_pid=$$!; \
	pids="$$pids $$tsc_pid"; \
	echo "Starting Electron preload watcher"; \
	pnpm exec vite build --config vite.electron.config.ts --watch & \
	preload_pid=$$!; \
	pids="$$pids $$preload_pid"; \
	echo "Waiting for portless route"; \
	electron_url=""; \
	until electron_url=$$(pnpm exec portless get $(PORTLESS_APP_NAME) 2>/dev/null) && curl -kfs -o /dev/null "$$electron_url"; do \
		sleep 0.5; \
	done; \
	echo "Launching Electron at $$electron_url"; \
	VITE_DEV_SERVER_URL="$$electron_url" node --watch --watch-path=dist-electron node_modules/electron/cli.js . & \
	electron_pid=$$!; \
	pids="$$pids $$electron_pid"; \
	while kill -0 "$$vite_pid" >/dev/null 2>&1 && kill -0 "$$tsc_pid" >/dev/null 2>&1 && kill -0 "$$preload_pid" >/dev/null 2>&1 && kill -0 "$$electron_pid" >/dev/null 2>&1; do \
		sleep 1; \
	done; \
	wait $$pids

format: format-start
	@echo "go-fmt format in $(ROOT_PATH)"; \
	$(GO_FMT_EXEC) format --cwd $(ROOT_PATH) --host-path $(ROOT_PATH)

format-start:
	@$(GO_FMT_COMPOSE) up -d $(GO_FMT_SERVICE)

format-stop:
	@$(GO_FMT_COMPOSE) stop $(GO_FMT_SERVICE)

format-login:
	@gh auth token | docker login ghcr.io -u $$(gh api user -q .login) --password-stdin
