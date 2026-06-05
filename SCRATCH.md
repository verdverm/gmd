# SCRATCH

Let's expand/update .design/web-providers.md in the following ways

## Design Decisions

2. should be in the other document
5. bad, SDKs are generally preferred
6. there should be no default, see 3/4
7. wasted space / info / context
10. even an issue at this point? seems like it belongs in the .design/web-browser-advanced.md
11. We have not made a final choice

generally too many specifics, not enough high-level design decisions

## Open Questions

1. what? this does not even make sense, what are we trying to actually ask here?
2. agree
3. not at this point, perhaps with a generic http-proxy typical in many applications, note it, but do not add to plan
4. what? is there even a question-answer here? clarify
5. ok, put those details where they belong
6. subprocess calls are definitely under consideration, it was the fucking point of the question

we do not need to include a line about moved content

---

## Review of .design/web-providers.md (2025-06-05)

### [x] Issue 1: SearchProvider has no cost return path

The proposed `SearchProvider.Search()` returns `([]SearchResult, error)` with no way to
surface cost. EXA search responses include `CostDollars` (used everywhere in the current CLI
via `printCost`). Either `SearchResult` needs a `Cost` field, or the signature needs a third
return value.

Same gap on `BrowserProvider.GetContent` — the current EXA `/contents` returns costs too,
but `GetContent` returns just `(string, error)`.

**Resolved:** Added `Cost *CostSummary` field to `SearchResult`. Wrapped `GetContent` return
in `GetContentResult` struct with `Content`, `Cost`, and `Extra` fields. Both interfaces now
carry cost through a first-class field.

### [x] Issue 2: ProviderConfig struct is undefined

`ProviderConstructor` references `ProviderConfig` but this type is never defined. Given that
API keys come from env vars and per-provider config blocks are typed differently
(`EXAConfig`, `CloudflareConfig`, `LocalConfig`), using a single `ProviderConfig` type seems
like it would be an untyped bag-of-keys or require a separate resolution step.

**Resolved:** Defined `ProviderConfig` struct with `Name string` and `Extra map[string]any`.
Added Design Decision 11 (two-layer architecture): provider-specific config flows through
`Extra`, interpreted by each provider's constructor. Provider packages own their native
types; shared interfaces use a minimal uniform surface with `Extra` for provider-specific
data. Added `Extra` to `CrawlOptions`, `Page`, and `Element` for consistency.

### [x] Issue 3: agent vs research naming collision

The CLI table lists `gmd web agent` as "existing" and `gmd web research` as "new." Phase 5's
goal is to "build `gmd web research`" and refactor `agent.go` to use the provider interface.
Is `research` replacing `agent`, complementing it, or renaming it? The relationship is
unclear.

**Resolved:** Added "Command Spectrum" section to `web-providers.md` describing three tiers
that build on each other:
1. Deterministic (search, fetch, crawl) — no LLM
2. Agent — conversational, iterative, quick synthesis
3. Research — structured deep pipeline with formal reports

Updated CLI mapping table with Tier column. Updated Phase 5 and Agent Refactoring sections
to clarify agent refactoring is provider-interface modernization (not replacement by
research). Both commands coexist; research builds on patterns proven by agent. Added
generalization note to `websearch.md` pointing readers to `web-providers.md`.

### [x] Issue 5: Missing error for "provider referenced but credentials missing"

The doc says "omitting a config block means don't use this provider," but what about when a
provider group references `cloudflare` but `CLOUDFLARE_API_KEY` is unset? The error taxonomy
has `ErrAuthFailed` but no sentinel for "provider group references a provider whose config is
absent or credentials are unset."

**Resolved:** Added `ErrAuthMissing` sentinel to the error taxonomy in `web-providers.md`
alongside a clarifying comment about the distinction between not-configured and
credentials-missing.


