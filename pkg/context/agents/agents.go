package agents

import (
	"fmt"
	"os"
	"path/filepath"
)

func GlobalAgentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gmd", "agents"), nil
}

func ProjectAgentsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".gmd", "agents")
}

func ResolveDir(global bool, projectRoot string) (string, error) {
	if global {
		return GlobalAgentsDir()
	}
	return ProjectAgentsDir(projectRoot), nil
}

func ListAgents(global bool, projectRoot string) ([]string, error) {
	dir, err := ResolveDir(global, projectRoot)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading agents dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func ShowAgent(name string, global bool, projectRoot string) (map[string]string, error) {
	dir, err := ResolveDir(global, projectRoot)
	if err != nil {
		return nil, err
	}

	agentDir := filepath.Join(dir, name)
	info, err := os.Stat(agentDir)
	if err != nil {
		return nil, fmt.Errorf("agent %q not found: %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", name)
	}

	entries, err := os.ReadDir(agentDir)
	if err != nil {
		return nil, fmt.Errorf("reading agent dir %q: %w", name, err)
	}

	files := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(agentDir, e.Name()))
		if err != nil {
			files[e.Name()] = fmt.Sprintf("[error reading: %v]", err)
		} else {
			files[e.Name()] = string(data)
		}
	}
	return files, nil
}
