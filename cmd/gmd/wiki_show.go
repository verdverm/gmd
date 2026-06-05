package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var wikiShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show wiki config details and chunk count",
	Long: `Displays the full configuration for a wiki including path, wikiDir,
rawDir, patterns, sourceRefs, and chunk count.

Example:
  gmd wiki show mywiki`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]
		cfg := r.Config()

		wc, ok := cfg.Wikis[name]
		if !ok {
			return fmt.Errorf("wiki %q not found", name)
		}

		fmt.Printf("name:              %s\n", name)
		fmt.Printf("path:              %s\n", wc.Path)
		fmt.Printf("wikiDir:           %s\n", wc.WikiDir)
		fmt.Printf("rawDir:            %s\n", wc.RawDir)
		fmt.Printf("indexFile:         %s\n", wc.IndexFile)
		fmt.Printf("logFile:           %s\n", wc.LogFile)
		fmt.Printf("graphLinks:        %v\n", wc.GraphLinks)
		fmt.Printf("patterns:          %v\n", wc.Patterns)
		if len(wc.Ignore) > 0 {
			fmt.Printf("ignore:            %v\n", wc.Ignore)
		}
		if wc.Context != "" {
			fmt.Printf("context:           %s\n", wc.Context)
		}
		fmt.Printf("excludeFromDefault: %v\n", wc.ExcludeFromDefault)
		if len(wc.SourceRefs) > 0 {
			fmt.Printf("sourceRefs:        %v\n", wc.SourceRefs)
		}

		key := cfg.CollectionKey(name)
		ctx := context.Background()
		counts, err := r.TSClient().CountByCollection(ctx, []string{key})
		if err != nil {
			fmt.Printf("chunks:            (error counting: %v)\n", err)
		} else {
			fmt.Printf("chunks:            %d\n", counts[key])
		}
		return nil
	},
}

func init() {
	wikiCmd.AddCommand(wikiShowCmd)
}
