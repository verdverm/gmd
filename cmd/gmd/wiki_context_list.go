package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiContextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all wiki context documents",
	Long: `Shows every wiki that has a context document attached and the path
to the context file stored in the config.

Example:
  gmd wiki context list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		ctxs := config.ListContextDocs(r.Config())
		wikiCtxs := make(map[string]string)
		for name, ctx := range ctxs {
			if _, ok := r.Config().Wikis[name]; ok {
				wikiCtxs[name] = ctx
			}
		}

		if len(wikiCtxs) == 0 {
			fmt.Println("No wiki context documents configured.")
			return nil
		}

		names := make([]string, 0, len(wikiCtxs))
		for name := range wikiCtxs {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			fmt.Printf("  %s -> %s\n", name, wikiCtxs[name])
		}
		return nil
	},
}

func init() {
	wikiContextCmd.AddCommand(wikiContextListCmd)
}
