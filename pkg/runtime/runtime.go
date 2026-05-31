package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/ts"
)

// Runtime is the core engine that orchestrates indexing, search, and lifecycle.
// It owns the Typesense client. There is no operational database.
type Runtime struct {
	mu       sync.RWMutex
	cfg      *config.Config
	tsClient *ts.Client
	closed   bool
}

// Open creates and initializes a new Runtime from configuration.
func Open(cfg *config.Config) (*Runtime, error) {
	r := &Runtime{
		cfg: cfg,
	}

	ctx := context.Background()

	r.tsClient = ts.New(ts.Config{
		Host:   cfg.Typesense.Host,
		APIKey: cfg.Typesense.APIKey,
	})

	if err := r.tsClient.EnsureSchema(ctx, 0); err != nil {
		return nil, fmt.Errorf("ensuring typesense schema: %w", err)
	}

	return r, nil
}

// Close shuts down the runtime and releases resources.
func (r *Runtime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

// Config returns the runtime's configuration.
func (r *Runtime) Config() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// TSClient returns the Typesense client wrapper.
func (r *Runtime) TSClient() *ts.Client {
	return r.tsClient
}
