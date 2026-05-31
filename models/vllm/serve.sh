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

EMBEDDING_MODEL="${GMD_EMBED_MODEL:-google/embeddinggemma-300m}"
EMBEDDING_PORT=8001

EXPANSION_MODEL="${GMD_EXPANSION_MODEL:-Qwen/Qwen3-1.7B}"
EXPANSION_PORT=8002

RERANK_MODEL="${GMD_RERANK_MODEL:-Qwen/Qwen3-Reranker-0.6B}"
RERANK_PORT=8003

EMBEDDING_MEM_GB="${EMBEDDING_MEM_GB:-2}"
EXPANSION_MEM_GB="${EXPANSION_MEM_GB:-4}"
RERANK_MEM_GB="${RERANK_MEM_GB:-1}"

TOTAL_GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits | head -1)

start_model() {
  local name=$1 model=$2 port=$3 task=$4 mem_gb=$5
  local gpu_mem_util
  gpu_mem_util=$(awk "BEGIN {printf \"%.2f\", ($mem_gb * 1024) / $TOTAL_GPU_MEM}")
  echo "Starting $name ($model) on port $port (task: $task, mem: ${mem_gb}GB, util: $gpu_mem_util)..."
  source "$VLLM_DIR/.venv/bin/activate"
  vllm serve "$model" \
    --port "$port" \
    --task "$task" \
    --max-model-len 8192 \
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
  start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed" "$EMBEDDING_MEM_GB"
  start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate" "$EXPANSION_MEM_GB"
  start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank" "$RERANK_MEM_GB"
else
  for arg in "$@"; do
    case "$arg" in
      embedding) start_model "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "embed" "$EMBEDDING_MEM_GB" ;;
      expansion) start_model "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "generate" "$EXPANSION_MEM_GB" ;;
      rerank)    start_model "rerank" "$RERANK_MODEL" "$RERANK_PORT" "rerank" "$RERANK_MEM_GB" ;;
      *)         echo "Unknown: $arg (use: embedding, expansion, rerank)" >&2; exit 1 ;;
    esac
  done
fi

echo "All requested models started. Waiting..."
wait
