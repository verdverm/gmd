# Development

## Build & test

```bash
make build                  # Build binary (CGO_ENABLED=0)
make test                   # Run unit tests (no external deps needed)
make test.integration       # Run all tests including integration
make cover.detailed         # Unit test coverage with HTML report
make cover.detailed.integration  # Full coverage including integration tests
make lint                   # go vet ./...
make tidy                   # go mod tidy
```

Integration tests (requiring Typesense or LLM endpoints) use the `//go:build integration` build tag.
Add it to any test file that needs external systems — it will be skipped by `make test` and only
run with `make test.integration`.
