package main

import (
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
)

func llmConfigFromConfig(cfg *config.Config) llm.Config {
	return llm.Config{
		APIKey:              cfg.LLM.APIKey,
		EmbeddingModel:      cfg.LLM.EmbeddingModel,
		ExpansionModel:      cfg.LLM.ExpansionModel,
		RerankModel:         cfg.LLM.RerankModel,
		EmbedURL:            cfg.LLM.EmbeddingBaseURL,
		ExpandURL:           cfg.LLM.ExpansionBaseURL,
		RerankURL:           cfg.LLM.RerankBaseURL,
		SummarizingModel:    cfg.LLM.SummarizingModel,
		SummarizingBaseURL:  cfg.LLM.SummarizingBaseURL,
		GeneralBigModel:     cfg.LLM.GeneralBigModel,
		GeneralBigBaseURL:   cfg.LLM.GeneralBigBaseURL,
		GeneralMidModel:     cfg.LLM.GeneralMidModel,
		GeneralMidBaseURL:   cfg.LLM.GeneralMidBaseURL,
		GeneralSmallModel:   cfg.LLM.GeneralSmallModel,
		GeneralSmallBaseURL: cfg.LLM.GeneralSmallBaseURL,
	}
}
