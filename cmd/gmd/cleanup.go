package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/indexer"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove stale chunks for files that no longer exist",
	Long: `Scans all collections and removes indexed chunks whose source files have
been deleted from disk. This keeps the Typesense index clean after files
are moved or removed.

Run this periodically or after large file reorganizations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		idx := indexer.New(r.Config(), r.TSClient(), nil)
		ctx := context.Background()
		progress := func(m string) { fmt.Println(m) }
		results := idx.CleanupAllCollections(ctx, progress)

		total := 0
		for name, count := range results {
			fmt.Printf("[%s] Removed %d stale chunks\n", name, count)
			total += count
		}
		fmt.Printf("\nCleanup complete: %d total stale chunks removed.\n", total)
		return nil
	},
}
