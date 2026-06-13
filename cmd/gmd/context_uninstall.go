package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextUninstallCmd = &cobra.Command{
	Use:   "uninstall [--target claude|codex|opencode|all]",
	Short: "Remove skills from harness discovery paths",
	Long: `Removes skill directories from harness discovery paths. Idempotent:
if a skill is already absent, it is reported as already absent.

Use --global to target global (home directory) scope.

Examples:
  gmd context uninstall --target all
  gmd context uninstall --target claude
  gmd context uninstall --global --target opencode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := contextTarget
		if target == "" {
			target = "all"
		}

		cfg, err := getConfig()
		if err != nil {
			return err
		}

		baseDir := cfg.ProjectRoot
		isGlobal := contextGlobal
		if isGlobal || baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			baseDir = home
			isGlobal = true
		}

		targets := []string{target}
		if target == "all" {
			targets = skills.HarnessNames()
		}

		skillNames, err := skills.ListSkillNames()
		if err != nil {
			return err
		}

		for _, h := range targets {
			for _, s := range skillNames {
				p, err := skills.SkillPath(baseDir, isGlobal, h, s)
				if err != nil {
					return err
				}
				if _, err := os.Stat(p); os.IsNotExist(err) {
					fmt.Printf("  %s/%s: already absent (%s)\n", h, s, p)
					continue
				}
				if err := os.RemoveAll(p); err != nil {
					return fmt.Errorf("removing %s: %w", p, err)
				}
				fmt.Printf("  Removed: %s\n", p)
			}
		}
		return nil
	},
}
