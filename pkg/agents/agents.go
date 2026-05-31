package agents

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed content/*.md
var contentFS embed.FS

type Size string

const (
	Oneline  Size = "oneline"
	Summary  Size = "summary"
	Detailed Size = "detailed"
	Full     Size = "full"
)

var ValidSizes = []string{string(Oneline), string(Summary), string(Detailed), string(Full)}

func IsValidSize(s string) bool {
	for _, v := range ValidSizes {
		if v == s {
			return true
		}
	}
	return false
}

func GetContent(size Size) (string, error) {
	data, err := contentFS.ReadFile("content/" + string(size) + ".md")
	if err != nil {
		return "", fmt.Errorf("reading agents content for size %q: %w", size, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func MustGetContent(size Size) string {
	content, err := GetContent(size)
	if err != nil {
		panic(err)
	}
	return content
}
