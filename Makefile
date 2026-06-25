.PHONY: all build build-all clean generate lint tidy gofmt vulncheck nilaway lint-all check \
	test test.integration cover cover.integration cover.detailed cover.detailed.integration \
	tools-install tools-update cloc

GO ?= go
CGO_ENABLED ?= 0
BINDIR ?= ./bin
TOOLS_BINDIR ?= $(BINDIR)

all: build

build: build-gmd

build-gmd:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/gmd ./cmd/gmd

build-all: build-gmd

tidy:
	$(GO) mod tidy

lint:
	$(GO) vet ./...

gofmt:
	@$(GO)fmt -s -l . | grep . && echo "Run 'gofmt -s -w' to fix" && exit 1 || true

# --- 3rd party tooling ---

tools-install:
	GOBIN=$(abspath $(TOOLS_BINDIR)) $(GO) install \
		github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
		golang.org/x/vuln/cmd/govulncheck@latest \
		go.uber.org/nilaway/cmd/nilaway@latest

tools-update: tools-install

lint-all: $(TOOLS_BINDIR)/golangci-lint
	$(TOOLS_BINDIR)/golangci-lint run ./...

vulncheck: $(TOOLS_BINDIR)/govulncheck
	$(TOOLS_BINDIR)/govulncheck ./...

nilaway: $(TOOLS_BINDIR)/nilaway
	$(TOOLS_BINDIR)/nilaway ./...

check: tidy gofmt lint lint-all vulncheck test

test:
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test ./... -v -count=1

test.integration: clean-ts
	CGO_ENABLED=$(CGO_ENABLED) $(GO) test -p 1 ./... -v -count=1 -tags=integration -timeout 30m

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

cloc:
	cloc . --exclude-dir=node_modules

clean-ts:
	-docker rm -f gmd-ts-integration
