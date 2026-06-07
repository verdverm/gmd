# Agent Harness Launcher

Created: 2026-06-06
Last Updated: 2026-06-06 (resolved all open questions from TASK.md)
Phase: 1 (design) — implementation TBD

## Context

`gmd` currently detects agent harnesses (opencode, claude, codex) via `pkg/wiki/skills.go`
and checks skill installation status via `wiki doctor`, but cannot launch them. Multiple
areas of the CLI would benefit from being able to launch a user's preferred agent harness
pre-populated with context (message, flags, custom config, env vars):

- `gmd wiki doctor --fix` → launch agent with wiki context
- `gmd web agent` → optionally launch external agent for follow-up research
- New `gmd agent` command → direct user-facing launcher

## Goal

Add a `pkg/agent/` package with an abstraction for launching external AI agent harnesses
via `exec`, a `gmd agent` CLI command group, and config schema (harnesses + profiles)
following the existing LLM/web providers+profiles pattern. Start with opencode support,
design for expansion to claude and others.

## Summary

| Area | Action |
|------|--------|
| `pkg/agent/` | New package: harness interface, exec launcher, config resolution, tmux session, git worktree, rollback |
| `pkg/config/` | New `AgentConfig` in Go struct + CUE schema (`AgentHarnessConfig`, `AgentProfile`) |
| `cmd/gmd/agent*.go` | New `gmd agent` command: `<name>` launches by default; subcommands: `list`, `show`, `session` |
| `cmd/gmd/wiki_doctor.go` | Doctor launches agent after fixes by default; `--async` flag skips attach |
| `pkg/wiki/doctor.go` | Refactor to enumerate agents from `AgentConfig.Harnesses` (config-driven, no hardcoded list) |
| `pkg/wiki/skills.go` | Keep hardcoded skill-file destination paths; agent names now config-driven |
| Global config | User adds harness entries + profiles to `~/.../gmd/config.cue` |

## Key Design Decisions

### 1. Use `os/exec` for launching — not IPC, not SDKs

TASK.md specifies "let's just use exec". Each harness is a CLI binary. We build a
`*exec.Cmd`, wire stdin/stdout/stderr, and run it. No harness-specific SDKs or
long-running process management. This keeps the surface area minimal.

### 2. Providers + profiles pattern (matching LLM / web config)

LLM and web both use a two-level config: **named providers** (what/where) and **named
profiles** (how to use them, with pre-configured options). Agent harness config follows
the same pattern:

- `harnesses` — defines available harness binaries (name, bin resolved from PATH or absolute path, flag style, default env)
- `profiles` — named presets that select a harness + pre-configure message, flags, env
- `defaultHarness` — which harness to use when none is specified

```cue
// CUE schema (additions to types.cue)
AgentHarnessConfig: {
    bin:        string               // executable name (PATH lookup) or absolute path
    flagStyle?: string | *"double-dash"  // "--flag value" vs "-flag value"
    env?:       [string]: string     // default env vars
}

AgentHarnessProfile: {
    harness?:    string              // which harness (from harnesses map)
    message?:    string              // prepopulated message/prompt
    flags?:      [string]: string    // CLI flags to pass (key = value)
    args?:       [...string]         // positional args
    env?:        [string]: string    // extra env vars (merged with harness env)
    configFile?: string              // path to harness-specific config for this run
    cwd?:        string              // working directory (relative paths resolved vs ProjectRoot)
    tmux?:       bool                // launch inside named tmux session
    workspace?:  bool                // create git worktree before launching
    async?:      bool                // don't block; return after launching (for server/background use)
    // NOTE: workspaceBase, Name, and DryRun are CLI-only flags, not profile fields.
}

// Added to ProjectConfig:
ProjectConfig: {
    // ... existing fields ...
    agent?: AgentConfig
}

AgentConfig: {
    defaultHarness?: string
    harnesses?: [string]: AgentHarnessConfig
    profiles?:  [string]: AgentHarnessProfile
}
```

### 3. Binaries resolved from PATH, with absolute path support

Config can specify a bare name (e.g., `"opencode"`) resolved via `exec.LookPath`
from the user's PATH, or an absolute path (e.g., `"/usr/local/bin/opencode"`)
used directly. This handles cases where the harness is not on PATH but the user
knows its location. If the binary is not found, the launch fails with a clear error.

### 4. Harness is an interface, not just a struct

Even though we start with `exec`, different harnesses may need different argument
formatting (e.g., opencode uses `--message`, claude uses `-p`). The interface
allows harness-specific `BuildCommand` logic without callers caring about the details.

```go
// pkg/agent/agent.go
type Harness interface {
    Name() string
    BuildCommand(opts LaunchOptions) (*exec.Cmd, error)
}
```

Default implementation (`genericHarness`) handles `flagStyle` to switch between
`--flag value` and `-flag value` formatting. opencode gets its own implementation
for known flags like `--message`, `--config`, `--add-dir`, etc.

### 5. CLI design: `gmd agent <name>` is the default (not a subcommand)

`gmd agent <name>` directly launches the named profile (or harness). This avoids
requiring `gmd agent launch my-profile` — the common case is the quickest path.
Subcommands that don't match profile/harness names are treated as actions:
`list`, `show`, `session`.

```
gmd agent my-profile              # launch "my-profile"
gmd agent my-profile "fix bug"    # launch with message override
gmd agent dev                     # launch "dev" profile
gmd agent list                    # list harnesses + profiles
gmd agent show <name>             # show resolved config
gmd agent session list            # list active sessions + workspaces
gmd agent session kill <name>     # kill session + remove workspace
gmd agent session merge <name>    # merge workspace back into main branch
```

The resolution order for the first positional arg is:
1. If it matches a known subcommand (`list`, `show`, `session`), treat as subcommand
2. If it matches a profile name, launch that profile
3. If it matches a harness name, launch with default settings
4. Otherwise, error: unknown profile/harness

**Shortcut:** `gmd agent <name> "<message>"` sets the message. Works with both
profile lookup and harness fallback. This is equivalent to `--message "..."` but
shorter for the common case.

| Mode | Caller | Example |
|------|--------|---------|
| Direct CLI | User | `gmd agent my-profile` |
| Direct CLI (msg) | User | `gmd agent my-profile "fix the bug"` |
| Library call | Other commands | `wiki doctor` calls `agent.Launch(ctx, cfg, profileName)` auto after fixes |

The `pkg/agent/` package exposes a `Launch()` function that resolves config, builds
the command, and execs it. Both the CLI command and internal callers use the same
function.

### 6. Config resolution: profile → harness merge

When launching, the resolution order is:
1. Look up the named profile (or use `defaultHarness` if no profile given)
2. Get the harness definition from `harnesses[profile.harness]`
3. Merge: harness.env → profile.env (profile overrides)
4. CLI flags (--message, --flag, --config) override profile values

This matches how LLM config resolves: provider base config → profile role overrides.

### 7. Global config only for harness definitions, project may add profiles

TASK.md says "update my global config for the new content and a couple of profiles."
Harness definitions (binary paths) are machine-specific and belong in global config.
Profiles could be either global (personal presets) or project-local (team-shared).
The merge logic supports both: global first, project overlays.

### 8. Direct stdin/stdout passthrough — no capture

The launched agent should take over the terminal (stdin/stdout/stderr connected
directly) so the user interacts with it naturally. We are not capturing output
or managing the agent process; we hand off control.

### 9. Tmux session launch with two windows — tabs (shell + agent)

`--tmux` wraps the agent launch inside a named tmux session with two windows:

- **Window 0:** a plain shell in the working directory
- **Window 1 (focused):** runs the agent harness via a generated shell script

This uses two temp files to avoid shell escaping issues (messages may contain
backticks, special chars, etc.):
- **Temp file A** — the user's message/input, written to disk
- **Temp file B** — a shell script that reads file A into a variable and invokes
  the harness with it (e.g., `INPUT=$(cat /tmp/msg); opencode --message "$INPUT"`)

The tmux command is built as:
```
tmux new-session -s <name> -c <cwd> -n shell \; \
  new-window -c <cwd> -n agent /tmp/script.sh \; \
  select-window -t 1
```

The `<name>` arg (first positional) becomes the tmux session name. When `--tmux`
is set, gmd runs tmux as a child process (via `cmd.Run()`) and blocks until the
session ends. The user switches windows with `ctrl-b n`/`ctrl-b p`, detaches with
`ctrl-b d`, or exits both windows to return to gmd.

**Session name collision:** If a tmux session with `<name>` already exists,
`tmux new-session -s <name>` fails with an error. gmd surfaces this error directly.
The user must choose a different name or kill the existing session via
`gmd agent session kill <name>`. See ### 11 for session management commands.

### 10. Git worktree under `.workspaces/<name>`

`--workspace` creates an isolated git worktree before launching the agent:

```
git worktree add .workspaces/<name> <base-ref>
```

- `<name>` is the first positional arg
- `<base-ref>` defaults to the **currently checked-out branch** (not HEAD — the
  branch name is more descriptive and reproducible). Overridable via `--workspace-base <ref>`.
- The agent's CWD is set to `.workspaces/<name>`
- Workspaces live at the **project root** (alongside `.gmd/`)
- `.workspaces/` is **automatically added to `.gitignore`** on first use if not
  already present (see `ensureGitignore()` in workspace.go)

**Cleanup:** Manual by default. Use `git worktree remove .workspaces/<name>` or
`git worktree prune`. See ### 12 for workspace lifecycle commands.

When combined with `--tmux`, the tmux session CWD and the shell window both use
the workspace directory. When used without `--tmux`, the agent runs directly in
the workspace directory.

Note: `git worktree add` fails if the main worktree has uncommitted changes. The
user must stash or commit before using `--workspace`. This is surfaced as a clear
git error, not swallowed.

### 11. Session management commands

`gmd agent session` provides management for active tmux sessions and workspaces:

```
gmd agent session list              # list active tmux sessions + orphaned workspaces
gmd agent session kill <name>       # kill tmux session + remove workspace
gmd agent session merge <name>      # merge workspace into main branch (see ### 12)
```

Session listing introspects tmux (`tmux list-sessions`) filtered to sessions
matching gmd naming conventions. Kill sends `tmux kill-session -t <name>` and
automatically removes the associated workspace via `git worktree remove --force`.
If there is no workspace for the session, only the tmux session is killed.

Sessions are not auto-attached on collision (see ### 9). The user must explicitly
kill the old session or choose a different name.

### 12. Workspace lifecycle management

Workspace management is exposed via `gmd agent session`:

```
gmd agent session list              # shows both tmux sessions and orphaned workspaces
gmd agent session kill <name>       # kills tmux session + removes workspace automatically
gmd agent session merge <name>      # merge workspace back into main branch
```

Workspaces can become orphaned if a tmux session is killed externally or if
`--workspace` was used without `--tmux`. The `session list` output notes orphaned
workspaces. Cleanup uses `git worktree remove --force`.

**Merge command:** `gmd agent session merge <name>` merges the workspace's
current branch back into the main worktree's current branch:

```
cd <projectRoot> && git merge .workspaces/<name>/<branch> [--squash]
```

- By default, performs a normal merge commit.
- `--squash` flag squashes all workspace commits into a single change on the
  main branch (user commits manually after review).
- If the merge fails (conflicts), the command exits with an error and git's
  conflict markers are left in the working tree for the user to resolve.
- The workspace must exist under `.workspaces/<name>`.
- After a successful merge, the workspace is NOT automatically removed — the
  user runs `gmd agent session kill <name>` to clean up when done.

Flow:
1. `cd <projectRoot>`
2. `git merge .workspaces/<name>/<ref> [--squash]` (ref = the branch name used at worktree creation)
3. Report result

### 13. Workspace rollback on tmux failure

When `--workspace` and `--tmux` are combined, the worktree is created first, then
the tmux session is launched. If tmux fails (e.g., tmux not on PATH, session name
collision), the worktree is automatically rolled back:

```
defer func() { if err != nil { git worktree remove .workspaces/<name> } }()
```

The rollback uses `git worktree remove --force` to clean up even if the worktree
has modifications (it was just created, so there are none). This prevents orphaned
workspaces from failed combined launches.

### 14. Wiki doctor refactored to use config-driven agent list

Currently `pkg/wiki/doctor.go` hardcodes agent names in two places:
- `Doctor()`: `agentNames := []string{"claude", "codex", "opencode"}`
- `DoctorFix()`: same hardcoded list

This is refactored to enumerate agents from `AgentConfig.Harnesses`. The doctor
accepts the `AgentConfig` and iterates over configured harness names instead of
hardcoded strings.

`pkg/wiki/skills.go` keeps its hardcoded agent discovery paths (these are gmd's
own skill template destinations and don't change per-user). The skill installation
keys off the harness name, and the hardcoded paths map covers the known set
(claude, codex, opencode). Unknown harnesses get the "generic" skill template at
a sensible default path.

**DoctorResult changes:**

```go
type DoctorResult struct {
    WikiName     string
    PageCount    int
    SourceCount  int
    TSConnected  bool
    LLMStatus    []llm.EndpointStatus
    Agents       []AgentStatus
    Errors       []string
    FixesApplied []string          // NEW: what DoctorFix() changed
}
```

### 15. `--dry-run` flag

`gmd agent <name> --dry-run` prints the resolved command and environment without
executing. Output shows:
- Resolved binary path
- All args (message, flags, config, extra args)
- Environment variables (merged from harness, profile, CLI)
- CWD (including workspace path if `--workspace`)
- Whether tmux would be used

Useful for debugging config resolution before launching.

### 16. `--async` flag

`gmd agent <name> --async` launches the agent without blocking. gmd returns
immediately after starting the process. stdin/stdout/stderr are NOT connected
(the agent runs detached).

This is the opposite of the default behavior (blocking, stdin/stdout/stderr
connected). Intended for:
- `gmd wiki doctor --fix` (launches agent in background after fixes)
- Server-mode or scripted use where the caller doesn't need to interact
- Chained workflows where multiple agents are launched in parallel

When `--async` is combined with `--tmux`, the tmux session is created and gmd
detaches immediately (equivalent to `tmux new-session -d ...`). The user can
attach later with `tmux attach-session -t <name>` or `gmd agent session list`.

## Architecture

```
┌─────────────────────────────────────────────────┐
│ cmd/gmd/                                        │
│  agent.go         gmd agent <name> [message]    │
│                   (default = launch)             │
│  agent_list.go    gmd agent list                │
│  agent_show.go    gmd agent show <name>         │
│  agent_session.go gmd agent session [list|kill|merge] │
│                                                 │
│  wiki_doctor.go   auto-launches agent after fix │
│                   --async skips attach           │
│  web_agent.go     auto-launch after research     │
│                   --async skips attach (future)   │
└───────────────┬─────────────────────────────────┘
                │ calls
┌───────────────▼─────────────────────────────────┐
│ pkg/agent/                                      │
│  agent.go       Harness interface, Launch(),    │
│                 LaunchOptions struct             │
│  config.go      ResolveAgentConfig(cfg),        │
│                 resolveHarness(), resolveProfile()│
│  opencode.go    opencodeHarness implementation  │
│  generic.go     genericHarness (fallback)        │
│  tmux.go        buildTmuxCmd()                  │
│  workspace.go   setupWorkspace(), rollback,     │
│                 ensureGitignore(), branch detect │
│  session.go     SessionManager (list/kill)      │
└───────────────┬─────────────────────────────────┘
                │ reads
┌───────────────▼─────────────────────────────────┐
│ pkg/config/                                      │
│  config.go      +AgentConfig field on Config    │
│  embeds/types.cue  +AgentConfig schema          │
│                 +AgentHarnessConfig             │
│                 +AgentHarnessProfile            │
└─────────────────────────────────────────────────┘
```

## Config Integration

### Go Config struct addition (`pkg/config/config.go`)

Add to the `Config` struct alongside existing optional sections (matching `WebConfig` pattern):

```go
type Config struct {
    LLM            LLMConfig                       `json:"llm,omitempty"`
    Typesense      TypesenseConfig                 `json:"typesense,omitempty"`
    Web            WebConfig                       `json:"web,omitempty"`
    Pipeline       PipelineConfig                  `json:"pipeline,omitempty"`
    Agent          AgentConfig                     `json:"agent,omitempty"`      // NEW
    Collections    map[string]CollectionConfig     `json:"collections,omitempty"`
    Wikis          map[string]WikiConfig           `json:"wikis,omitempty"`
    SearchDefaults map[string][]string             `json:"searchDefaults,omitempty"`
    ProjectRoot    string                          `json:"-"`
    Project        string                          `json:"-"`
}

type AgentConfig struct {
    DefaultHarness string                         `json:"defaultHarness,omitempty"`
    Harnesses      map[string]AgentHarnessConfig   `json:"harnesses,omitempty"`
    Profiles       map[string]AgentHarnessProfile  `json:"profiles,omitempty"`
}

type AgentHarnessConfig struct {
    Name      string            `json:"-"`           // key from map
    Bin       string            `json:"bin"`
    FlagStyle string            `json:"flagStyle,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
}

type AgentHarnessProfile struct {
    Name       string            `json:"-"`           // key from map
    Harness    string            `json:"harness,omitempty"`
    Message    string            `json:"message,omitempty"`
    Flags      map[string]string `json:"flags,omitempty"`
    Args       []string          `json:"args,omitempty"`
    Env        map[string]string `json:"env,omitempty"`
    ConfigFile string            `json:"configFile,omitempty"`
    Cwd        string            `json:"cwd,omitempty"`
    Tmux       bool              `json:"tmux,omitempty"`
    Workspace  bool              `json:"workspace,omitempty"`
    Async      bool              `json:"async,omitempty"`
}
```

### mergeConfigs logic (`pkg/config/config.go`)

```go
// In mergeConfigs(), added alongside existing section merges:
if src.Agent.DefaultHarness != "" {
    dst.Agent.DefaultHarness = src.Agent.DefaultHarness
}
for k, v := range src.Agent.Harnesses {
    if dst.Agent.Harnesses == nil {
        dst.Agent.Harnesses = make(map[string]AgentHarnessConfig)
    }
    hc := v
    hc.Name = k
    dst.Agent.Harnesses[k] = hc
}
for k, v := range src.Agent.Profiles {
    if dst.Agent.Profiles == nil {
        dst.Agent.Profiles = make(map[string]AgentHarnessProfile)
    }
    p := v
    p.Name = k
    dst.Agent.Profiles[k] = p
}
```

### Updated CUE ProjectConfig (`pkg/config/embeds/types.cue`)

Add `agent?` field to the existing `ProjectConfig` definition at `types.cue:227`:

```cue
ProjectConfig: {
    project?:        string
    llm?:            LLMConfig
    typesense?:      TypesenseConfig
    web?:            WebConfig
    pipeline?:       PipelineConfig
    agent?:          AgentConfig              // NEW
    collections:     [string]: CollectionConfig
    wikis:           [string]: WikiConfig
    searchDefaults?: [string]: [...string]
}
```

## Code Details

### pkg/agent/agent.go

```go
package agent

import (
    "context"
    "fmt"
    "os"
    "os/exec"
)

// LaunchOptions are the fully-resolved options for launching a harness.
// Built by merging profile defaults with CLI overrides.
type LaunchOptions struct {
    Name          string            // session/workspace name (first positional arg; must pass validateName when Tmux or Workspace is true)
    HarnessName   string            // which harness to use
    Message       string            // prepopulated message/prompt
    Flags         map[string]string // CLI flags to pass
    Args          []string          // positional args
    Env           map[string]string // extra env vars
    ConfigFile    string            // path to harness config
    Cwd           string            // working directory
    Tmux          bool              // launch inside named tmux session
    Workspace     bool              // create git worktree before launching
    WorkspaceBase string            // git ref for worktree (defaults to current branch)
    Async         bool              // don't block; return after launching
    DryRun        bool              // print resolved command without executing
}

// Harness builds an *exec.Cmd for a specific agent CLI.
type Harness interface {
    Name() string
    BuildCommand(opts LaunchOptions) (*exec.Cmd, error)
}

// Launch resolves config and runs the harness. By default it connects
// stdin/stdout/stderr and blocks until the harness exits. Use opts.Async
// to launch without blocking (no stdin/stdout/stderr attachment).
//
// Launch flow:
//  1. Resolve profile → harness, merge options
//  2. Validate name if tmux/workspace requested
//  3. If DryRun: print resolved command and return
//  4. If Workspace: setupWorkspace() → adjust CWD to .workspaces/<name>
//      (with rollback if subsequent steps fail, see ### 13)
//  5. If Tmux:     buildTmuxCmd(binPath, opts) → cmd.Start() or cmd.Run()
//  6. Else:        harness.BuildCommand(opts) → cmd.Start() or cmd.Run()
//  7. If !Async:   cmd.Stdin = os.Stdin; cmd.Stdout = os.Stdout; cmd.Stderr = os.Stderr
//                  cmd.Wait() — blocks, returns to gmd on exit
//      If Async:   cmd.Start() → return immediately (no pipes connected)
func Launch(ctx context.Context, cfg *config.Config, profileName string, overrides LaunchOptions) error {
    // 1. Resolve profile → harness, merge options
    // 2. Validate name if tmux/workspace requested
    // 3. If DryRun: print, return
    // 4. If Workspace: setup workspace, set CWD, defer rollback on error
    // 5. Build *exec.Cmd (harness or tmux wrapper)
    // 6. If Async: cmd.Start(); return
    // 7. Else: cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr; cmd.Run()
}
```

### pkg/agent/config.go

```go
// ErrNoAgentConfig is returned when no agent config is present.
var ErrNoAgentConfig = errors.New("no agent config: add an 'agent:' section to gmd config")

// ResolveAgentConfig resolves a profile name to a harness and merged options.
// Returns ErrNoAgentConfig if the agent section is absent from config.
func ResolveAgentConfig(cfg *config.Config, profileName string) (Harness, *LaunchOptions, error)

// ResolveHarness returns a harness by name, falling back to defaultHarness.
func ResolveHarness(cfg *config.Config, name string) (Harness, error)

// ListHarnesses returns all configured harness names.
func ListHarnesses(cfg *config.Config) []string

// ListProfiles returns all configured profile names.
func ListProfiles(cfg *config.Config) []string
```

### pkg/agent/opencode.go

```go
type opencodeHarness struct {
    bin       string
    flagStyle string
    env       map[string]string
}

func (h *opencodeHarness) Name() string { return "opencode" }

func (h *opencodeHarness) BuildCommand(opts LaunchOptions) (*exec.Cmd, error) {
    binPath, err := resolveBin(h.bin) // LookPath for bare names, direct use for absolute paths
    if err != nil {
        return nil, fmt.Errorf("harness 'opencode': %w", err)
    }
    args := []string{}
    if opts.Message != "" {
        args = append(args, "--message", opts.Message)
    }
    if opts.ConfigFile != "" {
        args = append(args, "--config", opts.ConfigFile)
    }
    for k, v := range opts.Flags {
        args = append(args, fmt.Sprintf("--%s", k), v)
    }
    args = append(args, opts.Args...)
    cmd := exec.Command(binPath, args...)
    if opts.Cwd != "" {
        cmd.Dir = opts.Cwd
    }
    // Merge with os.Environ() as base so PATH, HOME, etc. are inherited.
    cmd.Env = os.Environ()
    for k, v := range h.env {
        cmd.Env = append(cmd.Env, k+"="+v)
    }
    for k, v := range opts.Env {
        cmd.Env = append(cmd.Env, k+"="+v)
    }
    return cmd, nil
}
```

### pkg/agent/generic.go

```go
type genericHarness struct {
    name      string
    bin       string
    flagStyle string // "double-dash" or "single-dash"
    env       map[string]string
}

func (h *genericHarness) BuildCommand(opts LaunchOptions) (*exec.Cmd, error) {
    binPath, err := resolveBin(h.bin)
    if err != nil {
        return nil, fmt.Errorf("harness %q: %w", h.name, err)
    }
    prefix := "--"
    if h.flagStyle == "single-dash" {
        prefix = "-"
    }
    args := []string{}
    if opts.Message != "" {
        args = append(args, prefix+"message", opts.Message)
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
```

### pkg/agent/tmux.go

```go
// buildTmuxCmd creates an *exec.Cmd that launches a named tmux session with
// two windows (tabs): window 0 = shell, window 1 = agent harness (focused).
//
// Uses two temp files to avoid shell escaping issues:
//   - Temp file A: the user's message/input content
//   - Temp file B: a shell script that reads file A and invokes the harness
//
// buildTmuxCmd does NOT call harness.BuildCommand. Instead, it constructs
// the harness invocation directly from the bin path and LaunchOptions args,
// so all flags (--config, --flag overrides, etc.) are preserved.
func buildTmuxCmd(binPath string, opts LaunchOptions) (*exec.Cmd, error) {
    sessionName := opts.Name
    cwd := opts.Cwd
    if cwd == "" {
        cwd = "."
    }

    // 1. Write message to temp file A
    msgFile, err := os.CreateTemp("", "gmd-agent-msg-*.txt")
    if err != nil {
        return nil, fmt.Errorf("tmux: failed to create message temp file: %w", err)
    }
    if _, err := msgFile.WriteString(opts.Message); err != nil {
        msgFile.Close()
        return nil, err
    }
    msgFile.Close()

    // 2. Build harness invocation args (all non-message flags from opts)
    var harnessArgs []string
    if opts.ConfigFile != "" {
        harnessArgs = append(harnessArgs, "--config", opts.ConfigFile)
    }
    for k, v := range opts.Flags {
        harnessArgs = append(harnessArgs, "--"+k, v)
    }
    harnessArgs = append(harnessArgs, opts.Args...)

    // 3. Write shell script to temp file B
    //    Uses a trap to self-clean both temp files on exit.
    //    Reads message from file A; quotes all paths with %q.
    scriptFile, err := os.CreateTemp("", "gmd-agent-script-*.sh")
    if err != nil {
        return nil, fmt.Errorf("tmux: failed to create script temp file: %w", err)
    }
    script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
cleanup() { rm -f %[1]q "$0"; }
trap cleanup EXIT
GMD_AGENT_INPUT=$(cat %[1]q)
exec %[2]q --message "$GMD_AGENT_INPUT" %[3]s
`, msgFile.Name(), binPath, shellQuote(harnessArgs))
    if _, err := scriptFile.WriteString(script); err != nil {
        scriptFile.Close()
        return nil, err
    }
    scriptFile.Close()
    if err := os.Chmod(scriptFile.Name(), 0755); err != nil {
        return nil, err
    }

    // 4. Build tmux command with two windows.
    //    Window 0: shell in CWD (named "shell")
    //    Window 1: runs the script as initial command (named "agent", focused)
    tmuxPath, err := exec.LookPath("tmux")
    if err != nil {
        return nil, fmt.Errorf("tmux: tmux not found on PATH (required for --tmux)")
    }
    return exec.Command(tmuxPath,
        "new-session", "-s", sessionName, "-c", cwd, "-n", "shell",
        ";", "new-window", "-c", cwd, "-n", "agent", scriptFile.Name(),
        ";", "select-window", "-t", "1",
    ), nil
}

// shellQuote joins args with proper shell quoting for use in a bash script.
// Each arg is quoted with %q (Go's safe shell escaping).
func shellQuote(args []string) string {
    var quoted []string
    for _, a := range args {
        quoted = append(quoted, fmt.Sprintf("%q", a))
    }
    return strings.Join(quoted, " ")
}
```

### pkg/agent/workspace.go

```go
// validateName rejects names that could escape .workspaces/ or be
// interpreted as flags by git.
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

// getCurrentBranch returns the currently checked-out branch name.
// Falls back to "HEAD" if detection fails (detached HEAD, etc.).
func getCurrentBranch(projectRoot string) string {
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    cmd.Dir = projectRoot
    out, err := cmd.Output()
    if err != nil {
        return "HEAD"
    }
    branch := strings.TrimSpace(string(out))
    if branch == "HEAD" {
        return "HEAD" // detached HEAD
    }
    return branch
}

// ensureGitignore adds ".workspaces/" to the project's .gitignore if not present.
// Creates the file if it doesn't exist. Idempotent.
func ensureGitignore(projectRoot string) error {
    giPath := filepath.Join(projectRoot, ".gitignore")
    entry := ".workspaces/"
    
    data, err := os.ReadFile(giPath)
    if err != nil && !os.IsNotExist(err) {
        return err
    }
    if strings.Contains(string(data), entry) {
        return nil // already present
    }
    
    f, err := os.OpenFile(giPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()
    _, err = fmt.Fprintln(f, entry)
    return err
}

// setupWorkspace creates a git worktree under .workspaces/<name> and returns
// the path to use as CWD for the agent launch.
//
// Default base ref is the current branch (via getCurrentBranch), not HEAD.
// The .workspaces/ directory is created at the project root if it doesn't exist.
// Automatically adds .workspaces/ to .gitignore via ensureGitignore.
// Does not pre-check for existence — lets git worktree add report the error.
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

    // Auto-add .workspaces/ to .gitignore (idempotent)
    if err := ensureGitignore(projectRoot); err != nil {
        // Non-fatal: workspace still works, just warn.
        fmt.Fprintf(os.Stderr, "warning: could not update .gitignore: %v\n", err)
    }

    targetPath := filepath.Join(wsDir, name)

    gitPath, err := exec.LookPath("git")
    if err != nil {
        return "", fmt.Errorf("workspace: git not found on PATH (required for --workspace)")
    }

    cmd := exec.Command(gitPath, "worktree", "add", targetPath, baseRef)
    cmd.Dir = projectRoot
    cmd.Stdout = os.Stderr // forward git output to user
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("workspace: git worktree add failed: %w", err)
    }

    fmt.Fprintf(os.Stderr, "Created workspace: %s (from %s)\n", targetPath, baseRef)
    return targetPath, nil
}

// removeWorkspace removes a git worktree. Used for rollback on tmux failure.
func removeWorkspace(projectRoot, name string) error {
    targetPath := filepath.Join(projectRoot, ".workspaces", name)
    cmd := exec.Command("git", "worktree", "remove", "--force", targetPath)
    cmd.Dir = projectRoot
    cmd.Stdout = os.Stderr
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

### CLI: cmd/gmd/agent.go

```go
// gmd agent <name> [message] [flags]  — default action is launch
// gmd agent list                       — list harnesses + profiles
// gmd agent show <name>                — show resolved config
// gmd agent session list|kill|merge <name> — session management
var agentCmd = &cobra.Command{
    Use:   "agent [name] [message] [flags]",
    Short: "Launch external AI agent harnesses",
    Long: `Launch an external AI agent harness.

The default action is to launch. The first argument is interpreted as:
  1. A subcommand (list, show, session)
  2. A profile name
  3. A harness name (fallback with default settings)

Examples:
  gmd agent my-profile                    # launch "my-profile"
  gmd agent my-profile "fix the bug"      # launch with message override
  gmd agent dev --tmux --workspace        # launch "dev" in tmux + workspace
  gmd agent list                          # list configured harnesses/profiles
  gmd agent show wiki                     # show resolved config for "wiki"
  gmd agent session list                  # list active sessions
  gmd agent session kill my-session       # kill session + workspace
  gmd agent session merge my-session      # merge workspace into main branch`,
    Args: cobra.ArbitraryArgs,
    RunE: runAgent,
}

var agentListCmd = &cobra.Command{
    Use:   "list",
    Short: "List configured agent harnesses and profiles",
    RunE:  runAgentList,
}

var agentShowCmd = &cobra.Command{
    Use:   "show <profile|harness>",
    Short: "Show resolved configuration for a profile or harness",
    RunE:  runAgentShow,
}
```

### CLI flag → LaunchOptions translation (`agent.go`)

```go
func runAgent(cmd *cobra.Command, args []string) error {
    cfg := getConfig()

    // If first arg matches a subcommand, delegate. This is handled by
    // cobra subcommand routing normally, but since we use ArbitraryArgs
    // on the parent, we handle the "default is launch" case here.
    // Subcommands are registered as children and cobra routes them first.
    // We reach here only when the first arg is NOT a known subcommand.

    var sessionName string
    var message string
    if len(args) > 0 {
        sessionName = args[0]   // <name> — profile/harness, or tmux session/workspace dir
    }
    if len(args) > 1 {
        message = args[1]       // "<message>" shortcut
    }

    // Validate: <name> is required when --tmux or --workspace is set.
    if (tmuxFlag || workspaceFlag) && sessionName == "" {
        return fmt.Errorf("<name> is required when using --tmux or --workspace")
    }

    // --message flag overrides positional message
    if msgFlag != "" {
        message = msgFlag
    }

    overrides := agent.LaunchOptions{
        Name:          sessionName,
        Message:       message,           // positional or --message / -m
        ConfigFile:    configFlag,        // --config
        Cwd:           cwdFlag,           // --cwd
        Tmux:          tmuxFlag,          // --tmux
        Workspace:     workspaceFlag,     // --workspace
        WorkspaceBase: workspaceBaseFlag, // --workspace-base
        Async:         asyncFlag,         // --async
        DryRun:        dryRunFlag,        // --dry-run
        Flags:         parseFlagSlice(flagFlags),  // --flag key=value (repeatable)
        Env:           parseFlagSlice(envFlags),   // --env key=value (repeatable)
        Args:          extraArgs,         // -- trailing args
    }

    // If --profile is set, use it. Otherwise try <name> as a profile name.
    profileName := profileFlag
    if profileName == "" && sessionName != "" {
        profileName = sessionName
    }

    return agent.Launch(cmd.Context(), cfg, profileName, overrides)
}
```

Resolution priority: **CLI overrides > profile > harness defaults**. An empty field at
a higher priority level falls through to the next level (not reset to zero).

```go
// gmd agent list
var agentListCmd = &cobra.Command{
    Use:   "list",
    Short: "List configured agent harnesses and profiles",
    RunE:  runAgentList,
}

// gmd agent show <name>
var agentShowCmd = &cobra.Command{
    Use:   "show <profile|harness>",
    Short: "Show resolved configuration for a profile or harness",
    RunE:  runAgentShow,
}
```

### Integration: gmd wiki doctor auto-launches agent after fixes

When `gmd wiki doctor <name> --fix` applies fixes, the doctor automatically
launches the agent harness so the user can immediately work on the wiki.
No separate `--launch-agent` flag is needed — launching is the default
post-fix behavior.

Use `--async` to skip the blocking launch (gmd returns immediately after
starting the agent). This is useful for scripted or CI use.

**DoctorResult changes** (`pkg/wiki/doctor.go`):

```go
type DoctorResult struct {
    WikiName     string
    PageCount    int
    SourceCount  int
    TSConnected  bool
    LLMStatus    []llm.EndpointStatus
    Agents       []AgentStatus
    Errors       []string
    FixesApplied []string          // NEW: what DoctorFix() changed
}
```

**Integration flow in `runWikiDoctor()`:**

```go
if fixFlag && len(result.FixesApplied) > 0 {
    // Print summary of fixes applied.
    for _, fix := range result.FixesApplied {
        fmt.Printf("  Fixed: %s\n", fix)
    }

    // Try "wiki" profile first, fall back to defaultHarness.
    profileName := "wiki"
    if _, _, err := agent.ResolveAgentConfig(cfg, profileName); err != nil {
        profileName = ""
    }
    opts := agent.LaunchOptions{
        Name:    wikiName,
        Message: fmt.Sprintf("Work on the wiki '%s'. Run /help for tools.", wikiName),
        Async:   asyncFlag,  // --async flag (defaults to false = blocking)
    }
    return agent.Launch(ctx, cfg, profileName, opts)
}
```

If no agent config exists, the launch step prints a hint ("add an 'agent:'
section to gmd config to auto-launch after doctor fixes") and returns
normally. This is not an error — the doctor fixes were still applied.

## Config Example

User's global `~/Library/Application Support/gmd/config.cue` (macOS):

```cue
agent: {
    defaultHarness: "opencode"

    harnesses: {
        opencode: {
            bin: "opencode"
        }
        claude: {
            bin: "claude"
            flagStyle: "single-dash"
        }
    }

    profiles: {
        wiki: {
            harness: "opencode"
            message: "I'm working on a gmd wiki. Run /help for tools."
            flags: {
                "add-dir": "/Users/tony/verdverm/gmd"
            }
        }
        general: {
            harness: "opencode"
        }
        dev: {
            harness:   "opencode"
            tmux:      true
            workspace: true
            cwd:       "./"
        }
        background: {
            harness: "opencode"
            async:   true
            message: "Run the test suite and report back."
        }
    }
}
```

## Relationship with pkg/wiki/skills.go and pkg/wiki/doctor.go

`skills.go` and `doctor.go` currently hardcode agent names (`claude`, `codex`,
`opencode`) and skill paths. With the answers to Q8, the updated plan is:

- **`pkg/wiki/skills.go`** — manages *gmd skill file installation* for agent
  harnesses. Hardcoded agent discovery paths are kept here because the skill
  files are gmd's own templates and the destination paths are well-known
  conventions (e.g., `~/.claude/skills/`). Unknown harnesses get the
  "generic" skill template at a sensible default path.
- **`pkg/wiki/doctor.go`** — refactored to enumerate agents from
  `AgentConfig.Harnesses` instead of the hardcoded list. `Doctor()` accepts
  the `AgentConfig` and iterates over configured harness names. `DoctorFix()`
  does the same for skill installation.
- **`pkg/agent/`** — manages *launching* agent harnesses. Config-driven, not
  hardcoded.

**Decision: refactor now (not later).** The agent enumeration in doctor.go
must be config-driven from the start. skills.go keeps its hardcoded paths
for skill installation targets.

## Zero-Config Behavior

When no `agent:` section exists in config:

| Command | Behavior |
|---------|----------|
| `gmd agent list` | Prints "no agent harnesses configured" (not an error) |
| `gmd agent show` | Prints "no agent config found" |
| `gmd agent <name>` | Error: no agent config. The user must configure at least one harness. No hardcoded fallback list. |
| Library `agent.Launch(ctx, cfg, ...)` | Returns `ErrNoAgentConfig`; caller decides whether to handle gracefully |
| `gmd wiki doctor --fix` | Applies fixes, prints hint that agent config is needed for auto-launch, returns normally |

## gmd serve Interaction

`agent.Launch()` is designed for CLI use only — it blocks and takes over the
terminal. It is fundamentally incompatible with `gmd serve` (long-lived HTTP
server). The design documents this as a CLI-only feature. If server-mode agent
launch is needed later, a non-blocking `agent.Start()` variant could be added
that returns the `*exec.Cmd` without waiting.

## Implementation Plan

### Phase 1: Core package + config
1. Add `AgentConfig`, `AgentHarnessConfig`, `AgentHarnessProfile` to `pkg/config/config.go`
2. Add CUE schema to `pkg/config/embeds/types.cue`
3. Add merge logic in `mergeConfigs()`
4. Create `pkg/agent/` package:
   - `agent.go` — Harness interface, LaunchOptions, Launch() (with DryRun, Async support)
   - `config.go` — ResolveAgentConfig, ListHarnesses, ListProfiles
   - `opencode.go` — opencodeHarness
   - `generic.go` — genericHarness
   - `tmux.go` — buildTmuxCmd, temp file management
   - `workspace.go` — setupWorkspace, getCurrentBranch, ensureGitignore, removeWorkspace (rollback)
   - `session.go` — SessionManager (list/kill tmux sessions, find orphaned workspaces)
5. Unit tests for config resolution, BuildCommand output, merge behavior

### Phase 2: CLI commands
1. `cmd/gmd/agent.go` — parent command (default = launch, subcommands: list, show, session)
2. `cmd/gmd/agent_session.go` — session management (list, kill)
3. Register agent commands in `cmd/gmd/main.go` init()
4. CLI flags: `--tmux`, `--workspace`, `--workspace-base`, `--async`, `--dry-run`,
   `--profile`, `--message`/`-m`, `--config`, `--cwd`, `--flag`, `--env`
5. Trailing args support: `gmd agent <name> -- extra args for harness`
6. Integration tests (with `//go:build integration`)

### Phase 3: Doctor refactoring + internal integration
1. Refactor `pkg/wiki/doctor.go`: accept `AgentConfig`, enumerate agents from config
2. Refactor `cmd/gmd/wiki_doctor.go`: pass agent config, auto-launch after fixes, `--async` flag
3. Update `DoctorResult` with `FixesApplied` field
4. Wire agent launch into `cmd/gmd/web_agent.go` (future: auto-launch after research, --async to skip)

### Phase 4: Expansion
1. Add `claudeHarness` implementation (different flag style)
2. Add `codexHarness` implementation
3. Consider `gmd agent` subcommand aliases for common workflows

## Tests

| Scope | Type | What |
|-------|------|------|
| `pkg/agent/` | Unit | Config resolution (harness not found, profile not found, merge order) |
| `pkg/agent/` | Unit | BuildCommand output for opencode and generic harnesses |
| `pkg/agent/` | Unit | Env merging (os.Environ as base, harness + profile + overrides) |
| `pkg/agent/` | Unit | Binary validation (LookPath success + failure) |
| `pkg/agent/` | Unit | buildTmuxCmd output (args, session name, window layout) |
| `pkg/agent/` | Unit | buildTmuxCmd temp file creation, script content, cleanup trap |
| `pkg/agent/` | Unit | buildTmuxCmd error when tmux not on PATH |
| `pkg/agent/` | Unit | validateName (empty, path separators, leading dash, valid) |
| `pkg/agent/` | Unit | getCurrentBranch (normal branch, detached HEAD) |
| `pkg/agent/` | Unit | ensureGitignore (create, append, idempotent, missing file) |
| `pkg/agent/` | Unit | setupWorkspace path construction, branch detection, error cases |
| `pkg/agent/` | Unit | removeWorkspace (rollback) |
| `pkg/agent/` | Unit | Launch flow: default (blocking), --async, --dry-run |
| `pkg/agent/` | Unit | Launch flow: workspace-only, tmux-only, workspace+tmux combined |
| `pkg/agent/` | Unit | Launch flow: name validation enforced when tmux/workspace set |
| `pkg/agent/` | Unit | Launch flow: workspace rollback when combined tmux fails |
| `pkg/agent/` | Unit | Session list/kill (tmux introspection, orphan detection) |
| `pkg/config/` | Unit | CUE parsing of new schema, mergeConfigs for agent section |
| `pkg/wiki/` | Unit | Doctor enumerates agents from config (not hardcoded list) |
| `pkg/wiki/` | Unit | DoctorFix uses config-driven agent list |
| `cmd/gmd/` | Integration | `gmd agent my-profile` launch with test config |
| `cmd/gmd/` | Integration | `gmd agent my-profile "msg"` positional message |
| `cmd/gmd/` | Integration | `gmd agent list`, `gmd agent show` with test config |
| `cmd/gmd/` | Integration | `gmd agent session list` (requires tmux) |
| `cmd/gmd/` | Integration | `gmd agent session kill` removes tmux + workspace |
| `cmd/gmd/` | Integration | `gmd agent session merge` (normal + --squash) |
| `cmd/gmd/` | Integration | `gmd agent --dry-run` output verification |
| `cmd/gmd/` | Integration | `gmd wiki doctor --fix` auto-launches agent |
| `cmd/gmd/` | Integration | `gmd wiki doctor --fix --async` returns immediately |
| `cmd/gmd/` | Integration | `gmd wiki doctor --fix` with no agent config (graceful hint) |

## Risks & Dependencies

| Risk | Mitigation |
|------|------------|
| `exec.Command` with passthrough stdin blocks forever if harness hangs | OS process management handles this; user can Ctrl+C to kill both gmd and the harness |
| User config has stale binary paths after harness upgrade | `exec.LookPath` validation catches missing binaries at launch time with clear errors |
| Wiki doctor auto-launch surprises user by dropping into agent | Print summary of fixes before launching; user can use `--async` to skip |
| Harness flag conventions change over time (e.g., opencode renames `--message` to `-m`) | Harness-specific `BuildCommand` can be updated; `genericHarness` uses raw flag map |
| CUE schema additions break existing user configs | New fields are all `?` (optional), backward compatible |
| `pkg/agent/` has no Typesense dependency, but `getRuntime()` in CLI would create TS connection | `agent` commands use `getConfig()` directly, avoiding TS dependency |
| tmux not installed on user's machine | Pre-flight check with `exec.LookPath("tmux")` returns clear error; `--tmux` is opt-in |
| Temp files from tmux launch accumulate in /tmp | Script includes `trap cleanup EXIT` that removes both temp files on session end |
| `git worktree` fails (dirty working tree, no git repo, etc.) | Error surfaced immediately before agent launch; user can resolve and retry |
| Workspace directory naming collisions | Let `git worktree add` report the error directly (more descriptive than our own check) |
| Shell escaping in tmux script | Message goes through a temp file, not command-line args; script reads it as a variable; all paths use `%q` Go quoting; handles backticks and special chars |
| tmux session already exists with same name | Fail with clear error; user can `gmd agent session kill <name>` or choose different name |
| `<name>` contains path separators (escape attempt) | `validateName()` rejects names containing `/` or `\` before any file operations |
| `--workspace` succeeds but `--tmux` fails (tmux not found, session collision) | Worktree is automatically rolled back via `removeWorkspace()` in Launch() defer |
| `--workspace` without git repo | `exec.LookPath("git")` succeeds but `git worktree` fails with clear error from git |
| `.gitignore` modification surprises user | `ensureGitignore()` only appends `.workspaces/` entry; idempotent; prints warning on failure |
| Doctor refactoring breaks existing wiki doctor output | DoctorResult keeps same fields; Agents list comes from config instead of hardcoded list |
| `--async` + `--workspace` without `--tmux` leaves orphaned workspace | `gmd agent session list` shows orphaned workspaces; user cleans up with `session kill` |

## Resolved Questions (from TASK.md, 2026-06-06)

1. **Should `gmd agent launch` be the default subcommand?**
   **Yes.** `gmd agent <name>` directly launches. See ### 5 for CLI design.

2. **Should we support `gmd agent <message>` as a shortcut?**
   **Yes.** `gmd agent <name> "<message>"` works. Combined with Q1:
   first positional arg is name, second (optional) is message.

3. **Should harness binaries be resolved from PATH or require absolute paths?**
   **Both.** Bare name resolves via `exec.LookPath` from PATH; absolute path used
   directly. Config supports either. See ### 3.

4. **Should `gmd wiki doctor --launch-agent` block until agent exits?**
   **Yes, it's the default.** No `--launch-agent` flag. Doctor launches agent
   after fixes by default. `--async` flag skips attach. See ### 16 and the
   wiki doctor integration section.

5. **Do we need to persist agent sessions or logs?**
   **No.** Confirmed. gmd is just the launcher. The agent harness owns its history.

6. **Should profiles support `cwd` relative to project root?**
   **Yes.** If `cwd` starts with `./`, resolve relative to `cfg.ProjectRoot`.

7. **Should `gmd agent launch` support a `--dry-run` flag?**
   **Yes.** Print resolved command and env without executing. See ### 15.

8. **Should `gmd wiki doctor` refactor to enumerate agents from `AgentConfig.Harnesses`?**
   **Yes, refactor now.** No more hardcoded agent list in doctor.go. Config-driven
   from the start. skills.go keeps hardcoded destination paths. See ### 14.

9. **Should tmux sessions auto-attach if the session already exists?**
   **No, it should fail.** The user must choose a different name or kill the
   existing session via `gmd agent session kill`. See ### 9 and ### 11.

10. **Should `.workspaces/` be in `.gitignore` automatically?**
    **Yes.** Auto-add on first use if not already present. See `ensureGitignore()`
    in workspace.go.

11. **Should `--workspace` without `--workspace-base` default to HEAD or the current branch?**
    **Prefer the checked-out branch.** `getCurrentBranch()` reads `git rev-parse --abbrev-ref HEAD`.
    Falls back to `"HEAD"` on detached HEAD. See ### 10.

12. **Should workspace cleanup be manual or automatic?**
    **Lifecycle/management commands.** `gmd agent session list` shows orphaned
    workspaces, `gmd agent session kill <name>` auto-removes workspace,
    `gmd agent session merge <name>` merges changes back. See ### 11, ### 12.

13. **Should `--workspace --tmux` combined rollback the worktree if tmux fails?**
    **Yes, auto-rollback.** `Launch()` defers `removeWorkspace()` on error after
    successful worktree creation. See ### 13.
