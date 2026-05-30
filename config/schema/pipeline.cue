package gmd

// DefaultPipeline defines built-in default values for all pipeline parameters.
// These are unified with global and project-local config overrides.
DefaultPipeline: PipelineConfig & {
	chunk: {
		targetTokens: 900
		overlap:      0.15
		headingWeights: {
			h1: 100
			h2: 90
			h3: 80
			h4: 70
			h5: 60
			h6: 50
		}
		codeFenceWeight: 10
		newlineWeight:   1
	}
	strongSignal: {
		minScore: 0.85
		minGap:   0.15
	}
	rrf: {
		k:               60
		originalWeight:  2.0
		expansionWeight: 1.0
	}
	rerank: {
		candidateLimit: 40
		contextSize:    4096
	}
	blending: {
		thresholds: {
			top:    3
			middle: 10
		}
		weights: {
			top:    0.75
			middle: 0.60
			bottom: 0.40
		}
	}
	output: {
		defaultFormat: "cli"
		maxResults:    5
	}
}
