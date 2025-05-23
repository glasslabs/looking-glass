# Run CI tasks
ci: wasmexec lint test build
.PHONY: ci

# Format all files
fmt:
	@echo "==> Formatting source"
	@gofmt -s -w $(shell find . -type f -name '*.go' -not -path "./vendor/*")
	@echo "==> Done"
.PHONY: fmt

# Tidy the go.mod file
tidy:
	@echo "==> Cleaning go.mod"
	@go mod tidy
	@echo "==> Done"
.PHONY: tidy

# Lint the project
lint:
	@echo "==> Linting Go files"
	@golangci-lint run ./...
.PHONY: lint

# Run all tests
test:
	@go test -cover -race ./...
.PHONY: test

# Build the commands
build:
	@find ./cmd/* -maxdepth 1 -type d -exec go build {} \;
.PHONY: build

# Static Assets

wasmexec:
	@cp $(shell go env GOROOT)/lib/wasm/wasm_exec.js ./webui
.PHONY: wasmexec

wasmexec-check: wasmexec
	@echo "==> Checking wasm_exec"
	@git diff --exit-code --quiet ./
.PHONY: wasmexec-check

