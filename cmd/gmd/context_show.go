package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agents"
	"github.com/verdverm/gmd/pkg/context/agentsmd"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show content of a context item by name",
	Long: `Outputs the full content of a named context item. Disambiguates by
checking agentsmd detail levels, skill names, and agent
role definitions in order.

If the name is ambiguous across categories, an error is reported
listing all matches. Use category subcommands to disambiguate:

  gmd context agentsmd show <name>
  gmd context skills show <name>
  gmd context agents show <name>

Examples:
  gmd context show summary
  gmd context show gmd-wiki
  gmd context show opencode`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		var matches []string

		valid, err := agentsmd.ValidNames()
		if err != nil {
			return err
		}
		for _, v := range valid {
			if v == name {
				matches = append(matches, fmt.Sprintf("agentsmd/%s", v))
			}
		}

		skillNames, err := skills.ListSkillNames()
		if err != nil {
			return err
		}
		for _, n := range skillNames {
			if n == name {
				matches = append(matches, fmt.Sprintf("skills/%s", n))
			}
		}

		cfg, err := getConfig()
		if err != nil {
			return err
		}
		baseDir := cfg.ProjectRoot
		isGlobal := contextGlobal
		if isGlobal || baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseDir = home
			isGlobal = true
		}
		agentNames, err := agents.ListAgents(isGlobal, baseDir)
		if err != nil {
			return err
		}
		for _, a := range agentNames {
			if a == name {
				matches = append(matches, fmt.Sprintf("agents/%s", a))
			}
		}

		if len(matches) > 1 {
			return fmt.Errorf("ambiguous name %q matches: %s\nUse category subcommand to disambiguate", name, strings.Join(matches, ", "))
		}

		if len(matches) == 0 {
			return fmt.Errorf("unknown context item %q", name)
		}

		match := matches[0]
		switch {
		case strings.HasPrefix(match, "agentsmd/"):
			content, err := agentsmd.GetContent(name)
			if err != nil {
				return err
			}
			fmt.Println(content)
		case strings.HasPrefix(match, "skills/"):
			content, err := skills.GetSkillContent(name)
			if err != nil {
				return err
			}
			fmt.Println(content)
		case strings.HasPrefix(match, "agents/"):
			files, err := agents.ShowAgent(name, isGlobal, baseDir)
			if err != nil {
				return err
			}
			for fname, content := range files {
				fmt.Printf("=== %s ===\n%s\n", fname, content)
			}
		}
		return nil
	},
}
