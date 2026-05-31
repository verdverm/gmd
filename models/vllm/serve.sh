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

VLLM_DIR="${VLLM_DIR:-/home/ubuntu/vllm}"

# --- model configuration ---

EMBEDDING_MODEL="${GMD_EMBED_MODEL:-google/embeddinggemma-300m}"
EMBEDDING_PORT=8001
EMBEDDING_MEM_GB=2
EMBEDDING_FLAGS="--max-model-len 8192"

EXPANSION_MODEL="${GMD_EXPANSION_MODEL:-Qwen/Qwen3-1.7B}"
EXPANSION_PORT=8002
EXPANSION_MEM_GB=4
EXPANSION_FLAGS="--max-model-len 8192"

RERANK_MODEL="${GMD_RERANK_MODEL:-Qwen/Qwen3-Reranker-0.6B}"
RERANK_PORT=8003
RERANK_MEM_GB=1
RERANK_FLAGS="--max-model-len 8192"

# --- GPU detection ---

if ! command -v nvidia-smi &>/dev/null; then
  echo "Error: nvidia-smi not found. vLLM requires a GPU." >&2
  exit 1
fi

TOTAL_GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits | head -1)
if [ -z "$TOTAL_GPU_MEM" ] || [ "$TOTAL_GPU_MEM" = "0" ]; then
  echo "Error: No GPU memory detected." >&2
  exit 1
fi

# --- helpers ---

start_model() {
  local name=$1 model=$2 port=$3 task=$4 mem_gb=$5 flags=$6
  local gpu_mem_util
  gpu_mem_util=$(awk "BEGIN {printf \"%.2f\", ($mem_gb * 1024) / $TOTAL_GPU_MEM}")
  echo "Starting $name ($model) on port $port (task: $task, mem: ${mem_gb}GB, util: $gpu_mem_util)..."
  source "$VLLM_DIR/.venv/bin/activate"
  vllm serve "$model" \
    --port "$port" \
    --task "$task" \
    $flags \
    --gpu-memory-utilization "$gpu_mem_util" \
    --enforce-eager &
}

cleanup() {
  echo "Shutting down..."
  pkill -P $$ 2>/dev/null || true
  wait
}
trap cleanup EXIT INT TERM

if [ $# -eq 0 ]; then
  start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed" "$EMBEDDING_MEM_GB" "$EMBEDDING_FLAGS"
  start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate" "$EXPANSION_MEM_GB" "$EXPANSION_FLAGS"
  start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank" "$RERANK_MEM_GB" "$RERANK_FLAGS"
else
  for arg in "$@"; do
    case "$arg" in
      embedding) start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed" "$EMBEDDING_MEM_GB" "$EMBEDDING_FLAGS" ;;
      expansion) start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate" "$EXPANSION_MEM_GB" "$EXPANSION_FLAGS" ;;
      rerank)    start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank" "$RERANK_MEM_GB" "$RERANK_FLAGS" ;;
      *)         echo "Unknown: $arg (use: embedding, expansion, rerank)" >&2; exit 1 ;;
    esac
  done
fi

echo "All requested models started. Waiting..."
wait
