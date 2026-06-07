package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func filepathAbs(p string) bool {
	return filepath.IsAbs(p)
}

func filepathJoin(base, rel string) string {
	return filepath.Join(base, rel)
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required for --tmux / --workspace")
	}
	if strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid name %q: must not contain path separators", name)
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid name %q: must not start with '-'", name)
	}
	return nil
}

func getCurrentBranch(projectRoot string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectRoot
	out, err := cmd.Output()
	if err != nil {
		return "HEAD"
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "HEAD"
	}
	return branch
}

func ensureGitignore(projectRoot string) error {
	giPath := filepath.Join(projectRoot, ".gitignore")
	entry := ".workspaces/"

	data, err := os.ReadFile(giPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if strings.Contains(string(data), entry) {
		return nil
	}

	f, err := os.OpenFile(giPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		if _, err := fmt.Fprint(f, "\n"); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(f, entry)
	return err
}

func setupWorkspace(projectRoot, name, baseRef string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	if baseRef == "" {
		baseRef = getCurrentBranch(projectRoot)
	}

	wsDir := filepath.Join(projectRoot, ".workspaces")
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", fmt.Errorf("workspace: failed to create .workspaces/: %w", err)
	}

	if err := ensureGitignore(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update .gitignore: %v\n", err)
	}

	targetPath := filepath.Join(wsDir, name)

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("workspace: git not found on PATH (required for --workspace)")
	}

	cmd := exec.Command(gitPath, "worktree", "add", targetPath, baseRef)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("workspace: git worktree add failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Created workspace: %s (from %s)\n", targetPath, baseRef)
	return targetPath, nil
}

func removeWorkspace(projectRoot, name string) error {
	targetPath := filepath.Join(projectRoot, ".workspaces", name)
	cmd := exec.Command("git", "worktree", "remove", "--force", targetPath)
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
