package main

import (
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Re-embed all documents without re-chunking",
	Long: `Re-generates embeddings for all indexed documents and updates them in
Typesense. Chunks are not regenerated — only embeddings are recomputed.

use this after changing the embedding model in config to avoid a full
re-index. For a complete rebuild, use 'gmd update' instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndex("Embed")
	},
}
