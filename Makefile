BUILD_DIR := build
BINARY := $(BUILD_DIR)/clash-composer
GOCACHE ?= $(BUILD_DIR)/.gocache
SOURCES := $(shell find . -type f -name '*.go' -not -path './build/*')
WEBAPP_DIR := webapp
DEV_CONFIG_DIR ?= configs
DEV_TOKEN ?= dev
DEV_ADDR ?= 127.0.0.1:8080

.PHONY: build build-slim webapp go-build go-build-slim dev clean distclean

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
	mkdir -p $(DEV_CONFIG_DIR)
	set -e; \
	$(BINARY) serve -config-dir $(DEV_CONFIG_DIR) -addr $(DEV_ADDR) -token $(DEV_TOKEN) & \
	backend_pid=$$!; \
	trap 'kill $$backend_pid 2>/dev/null || true; wait $$backend_pid 2>/dev/null || true' EXIT; \
	trap 'exit 130' INT; \
	trap 'exit 143' TERM; \
	sleep 1; \
	if ! kill -0 $$backend_pid 2>/dev/null; then \
		wait $$backend_pid; \
		exit 1; \
	fi; \
	cd $(WEBAPP_DIR) && pnpm install --frozen-lockfile && pnpm run dev

$(BUILD_DIR)/:
	mkdir -p $@

clean:
	rm -rf $(BUILD_DIR)

# Remove webapp build artifacts in addition to the Go build dir.
distclean: clean
	rm -rf $(WEBAPP_DIR)/node_modules
	find $(WEBAPP_DIR)/dist -mindepth 1 ! -name .placeholder -delete 2>/dev/null || true
