package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiExportOutput string

var wikiExportCmd = &cobra.Command{
	Use:   "export <name> [--output <dir>]",
	Short: "Export wiki as a standalone directory",
	Long: `Export a wiki as a self-contained directory of markdown files
consumable by any OKF v0.1 compatible system.

Example:
  gmd wiki export mywiki
  gmd wiki export mywiki --output ./mywiki-export`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		name := args[0]

		wc, ok := cfg.Wikis[name]
		if !ok {
			return fmt.Errorf("wiki %q not found", name)
		}

		w, err := wiki.NewWiki(name, wc.Path, &wc)
		if err != nil {
			return err
		}

		outputDir := wikiExportOutput
		if outputDir == "" {
			outputDir = filepath.Join(w.Path, "wiki-export")
		}
		if !filepath.IsAbs(outputDir) {
			outputDir = filepath.Join(w.Path, outputDir)
		}

		report, err := wiki.ExportOKF(w, outputDir)
		if err != nil {
			return fmt.Errorf("exporting: %w", err)
		}

		fmt.Printf("Exported %d pages to %s\n", report.PassCount, outputDir)

		return nil
	},
}

func init() {
	wikiExportCmd.Flags().StringVar(&wikiExportOutput, "output", "", "Output directory (default: <wiki-dir>/wiki-export)")
	wikiCmd.AddCommand(wikiExportCmd)
}
