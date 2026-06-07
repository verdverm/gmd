package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/agent"
)

var (
	agentProfile       string
	agentMessage       string
	agentConfigFile    string
	agentCwd           string
	agentTmux          bool
	agentTmuxConf      string
	agentWorkspace     bool
	agentWorkspaceBase string
	agentAsync         bool
	agentDryRun        bool
	agentFlagFlags     []string
	agentEnvFlags      []string
	agentExtraArgs     []string
)

var agentCmd = &cobra.Command{
	Use:   "agent [task-name] [message] [flags]",
	Short: "Launch external AI agent harnesses",
	Long: `Launch an external AI agent harness.

The first argument is the task name (used for tmux session and/or git workspace naming).
The second argument is an optional message to pass to the agent.

Profile selection:
  1. --profile flag (preferred)
  2. Falls back to defaultHarness if set in config
  3. If no profile and no default, error

Examples:
  gmd agent mytask                            # launch with default harness
  gmd agent mytask "fix the bug"              # launch with message
  gmd agent mytask --profile wiki             # launch with "wiki" profile
  gmd agent mytask -p dev --tmux --workspace  # launch dev profile in tmux + workspace
  gmd agent mytask --tmux --tmux-conf ~/.tmux/minimal.conf  # launch with custom tmux config
  gmd agent list                              # list configured harnesses + profiles
  gmd agent profile list                      # list profiles
  gmd agent profile show wiki                 # show resolved config for "wiki"
  gmd agent session list                      # list active sessions
  gmd agent session kill mytask               # kill session + workspace
  gmd agent session merge mytask              # merge workspace into main branch`,
	Args: cobra.ArbitraryArgs,
	RunE: runAgent,
}

func runAgent(cmd *cobra.Command, args []string) error {
	cfg, err := getConfig()
	if err != nil {
		return err
	}

	var taskName string
	var message string
	if len(args) > 0 {
		taskName = args[0]
	}
	if len(args) > 1 {
		message = args[1]
	}

	if (agentTmux || agentWorkspace) && taskName == "" {
		return fmt.Errorf("<task-name> is required when using --tmux or --workspace")
	}

	if agentMessage != "" {
		message = agentMessage
	}

	overrides := agent.LaunchOptions{
		Name:          taskName,
		Message:       message,
		ConfigFile:    agentConfigFile,
		Cwd:           agentCwd,
		Tmux:          agentTmux,
		TmuxConf:      agentTmuxConf,
		Workspace:     agentWorkspace,
		WorkspaceBase: agentWorkspaceBase,
		Async:         agentAsync,
		DryRun:        agentDryRun,
		Verbose:       verboseFlag,
		Flags:         parseKeyValueSlice(agentFlagFlags),
		Env:           parseKeyValueSlice(agentEnvFlags),
		Args:          agentExtraArgs,
	}

	return agent.Launch(cmd.Context(), cfg, agentProfile, overrides)
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agent harnesses and profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		harnesses := agent.ListHarnesses(cfg)
		profiles := agent.ListProfiles(cfg)

		if len(harnesses) == 0 && len(profiles) == 0 {
			fmt.Println("No agent harnesses or profiles configured.")
			fmt.Println("Add an 'agent:' section to your gmd config.")
			return nil
		}

		if len(harnesses) > 0 {
			fmt.Println("Harnesses:")
			for _, name := range harnesses {
				hc := cfg.Agent.Harnesses[name]
				marker := ""
				if cfg.Agent.DefaultHarness == name {
					marker = " (default)"
				}
				fmt.Printf("  %s%s: bin=%s\n", name, marker, hc.Bin)
			}
		}

		if len(profiles) > 0 {
			fmt.Println("\nProfiles:")
			for _, name := range profiles {
				p := cfg.Agent.Profiles[name]
				fmt.Printf("  %s: harness=%s", name, p.Harness)
				if p.Message != "" {
					fmt.Printf(", message=%q", p.Message)
				}
				if p.Tmux {
					fmt.Printf(", tmux")
				}
				if p.Workspace {
					fmt.Printf(", workspace")
				}
				if p.Async {
					fmt.Printf(", async")
				}
				fmt.Println()
			}
		}

		return nil
	},
}

var agentProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage agent profiles",
}

var agentProfileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agent profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := getConfig()
		if err != nil {
			return err
		}

		profiles := agent.ListProfiles(cfg)

		if len(profiles) == 0 {
			fmt.Println("No agent profiles configured.")
			return nil
		}

		fmt.Println("Profiles:")
		for _, name := range profiles {
			p := cfg.Agent.Profiles[name]
			fmt.Printf("  %s: harness=%s", name, p.Harness)
			if p.Message != "" {
				fmt.Printf(", message=%q", p.Message)
			}
			if p.Tmux {
				fmt.Printf(", tmux")
			}
			if p.Workspace {
				fmt.Printf(", workspace")
			}
			if p.Async {
				fmt.Printf(", async")
			}
			fmt.Println()
		}

		return nil
	},
}

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

func parseKeyValueSlice(items []string) map[string]string {
	result := make(map[string]string)
	for _, item := range items {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func init() {
	agentCmd.Flags().StringVarP(&agentProfile, "profile", "p", "", "Profile or harness name to launch")
	agentCmd.Flags().StringVarP(&agentMessage, "message", "m", "", "Message/prompt for the agent")
	agentCmd.Flags().StringVar(&agentConfigFile, "config", "", "Path to harness-specific config file")
	agentCmd.Flags().StringVar(&agentCwd, "cwd", "", "Working directory (relative to project root unless absolute)")
	agentCmd.Flags().BoolVar(&agentTmux, "tmux", false, "Launch inside a named tmux session")
	agentCmd.Flags().StringVar(&agentTmuxConf, "tmux-conf", "", "Path to tmux config file for the session")
	agentCmd.Flags().BoolVar(&agentWorkspace, "workspace", false, "Create a git worktree before launching")
	agentCmd.Flags().StringVar(&agentWorkspaceBase, "workspace-base", "", "Git ref for worktree (defaults to current branch)")
	agentCmd.Flags().BoolVar(&agentAsync, "async", false, "Don't block; return after launching")
	agentCmd.Flags().BoolVar(&agentDryRun, "dry-run", false, "Print resolved command without executing")
	agentCmd.Flags().StringArrayVar(&agentFlagFlags, "flag", nil, "Extra flag for the harness KEY=VAL (repeatable)")
	agentCmd.Flags().StringArrayVar(&agentEnvFlags, "env", nil, "Extra env var KEY=VAL (repeatable)")
	agentCmd.Flags().StringArrayVar(&agentExtraArgs, "args", nil, "Extra positional args to pass to the harness")

	agentProfileCmd.AddCommand(agentProfileListCmd)
	agentProfileCmd.AddCommand(agentProfileShowCmd)

	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentProfileCmd)
	agentCmd.AddCommand(agentSessionCmd)
	rootCmd.AddCommand(agentCmd)
}
