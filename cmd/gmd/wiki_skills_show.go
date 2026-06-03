package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiSkillsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a skill template",
	Long: `Displays the full content of a named skill template.

Example:
  gmd wiki skills show research-agent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tmpl, err := wiki.GetSkillTemplate(args[0])
		if err != nil {
			return err
		}
		fmt.Println(tmpl.Content)
		return nil
	},
}

func init() {
	wikiSkillsCmd.AddCommand(wikiSkillsShowCmd)
}
