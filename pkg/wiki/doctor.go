package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/context/skills"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

type DoctorResult struct {
	WikiName      string
	PageCount     int
	SourceCount   int
	TSConnected   bool
	LLMStatus     []llm.EndpointStatus
	Agents        []AgentStatus
	Errors        []string
	FixesApplied  []string
	OKFIssues     []string
	MissingOKFVer bool
	PagesNoType   int
}

type AgentStatus struct {
	Name          string
	Installed     bool
	SkillInst     bool
	MCPConfigured bool
}

func Doctor(ctx context.Context, wiki *Wiki, cfg *config.Config, tsClient *ts.Client, llmClient *llm.Client) (*DoctorResult, error) {
	result := &DoctorResult{
		WikiName: wiki.Name,
	}

	if tsClient != nil {
		count, err := tsClient.CollectionCount(ctx)
		if err == nil && count >= 0 {
			result.TSConnected = true
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("Typesense: %v", err))
		}
	}

	if llmClient != nil {
		result.LLMStatus = llmClient.CheckAll(ctx)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("home directory: %v", err))
	} else {
		skillNames, err := skills.ListSkillNames()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("list skills: %v", err))
		} else {
			for _, name := range cfg.AgentHarnessNames() {
				installed, err := skills.CheckHarnessInstalled(home, true, name)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("check %s: %v", name, err))
					continue
				}
				skillInst := false
				for _, sn := range skillNames {
					si, err := skills.SkillInstalled(home, true, name, sn)
					if err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("check skill %s/%s: %v", name, sn, err))
						continue
					}
					if si {
						skillInst = true
						break
					}
				}

				result.Agents = append(result.Agents, AgentStatus{
					Name:      name,
					Installed: installed,
					SkillInst: skillInst,
				})
			}
		}
	}

	wikiDir := wiki.WikiPath
	if _, statErr := os.Stat(wikiDir); statErr == nil {
		indexPath := wiki.IndexFilePath()
		if indexData, readErr := os.ReadFile(indexPath); readErr == nil {
			fm, _, _ := ParseFrontmatter(string(indexData))
			if fm == nil || fm["okf_version"] == nil {
				result.MissingOKFVer = true
			}
		} else {
			result.MissingOKFVer = true
		}

		_ = filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}
			base := filepath.Base(path)
			if base == wiki.WikiConfig.IndexFile || base == wiki.WikiConfig.LogFile {
				return nil
			}
			result.PageCount++
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			fm, _, _ := ParseFrontmatter(string(data))
			if fm == nil || fm["type"] == nil || fm["type"] == "" {
				result.PagesNoType++
			}
			return nil
		})

		rawPath := wiki.RawPath
		if entries, readErr := os.ReadDir(rawPath); readErr == nil {
			result.SourceCount = len(entries)
		}
	}

	return result, nil
}

func DoctorFix(wiki *Wiki, cfg *config.Config) ([]string, error) {
	var fixes []string

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home directory: %w", err)
	}
	skillNames, err := skills.ListSkillNames()
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	for _, name := range cfg.AgentHarnessNames() {
		skillInst := false
		for _, sn := range skillNames {
			si, err := skills.SkillInstalled(home, true, name, sn)
			if err != nil {
				return nil, fmt.Errorf("check skill %s/%s: %w", name, sn, err)
			}
			if si {
				skillInst = true
				break
			}
		}
		agentInst, err := skills.CheckHarnessInstalled(home, true, name)
		if err != nil {
			return nil, fmt.Errorf("check harness %s: %w", name, err)
		}

		if !skillInst && agentInst {
			dest, err := skills.WriteSkillTo(home, true, name)
			if err != nil {
				fixes = append(fixes, fmt.Sprintf("  %s: failed to write skill: %v", name, err))
			} else {
				fixes = append(fixes, fmt.Sprintf("  %s: skill written to %s", name, dest))
			}
		}
	}

	return fixes, nil
}

func FormatDoctorResult(result *DoctorResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("  Wiki: %s\n", result.WikiName))

	if result.TSConnected {
		b.WriteString("  Typesense: \u2713 connected\n")
	} else {
		b.WriteString("  Typesense: \u2717 not connected\n")
	}

	if len(result.LLMStatus) > 0 {
		b.WriteString("  LLM:\n")
		for _, s := range result.LLMStatus {
			mark := "\u2713"
			if !s.OK {
				mark = "\u2717"
			}
			b.WriteString(fmt.Sprintf("    %s %s (%s)\n", mark, s.Label, s.Model))
			if s.Err != "" {
				b.WriteString(fmt.Sprintf("      error: %s\n", s.Err))
			}
		}
	}

	if len(result.Agents) > 0 {
		b.WriteString("  Agent discovery:\n")
		for _, a := range result.Agents {
			installed := "\u2717"
			if a.Installed {
				installed = "\u2713"
			}

			skill := "\u2717 skill"
			if a.SkillInst {
				skill = "\u2713 skill"
			}

			mcp := "\u2717 MCP"
			if a.MCPConfigured {
				mcp = "\u2713 MCP"
			}

			if !a.Installed {
				b.WriteString(fmt.Sprintf("    %s: %s not detected\n", a.Name, installed))
			} else {
				b.WriteString(fmt.Sprintf("    %s: %s, %s, %s\n", a.Name, installed, skill, mcp))
			}
		}
	}

	b.WriteString(fmt.Sprintf("  Pages: %d\n", result.PageCount))
	b.WriteString(fmt.Sprintf("  Raw sources: %d\n", result.SourceCount))

	if result.MissingOKFVer {
		b.WriteString("  OKF version: \u2717 missing (run 'gmd wiki doctor --fix' to add)\n")
	} else {
		b.WriteString("  OKF version: \u2713 declared\n")
	}

	if result.PagesNoType > 0 {
		b.WriteString(fmt.Sprintf("  Pages missing 'type': %d\n", result.PagesNoType))
	}

	if len(result.Errors) > 0 {
		b.WriteString("  Errors:\n")
		for _, e := range result.Errors {
			b.WriteString(fmt.Sprintf("    - %s\n", e))
		}
	}

	return b.String()
}
