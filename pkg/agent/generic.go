package agent

import (
	"fmt"
	"os"
	"os/exec"
)

type genericHarness struct {
	name      string
	bin       string
	flagStyle string
	env       map[string]string
}

func (h *genericHarness) Name() string { return h.name }

func (h *genericHarness) Bin() string { return h.bin }

func (h *genericHarness) BuildCommand(opts LaunchOptions) (*exec.Cmd, error) {
	binPath, err := resolveBinPath(h.bin)
	if err != nil {
		return nil, fmt.Errorf("harness %q: %w", h.name, err)
	}
	prefix := "--"
	if h.flagStyle == "single-dash" {
		prefix = "-"
	}
	args := make([]string, 0)
	if opts.Message != "" {
		args = append(args, prefix+"message", opts.Message)
	}
	if opts.ConfigFile != "" {
		args = append(args, prefix+"config", opts.ConfigFile)
	}
	for k, v := range opts.Flags {
		args = append(args, prefix+k, v)
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
