package main

import (
	"context"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
)

func newRegistry(cfg *config.Config) (*llm.Registry, error) {
	return llm.NewRegistry(context.Background(), cfg)
}
