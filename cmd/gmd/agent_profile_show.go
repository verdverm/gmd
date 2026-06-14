package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var agentProfileShowCmd = &cobra.Command{
	Use:   "show <profile>",
	Short: "Show resolved configuration for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		h, resolved, err := agent.ResolveAgentConfig(cfg, args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Profile: %s\n", args[0])
		fmt.Printf("  Harness: %s\n", h.Name())
		if resolved.Message != "" {
			fmt.Printf("  Message: %s\n", resolved.Message)
		}
		if resolved.ConfigFile != "" {
			fmt.Printf("  ConfigFile: %s\n", resolved.ConfigFile)
		}
		if resolved.Cwd != "" {
			fmt.Printf("  CWD: %s\n", resolved.Cwd)
		}
		if resolved.Tmux {
			fmt.Printf("  Tmux: true\n")
			if resolved.TmuxConf != "" {
				fmt.Printf("  TmuxConf: %s\n", resolved.TmuxConf)
			}
		}
		if resolved.Workspace {
			baseRef := resolved.WorkspaceBase
			if baseRef == "" {
				baseRef = "(current branch)"
			}
			fmt.Printf("  Workspace: %s\n", baseRef)
		}
		if resolved.Async {
			fmt.Printf("  Async: true\n")
		}
		if len(resolved.Flags) > 0 {
			fmt.Println("  Flags:")
			for k, v := range resolved.Flags {
				fmt.Printf("    --%s: %s\n", k, v)
			}
		}
		if len(resolved.Args) > 0 {
			fmt.Println("  Args:")
			for _, a := range resolved.Args {
				fmt.Printf("    %s\n", a)
			}
		}
		if len(resolved.Env) > 0 {
			fmt.Println("  Env:")
			for k, v := range resolved.Env {
				fmt.Printf("    %s=%s\n", k, v)
			}
		}

		return nil
	},
}
