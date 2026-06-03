package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiSkillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skill templates",
	Long:  "Shows all embedded wiki agent skill templates with their name, target, and description.",
	RunE: func(cmd *cobra.Command, args []string) error {
		templates, err := wiki.ListSkillTemplates()
		if err != nil {
			return err
		}
		fmt.Println("Available skill templates:")
		for _, t := range templates {
			fmt.Printf("  %-20s %-12s %s\n", t.Name, t.Target, t.Description)
		}
		return nil
	},
}

func init() {
	wikiSkillsCmd.AddCommand(wikiSkillsListCmd)
}
