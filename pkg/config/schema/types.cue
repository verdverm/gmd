package gmd

// LLMConfig defines the OpenAI-compatible provider settings.
// Each model role has its own endpoint URL (vLLM needs separate servers per model).
LLMConfig: {
	embedding_model:     string | *"google/embeddinggemma-300m"
	expansion_model:     string | *"Qwen/Qwen3-1.7B"
	rerank_model:        string | *"Qwen/Qwen3-Reranker-0.6B"
	embedding_base_url:  string
	expansion_base_url:  string
	rerank_base_url:     string
}

// TypesenseConfig defines the search engine connection settings.
TypesenseConfig: {
	host:    string
}

// ChunkConfig defines heading-aware chunking parameters.
ChunkConfig: {
	targetTokens:     int | *900
	overlap:          number | *0.15
	headingWeights: {
		h1: int | *100
		h2: int | *90
		h3: int | *80
		h4: int | *70
		h5: int | *60
		h6: int | *50
	}
	codeFenceWeight: int    | *10
	newlineWeight:   number | *1
}

// StrongSignalConfig defines the heuristic for detecting strong query signals.
StrongSignalConfig: {
	minScore: number | *0.85
	minGap:   number | *0.15
}

// RRFConfig defines rank fusion parameters.
RRFConfig: {
	k:               int | *60
	originalWeight:  number | *2.0
	expansionWeight: number | *1.0
}

// RerankConfig defines reranking pipeline parameters.
RerankConfig: {
	candidateLimit: int    | *40
	contextSize:    int    | *4096
}

// BlendingConfig defines position-aware blending thresholds and weights.
BlendingConfig: {
	thresholds: {
		top:    int | *3
		middle: int | *10
	}
	weights: {
		top:    number | *0.75
		middle: number | *0.60
		bottom: number | *0.40
	}
}

// OutputConfig defines result formatting parameters.
OutputConfig: {
	defaultFormat: "cli" | "json" | "csv" | "md" | "xml" | "files" | *"cli"
	maxResults:    int | *5
}

// PipelineConfig groups all pipeline knobs.
PipelineConfig: {
	chunk:        ChunkConfig
	strongSignal: StrongSignalConfig
	rrf:          RRFConfig
	rerank:       RerankConfig
	blending:     BlendingConfig
	output:       OutputConfig
}

// CollectionConfig defines a document collection to index.
// The collection name is the map key, not a field inside the struct.
CollectionConfig: {
	path:              string
	pattern:           string
	ignore?: [...string]
	context?:          string
	includeByDefault?: bool | *true
}

// ProjectConfig is the root configuration object.
ProjectConfig: {
	llm:         LLMConfig
	typesense:   TypesenseConfig
	pipeline?:   PipelineConfig
	collections: [string]: CollectionConfig
}
