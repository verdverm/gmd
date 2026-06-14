package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiOkfExportOutput string

var wikiOkfExportCmd = &cobra.Command{
	Use:   "export <name> [--output <dir>]",
	Short: "Export wiki as a standalone OKF bundle",
	Long: `Export a wiki as an Open Knowledge Format (OKF) v0.1 bundle directory.
Converts [[wikilinks]] to standard markdown links, ensures frontmatter compliance,
and writes a self-contained directory of markdown files consumable by any
OKF-compatible system.

Example:
  gmd wiki okf export mywiki
  gmd wiki okf export mywiki --output ./okf-export`,
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

		outputDir := wikiOkfExportOutput
		if outputDir == "" {
			outputDir = filepath.Join(w.Path, "okf-export")
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
	wikiOkfExportCmd.Flags().StringVar(&wikiOkfExportOutput, "output", "", "Output directory (default: <wiki-dir>/okf-export)")

	wikiOkfCmd.AddCommand(wikiOkfExportCmd)
	wikiCmd.AddCommand(wikiOkfCmd)
}

var wikiOkfCmd = &cobra.Command{
	Use:   "okf [export]",
	Short: "Open Knowledge Format (OKF) operations for wikis",
	Long:  `Export and validate wikis against the Open Knowledge Format (OKF) v0.1 specification.`,
}
