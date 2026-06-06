package main

import (
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
)

func llmConfigFromConfig(cfg *config.Config) (*llm.Client, error) {
	return llm.ResolveLLMConfig(cfg)
}
