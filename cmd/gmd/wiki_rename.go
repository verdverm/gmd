package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a wiki in the config",
	Long: `Renames a wiki without affecting its indexed chunks. The old name
is updated in the config only.

Example:
  gmd wiki rename mywiki research-wiki`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		oldName := args[0]
		newName := args[1]

		if err := config.RenameCollection(r.Config(), oldName, newName); err != nil {
			return err
		}

		fmt.Printf("Renamed wiki %q to %q.\n", oldName, newName)
		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiRenameCmd)
}
