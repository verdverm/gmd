package wiki

import (
	"context"
	"fmt"
	"strings"

	"github.com/verdverm/gmd/pkg/config"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

type DoctorResult struct {
	WikiName    string
	PageCount   int
	SourceCount int
	TSConnected bool
	LLMStatus   []llm.EndpointStatus
	Agents      []AgentStatus
	Errors      []string
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

	agentNames := []string{"claude", "codex", "opencode"}
	for _, name := range agentNames {
		installed := CheckAgentInstalled(name)
		skillInst := CheckSkillInstalled(name)

		result.Agents = append(result.Agents, AgentStatus{
			Name:      name,
			Installed: installed,
			SkillInst: skillInst,
		})
	}

	return result, nil
}

func DoctorFix(wiki *Wiki) ([]string, error) {
	var fixes []string

	for _, name := range []string{"claude", "codex", "opencode"} {
		if !CheckSkillInstalled(name) && CheckAgentInstalled(name) {
			written, err := WriteSkills(name)
			if err != nil {
				fixes = append(fixes, fmt.Sprintf("  %s: failed to write skill: %v", name, err))
			} else {
				for _, w := range written {
					fixes = append(fixes, fmt.Sprintf("  %s: skill written to %s", name, w))
				}
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

	if len(result.Errors) > 0 {
		b.WriteString("  Errors:\n")
		for _, e := range result.Errors {
			b.WriteString(fmt.Sprintf("    - %s\n", e))
		}
	}

	return b.String()
}
