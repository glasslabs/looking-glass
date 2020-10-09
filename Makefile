GO_TOOLS = \
    github.com/go-bindata/go-bindata/go-bindata@v3.1.2

GEN_GO_FILES?=$(shell find . -name '*.gen.go')

# Run CI tasks
ci: tools static-assets lint build test-coverage
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

# Install required tools to build TraefikEE
tools:
	@mkdir -p .gotools
	@set -e; \
	cd .gotools && if ! test -f go.mod; then \
		go mod init tools ; \
	fi
	cd .gotools && go get -v $(GO_TOOLS)

# Build the commands
build:
	@find ./cmd/* -maxdepth 1 -type d -exec go build {} \;
.PHONY: build

# Run all tests
test:
	@go test -cover -race ./...
.PHONY: test

# Run all tests with a coverage output
test-coverage:
	@go test -race -covermode=count -coverprofile=profile.cov ./...
.PHONY: test-coverage

# Lint the project
lint:
	@golangci-lint run ./...
.PHONY: lint

# Static Assets

# Remove generated static assets
static-assets-clean:
	@echo "==> Removing $(GEN_GO_FILES)"
	-@rm $(GEN_GO_FILES)

# Generate static assets
static-assets:
	@echo "==> Generating static assets"
	@go-bindata -pkg glass -prefix webui -nocompress -o assets.gen.go ./webui/...
	#go run ./internal/gensym -o=./module/internal/types/types.gen.go
