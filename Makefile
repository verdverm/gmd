.PHONY: all build build-all clean generate lint tidy test

GO ?= go
CGO_ENABLED ?= 0
BINDIR ?= ./bin

all: build

build: build-gmd build-mcp

build-gmd:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/gmd ./cmd/gmd

build-mcp:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/gmd-mcp ./cmd/gmd-mcp

build-all: build-gmd build-mcp

tidy:
	$(GO) mod tidy

lint:
	$(GO) vet ./...

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -v

clean:
	rm -rf $(BINDIR)
