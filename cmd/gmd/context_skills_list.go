package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextSkillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	Long: `Shows all embedded agent skills.

Example:
  gmd context skills list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		names, err := skills.ListSkillNames()
		if err != nil {
			return err
		}
		fmt.Println("Available skills:")
		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
		return nil
	},
}

func init() {
	contextSkillsCmd.AddCommand(contextSkillsListCmd)
}
