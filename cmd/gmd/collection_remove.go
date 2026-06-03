package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a collection and all its indexed chunks",
	Long: `Removes the collection from the config and deletes all associated chunks
from Typesense. This operation is immediate and cannot be undone.

Example:
  gmd collection remove mydocs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]

		if err := config.RemoveCollection(r.Config(), name); err != nil {
			return err
		}

		if err := r.TSClient().DeleteChunksByCollection(context.Background(), r.Config().CollectionKey(name)); err != nil {
			return fmt.Errorf("deleting chunks for %q: %w", name, err)
		}

		fmt.Printf("Removed collection %q and its chunks.\n", name)
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionRemoveCmd)
}
