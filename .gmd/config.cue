package gmd

Config: {
	llm: {
		providers: {
			vllm8000: {
				provider: "openai"
				base_url: "http://192.168.4.31:8000/v1"
				auth: "none"
				features: { embed: false, chat: true, rerank: false }
			}
			vllm8001: {
				provider: "openai"
				base_url: "http://192.168.4.31:8001/v1"
				auth: "none"
				features: { embed: true, chat: false, rerank: false }
			}
			vllm8002: {
				provider: "openai"
				base_url: "http://192.168.4.31:8002/v1"
				auth: "none"
				features: { embed: false, chat: true, rerank: false }
			}
			vllm8003: {
				provider: "openai"
				base_url: "http://192.168.4.31:8003/v1"
				auth: "none"
				features: { embed: false, chat: false, rerank: true }
			}
		}
		profiles: {
			default: {
				embedding:     { provider: "vllm8001", model: "google/embeddinggemma-300m" }
				expansion:     { provider: "vllm8002", model: "Qwen/Qwen3-1.7B" }
				rerank:        { provider: "vllm8003", model: "Qwen/Qwen3-Reranker-0.6B" }
				summarizing:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
				general_big:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
				general_mid:   { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
				general_small: { provider: "vllm8000", model: "Qwen/Qwen3.6-27B-FP8" }
			}
		}
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
