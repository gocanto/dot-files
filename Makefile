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
OXFMT := pnpm exec oxfmt
OXLINT := pnpm exec oxlint

.PHONY: dev format format-start format-stop format-login

dev:
	@echo "Dev server will be available at: $(PORTLESS_DEFAULT_DEV_SERVER_URL)"
	@pnpm --dir $(UI_DIR) exec tsx scripts/dev.ts

format: format-start
	@echo "go-fmt format in $(ROOT_PATH)"; \
	$(GO_FMT_EXEC) format --cwd $(ROOT_PATH) --host-path $(ROOT_PATH)
	@echo "oxfmt format in $(ROOT_PATH)"
	@$(OXFMT) --write packages/ui packages/bridge package.json turbo.json
	@echo "oxlint fix in $(ROOT_PATH)"
	@$(OXLINT) --fix --vue-plugin packages/ui packages/bridge

format-start:
	@$(GO_FMT_COMPOSE) up -d $(GO_FMT_SERVICE)

format-stop:
	@$(GO_FMT_COMPOSE) stop $(GO_FMT_SERVICE)

format-login:
	@gh auth token | docker login ghcr.io -u $$(gh api user -q .login) --password-stdin
