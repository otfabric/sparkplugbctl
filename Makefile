VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   = -X github.com/otfabric/sparkplugbctl/cmd.Version=$(VERSION) \
            -X github.com/otfabric/sparkplugbctl/cmd.Commit=$(COMMIT) \
            -X github.com/otfabric/sparkplugbctl/cmd.BuildDate=$(DATE)

.PHONY: build test lint install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/sparkplugbctl .

test:
	go test ./... -count=1 -timeout=60s

lint:
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -rf bin/
