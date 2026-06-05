package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var wikiContextRmCmd = &cobra.Command{
	Use:   "rm <wiki>",
	Short: "Remove a context document from a wiki",
	Long: `Removes the context document association from the wiki. The source
file on disk is not deleted — only the config reference is removed.

Example:
  gmd wiki context rm mywiki`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.RemoveContextDoc(r.Config(), args[0])
	},
}

func init() {
	wikiContextCmd.AddCommand(wikiContextRmCmd)
}
