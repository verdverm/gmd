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

VLLM_DIR="${VLLM_DIR:-$HOME/vllm}"

# --- model configuration ---

COMMON_FLAGS=""
DRY_RUN="${DRY_RUN:-}"

EMBEDDING_MODEL="${GMD_EMBED_MODEL:-google/embeddinggemma-300m}"
EMBEDDING_PORT=8001
EMBEDDING_MEM_GB=2
EMBEDDING_FLAGS="--port $EMBEDDING_PORT --max-model-len 8192"

EXPANSION_MODEL="${GMD_EXPANSION_MODEL:-Qwen/Qwen3-1.7B}"
EXPANSION_PORT=8002
EXPANSION_MEM_GB=4
EXPANSION_FLAGS="--port $EXPANSION_PORT --max-model-len 8192"

RERANK_MODEL="${GMD_RERANK_MODEL:-Qwen/Qwen3-Reranker-0.6B}"
RERANK_PORT=8003
RERANK_MEM_GB=1
RERANK_FLAGS="--port $RERANK_PORT --max-model-len 8192"

# --- GPU detection ---

if ! command -v nvidia-smi &>/dev/null; then
  echo "Error: nvidia-smi not found. vLLM requires a GPU." >&2
  exit 1
fi

TOTAL_GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits | head -1)
if ! [[ "$TOTAL_GPU_MEM" =~ ^[0-9]+$ ]] || [ "$TOTAL_GPU_MEM" = "0" ]; then
  # Fall back to system memory (unified memory GPUs like GB10)
  TOTAL_GPU_MEM=$(awk '/^MemTotal:/ {printf "%.0f", $2/1024}' /proc/meminfo)
fi
if [ -z "$TOTAL_GPU_MEM" ] || [ "$TOTAL_GPU_MEM" = "0" ]; then
  echo "Error: No GPU memory detected." >&2
  exit 1
fi

# --- helpers ---

start_model() {
  local name=$1 model=$2 port=$3 mem_gb=$4 flags=$5
  local gpu_mem_util
  gpu_mem_util=$(awk "BEGIN {printf \"%.2f\", ($mem_gb * 1024) / $TOTAL_GPU_MEM}" 2>/dev/null || echo "0.1")
  echo "Starting $name ($model) on port $port (mem: ${mem_gb}GB, util: $gpu_mem_util)..."
  if [ -n "$DRY_RUN" ]; then
    echo "[DRY-RUN] vllm serve $model $COMMON_FLAGS $flags --gpu-memory-utilization $gpu_mem_util"
  else
    vllm serve "$model" \
      $COMMON_FLAGS \
      $flags \
      --gpu-memory-utilization "$gpu_mem_util" &
  fi
}

cleanup() {
  echo "Shutting down..."
  pkill -P $$ 2>/dev/null || true
  wait
  deactivate 2>/dev/null || true
}
trap cleanup EXIT INT TERM

source "$VLLM_DIR/.venv/bin/activate"

# --- HuggingFace checks ---

if ! command -v huggingface-cli &>/dev/null; then
  echo "Error: huggingface-cli not found. Install with: pip install huggingface-hub" >&2
  exit 1
fi

HF_HUB_CACHE="${HF_HUB_CACHE:-$HOME/.cache/huggingface/hub}"

ensure_model() {
  local name=$1 model=$2
  local model_slug="${model//\//--}"
  local cache_dir="$HF_HUB_CACHE/models--$model_slug"
  if [ -d "$cache_dir" ]; then
    echo "Model $name ($model) found in cache."
  else
    echo "Downloading $name ($model)..."
    if [ -n "$DRY_RUN" ]; then
      echo "[DRY-RUN] huggingface-cli download $model"
    else
      huggingface-cli download "$model"
    fi
  fi
}

for_each_model() {
  local action=$1
  if [ $# -eq 1 ]; then
    $action "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "$EMBEDDING_MEM_GB" "$EMBEDDING_FLAGS"
    $action "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "$EXPANSION_MEM_GB" "$EXPANSION_FLAGS"
    $action "rerank" "$RERANK_MODEL" "$RERANK_PORT" "$RERANK_MEM_GB" "$RERANK_FLAGS"
  else
    shift
    for arg in "$@"; do
      case "$arg" in
        embedding) $action "embedding" "$EMBEDDING_MODEL" "$EMBEDDING_PORT" "$EMBEDDING_MEM_GB" "$EMBEDDING_FLAGS" ;;
        expansion) $action "expansion" "$EXPANSION_MODEL" "$EXPANSION_PORT" "$EXPANSION_MEM_GB" "$EXPANSION_FLAGS" ;;
        rerank)    $action "rerank" "$RERANK_MODEL" "$RERANK_PORT" "$RERANK_MEM_GB" "$RERANK_FLAGS" ;;
        *)         echo "Unknown: $arg (use: embedding, expansion, rerank)" >&2; exit 1 ;;
      esac
    done
  fi
}

# ensure all requested models are downloaded, then start them
for_each_model ensure_model "$@"
for_each_model start_model "$@"

echo "All requested models started. Waiting..."
wait
