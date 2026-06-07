package agent

import (
	"fmt"
	"os/exec"

	"github.com/verdverm/gmd/pkg/config"
)

func ResolveAgentConfig(cfg *config.Config, profileName string) (Harness, *LaunchOptions, error) {
	if cfg.Agent.Harnesses == nil && cfg.Agent.Profiles == nil {
		return nil, nil, ErrNoAgentConfig
	}

	var harnessName string
	var profile *config.AgentHarnessProfile

	if profileName != "" {
		if p, ok := cfg.Agent.Profiles[profileName]; ok {
			profile = &p
			harnessName = p.Harness
		}
		if harnessName == "" {
			if _, ok := cfg.Agent.Harnesses[profileName]; ok {
				harnessName = profileName
			}
		}
	}

	if harnessName == "" {
		harnessName = cfg.Agent.DefaultHarness
	}

	if harnessName == "" {
		return nil, nil, fmt.Errorf("no agent harness configured: set defaultHarness or specify a profile/harness name")
	}

	hc, ok := cfg.Agent.Harnesses[harnessName]
	if !ok {
		return nil, nil, fmt.Errorf("agent harness %q not found in config", harnessName)
	}

	h := newHarness(harnessName, hc)

	opts := LaunchOptions{
		Name:        profileName,
		HarnessName: harnessName,
	}

	if profile != nil {
		if profile.Message != "" {
			opts.Message = profile.Message
		}
		if profile.ConfigFile != "" {
			opts.ConfigFile = profile.ConfigFile
		}
		if profile.Cwd != "" {
			opts.Cwd = profile.Cwd
		}
		if profile.Tmux {
			opts.Tmux = true
		}
		if profile.Workspace {
			opts.Workspace = true
		}
		if profile.Async {
			opts.Async = true
		}
		if profile.Flags != nil {
			opts.Flags = make(map[string]string)
			for k, v := range profile.Flags {
				opts.Flags[k] = v
			}
		}
		if profile.Args != nil {
			opts.Args = append([]string{}, profile.Args...)
		}
		if profile.Env != nil {
			opts.Env = make(map[string]string)
			for k, v := range profile.Env {
				opts.Env[k] = v
			}
		}
	}

	return h, &opts, nil
}

func ListHarnesses(cfg *config.Config) []string {
	if cfg.Agent.Harnesses == nil {
		return nil
	}
	var names []string
	for k := range cfg.Agent.Harnesses {
		names = append(names, k)
	}
	return names
}

func ListProfiles(cfg *config.Config) []string {
	if cfg.Agent.Profiles == nil {
		return nil
	}
	var names []string
	for k := range cfg.Agent.Profiles {
		names = append(names, k)
	}
	return names
}

func newHarness(name string, hc config.AgentHarnessConfig) Harness {
	switch name {
	case "opencode":
		return &opencodeHarness{
			name: name,
			bin:  hc.Bin,
			env:  hc.Env,
		}
	default:
		return &genericHarness{
			name:      name,
			bin:       hc.Bin,
			flagStyle: hc.FlagStyle,
			env:       hc.Env,
		}
	}
}

func resolveBinPath(bin string) (string, error) {
	if bin == "" {
		return "", fmt.Errorf("bin path is empty")
	}
	p, err := exec.LookPath(bin)
	if err != nil {
		return "", fmt.Errorf("binary %q not found on PATH: %w", bin, err)
	}
	return p, nil
}
