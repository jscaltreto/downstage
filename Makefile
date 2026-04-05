.PHONY: all test lint fmt vet check clean render release-check release-snapshot wasm web web-dev web-clean

BINARY := downstage
BUILD_DIR := build
DS_FILES := $(filter-out testdata/errors.ds,$(wildcard testdata/*.ds))
PDF_FILES := $(patsubst testdata/%.ds,$(BUILD_DIR)/%.pdf,$(DS_FILES))
CONDENSED_FILES := $(patsubst testdata/%.ds,$(BUILD_DIR)/%_condensed.pdf,$(DS_FILES))

all: $(BINARY)

$(BINARY):
	go build -o $@ .

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

web-clean:
	rm -rf web/build web/dist web/node_modules

web/build:
	mkdir -p web/build
