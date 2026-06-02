.PHONY: all build build-all clean generate lint tidy test cover cover.detailed

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

cover:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -cover

cover.detailed:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out

clean:
	rm -rf $(BINDIR) coverage.out coverage.html
