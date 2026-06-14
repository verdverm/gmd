package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/agentsmd"
)

var contextAgentsmdShowCmd = &cobra.Command{
	Use:   "show <detail>",
	Short: "Output AGENTS.md content at a detail level",
	Long: `Prints AGENTS.md reference content at the specified detail level.

Detail levels:
  oneline   single-line description of GMD
  summary   essential commands and usage guidelines (default)
  detailed  full command reference, config, and pipeline details
  full      complete reference with architecture and design decisions

Examples:
  gmd context agentsmd show summary
  gmd context agentsmd show oneline`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		detail := "summary"
		if len(args) > 0 {
			detail = strings.ToLower(args[0])
		}

		content, err := agentsmd.GetContent(detail)
		if err != nil {
			valid, validErr := agentsmd.ValidNames()
			if validErr == nil {
				return fmt.Errorf("invalid detail %q - valid details: %s", args[0], strings.Join(valid, ", "))
			}
			return err
		}
		fmt.Println(content)
		return nil
	},
}

func init() {
	contextAgentsmdCmd.AddCommand(contextAgentsmdShowCmd)
}
