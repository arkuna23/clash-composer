BUILD_DIR := build
BINARY := $(BUILD_DIR)/clash-composer
GOCACHE ?= $(BUILD_DIR)/.gocache
SOURCES := $(shell find . -type f -name '*.go' -not -path './build/*')

.PHONY: build clean

build: $(BINARY)

$(BINARY): $(SOURCES) go.mod go.sum | $(BUILD_DIR)/
	GOCACHE=$(abspath $(GOCACHE)) go build -o $(BINARY) .

$(BUILD_DIR)/:
	mkdir -p $@

clean:
	rm -rf $(BUILD_DIR)
