package gmd

Config: {
	project: "{{PROJECT}}"
	llm: {
		embedding_base_url:    "http://localhost:8001/v1"
		expansion_base_url:    "http://localhost:8002/v1"
		rerank_base_url:       "http://localhost:8003/v1"
		embedding_model:       "google/embeddinggemma-300m"
		expansion_model:       "Qwen/Qwen3-1.7B"
		rerank_model:          "Qwen/Qwen3-Reranker-0.6B"
		summarizing_base_url:   "http://localhost:8000/v1"
		general_big_base_url:   "http://localhost:8000/v1"
		general_mid_base_url:   "http://localhost:8000/v1"
		general_small_base_url: "http://localhost:8000/v1"
	}
	typesense: {
		host: "http://localhost:8108"
	}
	collections: docs: {
		path:    "."
		pattern: "**/*.md"
		context: "Project documentation"
	}
}
