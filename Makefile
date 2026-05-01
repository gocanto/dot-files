ROOT_PATH := $(shell pwd)
GO_FMT_COMPOSE_FILE := go-fmt.compose.yaml
GO_FMT_SERVICE := go-fmt
GO_FMT_COMPOSE := docker compose -f $(GO_FMT_COMPOSE_FILE)
GO_FMT_BIN := /usr/local/bin/go-fmt
GO_FMT_EXEC := $(GO_FMT_COMPOSE) exec -T $(GO_FMT_SERVICE) $(GO_FMT_BIN)

.PHONY: format format-start format-stop

format: format-start
	@echo "go-fmt format in $(ROOT_PATH)"; \
	$(GO_FMT_EXEC) format --cwd $(ROOT_PATH) --host-path $(ROOT_PATH)

format-start:
	@$(GO_FMT_COMPOSE) up -d $(GO_FMT_SERVICE)

format-stop:
	@$(GO_FMT_COMPOSE) stop $(GO_FMT_SERVICE)
