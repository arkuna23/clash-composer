BUILD_DIR := build
BINARY := $(BUILD_DIR)/clash-composer
GOCACHE ?= $(BUILD_DIR)/.gocache
SOURCES := $(shell find . -type f -name '*.go' -not -path './build/*')
WEBAPP_DIR := webapp

.PHONY: build build-slim webapp go-build go-build-slim clean distclean

# Default build: build the webapp and embed it into the Go binary.
build: webapp go-build

# Slim build: skip the webapp and disable embed via the `noembed` build tag.
build-slim: go-build-slim

# Build the SPA into webapp/dist (runs `npm install` if needed).
webapp:
	cd $(WEBAPP_DIR) && npm install && npm run build

go-build: $(SOURCES) go.mod go.sum | $(BUILD_DIR)/
	GOCACHE=$(abspath $(GOCACHE)) go build -o $(BINARY) .

go-build-slim: $(SOURCES) go.mod go.sum | $(BUILD_DIR)/
	GOCACHE=$(abspath $(GOCACHE)) go build -tags noembed -o $(BINARY) .

$(BUILD_DIR)/:
	mkdir -p $@

clean:
	rm -rf $(BUILD_DIR)

# Remove webapp build artifacts in addition to the Go build dir.
distclean: clean
	rm -rf $(WEBAPP_DIR)/node_modules
	find $(WEBAPP_DIR)/dist -mindepth 1 ! -name .placeholder -delete 2>/dev/null || true
