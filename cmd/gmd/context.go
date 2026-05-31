package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var contextCmd = &cobra.Command{
	Use:   "context [add|list|rm]",
	Short: "Manage context documents attached to collections",
	Long: `Context documents provide additional text that is injected alongside
search results to give AI assistants domain knowledge.

Subcommands:
  add    attach a text file as context to a collection
  list   show all context documents
  rm     remove a context document from a collection`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var contextAddCmd = &cobra.Command{
	Use:   "add <collection> <path>",
	Short: "Attach a text file as context to a collection",
	Long: `Associates a text file with a collection. The file's content is stored in
the config and served alongside search results to provide AI assistants
with domain-specific knowledge about the collection.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddContextDoc(r.Config(), args[0], args[1])
	},
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all context documents by collection",
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		ctxs := config.ListContextDocs(r.Config())
		if len(ctxs) == 0 {
			fmt.Println("No context documents configured.")
			return nil
		}

		names := make([]string, 0, len(ctxs))
		for name := range ctxs {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("  %s -> %s\n", name, ctxs[name])
		}
		return nil
	},
}

var contextRmCmd = &cobra.Command{
	Use:   "rm <collection>",
	Short: "Remove a context document from a collection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.RemoveContextDoc(r.Config(), args[0])
	},
}

func init() {
	contextCmd.AddCommand(contextAddCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextRmCmd)
}
