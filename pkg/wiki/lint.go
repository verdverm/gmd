package wiki

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/verdverm/gmd/pkg/chunking"
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
	pageNameToID := make(map[string]string)

	pageContents := make(map[string]string)

	sourceDir := func(cid string) string {
		return filepath.Dir(filepath.Join(wikiDir, cid+".md"))
	}

	// Pass 1: build allPages and pageNameToID registry
	_ = filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		base := filepath.Base(path)
		if base == a.wiki.WikiConfig.IndexFile || base == a.wiki.WikiConfig.LogFile {
			return nil
		}

		cid := pageName(wikiDir, path)
		allPages[cid] = true

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fm, stripped, _ := ParseFrontmatter(string(data))
		pn := getPageName(fm, stripped)
		if pn != "" {
			pageNameToID[pn] = cid
		}
		pageContents[cid] = stripped

		return nil
	})

	// Pass 2: extract and resolve all links
	linkRefs := make(map[string]int)
	for cid, stripped := range pageContents {
		wLinks := chunking.ExtractWikilinks(stripped)
		mLinks := chunking.ExtractMarkdownLinks(stripped)

		seen := make(map[string]bool)
		for _, link := range wLinks {
			var resolved string
			if id, ok := pageNameToID[link]; ok {
				resolved = id
			} else {
				resolved = link
			}
			if !seen[resolved] && resolved != cid {
				seen[resolved] = true
				linkRefs[resolved]++
			}
		}
		for _, link := range mLinks {
			resolved := chunking.NormalizeConceptID(link, sourceDir(cid))
			if !seen[resolved] && resolved != cid {
				seen[resolved] = true
				linkRefs[resolved]++
			}
		}
	}

	// Orphan detection
	for page := range allPages {
		if linkRefs[page] == 0 {
			result.Orphans = append(result.Orphans, page)
		}
	}

	// Broken link detection
	for target := range linkRefs {
		if !allPages[target] {
			var fromPages []string
			for cid, stripped := range pageContents {
				wLinks := chunking.ExtractWikilinks(stripped)
				for _, link := range wLinks {
					if id, ok := pageNameToID[link]; ok && id == target {
						fromPages = append(fromPages, cid)
						break
					} else if link == target {
						fromPages = append(fromPages, cid)
						break
					}
				}
				if len(fromPages) > 0 && fromPages[len(fromPages)-1] == cid {
					continue
				}
				mLinks := chunking.ExtractMarkdownLinks(stripped)
				for _, link := range mLinks {
					if chunking.NormalizeConceptID(link, sourceDir(cid)) == target {
						fromPages = append(fromPages, cid)
						break
					}
				}
			}
			result.BrokenLinks = append(result.BrokenLinks, BrokenLink{
				FromPage:   strings.Join(fromPages, ", "),
				LinkTarget: target,
				Hint:       "missing page",
			})
		}
	}

	// Stale entry detection
	indexData, err := os.ReadFile(a.wiki.IndexFilePath())
	if err != nil {
		return
	}
	indexContent := string(indexData)

	pagesFoundInIndex := make(map[string]bool)
	indexWLinks := chunking.ExtractWikilinks(indexContent)
	for _, link := range indexWLinks {
		if id, ok := pageNameToID[link]; ok {
			pagesFoundInIndex[id] = true
		}
		if allPages[link] {
			pagesFoundInIndex[link] = true
		}
	}
	indexMLinks := chunking.ExtractMarkdownLinks(indexContent)
	for _, link := range indexMLinks {
		resolved := chunking.NormalizeConceptID(link, "")
		if allPages[resolved] {
			pagesFoundInIndex[resolved] = true
		}
	}

	for page := range allPages {
		if !pagesFoundInIndex[page] {
			result.StaleEntries = append(result.StaleEntries, page)
		}
	}
}

func (a *Agent) lintContent(ctx context.Context, result *LintResult) {
	if a.chat == nil {
		return
	}

	wikiDir := a.wiki.WikiPath
	var pages []struct {
		name    string
		content string
	}

	_ = filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if filepath.Base(path) == a.wiki.WikiConfig.IndexFile || filepath.Base(path) == a.wiki.WikiConfig.LogFile {
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
			resp, err := a.chat.Chat(ctx, "", prompt)
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
	if a.chat == nil {
		return
	}

	indexData, err := os.ReadFile(a.wiki.IndexFilePath())
	if err != nil {
		return
	}

	prompt := LintGapPrompt(string(indexData))
	resp, err := a.chat.Chat(ctx, "", prompt)
	if err != nil {
		return
	}
	result.Gaps = resp
}
