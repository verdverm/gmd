package gmd

Config: {
	llm: {
		embedding_model:     "google/embeddinggemma-300m"
		embedding_base_url:  "http://192.168.4.31:8001/v1"

		expansion_model:     "Qwen/Qwen3-1.7B"
		expansion_base_url:  "http://192.168.4.31:8002/v1"

		rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
		rerank_base_url:     "http://192.168.4.31:8003/v1"

		summarizing_model:      "Qwen/Qwen3.6-27B-FP8"
		summarizing_base_url:   "http://192.168.4.31:8000/v1"
		general_big_model:      "Qwen/Qwen3.6-27B-FP8"
		general_big_base_url:   "http://192.168.4.31:8000/v1"
		general_mid_model:      "Qwen/Qwen3.6-27B-FP8"
		general_mid_base_url:   "http://192.168.4.31:8000/v1"
		general_small_model:    "Qwen/Qwen3.6-27B-FP8"
		general_small_base_url: "http://192.168.4.31:8000/v1"
	}
	typesense: {
		host:    "http://192.168.4.26:31855"
	}
	collections: docs: {
		path:    "."
		patterns: ["**/*.md"]
		ignore:  [
			"qmd/**",
			"tmp/**",
			"pkg/agentsmd/content/**",
			"pkg/wiki/skills/**",
		]
		context: "Project documentation"

		// Optional: define frontmatter fields to index for faceted search/filtering.
		// Fields must match YAML frontmatter keys in your markdown files.
		// Supported types: string, string[], int32, float, bool
		// fields: {
		// 	tags:  { type: "string[]", facet: true }
		// 	author: { type: "string", facet: true }
		// 	rating: { type: "float", sort: true }
		// }
	}
}
