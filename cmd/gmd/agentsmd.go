package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agentsmd"
)

var agentsmdCmd = &cobra.Command{
	Use:   "agentsmd [name]",
	Short: "Output AGENTS.md content for AI coding assistants",
	Long: `Prints AGENTS.md reference content that teaches AI coding assistants
how to use GMD for searching and retrieving documentation.

Names:
  oneline   single-line description of GMD
  summary   essential commands and usage guidelines (default)
  detailed  full command reference, config, and pipeline details
  full      complete reference with architecture and design decisions

Pipe the output to a file or clipboard to share with your AI assistant:
  gmd agentsmd summary | pbcopy`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := "summary"
		if len(args) > 0 {
			name = strings.ToLower(args[0])
		}

		content, err := agentsmd.GetContent(name)
		if err != nil {
			valid, validErr := agentsmd.ValidNames()
			if validErr == nil {
				return fmt.Errorf("invalid name %q - valid names: %s", args[0], strings.Join(valid, ", "))
			}
			return err
		}
		fmt.Println(content)
		return nil
	},
}
