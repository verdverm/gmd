package wiki

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/verdverm/gmd/pkg/chunking"
	"github.com/verdverm/gmd/pkg/llm"
)

type LintResult struct {
	Orphans        []string
	BrokenLinks    []BrokenLink
	StaleEntries   []string
	Contradictions []Contradiction
	Gaps           string
	Errors         []string
}

type BrokenLink struct {
	FromPage   string
	LinkTarget string
	Hint       string
}

type Contradiction struct {
	PageA      string
	PageB      string
	ClaimA     string
	ClaimB     string
	Resolution string
}

type LintOpts struct {
	Watch bool
}

func (a *Agent) Lint(ctx context.Context, opts LintOpts) (*LintResult, error) {
	result := &LintResult{}

	a.lintStructure(ctx, result)

	if !opts.Watch {
		a.lintContent(ctx, result)
		a.lintGaps(ctx, result)
	}

	return result, nil
}

func (a *Agent) lintStructure(ctx context.Context, result *LintResult) {
	wikiDir := a.wiki.WikiPath

	allPages := make(map[string]bool)
	wikilinks := make(map[string]int)

	filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "_") && base != a.wiki.WikiConfig.IndexFile && base != a.wiki.WikiConfig.LogFile {
			return nil
		}

		name := pageName(wikiDir, path)
		allPages[name] = true

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		_, stripped, _ := ParseFrontmatter(string(data))
		links := chunking.ExtractWikilinks(stripped)
		for _, link := range links {
			wikilinks[link]++
		}

		return nil
	})

	for page := range allPages {
		if wikilinks[page] == 0 {
			result.Orphans = append(result.Orphans, page)
		}
	}

	for target := range wikilinks {
		if !allPages[target] {
			var fromPages []string
			filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
					return nil
				}
				data, _ := os.ReadFile(path)
				_, stripped, _ := ParseFrontmatter(string(data))
				links := chunking.ExtractWikilinks(stripped)
				for _, link := range links {
					if link == target {
						fromPages = append(fromPages, pageName(wikiDir, path))
					}
				}
				return nil
			})
			result.BrokenLinks = append(result.BrokenLinks, BrokenLink{
				FromPage:   strings.Join(fromPages, ", "),
				LinkTarget: target,
				Hint:       "missing page",
			})
		}
	}

	indexData, err := os.ReadFile(a.wiki.IndexFilePath())
	if err != nil {
		return
	}
	indexContent := string(indexData)
	for page := range allPages {
		if strings.Contains(indexContent, page) {
			continue
		}
		indexLinks := chunking.ExtractWikilinks(indexContent)
		for _, link := range indexLinks {
			if link == page && !allPages[link] {
				result.StaleEntries = append(result.StaleEntries, link)
			}
		}
	}
}

func (a *Agent) lintContent(ctx context.Context, result *LintResult) {
	if a.llmClient == nil {
		return
	}

	wikiDir := a.wiki.WikiPath
	var pages []struct {
		name    string
		content string
	}

	filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), "_") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		_, stripped, _ := ParseFrontmatter(string(data))
		pages = append(pages, struct {
			name    string
			content string
		}{name: pageName(wikiDir, path), content: stripped})
		return nil
	})

	for i := 0; i < len(pages) && i < 10; i++ {
		for j := i + 1; j < len(pages) && j < 10; j++ {
			prompt := LintContradictionPrompt(
				truncate(pages[i].content, 2000),
				truncate(pages[j].content, 2000),
			)
			resp, err := a.llmClient.Chat(ctx, []llm.ChatMessage{
				{Role: "user", Content: prompt},
			})
			if err != nil {
				continue
			}
			if resp != "" && !strings.Contains(strings.ToLower(resp), "no contradictions found") {
				result.Contradictions = append(result.Contradictions, Contradiction{
					PageA:      pages[i].name,
					PageB:      pages[j].name,
					Resolution: truncate(resp, 500),
				})
			}
		}
	}
}

func (a *Agent) lintGaps(ctx context.Context, result *LintResult) {
	if a.llmClient == nil {
		return
	}

	indexData, err := os.ReadFile(a.wiki.IndexFilePath())
	if err != nil {
		return
	}

	prompt := LintGapPrompt(string(indexData))
	resp, err := a.llmClient.Chat(ctx, []llm.ChatMessage{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return
	}
	result.Gaps = resp
}
