package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextSkillsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a skill",
	Long: `Displays the full content of a skill by name.

Example:
  gmd context skills show gmd-wiki`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := skills.GetSkillContent(args[0])
		if err != nil {
			return err
		}
		fmt.Println(content)
		return nil
	},
}
