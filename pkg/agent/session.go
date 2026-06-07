package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SessionInfo struct {
	Name      string
	Tmux      bool
	Workspace bool
	Orphaned  bool
}

type SessionManager struct {
	ProjectRoot string
}

func NewSessionManager(projectRoot string) *SessionManager {
	return &SessionManager{ProjectRoot: projectRoot}
}

func (m *SessionManager) ListSessions() ([]SessionInfo, error) {
	var sessions []SessionInfo
	seen := make(map[string]bool)

	tmuxPath, err := exec.LookPath("tmux")
	if err == nil {
		cmd := exec.Command(tmuxPath, "list-sessions", "-F", "#{session_name}")
		out, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				name := strings.TrimSpace(line)
				if name == "" {
					continue
				}
				si := SessionInfo{Name: name, Tmux: true}
				if m.hasWorkspace(name) {
					si.Workspace = true
				}
				sessions = append(sessions, si)
				seen[name] = true
			}
		}
	}

	wsDir := filepath.Join(m.ProjectRoot, ".workspaces")
	entries, err := os.ReadDir(wsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if seen[entry.Name()] {
				continue
			}
			sessions = append(sessions, SessionInfo{
				Name:      entry.Name(),
				Workspace: true,
				Orphaned:  true,
			})
		}
	}

	return sessions, nil
}

func (m *SessionManager) KillSession(name string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err == nil {
		cmd := exec.Command(tmuxPath, "kill-session", "-t", name)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}

	if m.hasWorkspace(name) {
		return removeWorkspace(m.ProjectRoot, name)
	}

	return nil
}

func (m *SessionManager) MergeSession(name string, squash bool) error {
	wsPath := filepath.Join(m.ProjectRoot, ".workspaces", name)
	if _, err := os.Stat(wsPath); os.IsNotExist(err) {
		return fmt.Errorf("workspace %q not found under .workspaces/", name)
	}

	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = wsPath
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("merge: failed to get workspace branch: %w", err)
	}
	wsBranch := strings.TrimSpace(string(out))

	mergeArgs := []string{"merge"}
	if squash {
		mergeArgs = append(mergeArgs, "--squash")
	}
	mergeArgs = append(mergeArgs, wsBranch)

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("merge: git not found on PATH")
	}

	mergeCmd := exec.Command(gitPath, mergeArgs...)
	mergeCmd.Dir = m.ProjectRoot
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr
	fmt.Fprintf(os.Stderr, "Merging %s/%s into %s\n", name, wsBranch, m.ProjectRoot)
	return mergeCmd.Run()
}

func (m *SessionManager) hasWorkspace(name string) bool {
	wsPath := filepath.Join(m.ProjectRoot, ".workspaces", name)
	_, err := os.Stat(wsPath)
	return err == nil
}
