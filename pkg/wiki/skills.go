package wiki

import (
	"fmt"
	"os"
	"path/filepath"
)

type SkillTemplate struct {
	Name        string
	Target      string
	Description string
	Content     string
}

func ListSkillTemplates() ([]SkillTemplate, error) {
	entries, err := wikiEmbedsFS.ReadDir("embeds/skills")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}

	templates := make([]SkillTemplate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := wikiEmbedsFS.ReadFile("embeds/skills/" + entry.Name())
		if err != nil {
			continue
		}
		name := entry.Name()
		target := ""
		desc := ""
		switch name {
		case "AGENTS.md":
			target = "universal"
			desc = "Universal agent instructions (ingest/query/lint workflows)"
		case "WIKI_SCHEMA.md":
			target = "reference"
			desc = "Wiki conventions, directory structure, page formats"
		case "claude-code.md":
			target = "claude"
			desc = "Claude Code-specific skill with tool mappings"
		case "codex-cli.md":
			target = "codex"
			desc = "Codex CLI-specific skill"
		case "opencode.md":
			target = "opencode"
			desc = "OpenCode-specific skill"
		case "generic.md":
			target = "generic"
			desc = "Fallback for any AGENTS.md-reading agent"
		}
		templates = append(templates, SkillTemplate{
			Name:        name,
			Target:      target,
			Description: desc,
			Content:     string(data),
		})
	}
	return templates, nil
}

func GetSkillTemplate(name string) (*SkillTemplate, error) {
	templates, err := ListSkillTemplates()
	if err != nil {
		return nil, err
	}
	for _, t := range templates {
		if t.Name == name {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("skill template %q not found", name)
}

var agentPaths = map[string]func() (string, error){
	"claude": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".claude", "skills", "gmd-wiki.md"), nil
	},
	"codex": func() (string, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return filepath.Join(cwd, ".agents", "skills", "gmd-wiki"), nil
	},
	"opencode": func() (string, error) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "opencode", "skills", "gmd-wiki.md"), nil
	},
}

var agentSkillFiles = map[string]string{
	"claude":   "claude-code.md",
	"codex":    "codex-cli.md",
	"opencode": "opencode.md",
	"generic":  "generic.md",
	"all":      "",
}

func WriteSkills(target string) ([]string, error) {
	var written []string

	writeOne := func(t string) error {
		skillFile, ok := agentSkillFiles[t]
		if !ok {
			return fmt.Errorf("unknown target %q", t)
		}

		tmpl, err := GetSkillTemplate(skillFile)
		if err != nil {
			return err
		}

		pathFn, ok := agentPaths[t]
		if !ok {
			return fmt.Errorf("unknown agent path for %q", t)
		}

		destPath, err := pathFn()
		if err != nil {
			return err
		}

		dir := filepath.Dir(destPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		if err := os.WriteFile(destPath, []byte(tmpl.Content), 0600); err != nil {
			return fmt.Errorf("writing %s: %w", destPath, err)
		}

		written = append(written, destPath)
		return nil
	}

	if target == "all" {
		for t := range agentPaths {
			if err := writeOne(t); err != nil {
				return written, err
			}
		}
	} else {
		if err := writeOne(target); err != nil {
			return written, err
		}
	}

	return written, nil
}

func AgentDiscoveryPaths() map[string]string {
	paths := make(map[string]string)
	for name, pathFn := range agentPaths {
		p, err := pathFn()
		if err != nil {
			paths[name] = "error: " + err.Error()
		} else {
			paths[name] = p
		}
	}
	return paths
}

func CheckAgentInstalled(name string) bool {
	switch name {
	case "claude":
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		_, err = os.Stat(filepath.Join(home, ".claude"))
		return err == nil
	case "codex":
		cwd, err := os.Getwd()
		if err != nil {
			return false
		}
		_, err = os.Stat(filepath.Join(cwd, ".agents"))
		return err == nil
	case "opencode":
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		_, err = os.Stat(filepath.Join(home, ".config", "opencode"))
		return err == nil
	}
	return false
}

func CheckSkillInstalled(name string) bool {
	pathFn, ok := agentPaths[name]
	if !ok {
		return false
	}
	p, err := pathFn()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}
