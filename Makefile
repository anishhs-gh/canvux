VERSION ?= 0.1.0
LDFLAGS  = -s -w -X main.version=$(VERSION)

.PHONY: build test bench demo clean

build:
	go build -ldflags "$(LDFLAGS)" -o canvux ./cmd/canvux

test:
	go vet ./...
	go test ./...

bench:
	go test -bench=. -benchmem -run=NONE ./internal/render/

demo: build
	./canvux export examples/demo.canvux --png examples/demo.png --scale 10

clean:
	rm -f canvux
