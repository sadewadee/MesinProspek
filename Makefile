# Standard commands, identical locally and in CI (docs/08 §0).
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2

.PHONY: generate lint test build

# Code generation: oapi-codegen, openapi-typescript, sqlc, design tokens.
# Each generator is added by the work package that introduces it.
generate:
	@echo "generate: no generators yet"

lint:
	cd apps/api && $(GOLANGCI_LINT) run ./...
	pnpm -r --if-present lint
	pnpm -r --if-present typecheck

test:
	cd apps/api && go test -race ./...
	pnpm -r --if-present test

build:
	cd apps/api && go build -o bin/msp ./cmd/msp
	pnpm -r --if-present build
