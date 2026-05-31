package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var contextCmd = &cobra.Command{
	Use:   "context [add|list|rm]",
	Short: "Manage search context documents",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var contextAddCmd = &cobra.Command{
	Use:   "add <name> <path>",
	Short: "Add a context document to a collection",
	Args:  cobra.ExactArgs(2),
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
	Short: "List context documents",
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
	Use:   "rm <name>",
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
