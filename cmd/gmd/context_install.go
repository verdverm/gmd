package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/context/skills"
)

var contextInstallCmd = &cobra.Command{
	Use:   "install [--target claude|codex|opencode|all]",
	Short: "Install skills to harness discovery paths",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if contextTarget != "" && contextTarget != "claude" && contextTarget != "codex" && contextTarget != "opencode" && contextTarget != "all" {
			return fmt.Errorf("invalid --target %q (valid: claude, codex, opencode, all)", contextTarget)
		}
		return nil
	},
	Long: `Writes skills to the appropriate harness discovery directories
so that AI coding tools discover and use them automatically.

Use --global to target global (home directory) scope.

Examples:
  gmd context install --target all
  gmd context install --target opencode
  gmd context install --global --target all`,
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

		harnesses := []string{target}
		if target == "all" {
			harnesses = skills.HarnessNames()
		}

		for _, h := range harnesses {
			dest, err := skills.WriteSkillTo(baseDir, isGlobal, h)
			if err != nil {
				return fmt.Errorf("writing skill for %s: %w", h, err)
			}
			fmt.Printf("  %s: installed to %s\n", h, dest)
		}
		return nil
	},
}

func init() {
	contextInstallCmd.Flags().StringVar(&contextTarget, "target", "", "Target agent harness (claude, codex, opencode, all)")

	contextCmd.AddCommand(contextInstallCmd)
}
