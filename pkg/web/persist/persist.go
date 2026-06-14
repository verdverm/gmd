package persist

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verdverm/gmd/pkg/web"
	"github.com/verdverm/gmd/pkg/web/fusion"
)

// Metadata captures invocation details for full reproducibility.
type Metadata struct {
	Timestamp     string            `json:"timestamp"`
	Caller        string            `json:"caller"`
	Command       string            `json:"command"`
	ProviderGroup string            `json:"providerGroup,omitempty"`
	Query         string            `json:"query,omitempty"`
	URL           string            `json:"url,omitempty"`
	StartURL      string            `json:"startUrl,omitempty"`
	Providers     []string          `json:"providers,omitempty"`
	LLMProfile    string            `json:"llmProfile,omitempty"`
	LLMModel      string            `json:"llmModel,omitempty"`
	Failures      map[string]string `json:"failures,omitempty"`
	Flags         map[string]any    `json:"flags,omitempty"`
}

// Fetch saves content from a single URL fetch.
// Creates: <dir>/fetch/<timestamp>-<slug>/result.json, metadata.json, content.md
func Fetch(dir string, urlStr string, result *web.GetContentResult, meta Metadata) error {
	if meta.Timestamp == "" {
		meta.Timestamp = timestamp()
	}
	meta.Command = "fetch"
	meta.URL = urlStr

	slug := urlSlug(urlStr)
	workDir := filepath.Join(dir, "fetch", timestampDir(meta.Timestamp, slug))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("creating persist directory: %w", err)
	}

	if result != nil {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		if err := os.WriteFile(filepath.Join(workDir, "result.json"), data, 0644); err != nil {
			return fmt.Errorf("writing result.json: %w", err)
		}

		if result.Content != "" {
			title := urlStr
			if t, ok := result.Extra["title"].(string); ok && t != "" {
				title = t
			}
			if err := writeContentMD(workDir, "content.md", urlStr, title, meta.Timestamp, result.Content); err != nil {
				return err
			}
		}
	}

	return writeMetadata(workDir, meta)
}

// Crawl saves all pages from a crawl.
// Creates: <dir>/crawl/<timestamp>-<slug>/result.json, metadata.json, pages/*.md
func Crawl(dir string, startURL string, pages []web.Page, meta Metadata) error {
	if meta.Timestamp == "" {
		meta.Timestamp = timestamp()
	}
	meta.Command = "crawl"
	meta.StartURL = startURL

	slug := urlSlug(startURL)
	workDir := filepath.Join(dir, "crawl", timestampDir(meta.Timestamp, slug))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("creating persist directory: %w", err)
	}

	if pages != nil {
		data, err := json.MarshalIndent(pages, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		if err := os.WriteFile(filepath.Join(workDir, "result.json"), data, 0644); err != nil {
			return fmt.Errorf("writing result.json: %w", err)
		}

		if len(pages) > 0 {
			pagesDir := filepath.Join(workDir, "pages")
			if err := os.MkdirAll(pagesDir, 0755); err != nil {
				return fmt.Errorf("creating pages directory: %w", err)
			}
			for i, p := range pages {
				filename := fmt.Sprintf("%06d-%s.md", i+1, web.Sluggify(p.Title))
				if err := writeContentMD(pagesDir, filename, p.URL, p.Title, meta.Timestamp, p.Content); err != nil {
					return err
				}
			}
		}
	}

	return writeMetadata(workDir, meta)
}

// Search saves fused search results AND raw per-provider results.
// Creates: <dir>/search/<timestamp>-<slug>/result.json, metadata.json, query.txt,
//
//	answer.md, results/*.md, raw/<provider>.json
//
//nolint:gocyclo
func Search(dir string, query string, result *fusion.Result, rawResults map[string][]web.SearchResult, meta Metadata) error {
	if meta.Timestamp == "" {
		meta.Timestamp = timestamp()
	}
	meta.Command = "search"
	meta.Query = query

	slug := web.Sluggify(query)
	workDir := filepath.Join(dir, "search", timestampDir(meta.Timestamp, slug))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("creating persist directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(workDir, "query.txt"), []byte(query), 0644); err != nil {
		return fmt.Errorf("writing query.txt: %w", err)
	}

	if result != nil {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		if err := os.WriteFile(filepath.Join(workDir, "result.json"), data, 0644); err != nil {
			return fmt.Errorf("writing result.json: %w", err)
		}

		if result.Answer != "" {
			if err := os.WriteFile(filepath.Join(workDir, "answer.md"), []byte(result.Answer), 0644); err != nil {
				return fmt.Errorf("writing answer.md: %w", err)
			}
		}

		if len(result.Results) > 0 {
			resultsDir := filepath.Join(workDir, "results")
			if err := os.MkdirAll(resultsDir, 0755); err != nil {
				return fmt.Errorf("creating results directory: %w", err)
			}
			for i, r := range result.Results {
				provider, _ := r.Extra["_provider"].(string)
				filename := fmt.Sprintf("%04d-%s.md", i+1, web.Sluggify(r.Title))
				if err := writeSearchResultMD(resultsDir, filename, r.URL, r.Title, provider, r.Score, r.Content); err != nil {
					return err
				}
			}
		}

		if meta.Failures == nil {
			meta.Failures = result.Failures
		}
	}

	if len(rawResults) > 0 {
		rawDir := filepath.Join(workDir, "raw")
		if err := os.MkdirAll(rawDir, 0755); err != nil {
			return fmt.Errorf("creating raw directory: %w", err)
		}
		for provider, results := range rawResults {
			cleaned := make([]web.SearchResult, len(results))
			for i, r := range results {
				cleaned[i] = r
				cp := make(map[string]any, len(r.Extra))
				for k, v := range r.Extra {
					if k != "_provider" {
						cp[k] = v
					}
				}
				cleaned[i].Extra = cp
			}
			data, err := json.MarshalIndent(cleaned, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling raw %s results: %w", provider, err)
			}
			destFile := filepath.Join(rawDir, provider+".json")
			if err := os.WriteFile(destFile, data, 0644); err != nil {
				return fmt.Errorf("writing raw/%s.json: %w", provider, err)
			}
		}
	}

	return writeMetadata(workDir, meta)
}

// Agent saves agent multi-step search results.
// Creates: <dir>/agent/<timestamp>-<slug>/result.json, metadata.json, query.txt,
//
//	answer.md, steps/*.json, sources/*.md
//
//nolint:gocyclo
func Agent(dir string, query string, result *web.AgentResult, steps []json.RawMessage, meta Metadata) error {
	if meta.Timestamp == "" {
		meta.Timestamp = timestamp()
	}
	meta.Command = "agent"
	meta.Query = query

	slug := web.Sluggify(query)
	workDir := filepath.Join(dir, "agent", timestampDir(meta.Timestamp, slug))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("creating persist directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(workDir, "query.txt"), []byte(query), 0644); err != nil {
		return fmt.Errorf("writing query.txt: %w", err)
	}

	if result != nil {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling result: %w", err)
		}
		if err := os.WriteFile(filepath.Join(workDir, "result.json"), data, 0644); err != nil {
			return fmt.Errorf("writing result.json: %w", err)
		}

		if result.Answer != "" {
			if err := os.WriteFile(filepath.Join(workDir, "answer.md"), []byte(result.Answer), 0644); err != nil {
				return fmt.Errorf("writing answer.md: %w", err)
			}
		}

		if len(result.Sources) > 0 {
			sourcesDir := filepath.Join(workDir, "sources")
			if err := os.MkdirAll(sourcesDir, 0755); err != nil {
				return fmt.Errorf("creating sources directory: %w", err)
			}
			for i, s := range result.Sources {
				filename := fmt.Sprintf("%04d-%s.md", i+1, web.Sluggify(s.Title))
				if err := writeContentMD(sourcesDir, filename, s.URL, s.Title, meta.Timestamp, s.Text); err != nil {
					return err
				}
			}
		}
	}

	if len(steps) > 0 {
		stepsDir := filepath.Join(workDir, "steps")
		if err := os.MkdirAll(stepsDir, 0755); err != nil {
			return fmt.Errorf("creating steps directory: %w", err)
		}
		for i, step := range steps {
			if step == nil {
				continue
			}
			filename := fmt.Sprintf("%04d-search.json", i+1)
			if err := os.WriteFile(filepath.Join(stepsDir, filename), step, 0644); err != nil {
				return fmt.Errorf("writing step %d: %w", i+1, err)
			}
		}
	}

	return writeMetadata(workDir, meta)
}

func timestamp() string {
	t := time.Now().UTC()
	s := t.Format("2006-01-02T15:04:05.000000000Z07:00")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "+", "_")
	return s
}

func timestampDir(ts, slug string) string {
	clean := strings.ReplaceAll(ts, "_", "-")
	if idx := strings.LastIndex(clean, "-"); idx > 0 {
		clean = clean[:idx] + "." + clean[idx+1:]
	}
	return clean + "-" + slug
}

func urlSlug(rawURL string) string {
	if rawURL == "" {
		return "untitled"
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return web.Sluggify(rawURL)
	}
	parts := strings.Split(u.Host, ".")
	host := strings.Join(parts, "-")
	path := strings.Trim(u.Path, "/")
	if path != "" {
		pathParts := strings.Split(path, "/")
		last := pathParts[len(pathParts)-1]
		if last != "" {
			return web.Sluggify(host + "-" + last)
		}
		if len(pathParts) >= 2 {
			second := pathParts[len(pathParts)-2]
			if second != "" {
				return web.Sluggify(host + "-" + second)
			}
		}
	}
	return web.Sluggify(host)
}

func writeMetadata(dir string, meta Metadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), data, 0644); err != nil {
		return fmt.Errorf("writing metadata.json: %w", err)
	}
	return nil
}

func writeContentMD(dir, filename, urlStr, title, ts, content string) error {
	frontmatter := fmt.Sprintf("---\nurl: %s\ntitle: %s\nretrieved: %s\n---\n\n", urlStr, title, ts)
	full := frontmatter + content
	return os.WriteFile(filepath.Join(dir, filename), []byte(full), 0644)
}

func writeSearchResultMD(dir, filename, urlStr, title, provider string, score float64, content string) error {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("url: %s\n", urlStr))
	sb.WriteString(fmt.Sprintf("title: %s\n", title))
	if provider != "" {
		sb.WriteString(fmt.Sprintf("provider: %s\n", provider))
	}
	if score > 0 {
		sb.WriteString(fmt.Sprintf("score: %.4f\n", score))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(content)
	return os.WriteFile(filepath.Join(dir, filename), []byte(sb.String()), 0644)
}
