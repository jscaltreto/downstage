.PHONY: all test lint fmt vet check clean render release-check release-snapshot wasm web web-dev web-e2e web-clean desktop-dev desktop-build desktop-debug desktop-clean

BINARY := downstage
BUILD_DIR := build
DS_FILES := $(filter-out testdata/errors.ds,$(wildcard testdata/*.ds))
PDF_FILES := $(patsubst testdata/%.ds,$(BUILD_DIR)/%.pdf,$(DS_FILES))
CONDENSED_FILES := $(patsubst testdata/%.ds,$(BUILD_DIR)/%_condensed.pdf,$(DS_FILES))

all: $(BINARY)

$(BINARY):
	go build -o $@ ./cmd/downstage

test:
	go test ./...

lint: fmt vet

fmt:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

vet:
	go vet ./...

check: lint test

clean:
	rm -f $(BINARY)
	rm -rf $(BUILD_DIR)

render: $(BINARY) $(PDF_FILES) $(CONDENSED_FILES)
	@echo "Rendered $(words $(PDF_FILES)) manuscripts and $(words $(CONDENSED_FILES)) condensed editions to $(BUILD_DIR)/"

release-check:
	goreleaser check

release-snapshot:
	goreleaser release --snapshot --clean

$(BUILD_DIR)/%.pdf: testdata/%.ds $(BINARY) | $(BUILD_DIR)
	./$(BINARY) render $< -o $@

$(BUILD_DIR)/%_condensed.pdf: testdata/%.ds $(BINARY) | $(BUILD_DIR)
	./$(BINARY) render --style condensed $< -o $@

$(BUILD_DIR):
	mkdir -p $@

# --- Web editor (WASM) ---

wasm: | web/build
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/build/downstage.wasm ./cmd/wasm/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" web/build/

web: wasm
	npm --prefix web run build

web-dev: wasm
	@echo "Serving web editor at http://localhost:5173/editor/"
	cd web && npm run dev -- --host 0.0.0.0

web-e2e: web
	npm --prefix web run test:e2e

web-clean:
	rm -rf web/build web/dist web/node_modules

web/build:
	mkdir -p web/build

# --- Desktop app (Wails) ---

# Version string surfaced in the About dialog via ldflags. `git describe`
# produces something like `v0.3.1-12-g329f5e6-dirty`; falls back to "dev"
# in non-git contexts. Overridable: `make desktop-build VERSION=1.2.3`.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DESKTOP_LDFLAGS := -X github.com/jscaltreto/downstage/internal/desktop.Version=$(VERSION)

desktop-dev:
	@echo "Starting desktop app in dev mode (version $(VERSION))..."
	cd cmd/downstage-write && wails dev -tags webkit2_41 -ldflags "$(DESKTOP_LDFLAGS)"

desktop-build:
	@echo "Building desktop app (version $(VERSION))..."
	cd cmd/downstage-write && wails build -tags webkit2_41 -ldflags "$(DESKTOP_LDFLAGS)"

# Debug build: enables the WebKit Web Inspector so you can right-click
# the running app and pick "Inspect Element". Wails -debug also opens
# the inspector on startup.
#
# To make webkit2gtk's compositing layers visible, launch the binary
# with:
#   WEBKIT_SHOW_COMPOSITING_INDICATORS=1 \
#     cmd/downstage-write/build/bin/downstage-write
# That overlays a translucent tint per compositing layer — useful for
# seeing which layer is holding stale paint after a reflow.
desktop-debug:
	@echo "Building desktop app in debug mode (version $(VERSION))..."
	cd cmd/downstage-write && wails build -tags webkit2_41 -debug -devtools -ldflags "$(DESKTOP_LDFLAGS)"

desktop-clean:
	rm -rf cmd/downstage-write/build/bin cmd/downstage-write/frontend
