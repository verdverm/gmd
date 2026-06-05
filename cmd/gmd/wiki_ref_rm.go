package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiRefRmCmd = &cobra.Command{
	Use:   "rm <wiki> <source>",
	Short: "Remove a source reference from a wiki",
	Long: `Removes a source reference from the wiki.

Example:
  gmd wiki ref rm mywiki docs`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		wikiName := args[0]
		srcName := args[1]

		if err := config.RemoveSourceRef(r.Config(), wikiName, srcName); err != nil {
			return err
		}

		fmt.Printf("Removed source reference %q from wiki %q.\n", srcName, wikiName)
		return nil
	},
}

func init() {
	wikiRefCmd.AddCommand(wikiRefRmCmd)
}
