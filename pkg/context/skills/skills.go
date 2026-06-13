package skills

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed embeds
var skillsEmbedsFS embed.FS

func ListSkillNames() ([]string, error) {
	entries, err := skillsEmbedsFS.ReadDir("embeds")
	if err != nil {
		return nil, fmt.Errorf("reading embedded skills: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func GetSkillContent(name string) (string, error) {
	data, err := skillsEmbedsFS.ReadFile("embeds/" + name + "/SKILL.md")
	if err != nil {
		return "", fmt.Errorf("skill %q not found", name)
	}
	return string(data), nil
}

func HarnessNames() []string {
	return []string{"claude", "codex", "opencode"}
}

func harnessDir(baseDir string, global bool, name string) (string, error) {
	switch name {
	case "claude":
		return filepath.Join(baseDir, ".claude"), nil
	case "codex":
		return filepath.Join(baseDir, ".agents"), nil
	case "opencode":
		if global {
			return filepath.Join(baseDir, ".config", "opencode"), nil
		}
		return filepath.Join(baseDir, ".opencode"), nil
	default:
		return "", fmt.Errorf("unknown harness %q", name)
	}
}

func harnessSkillsDir(baseDir string, global bool, name string) (string, error) {
	hd, err := harnessDir(baseDir, global, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(hd, "skills"), nil
}

func WriteSkillTo(baseDir string, global bool, harness string) (string, error) {
	skillsDir, err := harnessSkillsDir(baseDir, global, harness)
	if err != nil {
		return "", err
	}

	names, err := ListSkillNames()
	if err != nil {
		return "", err
	}

	var copied int
	for _, skill := range names {
		entries, err := skillsEmbedsFS.ReadDir("embeds/" + skill)
		if err != nil {
			return "", fmt.Errorf("reading skill %q: %w", skill, err)
		}
		dest := filepath.Join(skillsDir, skill)
		if err := os.RemoveAll(dest); err != nil {
			return "", fmt.Errorf("cleaning %s: %w", dest, err)
		}
		if err := os.MkdirAll(dest, 0755); err != nil {
			return "", fmt.Errorf("creating %s: %w", dest, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := skillsEmbedsFS.ReadFile("embeds/" + skill + "/" + entry.Name())
			if err != nil {
				return "", fmt.Errorf("reading %s/%s: %w", skill, entry.Name(), err)
			}
			out := filepath.Join(dest, entry.Name())
			if err := os.WriteFile(out, data, 0600); err != nil {
				return "", fmt.Errorf("writing %s: %w", out, err)
			}
		}
		copied++
	}

	if copied == 0 {
		return "", fmt.Errorf("no skills found in embed")
	}
	return skillsDir, nil
}

func SkillPath(baseDir string, global bool, harness, skill string) (string, error) {
	sd, err := harnessSkillsDir(baseDir, global, harness)
	if err != nil {
		return "", err
	}
	return filepath.Join(sd, skill), nil
}

func CheckHarnessInstalled(baseDir string, global bool, name string) (bool, error) {
	hd, err := harnessDir(baseDir, global, name)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(hd)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking %s: %w", hd, err)
	}
	return true, nil
}

func SkillInstalled(baseDir string, global bool, harness, skill string) (bool, error) {
	sp, err := SkillPath(baseDir, global, harness, skill)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(sp)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking %s: %w", sp, err)
	}
	return info.IsDir(), nil
}
