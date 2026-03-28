# OctoManger — developer convenience targets
# Run `make help` for a summary.

PROTO_DIR     := proto
GO_GEN_DIR    := internal/gen
PYTHON_SDK    := plugins/sdk/python/octo
PROTOC        ?= $(shell which protoc 2>/dev/null || echo ~/.local/bin/protoc)
PROTOC_GO     ?= $(shell which protoc-gen-go 2>/dev/null || echo ~/go/bin/protoc-gen-go)
PROTOC_GO_GRPC ?= $(shell which protoc-gen-go-grpc 2>/dev/null || echo ~/go/bin/protoc-gen-go-grpc)
PYTHON        ?= python3

.PHONY: help proto-gen proto-gen-go proto-gen-python build build-web-assets test lint

## ── help ─────────────────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ── proto-gen ────────────────────────────────────────────────────────────────

proto-gen: proto-gen-go proto-gen-python ## Regenerate all gRPC stubs (Go + Python)

proto-gen-go: ## Generate Go stubs from proto/plugin/v1/plugin.proto
	@echo "[proto-gen-go] generating..."
	@mkdir -p $(GO_GEN_DIR)/plugin/v1
	PATH="$(dir $(PROTOC_GO)):$(dir $(PROTOC_GO_GRPC)):$$PATH" \
	$(PROTOC) \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_GEN_DIR)/plugin/v1 \
		--go_opt=paths=import \
		--go_opt=Mplugin/v1/plugin.proto=octomanger/internal/gen/plugin/v1 \
		--go-grpc_out=$(GO_GEN_DIR)/plugin/v1 \
		--go-grpc_opt=paths=import \
		--go-grpc_opt=Mplugin/v1/plugin.proto=octomanger/internal/gen/plugin/v1 \
		$(PROTO_DIR)/plugin/v1/plugin.proto
	@echo "[proto-gen-go] done → $(GO_GEN_DIR)/plugin/v1/"

proto-gen-python: ## Generate Python stubs from proto/plugin/v1/plugin.proto
	@echo "[proto-gen-python] generating..."
	$(PYTHON) -m grpc_tools.protoc \
		--proto_path=$(PROTO_DIR) \
		--python_out=$(PYTHON_SDK) \
		--grpc_python_out=$(PYTHON_SDK) \
		$(PROTO_DIR)/plugin/v1/plugin.proto
	@# Fix import path: generated file uses `import plugin_pb2` but we need
	@# relative imports inside the octo package.
	sed -i 's/^import plugin_v1_pb2/from . import plugin_v1_pb2/' \
		$(PYTHON_SDK)/plugin_v1_pb2_grpc.py 2>/dev/null || true
	@echo "[proto-gen-python] done → $(PYTHON_SDK)/"

## ── build / test ─────────────────────────────────────────────────────────────

build-web-assets: ## Build the frontend and sync embedded assets
	./scripts/prepare-webui.sh

build: build-web-assets ## Build the unified binary with embedded web assets
	go build -o bin/octomanger ./apps/octomanger

test: ## Run all Go tests
	go test ./...

lint: ## Run Go vet
	go vet ./...
