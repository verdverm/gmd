# Search pipeline

`gmd query` runs every search through a multi-stage pipeline:

```
gmd query "deploy config"
       │
       ▼
Strong signal check ──── if score ≥ 0.85 and gap ≥ 0.15 ────► use query directly
       │
       ▼
LLM query expansion ──── generates lex / vec / hyde variants
       │
       ▼
For each variant ──────── embed → Typesense hybrid search (text + vector, grouped by doc)
       │
       ▼
RRF fusion ────────────── Σ(w / (k + rank)) across all variants
       │
       ▼
LLM reranking ─────────── /v1/rerank endpoint (skipped if unsupported)
       │
       ▼
Position blending ─────── top/middle/bottom tiers with configurable weights
       │
       ▼
Results
```

## Stage details

**Signal detection.** Before expansion, the query is checked for a direct strong signal. If one
variant has score ≥ 0.85 and the gap to the next best is ≥ 0.15, the query passes through
unexpanded — saves a round-trip for unambiguous queries.

**Query expansion.** The LLM generates lexical variants (synonyms, rephrasings), vector variants
(more abstract semantic formulations), and HyDE (Hypothetical Document Embedding — a synthetic
answer snippet). Each variant is searched independently.

**Hybrid search.** Each variant is embedded and searched against Typesense with hybrid search
(text + vector similarity), grouped by document path to avoid chunk-level noise.

**RRF fusion.** Reciprocal Rank Fusion combines results across all variants:
`Σ(w / (k + rank))`, where `w` is variant weight and `k` is smoothing constant (default 60).

**LLM reranking.** Results pass through the `/v1/rerank` endpoint for relevance rescoring.
Skipped automatically if the provider doesn't support it.

**Position blending.** Final scores blend chunk position within the document — top chunks get
higher weight than bottom chunks, configurable in three tiers.
