package main

import (
	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var contextRmCmd = &cobra.Command{
	Use:   "rm <collection>",
	Short: "Remove a context document from a collection",
	Long: `Removes the context document association from the collection. The source
file on disk is not deleted — only the config reference is removed.

Example:
  gmd context rm docs`,
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
	contextCmd.AddCommand(contextRmCmd)
}
