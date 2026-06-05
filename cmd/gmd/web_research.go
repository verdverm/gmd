package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	webResearchDepth   string
	webResearchMaxSrc  int
	webResearchOutput  string
	webResearchOutFile string
	webResearchFormat  string
	webResearchSave    bool
	webResearchWiki    string
)

var webResearchCmd = &cobra.Command{
	Use:   "research <query>",
	Short: "Deep research with structured multi-phase pipeline",
	Long: `Deep research agent that performs multi-layered exploration with
assumption testing, cross-validation, and structured reporting.

Tier 3 — builds on the Tier 2 agent but adds structured phases:
decompose → explore → cross-reference → validate → fill → synthesize.

Requires SearchProvider + BrowserProvider + LLM.

Examples:
  gmd web research "environmental impact of EV batteries"
  gmd web research "WebAssembly in production" --depth deep -o wasm-report.md
  gmd web research "post-quantum cryptography standards" --save`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not yet implemented: gmd web research (Phase 5)")
	},
}

func init() {
	webResearchCmd.Flags().StringVar(&webResearchDepth, "depth", "medium", "Research depth: shallow, medium, deep")
	webResearchCmd.Flags().IntVar(&webResearchMaxSrc, "max-sources", 20, "Maximum unique sources to consult")
	webResearchCmd.Flags().StringVar(&webResearchOutput, "output", "stdout", "Output destination: stdout or file")
	webResearchCmd.Flags().StringVarP(&webResearchOutFile, "out", "o", "", "File to write report to")
	webResearchCmd.Flags().StringVar(&webResearchFormat, "format", "markdown", "Output format: json or markdown")
	webResearchCmd.Flags().BoolVarP(&webResearchSave, "save", "s", false, "Save to wiki (requires wiki)")
	webResearchCmd.Flags().StringVar(&webResearchWiki, "wiki", "", "Wiki name")
}
