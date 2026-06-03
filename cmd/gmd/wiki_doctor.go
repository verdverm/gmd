package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiDoctorCmd = &cobra.Command{
	Use:   "doctor [--name <name>] [--fix]",
	Short: "Run wiki diagnostics and auto-configure agents",
	Long: `Checks wiki configuration, file system structure, Typesense sync state,
and agent compatibility. Reports issues and suggests fixes.

Use --fix to automatically apply safe fixes (fixable issues only).

Example:
  gmd wiki doctor --name mywiki --fix`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		cfg := r.Config()

		if wikiName == "" {
			return fmt.Errorf("wiki name required (--name)")
		}

		col, ok := cfg.Collections[wikiName]
		if !ok {
			return fmt.Errorf("wiki collection %q not found", wikiName)
		}

		w, err := wiki.NewWiki(wikiName, col.Path, col)
		if err != nil {
			return err
		}

		tsClient := r.TSClient()
		llmClient := llm.New(llmConfigFromConfig(cfg))

		ctx := context.Background()
		result, err := wiki.Doctor(ctx, w, cfg, tsClient, llmClient)
		if err != nil {
			return err
		}

		fmt.Print(wiki.FormatDoctorResult(result))

		doFix, _ := cmd.Flags().GetBool("fix")
		if doFix {
			fixes, err := wiki.DoctorFix(w)
			if err != nil {
				return err
			}
			if len(fixes) > 0 {
				fmt.Printf("\n  Fixes applied:\n")
				for _, f := range fixes {
					fmt.Println(f)
				}
			}
		}

		return nil
	},
}

func init() {
	wikiDoctorCmd.Flags().Bool("fix", false, "Apply safe fixes automatically")
	wikiCmd.AddCommand(wikiDoctorCmd)
}
