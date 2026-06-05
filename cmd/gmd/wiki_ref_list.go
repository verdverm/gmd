package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var wikiRefListCmd = &cobra.Command{
	Use:   "list <wiki>",
	Short: "List source references for a wiki",
	Long: `Lists all source references for a wiki.

Example:
  gmd wiki ref list mywiki`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]

		wc, ok := r.Config().Wikis[name]
		if !ok {
			return fmt.Errorf("wiki %q not found", name)
		}

		if len(wc.SourceRefs) == 0 {
			fmt.Printf("Wiki %q has no source references.\n", name)
			return nil
		}

		fmt.Printf("Source references for %q:\n", name)
		for _, ref := range wc.SourceRefs {
			fmt.Printf("  %s\n", ref)
		}
		return nil
	},
}

func init() {
	wikiRefCmd.AddCommand(wikiRefListCmd)
}
