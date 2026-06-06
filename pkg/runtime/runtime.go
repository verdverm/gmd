package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/ts"
)

// mapping from config FrontmatterField type strings to Typesense field types.
var configTypeToTS = map[string]string{
	"string":   "string",
	"string[]": "string[]",
	"int32":    "int32",
	"float":    "float",
	"bool":     "bool",
}

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

	fieldSchemas, err := buildFieldSchemas(cfg.Collections)
	if err != nil {
		return nil, fmt.Errorf("building field schemas: %w", err)
	}

	if err := r.tsClient.EnsureAllSchemas(ctx, 0, fieldSchemas); err != nil {
		return nil, fmt.Errorf("ensuring typesense schema: %w", err)
	}

	return r, nil
}

// buildFieldSchemas merges per-collection frontmatter field definitions into a single
// slice of SchemaFields, validating that no two collections use the same field name
// with different Typesense types.
func buildFieldSchemas(collections map[string]config.CollectionConfig) ([]ts.SchemaField, error) {
	var fields []ts.SchemaField
	seen := make(map[string]string) // field name -> Typesense type
	for _, col := range collections {
		for name, f := range col.Fields {
			tsType, ok := configTypeToTS[f.Type]
			if !ok {
				return nil, fmt.Errorf("field %q: unknown type %q", name, f.Type)
			}
			if existing, ok := seen[name]; ok {
				if existing != tsType {
					return nil, fmt.Errorf("field %q has conflicting types across collections: %q vs %q", name, existing, tsType)
				}
				continue
			}
			seen[name] = tsType
			fields = append(fields, ts.SchemaField{
				Name:  name,
				Type:  tsType,
				Facet: f.Facet,
				Sort:  f.Sort,
			})
		}
	}
	return fields, nil
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
	if r == nil {
		return &config.Config{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cfg
}

// TSClient returns the Typesense client wrapper.
func (r *Runtime) TSClient() *ts.Client {
	return r.tsClient
}
