package wiki

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/search"
	"github.com/verdverm/gmd/pkg/ts"
)

type Agent struct {
	wiki       *Wiki
	cfg        *config.Config
	tsClient   *ts.Client
	llmClient  *llm.Client
	schema     string
	indexCache map[string]string
}

func NewAgent(wiki *Wiki, cfg *config.Config, tsClient *ts.Client, llmClient *llm.Client) *Agent {
	return &Agent{
		wiki:       wiki,
		cfg:        cfg,
		tsClient:   tsClient,
		llmClient:  llmClient,
		schema:     SchemaPrompt(),
		indexCache: make(map[string]string),
	}
}

type IngestOpts struct {
	Batch       bool
	Interactive bool
}

type IngestAction struct {
	Name          string                 `json:"name"`
	Page          string                 `json:"page"`
	Action        string                 `json:"action"`
	Content       string                 `json:"content,omitempty"`
	Frontmatter   map[string]interface{} `json:"frontmatter,omitempty"`
	LinksTo       []string               `json:"links_to,omitempty"`
	Claims        []string               `json:"claims,omitempty"`
	MergeSection  string                 `json:"merge_section,omitempty"`
	AppendContent string                 `json:"append_content,omitempty"`
}

type IngestReport struct {
	Source         string
	CreatedPages   []string
	UpdatedPages   []string
	Contradictions []string
	Errors         []string
}

type ingestResponse struct {
	SourceSummary struct {
		Title       string                 `json:"title"`
		Page        string                 `json:"page"`
		Frontmatter map[string]interface{} `json:"frontmatter"`
	} `json:"source_summary"`
	Entities       []IngestAction `json:"entities"`
	Concepts       []IngestAction `json:"concepts"`
	Comparisons    []IngestAction `json:"comparisons"`
	Contradictions []struct {
		Claim           string `json:"claim"`
		SourcePage      string `json:"source_page"`
		ContradictsPage string `json:"contradicts_page"`
		ExistingClaim   string `json:"existing_claim"`
		ResolutionHint  string `json:"resolution_hint"`
	} `json:"contradictions"`
	IndexUpdates []struct {
		Page     string `json:"page"`
		Summary  string `json:"summary"`
		Category string `json:"category"`
	} `json:"index_updates"`
	LogEntry string `json:"log_entry"`
}

func (a *Agent) Ingest(ctx context.Context, sourcePath string, opts IngestOpts) (*IngestReport, error) {
	report := &IngestReport{Source: sourcePath}

	sourceContent, err := readSource(sourcePath, a.wiki.RawPath)
	if err != nil {
		return report, fmt.Errorf("reading source: %w", err)
	}

	existingPages, err := a.loadIndexContext(ctx)
	if err != nil {
		return report, fmt.Errorf("loading index context: %w", err)
	}

	overlap, err := a.searchOverlap(ctx, sourceContent)
	if err != nil {
		return report, fmt.Errorf("searching overlap: %w", err)
	}

	if len(overlap) > 0 {
		for _, page := range overlap {
			content, err := a.readWikiPage(page)
			if err == nil {
				existingPages += fmt.Sprintf("\n\n### %s\n%s", page, truncate(content, 2000))
			}
		}
	}

	systemPrompt := IngestSystemPrompt(existingPages)
	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: sourceContent},
	}

	response, err := a.llmClient.Chat(ctx, messages)
	if err != nil {
		return report, fmt.Errorf("LLM ingest: %w", err)
	}

	var result ingestResponse
	if err := json.Unmarshal(cleanJSON(response), &result); err != nil {
		return report, fmt.Errorf("parsing LLM response: %w\nResponse: %s", err, truncate(response, 500))
	}

	var allActions []IngestAction
	allActions = append(allActions, result.Entities...)
	allActions = append(allActions, result.Concepts...)
	allActions = append(allActions, result.Comparisons...)

	if result.SourceSummary.Page != "" {
		allActions = append(allActions, IngestAction{
			Name:        result.SourceSummary.Title,
			Page:        result.SourceSummary.Page,
			Action:      "create",
			Frontmatter: result.SourceSummary.Frontmatter,
		})
	}

	for _, action := range allActions {
		switch action.Action {
		case "create":
			if err := a.createWikiPage(action); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("create %s: %v", action.Page, err))
			} else {
				report.CreatedPages = append(report.CreatedPages, action.Page)
			}
		case "update", "merge":
			if err := a.updateWikiPage(action); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("update %s: %v", action.Page, err))
			} else {
				report.UpdatedPages = append(report.UpdatedPages, action.Page)
			}
		}
	}

	for _, c := range result.Contradictions {
		report.Contradictions = append(report.Contradictions,
			fmt.Sprintf("%s: %s vs %s", c.SourcePage, c.Claim, c.ContradictsPage))
	}

	if err := a.updateIndexFile(result.IndexUpdates); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("index update: %v", err))
	}

	if err := a.appendLogFile(result.LogEntry); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("log update: %v", err))
	}

	return report, nil
}

func (a *Agent) createWikiPage(action IngestAction) error {
	pagePath := filepath.Join(a.wiki.WikiPath, action.Page)
	dir := filepath.Dir(pagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fm := action.Frontmatter
	if fm == nil {
		fm = make(map[string]interface{})
	}
	fmYAML, err := marshalYAML(fm)
	if err != nil {
		return fmt.Errorf("marshaling frontmatter: %w", err)
	}

	content := fmt.Sprintf("---\n%s\n---\n\n%s", fmYAML, action.Content)
	return os.WriteFile(pagePath, []byte(content), 0644)
}

func (a *Agent) updateWikiPage(action IngestAction) error {
	pagePath := filepath.Join(a.wiki.WikiPath, action.Page)
	existing, err := os.ReadFile(pagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return a.createWikiPage(action)
		}
		return err
	}

	content := string(existing)

	if action.MergeSection != "" {
		idx := strings.Index(content, action.MergeSection)
		if idx >= 0 {
			content = content[:idx] + action.Content + "\n\n" + content[idx:]
		}
	}

	if action.AppendContent != "" {
		content = strings.TrimRight(content, "\n") + "\n\n" + action.AppendContent + "\n"
	}

	return os.WriteFile(pagePath, []byte(content), 0644)
}

func (a *Agent) updateIndexFile(updates []struct {
	Page     string `json:"page"`
	Summary  string `json:"summary"`
	Category string `json:"category"`
}) error {
	if len(updates) == 0 {
		return nil
	}

	indexPath := a.wiki.IndexFilePath()
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	indexStr := string(content)

	for _, update := range updates {
		line := fmt.Sprintf("- [[%s]] — %s", strings.TrimSuffix(filepath.Base(update.Page), ".md"), update.Summary)
		categoryHeader := "## " + strings.Title(strings.ReplaceAll(update.Category, "_", " "))

		if !strings.Contains(indexStr, categoryHeader) {
			indexStr = strings.TrimRight(indexStr, "\n") + "\n\n" + categoryHeader + "\n" + line + "\n"
		} else {
			idx := strings.Index(indexStr, categoryHeader)
			insertPos := strings.Index(indexStr[idx:], "\n\n")
			if insertPos < 0 {
				insertPos = strings.Index(indexStr[idx:], "\n##")
			}
			if insertPos > 0 {
				insertPos += idx
				indexStr = indexStr[:insertPos] + "\n" + line + indexStr[insertPos:]
			} else {
				indexStr = strings.TrimRight(indexStr, "\n") + "\n" + line + "\n"
			}
		}
	}

	updatedLine := fmt.Sprintf("## Last Updated\n%s", time.Now().Format("2006-01-02 15:04"))
	if strings.Contains(indexStr, "## Last Updated") {
		re := strings.Index(indexStr, "## Last Updated")
		end := strings.Index(indexStr[re:], "\n##")
		if end < 0 {
			indexStr = indexStr[:re] + updatedLine
		} else {
			indexStr = indexStr[:re] + updatedLine + indexStr[re+end:]
		}
	} else {
		indexStr = strings.TrimRight(indexStr, "\n") + "\n\n" + updatedLine + "\n"
	}

	return os.WriteFile(indexPath, []byte(indexStr), 0644)
}

func (a *Agent) appendLogFile(entry string) error {
	if entry == "" {
		return nil
	}
	logPath := a.wiki.LogFilePath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		return err
	}
	logStr := strings.TrimRight(string(content), "\n") + "\n\n" + entry + "\n"
	return os.WriteFile(logPath, []byte(logStr), 0644)
}

type QueryOpts struct {
	Save  bool
	Limit int
}

type QueryResult struct {
	Answer  string
	Sources []string
}

func (a *Agent) Query(ctx context.Context, question string, opts QueryOpts) (*QueryResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 5
	}

	collections := []string{a.cfg.CollectionKey(a.wiki.Name)}
	filterBy := fmt.Sprintf("collection:=%s", collections[0])

	searchResults, err := a.tsClient.TextSearch(ctx, ts.HybridSearchParams{
		Query:       question,
		Collections: collections,
		FilterBy:    filterBy,
		Limit:       opts.Limit,
		GroupLimit:  1,
	})
	if err != nil {
		return nil, fmt.Errorf("searching wiki: %w", err)
	}

	var pageContents string
	var sources []string
	for _, r := range searchResults {
		content, err := a.readWikiPage(r.Path)
		if err != nil {
			continue
		}
		pageName := strings.TrimSuffix(strings.TrimPrefix(r.Path, "wiki/"), ".md")
		pageContents += fmt.Sprintf("### [[%s]]\n%s\n\n", pageName, content)
		sources = append(sources, r.Path)
	}

	systemPrompt := QuerySystemPrompt(pageContents)
	messages := []llm.ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}

	answer, err := a.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM query: %w", err)
	}

	result := &QueryResult{
		Answer:  answer,
		Sources: sources,
	}

	if opts.Save {
		savedPath, err := a.saveQueryResult(question, answer, sources)
		if err != nil {
			return result, fmt.Errorf("saving result: %w", err)
		}
		_ = savedPath
	}

	return result, nil
}

func (a *Agent) saveQueryResult(question, answer string, sources []string) (string, error) {
	slug := slugify(question)
	if len(slug) > 50 {
		slug = slug[:50]
	}
	filename := fmt.Sprintf("synthesis/%s-%s.md", time.Now().Format("2006-01-02"), slug)
	pagePath := filepath.Join(a.wiki.WikiPath, filename)

	dir := filepath.Dir(pagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	var sourceList string
	for _, s := range sources {
		sourceList += fmt.Sprintf("- [[%s]]\n", strings.TrimSuffix(strings.TrimPrefix(s, "wiki/"), ".md"))
	}

	content := fmt.Sprintf(`---
type: synthesis
tags: [query-result]
status: draft
sources: [%s]
---

# %s

%s

## Sources
%s
`, strings.Join(sources, ", "), question, answer, sourceList)

	return filename, os.WriteFile(pagePath, []byte(content), 0644)
}

func (a *Agent) readWikiPage(path string) (string, error) {
	fullPath := filepath.Join(a.wiki.Path, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	content := string(data)
	_, content, _ = ParseFrontmatter(content)
	return content, nil
}

func (a *Agent) loadIndexContext(ctx context.Context) (string, error) {
	indexPath := a.wiki.IndexFilePath()
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

func (a *Agent) searchOverlap(ctx context.Context, sourceContent string) ([]string, error) {
	terms := extractKeyTerms(sourceContent, 5)
	var overlapping []string
	seen := make(map[string]bool)

	collections := []string{a.cfg.CollectionKey(a.wiki.Name)}
	for _, term := range terms {
		results, err := a.tsClient.TextSearch(ctx, ts.HybridSearchParams{
			Query:       term,
			Collections: collections,
			Limit:       3,
			GroupLimit:  1,
		})
		if err != nil {
			continue
		}
		for _, r := range results {
			if !seen[r.Path] {
				overlapping = append(overlapping, r.Path)
				seen[r.Path] = true
			}
		}
	}
	return overlapping, nil
}

func extractKeyTerms(content string, n int) []string {
	lines := strings.Split(content, "\n")
	var terms []string
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || len(line) < 10 {
			continue
		}
		words := strings.Fields(line)
		for i := 0; i < len(words)-1 && len(terms) < n; i++ {
			pair := strings.ToLower(words[i] + " " + words[i+1])
			if !seen[pair] && len(words[i]) > 3 && len(words[i+1]) > 3 {
				terms = append(terms, pair)
				seen[pair] = true
			}
		}
		if len(terms) >= n {
			break
		}
	}
	return terms
}

func readSource(sourcePath, rawPath string) (string, error) {
	if strings.HasPrefix(sourcePath, "http://") || strings.HasPrefix(sourcePath, "https://") {
		return "", fmt.Errorf("URL fetching not yet implemented, save the source to raw/ first")
	}

	fullPath := sourcePath
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(rawPath, sourcePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fullPath = sourcePath
		}
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", fullPath, err)
	}
	return string(data), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func cleanJSON(s string) []byte {
	s = strings.TrimSpace(s)
	// Strip  blocks (DeepSeek-style reasoning)
	for {
		start := strings.Index(s, "<think>")
		end := strings.Index(s, "</think>")
		if start == -1 || end == -1 || end < start {
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return []byte(s)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == ' ' {
			return r
		}
		return '-'
	}, s)
	s = strings.ReplaceAll(s, " ", "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

func marshalYAML(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}

	var lines []string
	for k, v := range result {
		switch val := v.(type) {
		case string:
			lines = append(lines, fmt.Sprintf("%s: %s", k, val))
		case []interface{}:
			var items []string
			for _, item := range val {
				items = append(items, fmt.Sprintf("%v", item))
			}
			lines = append(lines, fmt.Sprintf("%s: [%s]", k, strings.Join(items, ", ")))
		default:
			lines = append(lines, fmt.Sprintf("%s: %v", k, val))
		}
	}
	return strings.Join(lines, "\n"), nil
}

var _ = chunking.ChunkMarkdown
var _ = search.New
