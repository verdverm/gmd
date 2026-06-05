package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiContextAddCmd = &cobra.Command{
	Use:   "add <wiki> <path>",
	Short: "Attach a text file as context to a wiki",
	Long: `Associates a text file with a wiki. The file's content is stored in
the config and served alongside search results.

Example:
  gmd wiki context add mywiki ./CONTEXT.md`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddContextDoc(r.Config(), args[0], args[1])
	},
}

func init() {
	wikiContextCmd.AddCommand(wikiContextAddCmd)
}
