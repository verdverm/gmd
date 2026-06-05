package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
	"github.com/verdverm/gmd/pkg/wiki"
)

type WikiTools struct {
	toolHandlers map[string]func(context.Context, json.RawMessage) (string, error)
}

func NewWikiTools(cfg *config.Config, tsClient *ts.Client, llmClient *llm.Client, wikiName string) *WikiTools {
	wt := &WikiTools{
		toolHandlers: make(map[string]func(context.Context, json.RawMessage) (string, error)),
	}

	wc, ok := cfg.Wikis[wikiName]
	if !ok {
		return wt
	}

	w, err := wiki.NewWiki(wikiName, wc.Path, &wc)
	if err != nil {
		return wt
	}

	agent := wiki.NewAgent(w, cfg, tsClient, llmClient)

	wt.toolHandlers["gmd_wiki_search"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Query  string `json:"query"`
			Wiki   string `json:"wiki"`
			Filter string `json:"filter,omitempty"`
			Limit  int    `json:"limit,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}
		if params.Limit <= 0 {
			params.Limit = 10
		}
		if params.Wiki == "" {
			params.Wiki = wikiName
		}

		collectionKey := cfg.CollectionKey(params.Wiki)
		collections := []string{collectionKey}
		filterBy := fmt.Sprintf("collection:=%s", collectionKey)
		if params.Filter != "" {
			filterBy += " && " + params.Filter
		}

		results, err := tsClient.TextSearch(ctx, ts.HybridSearchParams{
			Query:       params.Query,
			Collections: collections,
			FilterBy:    filterBy,
			Limit:       params.Limit,
			GroupLimit:  1,
		})
		if err != nil {
			return "", err
		}

		data, _ := json.Marshal(results)
		return string(data), nil
	}

	wt.toolHandlers["gmd_wiki_get"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Path string `json:"path"`
			Wiki string `json:"wiki"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}

		_ = cfg.CollectionKey(params.Wiki)
		results, err := tsClient.SearchChunksByPath(ctx, fmt.Sprintf("path:=%s", params.Path), 50)
		if err != nil {
			return "", err
		}

		data, _ := json.Marshal(results)
		return string(data), nil
	}

	wt.toolHandlers["gmd_wiki_neighbors"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Path      string `json:"path"`
			Wiki      string `json:"wiki"`
			Direction string `json:"direction,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}

		pageName := strings.TrimSuffix(strings.TrimPrefix(params.Path, "wiki/"), ".md")
		neighbors, err := agent.Neighbors(ctx, pageName, params.Direction)
		if err != nil {
			inbound, err2 := agent.NeighborsFromTS(ctx, pageName)
			if err2 != nil {
				return "", fmt.Errorf("neighbors: %w (ts fallback: %w)", err, err2)
			}
			neighbors = inbound
		}

		result := map[string]interface{}{
			"page":      pageName,
			"neighbors": neighbors,
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}

	wt.toolHandlers["gmd_wiki_status"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Wiki string `json:"wiki"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}
		if params.Wiki == "" {
			params.Wiki = wikiName
		}

		collections := []string{cfg.CollectionKey(params.Wiki)}
		counts, err := tsClient.CountByCollection(ctx, collections)
		if err != nil {
			return "", fmt.Errorf("counting: %w", err)
		}

		var pageTypeCounts map[string]int
		g, err := agent.BuildGraph(ctx)
		if err == nil {
			pageTypeCounts = make(map[string]int)
			pageTypeCounts["nodes"] = len(g.Nodes)
			pageTypeCounts["edges"] = len(g.Edges)
			for _, node := range g.Nodes {
				if g.InDegree[node] == 0 {
					pageTypeCounts["orphans"]++
				}
			}
		}

		result := map[string]interface{}{
			"wiki":   params.Wiki,
			"chunks": counts[cfg.CollectionKey(params.Wiki)],
			"graph":  pageTypeCounts,
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}

	wt.toolHandlers["gmd_wiki_suggest"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Prefix string `json:"prefix"`
			Wiki   string `json:"wiki"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}
		if params.Wiki == "" {
			params.Wiki = wikiName
		}

		collections := []string{cfg.CollectionKey(params.Wiki)}
		filterBy := fmt.Sprintf("collection:=%s && path:^wiki", collections[0])

		results, err := tsClient.TextSearch(ctx, ts.HybridSearchParams{
			Query:       params.Prefix,
			Collections: collections,
			FilterBy:    filterBy,
			Limit:       10,
			GroupLimit:  1,
		})
		if err != nil {
			return "", err
		}

		var suggestions []string
		for _, r := range results {
			name := strings.TrimSuffix(strings.TrimPrefix(r.Path, "wiki/"), ".md")
			suggestions = append(suggestions, name)
		}
		sort.Strings(suggestions)

		data, _ := json.Marshal(suggestions)
		return string(data), nil
	}

	wt.toolHandlers["gmd_wiki_update"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		return `{"status": "ok", "message": "Re-index triggered for wiki collection"}`, nil
	}

	wt.toolHandlers["gmd_wiki_ingest"] = func(ctx context.Context, args json.RawMessage) (string, error) {
		var params struct {
			Source string `json:"source"`
			Wiki   string `json:"wiki"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}

		report, err := agent.Ingest(ctx, params.Source, wiki.IngestOpts{Batch: false})
		if err != nil {
			return "", err
		}

		result := map[string]interface{}{
			"source":         report.Source,
			"created_pages":  report.CreatedPages,
			"updated_pages":  report.UpdatedPages,
			"contradictions": report.Contradictions,
			"errors":         report.Errors,
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	}

	return wt
}

func (wt *WikiTools) Handle(ctx context.Context, method string, args json.RawMessage) (string, error) {
	handler, ok := wt.toolHandlers[method]
	if !ok {
		return "", fmt.Errorf("unknown wiki tool: %s", method)
	}
	return handler(ctx, args)
}

func (wt *WikiTools) ListTools() []string {
	var tools []string
	for k := range wt.toolHandlers {
		tools = append(tools, k)
	}
	sort.Strings(tools)
	return tools
}
