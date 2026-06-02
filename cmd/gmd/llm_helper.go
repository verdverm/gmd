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
		EmbeddingAPIKey:     cfg.LLM.EmbeddingAPIKey,
		ExpansionAPIKey:     cfg.LLM.ExpansionAPIKey,
		RerankAPIKey:        cfg.LLM.RerankAPIKey,
		SummarizingModel:    cfg.LLM.SummarizingModel,
		SummarizingBaseURL:  cfg.LLM.SummarizingBaseURL,
		SummarizingAPIKey:   cfg.LLM.SummarizingAPIKey,
		GeneralBigModel:     cfg.LLM.GeneralBigModel,
		GeneralBigBaseURL:   cfg.LLM.GeneralBigBaseURL,
		GeneralBigAPIKey:    cfg.LLM.GeneralBigAPIKey,
		GeneralMidModel:     cfg.LLM.GeneralMidModel,
		GeneralMidBaseURL:   cfg.LLM.GeneralMidBaseURL,
		GeneralMidAPIKey:    cfg.LLM.GeneralMidAPIKey,
		GeneralSmallModel:   cfg.LLM.GeneralSmallModel,
		GeneralSmallBaseURL: cfg.LLM.GeneralSmallBaseURL,
		GeneralSmallAPIKey:  cfg.LLM.GeneralSmallAPIKey,
	}
}
