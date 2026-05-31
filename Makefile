.PHONY: all build build-all clean generate lint tidy test

GO ?= go
CGO_ENABLED ?= 0
BINDIR ?= ./bin

all: build

build: build-gmd

build-gmd:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/gmd ./cmd/gmd

build-all: build-gmd

tidy:
	$(GO) mod tidy

lint:
	$(GO) vet ./...

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -v

clean:
	rm -rf $(BINDIR)
