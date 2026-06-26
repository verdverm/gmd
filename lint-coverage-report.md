# GMD Lint and Coverage Report

Generated: 2026-06-26

## 1. Lint Results

### `make tidy` — go mod tidy

**Status: PASS**

### `make lint` — go vet

**Status: PASS**

### `make gofmt` — formatting check

**Status: PASS**

### `make lint-all` — golangci-lint

**Status: FAIL** (58 issues)

#### By linter

| Linter | Count |
|---|---|
| gocyclo | 34 |
| prealloc | 9 |
| errcheck | 4 |
| gosec | 4 |
| unconvert | 3 |
| unparam | 2 |
| unused | 1 |
| usetesting | 1 |

#### Details

**errcheck** (4) — unchecked error returns

| File | Line | Issue |
|---|---|---|
| `pkg/context/agents/agents_test.go` | 193 | `os.Chmod` return not checked |
| `pkg/llm/openai_model_test.go` | 171 | type assertion error not checked |
| `pkg/wiki/wiki_tape_helpers_test.go` | 22 | `os.MkdirAll` return not checked |
| `pkg/wiki/wiki_tape_helpers_test.go` | 27 | `os.WriteFile` return not checked |
| `pkg/web/fusion/fusion_replay_test.go` | 23, 34, 45 | `tape.Stop` return not checked |

**gocyclo** (34) — cyclomatic complexity > 15

| File | Line | Function | Complexity |
|---|---|---|---|
| `pkg/config/config.go` | 611 | `Load` | 39 |
| `pkg/wiki/lint.go` | 52 | `(*Agent).lintStructure` | 37 |
| `pkg/config/config.go` | 821 | `mergeConfigs` | 27 |
| `pkg/config/edit.go` | 884 | `RemoveSourceRef` | 21 |
| `cmd/gmd/env.go` | 77 | `printConfigSources` | 26 |
| `pkg/llm/registry.go` | 122 | `NewRegistry` | 26 |
| `pkg/web/providers/exa/search.go` | 26 | `(*SearchAdapter).Search` | 26 |
| `pkg/wiki/doctor.go` | 37 | `Doctor` | 26 |
| `pkg/search/pipeline_test.go` | 271 | `TestSearch_GenerateVariantsParsing` | 25 |
| `pkg/agent/agent.go` | 38 | `Launch` | 24 |
| `pkg/indexer/indexer.go` | 111 | `(*Indexer).updateCollection` | 24 |
| `pkg/web/agent.go` | 55 | `(*Agent).Run` | 23 |
| `pkg/context/agents/agents_test.go` | 102 | `TestAgents_ShowAgent` | 23 |
| `pkg/agent/config.go` | 10 | `ResolveAgentConfig` | 22 |
| `pkg/config/edit.go` | 805 | `AddSourceRef` | 18 |
| `pkg/wiki/frontmatter.go` | 35 | `ValidateFrontmatter` | 18 |
| `pkg/config/edit.go` | 313 | `AddCollectionPatterns` | 16 |
| `pkg/config/edit.go` | 423 | `AddIgnorePatterns` | 16 |
| `pkg/config/edit.go` | 532 | `RemoveIgnorePattern` | 16 |
| `pkg/ts/client.go` | 331 | `(*Client).searchDocsByPattern` | 16 |
| `pkg/wiki/agent.go` | 89 | `(*Agent).Ingest` | 16 |
| `pkg/wiki/doctor.go` | 174 | `FormatDoctorResult` | 16 |
| `pkg/wiki/okf.go` | 124 | `ExportOKF` | 16 |
| `pkg/web/providers/cloudflare/client.go` | 89 | `(*BrowserClient).Crawl` | 17 |
| `pkg/wiki/frontmatter.go` | 80 | `FrontmatterToFilter` | 17 |
| `pkg/wiki/okf.go` | 31 | `ValidateOKF` | 17 |
| `pkg/config/edit_test.go` | 266 | `TestEdit_IgnorePatterns` | 17 |
| `pkg/wiki/wiki_test.go` | 158 | `TestWiki_ValidateFrontmatter` | 17 |
| `pkg/agent/agent.go` | 133 | `mergeOptions` | 20 |
| `pkg/indexer/indexer_test.go` | 15 | `TestIndexer_ScanFilesFS` | 20 |
| `pkg/agent/agent.go` | 193 | `printDryRun` | 18 |
| `pkg/wiki/wiki_test.go` | 16 | `TestWiki_ParseFrontmatter` | 28 |
| `pkg/web/providers/exa/search_test.go` | 101 | `TestSearchAdapter_ExtraMapping` | 16 |

**gosec** (4) — security issues

| File | Line | Issue |
|---|---|---|
| `pkg/agent/agent.go` | 101 | G204: Subprocess launched with a potential tainted input or cmd arguments |
| `pkg/wiki/wiki_replay_test.go` | 90, 155, 196 | G306: Expect WriteFile permissions to be 0600 or less |
| `pkg/context/agents/agents_test.go` | 82, 110, 113 | G306: WriteFile permissions should be <= 0600 |

**unconvert** (3) — unnecessary type conversions

| File | Line | Detail |
|---|---|---|
| `pkg/llm/openai_convert.go` | 33 | `openai.ChatModel(m.modelName)` |
| `pkg/llm/openai_convert.go` | 222 | `convertFinishReason(string(...))` |
| `pkg/llm/openai_model.go` | 217 | `convertFinishReason(string(...))` |

**unused** (1) — unused function

| File | Line | Function |
|---|---|---|
| `pkg/config/config.go` | 941 | `mergeBoolField` |

**prealloc** (9) — consider pre-allocating slices

| File | Line |
|---|---|
| `pkg/agent/config.go` | 93, 104 |
| `pkg/agent/tmux.go` | 75, 141, 171 |
| `pkg/config/config_test.go` | 150 |
| `pkg/llm/openai_convert.go` | 169 |
| `pkg/llm/registry.go` | 73 |

**unparam** (2) — unused parameter / always-nil result

| File | Line | Function |
|---|---|---|
| `pkg/llm/openai_convert.go` | 227 | `convertTools` — error always nil |
| `pkg/wiki/wiki_tape_helpers_test.go` | 70 | `newTestWikiAgent` — *Wiki result never used |

**usetesting** (1) — could use `t.Context()`

| File | Line |
|---|---|
| `pkg/web/fusion/fusion_replay_test.go` | 59 |

---

## 2. Test Coverage Results

### Overall: **38.9%** of statements (unit tests)

Run via `make cover.detailed`. All packages passed (`ok`).

### Per-package coverage (unit)

| Package | Coverage | Time |
|---|---|---|
| `cmd/gmd` | 0.0% | — |
| `pkg/agent` | 0.0% | — |
| `pkg/chunking` | 66.9% | 0.2s |
| `pkg/config` | 32.6% | 0.3s |
| `pkg/context/agents` | 88.1% | 0.2s |
| `pkg/context/agentsmd` | 85.7% | 0.3s |
| `pkg/context/skills` | 16.0% | 0.3s |
| `pkg/indexer` | 42.9% | 0.4s |
| `pkg/llm` | 19.7% | 0.8s |
| `pkg/llm/auth` | 40.0% | 0.6s |
| `pkg/mcp` | 0.0% | — |
| `pkg/output` | 95.3% | 1.1s |
| `pkg/runtime` | 0.0% | — |
| `pkg/search` | 36.4% | 1.2s |
| `pkg/testutil` | 91.7% | 1.4s |
| `pkg/ts` | 80.3% | 1.6s |
| `pkg/ts/testserver` | 0.0% | — |
| `pkg/web` | 16.9% | 1.5s |
| `pkg/web/builders` | 0.0% | — |
| `pkg/web/exa` | 0.0% | — |
| `pkg/web/fusion` | 57.6% | 1.5s |
| `pkg/web/persist` | 80.2% | 1.8s |
| `pkg/web/providers/cloudflare` | 85.6% | 1.5s |
| `pkg/web/providers/exa` | 79.0% | 3.4s |
| `pkg/web/providers/searxng` | 89.6% | 1.4s |
| `pkg/web/providers/tavily` | 87.5% | 1.4s |
| `pkg/wiki` | 44.2% | 1.4s |

---

## 3. Integration Coverage Results

### Overall: Integration tests required external services (Typesense Docker, LLM endpoints)

Run via `make cover.detailed.integration`. Some tests failed due to missing LLM/TS services:

- Fusions synthesis tests failed with 404 Not Found for model `Qwen/Qwen3.6-35B-A3B-FP8`
- Wiki integration tests failed with connection refused to LLM embedding endpoints (`192.168.4.31:8001`)
- LLM chat endpoints returned 404 Not Found

### Per-package coverage (integration where passed)

| Package | Coverage | Time |
|---|---|---|
| `cmd/gmd` | 0.0% | — |
| `pkg/agent` | 0.0% | — |
| `pkg/chunking` | 66.9% | 0.2s |
| `pkg/config` | 32.6% | 0.3s |
| `pkg/context/agents` | 88.1% | 0.2s |
| `pkg/context/agentsmd` | 85.7% | 0.2s |
| `pkg/context/skills` | 52.0% | 0.2s |
| `pkg/indexer` | 42.9% | 0.3s |
| `pkg/llm` | 23.2% | 13.1s |
| `pkg/llm/auth` | 40.0% | 0.2s |
| `pkg/mcp` | 0.0% | — |
| `pkg/output` | 95.3% | 0.3s |
| `pkg/runtime` | 0.0% | — |
| `pkg/search` | 36.4% | 0.3s |
| `pkg/testutil` | 91.7% | 0.2s |
| `pkg/ts` | 82.7% | 4.7s |
| `pkg/ts/testserver` | 0.0% | — |
| `pkg/web` | 16.9% | 0.3s |
| `pkg/web/builders` | 0.0% | — |
| `pkg/web/exa` | 0.0% | — |
| `pkg/web/fusion` | 88.9% | 8.9s |
| `pkg/web/persist` | 80.2% | 0.3s |
| `pkg/web/providers/cloudflare` | 86.4% | 1.6s |
| `pkg/web/providers/exa` | 79.0% | 3.4s |
| `pkg/web/providers/searxng` | 89.6% | 1.5s |
| `pkg/web/providers/tavily` | 87.5% | 1.2s |
| `pkg/wiki` | 69.8% | 27.4s |

---

## 4. Project Stats

| Metric | Value |
|---|---|
| Go source files | 384 |
| Test files | 159 |
| Total Go LOC | 88,081 |
| Total project LOC | 162,047 |
| Go blank lines | 11,845 |
| Go comment lines | 10,045 |

---

## 5. Summary

- **go vet**: clean
- **gofmt**: clean
- **golangci-lint**: 58 issues — dominated by `gocyclo` (34 high-complexity functions) and `prealloc` (9 slice pre-allocation hints). Test files account for ~12 of the 58 issues.
- **Unit test coverage**: 38.9% overall.
- **Integration test coverage**: 69.8% for `pkg/wiki`, 88.9% for `pkg/web/fusion`, but integration tests require Typesense and LLM endpoints to be running.
- **Coverage profile**: written to `coverage.out`; HTML report in `coverage.html`.

(End of file - total 197 lines)
