VERSION ?= 1.0.0
LDFLAGS ?= -s -w -X main.version=$(VERSION)

# Platforms built by `make release`, as GOOS/GOARCH pairs.
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build vet test bench demo release clean

build:
	go build -ldflags "$(LDFLAGS)" -o canvux ./cmd/canvux

vet:
	go vet ./...

test: vet
	go test ./...

bench:
	go test -bench=. -benchmem -run=NONE ./internal/render/

demo: build
	./canvux export examples/demo.canvux --png examples/demo.png --scale 10

# Cross-compile release binaries into dist/ and write a SHA256SUMS file.
# Binary names match what install.sh expects: canvux-<os>-<arch>[.exe].
release:
	@rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out="dist/canvux-$$os-$$arch"; \
		[ "$$os" = "windows" ] && out="$$out.exe"; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
			go build -ldflags "$(LDFLAGS)" -o "$$out" ./cmd/canvux || exit 1; \
	done
	@cd dist && (command -v sha256sum >/dev/null 2>&1 && sha256sum canvux-* || shasum -a 256 canvux-*) > SHA256SUMS
	@ls -lh dist

clean:
	rm -f canvux
	rm -rf dist
