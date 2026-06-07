package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func generateTmuxScript(msgPath, binPath string, harnessArgs []string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
cleanup() { rm -f "%[1]s" "$0"; }
trap cleanup EXIT

GMD_AGENT_INPUT=$(cat "%[1]s")

echo "=== gmd agent launching ===" >&2
echo "  binary: %[2]s" >&2
if [ -n "$GMD_AGENT_INPUT" ]; then
  echo "  message: $GMD_AGENT_INPUT" >&2
fi
echo "" >&2

if [ -n "$GMD_AGENT_INPUT" ]; then
  "%[2]s" --prompt "$GMD_AGENT_INPUT" %[3]s
else
  "%[2]s" %[3]s
fi
RC=$?

echo "" >&2
echo "=== agent exited (code: $RC) ===" >&2
echo "Press enter to close window..."
read -r
`, msgPath, binPath, shellQuote(harnessArgs))
}

func generateTmuxSetupScript(tmuxPath, sessionName, cwd, agentScriptPath, tmuxConf string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("set -euo pipefail\n")
	fmt.Fprintf(&sb, "%s new-session -d -s %s -c %s -n shell", tmuxPath, sessionName, cwd)
	if tmuxConf != "" {
		fmt.Fprintf(&sb, " -f %s", tmuxConf)
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "%s new-window -t %s -c %s -n agent %s\n", tmuxPath, sessionName, cwd, agentScriptPath)
	return sb.String()
}

type tmuxFiles struct {
	msgFile   string
	agentFile string
	setupFile string
}

func buildTmuxFiles(binPath string, opts LaunchOptions) (*tmuxFiles, error) {
	sessionName := opts.Name
	cwd := opts.Cwd
	if cwd == "" {
		cwd = "."
	}

	msgFile, err := os.CreateTemp("", "gmd-agent-msg-*.txt")
	if err != nil {
		return nil, fmt.Errorf("tmux: failed to create message temp file: %w", err)
	}
	if _, err := msgFile.WriteString(opts.Message); err != nil {
		msgFile.Close()
		return nil, err
	}
	msgFile.Close()

	var harnessArgs []string
	if opts.ConfigFile != "" {
		harnessArgs = append(harnessArgs, "--config", opts.ConfigFile)
	}
	for k, v := range opts.Flags {
		harnessArgs = append(harnessArgs, "--"+k, v)
	}
	harnessArgs = append(harnessArgs, opts.Args...)

	agentScript := generateTmuxScript(msgFile.Name(), binPath, harnessArgs)

	agentFile, err := os.CreateTemp("", "gmd-agent-script-*.sh")
	if err != nil {
		return nil, fmt.Errorf("tmux: failed to create agent script temp file: %w", err)
	}
	if _, err := agentFile.WriteString(agentScript); err != nil {
		agentFile.Close()
		return nil, err
	}
	agentFile.Close()
	if err := os.Chmod(agentFile.Name(), 0755); err != nil {
		return nil, err
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux: tmux not found on PATH (required for --tmux)")
	}

	setupScript := generateTmuxSetupScript(tmuxPath, sessionName, cwd, agentFile.Name(), opts.TmuxConf)

	setupFile, err := os.CreateTemp("", "gmd-tmux-setup-*.sh")
	if err != nil {
		return nil, fmt.Errorf("tmux: failed to create setup script temp file: %w", err)
	}
	if _, err := setupFile.WriteString(setupScript); err != nil {
		setupFile.Close()
		return nil, err
	}
	setupFile.Close()
	if err := os.Chmod(setupFile.Name(), 0755); err != nil {
		return nil, err
	}

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "[gmd] tmux: msg file = %s\n", msgFile.Name())
		fmt.Fprintf(os.Stderr, "[gmd] tmux: msg file content: %s\n", opts.Message)
		fmt.Fprintf(os.Stderr, "[gmd] tmux: agent file = %s\n", agentFile.Name())
		fmt.Fprintf(os.Stderr, "[gmd] tmux: agent file content:\n%s\n", agentScript)
		fmt.Fprintf(os.Stderr, "[gmd] tmux: setup file = %s\n", setupFile.Name())
		fmt.Fprintf(os.Stderr, "[gmd] tmux: setup file content:\n%s\n", setupScript)
		fmt.Fprintf(os.Stderr, "[gmd] tmux: attach cmd = %s attach -t %s\n", tmuxPath, sessionName)
	}

	return &tmuxFiles{
		msgFile:   msgFile.Name(),
		agentFile: agentFile.Name(),
		setupFile: setupFile.Name(),
	}, nil
}

func tmuxDryRunInfo(binPath string, opts LaunchOptions) {
	msgPath := filepath.Join(os.TempDir(), "gmd-agent-msg-XXXX.txt")
	agentPath := filepath.Join(os.TempDir(), "gmd-agent-script-XXXX.sh")
	setupPath := filepath.Join(os.TempDir(), "gmd-tmux-setup-XXXX.sh")

	var harnessArgs []string
	if opts.ConfigFile != "" {
		harnessArgs = append(harnessArgs, "--config", opts.ConfigFile)
	}
	for k, v := range opts.Flags {
		harnessArgs = append(harnessArgs, "--"+k, v)
	}
	harnessArgs = append(harnessArgs, opts.Args...)

	agentScript := generateTmuxScript(msgPath, binPath, harnessArgs)

	cwd := opts.Cwd
	if cwd == "" {
		cwd = "."
	}

	setupScript := generateTmuxSetupScript("tmux", opts.Name, cwd, agentPath, opts.TmuxConf)

	fmt.Fprintf(os.Stderr, "[gmd] tmux: session = %s, cwd = %s\n", opts.Name, cwd)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: agent binary = %s\n", binPath)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: msg file = %s\n", msgPath)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: msg file content: %s\n", opts.Message)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: agent file = %s\n", agentPath)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: agent file content:\n%s\n", agentScript)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: setup file = %s\n", setupPath)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: setup file content:\n%s\n", setupScript)
	fmt.Fprintf(os.Stderr, "[gmd] tmux: exec cmd = bash %s ; tmux attach-session -d -t %s\n", setupPath, opts.Name)
}

func shellQuote(args []string) string {
	var quoted []string
	for _, a := range args {
		quoted = append(quoted, fmt.Sprintf("%q", a))
	}
	return strings.Join(quoted, " ")
}
