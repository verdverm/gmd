package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	searchLimit  int
	searchFormat string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text keyword search (no vector, no expansion)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Printf("search: %q (not yet implemented, Phase 3)\n", args[0])
		return nil
	},
}

var vsearchCmd = &cobra.Command{
	Use:   "vsearch <query>",
	Short: "Vector similarity search (no text)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Printf("vsearch: %q (not yet implemented, Phase 3)\n", args[0])
		return nil
	},
}

var queryCmd = &cobra.Command{
	Use:   "query <query>",
	Short: "Full hybrid pipeline (expansion, RRF, rerank, blend)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Printf("query: %q (not yet implemented, Phase 3)\n", args[0])
		return nil
	},
}

func init() {
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	searchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	vsearchCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	vsearchCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
	queryCmd.Flags().IntVarP(&searchLimit, "limit", "n", 5, "max results")
	queryCmd.Flags().StringVarP(&searchFormat, "format", "f", "cli", "output format")
}
