package gmd

Config: {
	llm: {
		embedding_base_url:  "http://192.168.4.31:8001/v1"
		expansion_base_url:  "http://192.168.4.31:8002/v1"
		rerank_base_url:     "http://192.168.4.31:8003/v1"
		api_key:             ""
		embedding_model:     "google/embeddinggemma-300m"
		expansion_model:     "Qwen/Qwen3-1.7B"
		rerank_model:        "Qwen/Qwen3-Reranker-0.6B"
	}
	typesense: {
		host:    "http://192.168.4.26:30336"
		api_key: "fnBHJWjCw1BBZC8DSvE9X99aDj6goWW0cMukKJ6nv3WUmuX8vaqoM1z/y31C34ob9C3AL8MhHeOXqwxBnPzULRNtLGQxrnQv1aRSJKvsp+vnd9sbCSLHcdN6YB7ZAicTE/YBrrACYCPrZKVmeBzeSk9Fa6+vgEitt/CRomL1CUPB/DstFw/uJdvdqrS44fsYGup5gYgLjLU3eIF846u5wc5KgF1jNm17uZJVLYjMq12gZRMhr7CXSaggvmwdn56A1VwTJs9GnSZS9EyhrKoVNGnW/ruX8f5RZac0Dww+EXUQwuRvCIkAlySz6rjIogWqAkR5i9fy9qdkyvRM3DcpjQ=="
	}
	collections: docs: {
		path:    "."
		pattern: "**/*.md"
		context: "Project documentation"
	}
}
