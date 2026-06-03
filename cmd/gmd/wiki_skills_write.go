package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiSkillsWriteCmd = &cobra.Command{
	Use:   "write [--target claude|codex|opencode|all]",
	Short: "Write skill templates to agent discovery paths",
	Long: `Installs skill templates to the appropriate agent discovery directories
so that AI coding assistants discover and use them automatically.

Examples:
  gmd wiki skills write --target all
  gmd wiki skills write --target opencode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := wikiTarget
		if target == "" {
			target = "all"
		}

		written, err := wiki.WriteSkills(target)
		if err != nil {
			return fmt.Errorf("writing skills: %w", err)
		}
		for _, w := range written {
			fmt.Printf("  Written: %s\n", w)
		}
		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiSkillsWriteCmd)
}
