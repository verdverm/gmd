package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all context documents by collection",
	Long: `Shows every collection that has a context document attached and the path
to the context file stored in the config.

Example:
  gmd context list`,
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

func init() {
	contextCmd.AddCommand(contextListCmd)
}
