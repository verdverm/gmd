# LLM models

gmd requires up to seven OpenAI-compatible models — you need them running before `gmd update` or `gmd query` will work. Only embedding, expansion, and rerank are required for basic search; summarizing and general models add synthesis and agent capabilities.

## Models

| Role | Required | Model ID | Params | Source |
|---|---|---|---|---|
| **embedding** | yes | `google/embeddinggemma-300m` | 300M | Google |
| **expansion** | yes | `Qwen/Qwen3-1.7B` | 1.7B | Alibaba |
| **rerank** | yes | `Qwen/Qwen3-Reranker-0.6B` | 600M | Alibaba |
| **summarizing** | no | (configurable) | — | — |
| **general-big** | no | (configurable) | — | — |
| **general-mid** | no | (configurable) | — | — |
| **general-small** | no | (configurable) | — | — |

> `Qwen/Qwen3-Reranker-0.6B` is Apache 2.0, openly available, and confirmed compatible with vLLM's `--task rerank`.

## vLLM

See [`vllm/`](vllm/) for serve scripts.

## systemd

See [`systemd/`](systemd/) for systemd service files.

## Ollama

If you prefer Ollama, the standard Ollama equivalents work as alternatives.

## Configuration

Each model role has its own `*_model`, `*_base_url`, and optional `*_api_key` in the CUE config. API keys fall back to `OPENAI_API_KEY` if not set per-role.

```cue
llm: {
  embedding_model:        "google/embeddinggemma-300m"
  embedding_base_url:     "http://localhost:8001/v1"
  expansion_model:        "Qwen/Qwen3-1.7B"
  expansion_base_url:     "http://localhost:8002/v1"
  rerank_model:           "Qwen/Qwen3-Reranker-0.6B"
  rerank_base_url:        "http://localhost:8003/v1"
  summarizing_model:      "Qwen/Qwen3.6-27B-FP8"
  summarizing_base_url:   "http://localhost:8000/v1"
  general_big_model:      "Qwen/Qwen3.6-27B-FP8"
  general_big_base_url:   "http://localhost:8000/v1"
  general_mid_model:      "Qwen/Qwen3.6-27B-FP8"
  general_mid_base_url:   "http://localhost:8000/v1"
  general_small_model:    "Qwen/Qwen3.6-27B-FP8"
  general_small_base_url:  "http://localhost:8000/v1"
}
```

Per-role API key overrides: `GMD_EMBEDDING_API_KEY`, `GMD_EXPANSION_API_KEY`, `GMD_RERANK_API_KEY`, `GMD_SUMMARIZING_API_KEY`, `GMD_GENERAL_BIG_API_KEY`, `GMD_GENERAL_MID_API_KEY`, `GMD_GENERAL_SMALL_API_KEY`.

> Note: `Qwen/Qwen3-1.7B` has thinking mode enabled by default. For query expansion, pass `enable_thinking=False` in the chat template or include `/no_think` in the system prompt to suppress it.
