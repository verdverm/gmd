package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/search"
)

var vsearchCmd = &cobra.Command{
	Use:   "vsearch <query>",
	Short: "Vector similarity search — semantic matching via embeddings",
	Long: `Embeds the query and performs a vector similarity search using cosine
distance against document chunk embeddings in Typesense.

This finds semantically related content even when exact keywords don't
match. No query expansion or reranking is performed — for the full
pipeline with those features, use 'gmd query' instead.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return searchRun(args, search.ModeVector)
	},
}

func init() {
	vsearchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	vsearchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	vsearchCmd.Flags().StringSliceVarP(&searchCollections, "collection", "c", nil, "collection(s) to search (default: auto-detect from CWD)")
	vsearchCmd.Flags().BoolVar(&searchNoExpansion, "no-expansion", false, "Do not expand wiki sourceRefs")
	vsearchCmd.Flags().StringVarP(&searchPreset, "search", "s", "", "Named search preset from searchDefaults")
}
