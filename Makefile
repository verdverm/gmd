.PHONY: all build build-all clean generate lint tidy test test.integration cover cover.integration cover.detailed cover.detailed.integration

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
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -v -count=1

test.integration: clean-ts
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 ./... -v -count=1 -tags=integration

cover:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -cover -count=1

cover.integration:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -cover -count=1 -tags=integration

cover.detailed:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -coverprofile=coverage.out -count=1
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out

cover.detailed.integration:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -coverprofile=coverage.out -count=1 -tags=integration
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out

clean:
	rm -rf $(BINDIR) coverage.out coverage.html

clean-ts:
	-docker rm -f gmd-ts-integration
