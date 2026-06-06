package chunking

import (
	"bytes"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

const charsPerToken = 4

type Chunk struct {
	Content     string
	Title       string
	ChunkSeq    int
	TotalChunks int
	Links       []string               // outgoing [[wikilinks]]
	Frontmatter map[string]interface{} // extracted frontmatter fields
}

type Config struct {
	TargetTokens    int
	Overlap         float64
	HeadingWeights  HeadingWeights
	CodeFenceWeight int
	NewlineWeight   float64
}

type HeadingWeights struct {
	H1, H2, H3, H4, H5, H6 int
}

func DefaultConfig() Config {
	return Config{
		TargetTokens:    900,
		Overlap:         0.15,
		HeadingWeights:  HeadingWeights{H1: 100, H2: 90, H3: 80, H4: 70, H5: 60, H6: 50},
		CodeFenceWeight: 10,
		NewlineWeight:   1,
	}
}

func tokenCount(s string) int {
	if s == "" {
		return 0
	}
	return int(math.Ceil(float64(utf8.RuneCountInString(s)) / charsPerToken))
}

type headingInfo struct {
	level   int
	text    string
	lineIdx int
}

type segment struct {
	heading   headingInfo
	lines     []string
	startLine int
}

func extractHeadings(content string) []headingInfo {
	var headings []headingInfo
	reader := text.NewReader([]byte(content))
	md := goldmark.New()
	doc := md.Parser().Parse(reader)

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Kind() == ast.KindHeading {
			h, ok := n.(*ast.Heading)
			if !ok {
				return ast.WalkContinue, nil
			}
			var buf bytes.Buffer
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if c.Kind() == ast.KindText || c.Kind() == ast.KindString {
					buf.Write(c.Text([]byte(content)))
				}
			}
			seg := n.Lines().At(0)
			lineIdx := bytes.Count([]byte(content[:seg.Start]), []byte("\n"))
			headings = append(headings, headingInfo{
				level:   h.Level,
				text:    strings.TrimSpace(buf.String()),
				lineIdx: lineIdx,
			})
		}
		return ast.WalkContinue, nil
	})
	return headings
}

var wikilinkRe = regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]+)?\]\]`)

func ExtractWikilinks(content string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var links []string
	for _, m := range matches {
		target := strings.TrimSpace(m[1])
		if !seen[target] && target != "" {
			seen[target] = true
			links = append(links, target)
		}
	}
	return links
}

var frontmatterRe = regexp.MustCompile(`^---\s*\n([\s\S]*?)\n---\s*\n`)

func ExtractFrontmatter(content string) (map[string]interface{}, string) {
	match := frontmatterRe.FindStringSubmatch(content)
	if match == nil {
		return nil, content
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(match[1]), &fm); err != nil {
		return nil, content
	}
	remaining := content[len(match[0]):]
	return fm, remaining
}

func ChunkMarkdownWithMeta(content string, cfg Config) ([]Chunk, []string, map[string]interface{}) {
	fm, stripped := ExtractFrontmatter(content)
	links := ExtractWikilinks(stripped)
	chunks := ChunkMarkdown(stripped, cfg)
	for i := range chunks {
		chunks[i].Links = links
		chunks[i].Frontmatter = fm
	}
	return chunks, links, fm
}

func ChunkMarkdown(content string, cfg Config) []Chunk {
	lines := strings.Split(content, "\n")
	headings := extractHeadings(content)

	docTitle := ""
	if len(headings) > 0 && headings[0].level == 1 {
		docTitle = headings[0].text
	}

	segs := splitIntoSegments(lines, headings)

	var rawChunks []string
	var current strings.Builder
	var currentTokens int
	targetChars := cfg.TargetTokens * charsPerToken
	overlapChars := int(float64(targetChars) * cfg.Overlap)

	for _, seg := range segs {
		segText := buildSegmentText(seg)
		segTokens := tokenCount(segText)
		segChars := len(segText)

		if currentTokens > 0 && currentTokens+segTokens > cfg.TargetTokens {
			rawChunks = append(rawChunks, current.String())
			current.Reset()
			// carry overlap
			if len(rawChunks) > 0 {
				prev := rawChunks[len(rawChunks)-1]
				if len(prev) > overlapChars {
					current.WriteString(prev[len(prev)-overlapChars:])
				} else {
					current.WriteString(prev)
				}
			}
			currentTokens = tokenCount(current.String())
		}

		if current.Len() > 0 && segText != "" {
			current.WriteString("\n")
		}
		current.WriteString(segText)
		currentTokens += segTokens

		if segChars > targetChars {
			rawChunks = append(rawChunks, current.String())
			current.Reset()
			currentTokens = 0
		}
	}

	if current.Len() > 0 {
		rawChunks = append(rawChunks, current.String())
	}

	chunks := make([]Chunk, len(rawChunks))
	for i, c := range rawChunks {
		chunks[i] = Chunk{
			Content:     strings.TrimSpace(c),
			Title:       docTitle,
			ChunkSeq:    i,
			TotalChunks: len(rawChunks),
		}
	}

	return chunks
}

func splitIntoSegments(lines []string, headings []headingInfo) []segment {
	if len(headings) == 0 {
		return []segment{{
			heading:   headingInfo{level: 0, text: ""},
			lines:     lines,
			startLine: 0,
		}}
	}

	segs := make([]segment, 0, len(headings))
	for i, h := range headings {
		endLine := len(lines)
		if i+1 < len(headings) {
			endLine = headings[i+1].lineIdx
		}
		segLines := lines[h.lineIdx:endLine]
		segs = append(segs, segment{
			heading:   h,
			lines:     segLines,
			startLine: h.lineIdx,
		})
	}

	return segs
}

func buildSegmentText(seg segment) string {
	return strings.Join(seg.lines, "\n")
}
