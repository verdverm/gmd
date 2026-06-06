package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiDoctorCmd = &cobra.Command{
	Use:   "doctor <name> [--fix]",
	Short: "Run wiki diagnostics and auto-configure agents",
	Long: `Checks wiki configuration, file system structure, Typesense sync state,
and agent compatibility. Reports issues and suggests fixes.

Use --fix to automatically apply safe fixes (fixable issues only).

Example:
  gmd wiki doctor mywiki --fix`,
	Args: cobra.MinimumNArgs(1),
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

		tsClient := r.TSClient()
		llmClient, err := llmConfigFromConfig(cfg)
		if err != nil {
			return fmt.Errorf("resolving LLM config: %w", err)
		}

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
