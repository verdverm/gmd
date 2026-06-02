package wiki

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/verdverm/gmd/pkg/config"
	"gopkg.in/yaml.v3"
)

var fmRe = regexp.MustCompile(`^---\s*\n([\s\S]*?)\n---\s*\n`)

func ParseFrontmatter(content string) (map[string]interface{}, string, error) {
	match := fmRe.FindStringSubmatch(content)
	if match == nil {
		return nil, content, nil
	}
	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(match[1]), &fm); err != nil {
		return nil, content, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}
	remaining := content[len(match[0]):]
	return fm, remaining, nil
}

func StripFrontmatter(content string) string {
	match := fmRe.FindStringSubmatch(content)
	if match == nil {
		return content
	}
	return content[len(match[0]):]
}

func ValidateFrontmatter(fm map[string]interface{}, cfg *config.FrontmatterConfig) error {
	if cfg == nil || len(cfg.Fields) == 0 {
		return nil
	}
	for name, field := range cfg.Fields {
		val, ok := fm[name]
		if !ok {
			continue
		}
		switch field.Type {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("frontmatter field %q: expected string, got %T", name, val)
			}
		case "string[]":
			switch v := val.(type) {
			case []interface{}:
				for i, item := range v {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("frontmatter field %q[%d]: expected string, got %T", name, i, item)
					}
				}
			case []string:
			default:
				return fmt.Errorf("frontmatter field %q: expected string array, got %T", name, val)
			}
		case "int32":
			if _, ok := val.(int); !ok {
				return fmt.Errorf("frontmatter field %q: expected int, got %T", name, val)
			}
		case "float":
			if _, ok := val.(float64); !ok {
				return fmt.Errorf("frontmatter field %q: expected float, got %T", name, val)
			}
		case "bool":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("frontmatter field %q: expected bool, got %T", name, val)
			}
		default:
			return fmt.Errorf("frontmatter field %q: unknown type %q", name, field.Type)
		}
	}
	return nil
}

func FrontmatterToFilter(fm map[string]interface{}, cfg *config.FrontmatterConfig) string {
	if cfg == nil || len(cfg.Fields) == 0 {
		return ""
	}
	var parts []string
	for name, field := range cfg.Fields {
		val, ok := fm[name]
		if !ok {
			continue
		}
		switch field.Type {
		case "string":
			if s, ok := val.(string); ok {
				parts = append(parts, fmt.Sprintf("%s:=%s", name, s))
			}
		case "string[]":
			switch v := val.(type) {
			case []interface{}:
				var items []string
				for _, item := range v {
					if s, ok := item.(string); ok {
						items = append(items, s)
					}
				}
				if len(items) > 0 {
					parts = append(parts, fmt.Sprintf("%s:=[%s]", name, strings.Join(items, ",")))
				}
			case []string:
				if len(v) > 0 {
					parts = append(parts, fmt.Sprintf("%s:=[%s]", name, strings.Join(v, ",")))
				}
			}
		case "int32":
			parts = append(parts, fmt.Sprintf("%s:=%v", name, val))
		case "float":
			parts = append(parts, fmt.Sprintf("%s:=%v", name, val))
		case "bool":
			parts = append(parts, fmt.Sprintf("%s:=%v", name, val))
		}
	}
	return strings.Join(parts, " && ")
}
