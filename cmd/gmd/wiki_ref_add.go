package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiRefAddCmd = &cobra.Command{
	Use:   "add <wiki> <source>",
	Short: "Add a source reference to a wiki",
	Long: `Adds a source reference enabling the wiki to aggregate content from
the named collection or wiki for search. Validates the target exists
and rejects circular references.

Example:
  gmd wiki ref add mywiki docs`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		wikiName := args[0]
		srcName := args[1]

		if err := config.AddSourceRef(r.Config(), wikiName, srcName); err != nil {
			return err
		}

		fmt.Printf("Added source reference %q to wiki %q.\n", srcName, wikiName)
		return nil
	},
}

func init() {
	wikiRefCmd.AddCommand(wikiRefAddCmd)
}
