package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a collection in the config",
	Long: `Renames a collection without affecting its indexed chunks. The old name
is updated in the config only — existing Typesense data is preserved
under the new collection key.

Example:
  gmd collection rename mydocs docs`,
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

		fmt.Printf("Renamed collection %q to %q.\n", oldName, newName)
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionRenameCmd)
}
