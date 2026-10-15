# Standard commands, identical locally and in CI (docs/08 §0).
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0
GITLEAKS := go run github.com/zricethezav/gitleaks/v8@v8.30.1

.PHONY: generate lint test build

# Code generation: oapi-codegen, openapi-typescript, sqlc, design tokens.
# One generator per line; each work package appends the one it introduces.
generate:
	cd apps/api && $(OAPI_CODEGEN) -config internal/api/oapi-codegen.yaml ../../contracts/openapi.yaml
	pnpm --filter @msp/api-client generate
	node docs/02-design-system/tokens/build.mjs
	node docs/02-design-system/tokens/build.mjs --out packages/design-tokens/tokens.css
	cd apps/api && $(SQLC) generate

# STD-FE-01: apps/web styles only via design tokens (no raw hex/rgb/px, no Tailwind arbitrary values).
lint:
	! grep -rnIE '#[0-9a-fA-F]{3,8}\b|[0-9]px\b|rgba?\(|[a-z]-(\[|\(--)' $(wildcard apps/web/app apps/web/components) || (echo "STD-FE-01: use design tokens" && false)
	cd apps/api && $(GOLANGCI_LINT) run ./...
	cd apps/api && $(SQLC) vet
	pnpm -r --if-present lint
	pnpm -r --if-present typecheck
	$(GITLEAKS) git --log-opts=HEAD --redact --no-banner .   # STD-AGENT-06: no secret in the history of HEAD
# TODO(WP-10): squawk (new migrations) and oasdiff breaking vs main (08 §0) arrive with the F1 migrations and contract.
# TODO(WP-26): coverage gate >= 80% for the core packages in `make test` (NFR-09, STD-TEST-05), once estimate exists.

test:
	cd apps/api && go test -race ./...
	pnpm -r --if-present test

build:
	cd apps/api && go build -o bin/msp ./cmd/msp
	pnpm -r --if-present build
