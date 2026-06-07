package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/verdverm/gmd/pkg/config"
)

var ErrNoAgentConfig = errors.New("no agent config: add an 'agent:' section to gmd config")

type LaunchOptions struct {
	Name          string
	HarnessName   string
	Message       string
	Flags         map[string]string
	Args          []string
	Env           map[string]string
	ConfigFile    string
	Cwd           string
	Tmux          bool
	TmuxConf      string
	Workspace     bool
	WorkspaceBase string
	Async         bool
	DryRun        bool
	Verbose       bool
}

type Harness interface {
	Name() string
	BuildCommand(opts LaunchOptions) (*exec.Cmd, error)
}

func Launch(ctx context.Context, cfg *config.Config, profileName string, overrides LaunchOptions) (err error) {
	if cfg.Agent.Harnesses == nil && cfg.Agent.Profiles == nil {
		return ErrNoAgentConfig
	}

	h, resolved, resolveErr := ResolveAgentConfig(cfg, profileName)
	if resolveErr != nil {
		return resolveErr
	}

	opts := mergeOptions(*resolved, overrides)

	if opts.DryRun {
		printDryRun(h, opts)
		return nil
	}

	if opts.Tmux {
		if opts.Name == "" {
			return fmt.Errorf("<name> is required when using --tmux or --workspace")
		}
		if e := validateName(opts.Name); e != nil {
			return e
		}
	}
	if opts.Workspace {
		if opts.Name == "" {
			return fmt.Errorf("<name> is required when using --tmux or --workspace")
		}
	}

	if opts.Workspace {
		wsPath, wsErr := setupWorkspace(cfg.ProjectRoot, opts.Name, opts.WorkspaceBase)
		if wsErr != nil {
			return wsErr
		}
		if opts.Tmux {
			defer func() {
				if err != nil {
					_ = removeWorkspace(cfg.ProjectRoot, opts.Name)
				}
			}()
		}
		opts.Cwd = wsPath
	} else if opts.Cwd != "" {
		if !filepathAbs(opts.Cwd) {
			opts.Cwd = filepathJoin(cfg.ProjectRoot, opts.Cwd)
		}
	}

	if opts.Tmux {
		binPath, binErr := resolveBin(h)
		if binErr != nil {
			return fmt.Errorf("tmux: %w", binErr)
		}
		files, fileErr := buildTmuxFiles(binPath, opts)
		if fileErr != nil {
			return fileErr
		}
		if opts.Async {
			return nil
		}

		setup := exec.Command("bash", files.setupFile)
		if out, runErr := setup.CombinedOutput(); runErr != nil {
			return fmt.Errorf("tmux: setup failed: %w\n%s", runErr, out)
		}

		tmuxPath, lookErr := exec.LookPath("tmux")
		if lookErr != nil {
			return fmt.Errorf("tmux: tmux not found on PATH")
		}
		attach := exec.Command(tmuxPath, "attach-session", "-d", "-t", opts.Name)
		attach.Stdin = os.Stdin
		attach.Stdout = os.Stdout
		attach.Stderr = os.Stderr
		return attach.Run()
	}

	cmd, buildErr := h.BuildCommand(opts)
	if buildErr != nil {
		return buildErr
	}

	if opts.Async {
		return cmd.Start()
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return err
}

func mergeOptions(resolved LaunchOptions, overrides LaunchOptions) LaunchOptions {
	opts := resolved
	if overrides.Name != "" {
		opts.Name = overrides.Name
	}
	if overrides.HarnessName != "" {
		opts.HarnessName = overrides.HarnessName
	}
	if overrides.Message != "" {
		opts.Message = overrides.Message
	}
	if overrides.ConfigFile != "" {
		opts.ConfigFile = overrides.ConfigFile
	}
	if overrides.Cwd != "" {
		opts.Cwd = overrides.Cwd
	}
	if overrides.Tmux {
		opts.Tmux = true
	}
	if overrides.TmuxConf != "" {
		opts.TmuxConf = overrides.TmuxConf
	}
	if overrides.Workspace {
		opts.Workspace = true
	}
	if overrides.WorkspaceBase != "" {
		opts.WorkspaceBase = overrides.WorkspaceBase
	}
	if overrides.Async {
		opts.Async = true
	}
	if overrides.DryRun {
		opts.DryRun = true
	}
	if overrides.Verbose {
		opts.Verbose = true
	}
	if overrides.Flags != nil {
		if opts.Flags == nil {
			opts.Flags = make(map[string]string)
		}
		for k, v := range overrides.Flags {
			opts.Flags[k] = v
		}
	}
	if overrides.Args != nil {
		opts.Args = append(opts.Args, overrides.Args...)
	}
	if overrides.Env != nil {
		if opts.Env == nil {
			opts.Env = make(map[string]string)
		}
		for k, v := range overrides.Env {
			opts.Env[k] = v
		}
	}
	return opts
}

func printDryRun(h Harness, opts LaunchOptions) {
	fmt.Printf("Dry run: agent launch\n")
	fmt.Printf("  Harness: %s\n", h.Name())
	if opts.Name != "" {
		fmt.Printf("  Name: %s\n", opts.Name)
	}
	if opts.Message != "" {
		fmt.Printf("  Message: %s\n", opts.Message)
	}
	if opts.ConfigFile != "" {
		fmt.Printf("  Config: %s\n", opts.ConfigFile)
	}
	if opts.Cwd != "" {
		fmt.Printf("  CWD: %s\n", opts.Cwd)
	}
	if opts.Tmux {
		fmt.Printf("  Tmux: true\n")
		if opts.TmuxConf != "" {
			fmt.Printf("  TmuxConf: %s\n", opts.TmuxConf)
		}
	}
	if opts.Workspace {
		fmt.Printf("  Workspace: true (base ref: %s)\n", opts.WorkspaceBase)
	}
	if opts.Async {
		fmt.Printf("  Async: true\n")
	}
	if len(opts.Flags) > 0 {
		fmt.Printf("  Flags:\n")
		for k, v := range opts.Flags {
			fmt.Printf("    --%s: %s\n", k, v)
		}
	}
	if len(opts.Args) > 0 {
		fmt.Printf("  Args: %v\n", opts.Args)
	}
	if len(opts.Env) > 0 {
		fmt.Printf("  Env:\n")
		for k, v := range opts.Env {
			fmt.Printf("    %s=%s\n", k, v)
		}
	}

	if opts.Tmux && opts.Verbose {
		binPath, err := resolveBin(h)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gmd] tmux: failed to resolve binary: %v\n", err)
		} else {
			tmuxDryRunInfo(binPath, opts)
		}
	}

	cmd, err := h.BuildCommand(opts)
	if err != nil {
		fmt.Printf("  Error building command: %v\n", err)
		return
	}
	fmt.Printf("\n  Resolved binary: %s\n", cmd.Path)
	fmt.Printf("  Full command: %v\n", cmd.Args)
}

func resolveBin(h Harness) (string, error) {
	type binResolver interface {
		Bin() string
	}
	if br, ok := h.(binResolver); ok {
		return resolveBinPath(br.Bin())
	}
	return "", fmt.Errorf("harness %q does not expose bin path", h.Name())
}
