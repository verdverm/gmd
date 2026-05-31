package chunking

import (
	"math"
	"strings"
	"unicode/utf8"
)

const charsPerToken = 4

type Chunk struct {
	Content     string
	Title       string
	ChunkSeq    int
	TotalChunks int
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

func extractHeadings(lines []string) []headingInfo {
	var headings []headingInfo
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		if trimmed[0] == '#' {
			level := 0
			for _, ch := range trimmed {
				if ch == '#' {
					level++
				} else if ch == ' ' {
					break
				} else {
					break
				}
			}
			if level > 0 && level <= 6 && len(trimmed) > level && trimmed[level] == ' ' {
				headings = append(headings, headingInfo{
					level:   level,
					text:    strings.TrimSpace(trimmed[level:]),
					lineIdx: i,
				})
			}
		}
	}
	return headings
}

func ChunkMarkdown(content string, cfg Config) []Chunk {
	lines := strings.Split(content, "\n")
	headings := extractHeadings(lines)

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

	var segs []segment
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
