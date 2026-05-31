# Configuration

gmd uses [CUE](https://cuelang.org) for configuration. Global config lives at `~/.config/gmd/config.cue` and project-local config at `<project-root>/.gmd/config.cue`. They are merged at load time — project values override global defaults.

## Global config

```cue
package gmd

Config: {
  llm: {
    base_url:            "http://localhost:11434/v1"
    api_key:             ""
    embedding_model:     "google/embeddinggemma-300m"
    expansion_model:     "Qwen/Qwen3-1.7B"
    rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
    // Optional per-model endpoint overrides (for separate vLLM servers)
    // embedding_base_url: "http://localhost:8001/v1"
    // expansion_base_url: "http://localhost:8002/v1"
    // rerank_base_url:    "http://localhost:8003/v1"
  }
  typesense: {
    host:    "http://localhost:8108"
    api_key: "xyz"
  }
  collections: docs: {
    path:    "~/documents"
    pattern: "**/*.md"
    context: "Technical documentation"
  }
}
```

If `api_key` is empty in both config and env, gmd will fail.

## Pipeline reference

All parameters have sensible defaults — you only need to set what you want to override.

| Parameter | Default | Description |
|---|---|---|
| `llm.base_url` | — | OpenAI-compatible API endpoint |
| `llm.api_key` | `OPENAI_API_KEY` env | API key |
| `llm.embedding_model` | `google/embeddinggemma-300m` | Model for embeddings |
| `llm.expansion_model` | `Qwen/Qwen3-1.7B` | Model for query expansion |
| `llm.rerank_model` | `Qwen/Qwen3-Reranker-0.6B` | Model for reranking |
| `llm.embedding_base_url` | global `base_url` | Per-model endpoint override for embeddings |
| `llm.expansion_base_url` | global `base_url` | Per-model endpoint override for query expansion |
| `llm.rerank_base_url` | global `base_url` | Per-model endpoint override for reranking |
| `typesense.host` | — | Typesense server URL |
| `typesense.api_key` | — | Typesense API key |
| `pipeline.chunk.targetTokens` | 900 | Target tokens per chunk |
| `pipeline.chunk.overlap` | 0.15 | Fraction overlap between chunks |
| `pipeline.strongSignal.minScore` | 0.85 | BM25 score threshold for strong signal |
| `pipeline.strongSignal.minGap` | 0.15 | Min gap between top 2 scores |
| `pipeline.rrf.k` | 60 | RRF rank scaling constant |
| `pipeline.rrf.originalWeight` | 2.0 | RRF weight for original query |
| `pipeline.rrf.expansionWeight` | 1.0 | RRF weight for expansion variants |
| `pipeline.rerank.candidateLimit` | 40 | Max docs to rerank |
| `pipeline.rerank.contextSize` | 4096 | Token budget per doc for reranking |
| `pipeline.blending.thresholds.top` | 3 | Rank cutoff for top tier |
| `pipeline.blending.thresholds.middle` | 10 | Rank cutoff for middle tier |
| `pipeline.blending.weights.top` | 0.75 | RRF weight in top tier |
| `pipeline.blending.weights.middle` | 0.60 | RRF weight in middle tier |
| `pipeline.blending.weights.bottom` | 0.40 | RRF weight in bottom tier |
| `pipeline.output.defaultFormat` | `cli` | Output format |
| `pipeline.output.maxResults` | 5 | Default result count |
