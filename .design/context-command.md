# context-command.md

**created:** 2026-06-10
**updated:** 2026-06-13
**phase:** implemented
**status:** done

## Context

`gmd` had several separate mechanisms for surfacing "agent context" to AI
coding assistants, plus dead top-level `gmd context` commands. These have been
consolidated into a single `gmd context` command tree backed by packages under
`pkg/context/`.

## Final Command Tree

```
gmd context
├── status                              # installed skills per harness, agent roles, AGENTS.md docs
├── install [--target <name>]           # copy skill dirs into harness discovery paths
├── uninstall [--target <name>]         # remove skill dirs from harness discovery paths
├── list                                # flat list across all categories
├── show <name>                         # disambiguate and output a named item
├── agentsmd [list|show]                # AGENTS.md reference documents (embedded)
├── skills [list|show]                  # skill directories (embedded)
└── agents [list|show]                  # agent role definitions (filesystem, dir-of-dirs)
```

### --global flag

`contextCmd` has a persistent `--global` flag. When set, operations target the
global (user home directory) scope. Default is project-local.

| Scope | Base path | Example |
|---|---|---|
| Global (`--global`) | `os.UserHomeDir()` | `~/.claude/skills/`, `~/.config/opencode/skills/` |
| Project-local (default) | Project root (via `.gmd/` sentinel) | `./.agents/skills/`, `./.opencode/skills/` |

### --target flag (harness selection)

`contextCmd` has a persistent `--target` flag for `install`/`uninstall`,
accepting: `claude`, `codex`, `opencode`, `all` (default when empty).
Selects which harness to install/uninstall skill directories to.

### show <name> disambiguation

`context show <name>` resolves by checking:
1. `agentsmd` detail levels (exact match)
2. Skill directory names from embed FS (exact match)
3. Agent role definition names from filesystem

Ambiguous match across categories is an error. Use category subcommand to disambiguate.

### uninstall semantics

Idempotent: removes skill directories from harness paths via `os.RemoveAll`.
Reports already-absent as informational, not an error.

### status output

Shows combined overview:
1. AGENTS.md: available detail levels
2. Skills: per-skill per-harness installation status with paths
3. Harness discovery: whether each harness is detected on the system
4. Agent roles: listed by name

## Package Layout

```
pkg/context/
├── agentsmd/                     # AGENTS.md embeds + GetContent/ValidNames
│   ├── agents.go
│   └── embeds/                   # oneline.md, summary.md, detailed.md, full.md
├── skills/                       # skill management
│   ├── skills.go                 # ListSkillNames, GetSkillContent, WriteSkillTo, harness paths
│   └── embeds/
│       └── gmd-wiki/
│           └── SKILL.md          # the one skill (harness-agnostic)
└── agents/
    └── agents.go                 # agent role dir-of-dirs: list/show
```

## Skills Design

Skills are **directories** containing a `SKILL.md` file. The directory name is the
skill name. Skills are harness-agnostic — the same skill directory is copied into
each target harness's skills path at install time.

```
embeds/
  gmd-wiki/          # skill directory (name = skill name)
    SKILL.md         # skill content
```

`ListSkillNames()` reads the embed FS directories at runtime. No hardcoded
constants, no metadata maps, no structs. Everything is derived from the
filesystem.

### Skills API (`pkg/context/skills/`)

| Function | Returns | Description |
|---|---|---|
| `ListSkillNames()` | `([]string, error)` | Reads embed FS dirs listing |
| `GetSkillContent(name)` | `(string, error)` | Reads `SKILL.md` from named skill dir |
| `HarnessNames()` | `[]string` | Well-known harnesses: claude, codex, opencode |
| `WriteSkillTo(baseDir, global, harness)` | `(string, error)` | Copies all skill dirs into harness skills path |
| `SkillPath(baseDir, global, harness, skill)` | `(string, error)` | Path where a skill dir would be installed |
| `CheckHarnessInstalled(baseDir, global, name)` | `(bool, error)` | Whether harness config dir exists |
| `SkillInstalled(baseDir, global, harness, skill)` | `(bool, error)` | Whether skill dir exists at harness path |

All functions return errors. No silent empty-string returns, no swallowed
`os.Stat` failures (uses `os.IsNotExist` to distinguish "not found" from real
errors).

### Harness paths

| Harness | Config dir | Skills dir |
|---|---|---|
| claude | `{base}/.claude` | `{base}/.claude/skills` |
| codex | `{base}/.agents` | `{base}/.agents/skills` |
| opencode (global) | `{base}/.config/opencode` | `{base}/.config/opencode/skills` |
| opencode (project) | `{base}/.opencode` | `{base}/.opencode/skills` |

`harnessSkillsDir` calls `harnessDir` + `/skills` — no duplicated switch logic.

## Wiki Reference Document

`WIKI_SCHEMA.md` was moved from the skills package into `pkg/wiki/embeds/` as
`wiki_schema.md` (matching the existing `snake_case.md` convention of other wiki
embeds: `ingest_system.md`, `query_system.md`, etc.).

- `pkg/wiki/wiki.go` — reads `wikiEmbedsFS.ReadFile("embeds/wiki_schema.md")` during wiki init scaffolding
- `pkg/wiki/agent_prompts.go` — reads from wiki embed for LLM prompts (replaced `skills.GetSkillTemplate("WIKI_SCHEMA.md")`)

No more dependency on the skills package for wiki reference content.

## Callers

| Caller | Uses |
|---|---|
| `cmd/gmd/context_status.go` | `ListSkillNames`, `HarnessNames`, `SkillInstalled`, `SkillPath`, `CheckHarnessInstalled` |
| `cmd/gmd/context_list.go` | `ListSkillNames` |
| `cmd/gmd/context_show.go` | `ListSkillNames`, `GetSkillContent` |
| `cmd/gmd/context_skills_list.go` | `ListSkillNames` |
| `cmd/gmd/context_skills_show.go` | `GetSkillContent` |
| `cmd/gmd/context_install.go` | `HarnessNames`, `WriteSkillTo` |
| `cmd/gmd/context_uninstall.go` | `HarnessNames`, `ListSkillNames`, `SkillPath` |
| `cmd/gmd/wiki_create.go` | `HarnessNames`, `WriteSkillTo` (via `os.UserHomeDir()`) |
| `pkg/wiki/doctor.go` | `ListSkillNames`, `CheckHarnessInstalled`, `SkillInstalled`, `WriteSkillTo` |
| `pkg/wiki/wiki.go` | reads `wiki_schema.md` from own embed |
| `pkg/wiki/agent_prompts.go` | reads `wiki_schema.md` from own embed |

## What Was Deleted

| Item | Reason |
|---|---|
| `cmd/gmd/agentsmd.go` | Replaced by `context agentsmd` |
| `cmd/gmd/context_add.go`, `context_rm.go` | Dead commands, deleted |
| `cmd/gmd/wiki_context*.go` (4 files) | Dead commands, deleted |
| `cmd/gmd/wiki_skills*.go` (4 files) | Replaced by `context skills` + `context install` |
| `pkg/agentsmd/` | Moved to `pkg/context/agentsmd/` |
| `pkg/wiki/skills.go` | Replaced by `pkg/context/skills/skills.go` |
| `pkg/wiki/embeds/skills/` (6 files) | 5 flat files deleted; WIKI_SCHEMA.md moved to `pkg/wiki/embeds/wiki_schema.md` |
| `SkillTemplate` struct | Skills are now just directory names + string content |
| Backward-compat wrappers | All 5 wrappers deleted; callers use real API with proper error handling |
| `skllDir` constant | Derived from embed FS at runtime |

## What Was Modified

| File | Change |
|---|---|
| `cmd/gmd/main.go` | Removed `agentsmdCmd`; kept `contextCmd`; updated help text |
| `cmd/gmd/init.go` | Import path `pkg/agentsmd` → `pkg/context/agentsmd` |
| `cmd/gmd/wiki.go` | Removed `[skills]` and `[context]` from Use string |
| `cmd/gmd/collection_list.go`, `collection_show.go`, `wiki_show.go` | Removed Context field display |
| `pkg/wiki/doctor.go` | Uses real API: `CheckHarnessInstalled`, `SkillInstalled`, `ListSkillNames`, `WriteSkillTo` (no wrappers) |
| `pkg/wiki/wiki.go` | Reads `wiki_schema.md` from own embed FS; no skills import |
| `pkg/wiki/agent_prompts.go` | Reads `wiki_schema.md` from own embed FS; no skills import |
| `pkg/config/` | Context field kept as dead storage (pruning deferred) |

## Design Decisions

### No metadata in code

`ListSkillNames()` reads embed FS directories at runtime. There is no
`SkillTemplate` struct, no target/description metadata, no hardcoded maps. Skill
names are directory names. Skill content is `SKILL.md` content. Everything
derived from the filesystem.

### No backward compatibility

Alpha features. Old commands removed entirely. No backward-compat wrappers.
Callers use scoped API functions directly with proper error handling.

### No silent error suppression

Every function that can fail returns an error. Functions that use `os.Stat`
distinguish "not found" (`os.IsNotExist`) from real errors (permission denied,
etc.). No empty strings returned in place of errors.

### Wiki reference doc lives in wiki package

`WIKI_SCHEMA.md` was never a skill. It's wiki infrastructure — scaffolded into
wiki directories and injected into LLM prompts. It now lives in
`pkg/wiki/embeds/wiki_schema.md` alongside the other wiki embed files.

## Code Review Findings

### Alignment: What the staged implementation gets right

- **Full command tree.** Every command in the tree (`status`, `install`,
  `uninstall`, `list`, `show`, `agentsmd [list|show]`, `skills [list|show]`,
  `agents [list|show]`) is implemented with correct registration in `context.go`
  init(). No missing subcommands.

- **`--global` flag behavior.** The persistent flag on `contextCmd` correctly
  targets `os.UserHomeDir()` when set, and falls back from empty project root.
  Fallthrough logic in `status`, `install`, `uninstall` is consistent: when
  `--global` is set OR `baseDir` is empty (no project detected), the base
  directory switches to home and `isGlobal = true`.

- **`show <name>` disambiguation resolution order.** The three-tier check
  (agentsmd exact match -> skills exact match -> agents filesystem) matches the
  spec exactly. Ambiguous matches across categories produce a descriptive error
  listing all matches. Category subcommands (`agentsmd show`, `skills show`,
  `agents show`) bypass disambiguation entirely.

- **`uninstall` idempotence.** Uses `os.IsNotExist` to distinguish "already
  absent" (reported as informational) from real `os.Stat` failures. No silent
  error swallowing.

- **`status` output format.** Prints all four sections in order: AGENTS.md
  available levels, per-skill per-harness installation status with paths,
  harness detection (absent/detected), and agent role names.

- **Skills API surface.** All eight exported functions in `pkg/context/skills/`
  match the design table: `ListSkillNames`, `GetSkillContent`, `HarnessNames`,
  `WriteSkillTo`, `SkillPath`, `CheckHarnessInstalled`, `SkillInstalled`. All
  return errors properly.

- **Package moves and deletions.** `pkg/agentsmd/` -> `pkg/context/agentsmd/`,
  `pkg/wiki/embeds/skills/` -> `pkg/context/skills/embeds/`, dead commands
  removed (`context_add.go`, `context_rm.go`, all `wiki_context*.go`, all
  `wiki_skills*.go`). All match the deletion table.

- **Doctor.go migration.** `pkg/wiki/doctor.go` uses the real API
  (`CheckHarnessInstalled`, `SkillInstalled`, `ListSkillNames`, `WriteSkillTo`)
  with proper error propagation. No backward-compat wrappers.

- **WIKI_SCHEMA.md isolation.** `pkg/wiki/wiki.go` and `pkg/wiki/agent_prompts.go`
  read `embeds/wiki_schema.md` from their own `wikiEmbedsFS` embed. No import
  of the skills package.

- **Agent harness path consistency.** `harnessSkillsDir` calls `harnessDir` +
  `/skills` for all three harnesses — no duplicated switch logic.

- **WriteSkillTo cleans before writing.** Calls `os.RemoveAll(dest)` before
  `os.MkdirAll(dest)` and re-copies all files. No stale files remain from
  previous installs.

- **Agent show reads all files in a role dir.** `agents.ShowAgent` iterates
  directory entries and reads every non-directory file, returning them as a
  name->content map. No hardcoded filenames.

### Gaps and deviations

- **`--target` flag scope too broad.** The design specifies `--target` for
  `install`/`uninstall` only, but it is registered as a persistent flag on
  `contextCmd` (line 40 of `context.go`). This makes it appear in help output
  and accept values for every subcommand (`status`, `list`, `show`, `skills`,
  `agents`, `agentsmd`), where it is silently ignored. Fix: move the flag to
  `installCmd` and `uninstallCmd` only, or add a runtime check that errors if
  `--target` is provided to a command that does not use it.

- **`rootCmd` help text is misleading.** `main.go:22` shows:
  ```
  gmd context    output AGENTS.md content for AI coding assistants
  ```
  This understates the command scope — `gmd context` also manages status,
  install/uninstall, skills, and agent roles. It is no longer just an
  AGENTS.md output command. Fix: update to something like:
  ```
  gmd context    manage agent context (skills, AGENTS.md, agent roles)
  ```

- **`context skills show` example references deleted skill.** `context_skills.go:17`
  shows `gmd context skills show AGENTS.md` in the Long help text. `AGENTS.md`
  is not a skill — skills are directories containing `SKILL.md`, and the only
  embedded skill is `gmd-wiki`. Running this example returns:
  `skill "AGENTS.md" not found`. Fix: change example to `gmd context skills
  show gmd-wiki`.

- **SKILL.md content changed without documentation.** When `AGENTS.md` (the old
  universal skill) was moved to `pkg/context/skills/embeds/gmd-wiki/SKILL.md`,
  two content changes were made that are not described in the design doc:
  1. The "Frontmatter Conventions" table (type, tags, status, sources,
     difficulty, source_url) was removed entirely.
  2. All `---` horizontal rule markers were changed to `--` in list
     continuations (e.g., `---` -> `--` after: "entities/ -- people, orgs...").
     This is a downgrade — `---` renders as a horizontal rule in Markdown,
     `--` renders as a literal dash pair.

- **`pkg/context/agents/agents.go` has no tests.** Unlike `pkg/context/skills/`
  which has both unit and integration test files, the agents package has zero
  test coverage. `ListAgents` and `ShowAgent` both do filesystem I/O and should
  have at minimum unit tests with temp dirs.

- **Agent role directory paths undocumented in spec.** The design's scope table
  (Global vs Project-local) only covers harness paths for claude/codex/opencode.
  The agents package uses `~/.config/gmd/agents/` (global) and `.gmd/agents/`
  (project-local). These paths are referenced only in the `agents` command's
  Long help string. They should be added to the scope table.

- **`context show` disambiguation lacks concrete examples.** The show command's
  Long help says "Use category subcommand to disambiguate" but does not provide
  examples. A user seeing "ambiguous name 'foo' matches: agentsmd/foo,
  skills/foo" has to infer that `gmd context agentsmd show foo` or
  `gmd context skills show foo` would work.
