package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var wikiListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all wikis",
	Long: `List all wikis configured in the project.

Example:
  gmd wiki list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		wikis := r.Config().Wikis
		names := make([]string, 0, len(wikis))
		for name := range wikis {
			names = append(names, name)
		}
		sort.Strings(names)

		if len(names) == 0 {
			fmt.Println("No wikis configured.")
			return nil
		}

		for _, name := range names {
			wc := wikis[name]
			fmt.Printf("  %s\n", name)
			fmt.Printf("    path:       %s\n", wc.Path)
			fmt.Printf("    wikiDir:    %s\n", wc.WikiDir)
			fmt.Printf("    rawDir:     %s\n", wc.RawDir)
			fmt.Printf("    patterns:   %v\n", wc.Patterns)
			if len(wc.SourceRefs) > 0 {
				fmt.Printf("    sourceRefs: %v\n", wc.SourceRefs)
			}
		}
		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiListCmd)
}
