BUILD_DIR := build
BINARY := $(BUILD_DIR)/clash-composer
GOCACHE ?= $(BUILD_DIR)/.gocache
SOURCES := $(shell find . -type f -name '*.go' -not -path './build/*')
WEBAPP_DIR := webapp
DEV_CONFIG_DIR ?= configs
DEV_TOKEN ?= dev
DEV_ADDR ?= 127.0.0.1:8080
DEV_FIXTURE_ADDR := 127.0.0.1:8091

.PHONY: build build-slim webapp go-build go-build-slim dev deploy clean distclean

# Default build: build the webapp and embed it into the Go binary.
build: webapp go-build

# Slim build: skip the webapp and disable embed via the `noembed` build tag.
build-slim: go-build-slim

# Build the SPA into webapp/dist (runs `pnpm install` if needed).
webapp:
	cd $(WEBAPP_DIR) && pnpm install --frozen-lockfile && pnpm run build

go-build: $(SOURCES) go.mod go.sum | $(BUILD_DIR)/
	GOCACHE=$(abspath $(GOCACHE)) go build -o $(BINARY) .

go-build-slim: $(SOURCES) go.mod go.sum | $(BUILD_DIR)/
	GOCACHE=$(abspath $(GOCACHE)) go build -tags noembed -o $(BINARY) .

dev: go-build
	bash scripts/seed-dev.sh "$(DEV_CONFIG_DIR)"
	set -e; \
	$(BINARY) serve -config-dir $(DEV_CONFIG_DIR) -addr $(DEV_ADDR) -token $(DEV_TOKEN) & \
	backend_pid=$$!; \
	GOCACHE=$(abspath $(GOCACHE)) go run ./scripts/dev-fixture.go -addr $(DEV_FIXTURE_ADDR) -dir "$(abspath $(DEV_CONFIG_DIR))" & \
	fixture_pid=$$!; \
	cleanup() { \
		kill $$backend_pid $$fixture_pid 2>/dev/null || true; \
		wait $$backend_pid 2>/dev/null || true; \
		wait $$fixture_pid 2>/dev/null || true; \
	}; \
	trap cleanup EXIT; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	sleep 1; \
	if ! kill -0 $$backend_pid 2>/dev/null; then \
		wait $$backend_pid; \
		exit 1; \
	fi; \
	if ! kill -0 $$fixture_pid 2>/dev/null; then \
		wait $$fixture_pid; \
		exit 1; \
	fi; \
	cd $(WEBAPP_DIR) && pnpm install --frozen-lockfile && pnpm exec vite --host 127.0.0.1

deploy:
	./scripts/deploy.sh

$(BUILD_DIR)/:
	mkdir -p $@

clean:
	rm -rf $(BUILD_DIR)

# Remove webapp build artifacts in addition to the Go build dir.
distclean: clean
	rm -rf $(WEBAPP_DIR)/node_modules
	find $(WEBAPP_DIR)/dist -mindepth 1 ! -name .placeholder -delete 2>/dev/null || true
