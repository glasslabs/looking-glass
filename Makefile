# Run CI tasks
ci: lint test build
.PHONY: ci

# Download Roboto and Roboto Condensed TTF font files into ui/fonts/.
# These are compiled into the binary via //go:embed. Run this once before
# building. Fonts are Apache 2.0 licensed (https://fonts.google.com/specimen/Roboto).
fonts:
	@echo "==> Downloading Roboto fonts"
	@mkdir -p ui/fonts
	@curl -sSL "https://github.com/googlefonts/roboto-3-classic/releases/latest/download/Roboto_v3.015.zip" \
		-o /tmp/roboto.zip
	@unzip -o -j /tmp/roboto.zip "android/static/*.ttf" -d ui/fonts/
	@rm /tmp/roboto.zip
	@echo "==> Done ($(shell ls ui/fonts/*.ttf 2>/dev/null | wc -l | tr -d ' ') variants)"
.PHONY: fonts

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

# Build all WASM plugin modules and install them into .test/modules/
# Run this before `make run` whenever a plugin source changes.
build-modules:
	@echo "==> Building WASM modules"
	@mkdir -p .test/modules
	@for plugin in clock calendar weather solar water hass-floorplan; do \
		echo "    $$plugin..."; \
		(cd ../$$plugin && GOOS=wasip1 GOARCH=wasm go build -o $(CURDIR)/.test/modules/$$plugin.wasm .); \
	done
	@echo "==> Done"
.PHONY: build-modules

# Run the desktop app with the .test config (requires build-modules first)
run:
	@go run ./cmd/glass run \
		--config    .test/config.yaml \
		--secrets   .test/secrets.yaml \
		--assets    .test/assets \
		--modules   .test/modules \
		--log.level=trace
.PHONY: run

