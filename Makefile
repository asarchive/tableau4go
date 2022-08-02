GO ?= go

# Installs golangci cli tool
lint-install:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.47.3

# Runs golangci lint rules and reports any errors to the console
.PHONY: lint
lint: lint-install
	golangci-lint run ./...

# Runs golangci lint rules and applies auto-fix if available (not available very often, mostly just for imports formatting)
.PHONY: lint-fix
lint-fix: lint-install
	golangci-lint run ./... --fix
