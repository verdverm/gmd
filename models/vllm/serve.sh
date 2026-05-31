#!/usr/bin/env bash
set -euo pipefail

# Start all three LLM models required by gmd using vLLM.
# Each model runs on a separate port. Point gmd's base_url at a reverse
# proxy that routes /v1/embeddings, /v1/chat/completions, and /v1/rerank
# to the correct backend port.
#
# Usage:
#   ./serve.sh                    # start all three
#   ./serve.sh embedding          # start one
#   ./serve.sh expansion rerank   # start specific models
#
# Prerequisites:
#   pip install vllm
#   huggingface-cli login  # required for google/embeddinggemma-300m (gated)

CUDA_VISIBLE_DEVICES="${CUDA_VISIBLE_DEVICES:-0}"

EMBEDDING_MODEL="${GMD_EMBED_MODEL:-google/embeddinggemma-300m}"
EMBEDDING_PORT=8001

EXPANSION_MODEL="${GMD_EXPANSION_MODEL:-Qwen/Qwen3-1.7B}"
EXPANSION_PORT=8002

RERANK_MODEL="${GMD_RERANK_MODEL:-Qwen/Qwen3-Reranker-0.6B}"
RERANK_PORT=8003

start_model() {
  local name=$1 model=$2 port=$3 task=$4
  echo "Starting $name ($model) on port $port (task: $task)..."
  vllm serve "$model" \
    --port "$port" \
    --task "$task" \
    --max-model-len 8192 \
    --gpu-memory-utilization 0.30 \
    --enforce-eager &
}

cleanup() {
  echo "Shutting down..."
  pkill -P $$ 2>/dev/null || true
  wait
}
trap cleanup EXIT INT TERM

if [ $# -eq 0 ]; then
  start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed"
  start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate"
  start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank"
else
  for arg in "$@"; do
    case "$arg" in
      embedding) start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed" ;;
      expansion) start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate" ;;
      rerank)    start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank" ;;
      *)         echo "Unknown: $arg (use: embedding, expansion, rerank)" >&2; exit 1 ;;
    esac
  done
fi

echo "All requested models started. Waiting..."
wait
