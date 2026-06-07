package agent

import (
	"fmt"
	"os"
	"os/exec"
)

type opencodeHarness struct {
	name string
	bin  string
	env  map[string]string
}

func (h *opencodeHarness) Name() string { return h.name }

func (h *opencodeHarness) Bin() string { return h.bin }

func (h *opencodeHarness) BuildCommand(opts LaunchOptions) (*exec.Cmd, error) {
	binPath, err := resolveBinPath(h.bin)
	if err != nil {
		return nil, fmt.Errorf("harness 'opencode': %w", err)
	}
	args := make([]string, 0)
	if opts.Message != "" {
		args = append(args, "run", opts.Message)
	}
	if opts.ConfigFile != "" {
		args = append(args, "--config", opts.ConfigFile)
	}
	for k, v := range opts.Flags {
		args = append(args, "--"+k, v)
	}
	args = append(args, opts.Args...)
	cmd := exec.Command(binPath, args...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	cmd.Env = os.Environ()
	for k, v := range h.env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd, nil
}
