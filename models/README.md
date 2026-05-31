# LLM models

gmd requires three OpenAI-compatible models — you need them running before `gmd update` or `gmd query` will work.

## Models

| Role | Model ID | Params | Source |
|---|---|---|---|
| **embedding** | `google/embeddinggemma-300m` | 300M | Google |
| **expansion** | `Qwen/Qwen3-1.7B` | 1.7B | Alibaba |
- **Qwen/Qwen3-Reranker-0.6B** is Apache 2.0, openly available, and confirmed compatible with vLLM's `--task rerank`.

## vLLM

See [`vllm/`](vllm/) for serve scripts.

## systemd

See [`systemd/`](systemd/) for systemd service files.

## Ollama

If you prefer Ollama, the standard Ollama equivalents work as alternatives.

## Environment variables

Each model can be overridden via environment variable or config:

```cue
llm: {
  embedding_model: "google/embeddinggemma-300m"
  expansion_model: "Qwen/Qwen3-1.7B"
  rerank_model:    "Qwen/Qwen3-Reranker-0.6B"
}
```

Or via env vars: `GMD_EMBED_MODEL`, `GMD_EXPANSION_MODEL`, `GMD_RERANK_MODEL`.

> Note: `Qwen/Qwen3-1.7B` has thinking mode enabled by default. For query expansion, pass `enable_thinking=False` in the chat template or include `/no_think` in the system prompt to suppress it.
