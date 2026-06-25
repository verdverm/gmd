package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
	"github.com/verdverm/gmd/pkg/wiki"
)

var wikiDoctorCmd = &cobra.Command{
	Use:   "doctor <name> [--fix]",
	Short: "Run wiki diagnostics and auto-configure agents",
	Long: `Checks wiki configuration, file system structure, Typesense sync state,
and agent compatibility. Reports issues and suggests fixes.

Use --fix to automatically apply safe fixes (fixable issues only).
After fixes, automatically launches the agent harness (use --async to skip).

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
		registry, err := newRegistry(cfg)
		if err != nil {
			return fmt.Errorf("resolving LLM config: %w", err)
		}

		ctx := context.Background()
		result, err := wiki.Doctor(ctx, w, cfg, tsClient, registry)
		if err != nil {
			return err
		}

		fmt.Print(wiki.FormatDoctorResult(result))

		doFix, _ := cmd.Flags().GetBool("fix")
		asyncFlag, _ := cmd.Flags().GetBool("async")

		if doFix {
			fixes, err := wiki.DoctorFix(w, cfg)
			if err != nil {
				return err
			}
			if len(fixes) > 0 {
				fmt.Printf("\n  Fixes applied:\n")
				for _, f := range fixes {
					fmt.Println(f)
				}
			}

			if len(fixes) > 0 {
				profileName := "wiki"
				if _, _, err := agent.ResolveAgentConfig(cfg, profileName); err != nil {
					profileName = ""
				}

				opts := agent.LaunchOptions{
					Name:    name,
					Message: fmt.Sprintf("Work on the wiki '%s'. Run /help for tools.", name),
					Async:   asyncFlag,
				}
				err := agent.Launch(ctx, cfg, profileName, opts)
				if err != nil {
					if err == agent.ErrNoAgentConfig {
						fmt.Printf("\nHint: add an 'agent:' section to gmd config to auto-launch after doctor fixes.\n")
					} else {
						return err
					}
				}
			}
		}

		return nil
	},
}

func init() {
	wikiDoctorCmd.Flags().Bool("fix", false, "Apply safe fixes automatically")
	wikiDoctorCmd.Flags().Bool("async", false, "Launch agent in background after fixes (don't block)")
	wikiCmd.AddCommand(wikiDoctorCmd)
}
