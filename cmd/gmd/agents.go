package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agents"
)

var agentsCmd = &cobra.Command{
	Use:   "agents [size]",
	Short: "Output AGENTS.md content for AI coding assistants",
	Long: `Prints AGENTS.md reference content that teaches AI coding assistants
how to use GMD for searching and retrieving documentation.

Sizes:
  oneline   single-line description of GMD
  summary   essential commands and usage guidelines (default)
  detailed  full command reference, config, and pipeline details
  full      complete reference with architecture and design decisions

Pipe the output to a file or clipboard to share with your AI assistant:
  gmd agents summary | pbcopy`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		size := agents.Summary
		if len(args) > 0 {
			s := strings.ToLower(args[0])
			if !agents.IsValidSize(s) {
				return fmt.Errorf("invalid size %q - valid sizes: %s", args[0], strings.Join(agents.ValidSizes, ", "))
			}
			size = agents.Size(s)
		}

		content, err := agents.GetContent(size)
		if err != nil {
			return err
		}
		fmt.Println(content)
		return nil
	},
}
