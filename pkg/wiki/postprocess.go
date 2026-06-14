package wiki

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/verdverm/gmd/pkg/llm"
)

func setTimestamp(pagePath string) error {
	data, err := os.ReadFile(pagePath)
	if err != nil {
		return fmt.Errorf("reading page for timestamp: %w", err)
	}
	content := string(data)

	fm, stripped, err := ParseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("parsing frontmatter for timestamp: %w", err)
	}
	if fm == nil {
		return fmt.Errorf("no frontmatter found in %s", pagePath)
	}

	fm["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	fmYAML, err := marshalYAML(fm)
	if err != nil {
		return fmt.Errorf("marshaling frontmatter for timestamp: %w", err)
	}

	newContent := fmt.Sprintf("---\n%s\n---\n\n%s", fmYAML, stripped)
	return os.WriteFile(pagePath, []byte(newContent), 0600)
}

func generateDescription(ctx context.Context, pagePath string, llmClient *llm.Client) error {
	if llmClient == nil {
		return nil
	}
	data, err := os.ReadFile(pagePath)
	if err != nil {
		return fmt.Errorf("reading page for description: %w", err)
	}
	content := string(data)

	fm, stripped, err := ParseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("parsing frontmatter for description: %w", err)
	}
	if fm == nil {
		return fmt.Errorf("no frontmatter found in %s", pagePath)
	}

	if _, hasDesc := fm["description"]; hasDesc {
		return nil
	}

	// Truncate to reasonable size for summarization
	toSummarize := stripped
	if len(toSummarize) > 4000 {
		toSummarize = toSummarize[:4000]
	}

	msg := "Generate a single-sentence description (max 150 chars) for this wiki page: " + toSummarize
	resp, err := llmClient.Chat(ctx, []llm.ChatMessage{
		{Role: "user", Content: msg},
	})
	if err != nil {
		return fmt.Errorf("generating description: %w", err)
	}

	summary := resp
	if len(summary) > 200 {
		summary = summary[:200]
	}

	fm["description"] = summary
	fm["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	fmYAML, err := marshalYAML(fm)
	if err != nil {
		return fmt.Errorf("marshaling frontmatter for description: %w", err)
	}

	newContent := fmt.Sprintf("---\n%s\n---\n\n%s", fmYAML, stripped)
	return os.WriteFile(pagePath, []byte(newContent), 0600)
}
