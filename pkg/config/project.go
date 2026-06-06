package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	sentinelDir      = ".gmd"
	globalConfigFile = "config.cue"
)

// GlobalConfigDir returns the platform-appropriate global config directory for gmd.
// Uses os.UserConfigDir() which respects XDG (Linux), Library/Application Support (macOS), etc.
// Falls back to ~/.config/gmd if UserConfigDir/gmd does not exist (backward compat on macOS).
func GlobalConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	primary := filepath.Join(dir, "gmd")
	if info, err := os.Stat(primary); err == nil && info.IsDir() {
		return primary, nil
	}

	// Fallback to ~/.config/gmd for backward compatibility
	home, err := os.UserHomeDir()
	if err != nil {
		return primary, nil
	}
	fallback := filepath.Join(home, ".config", "gmd")
	if info, err := os.Stat(fallback); err == nil && info.IsDir() {
		return fallback, nil
	}

	return primary, nil
}

// FindProjectRoot walks up from start looking for a .gmd/ directory.
// Returns the absolute path to the project root, or empty string if not found.
func FindProjectRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return ""
	}
	dir := abs
	for {
		info, err := os.Stat(filepath.Join(dir, sentinelDir))
		if err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// GlobalConfigPath returns the path to the global CUE config file.
// Uses os.UserConfigDir() (e.g. ~/.config/gmd/config.cue on Linux, ~/Library/Application Support/gmd/config.cue on macOS).
func GlobalConfigPath() (string, error) {
	dir, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, globalConfigFile), nil
}

// MatchSourcesByCWD returns the names of sources (collections and wikis) whose path encompasses cwd.
func MatchSourcesByCWD(cfg *Config, cwd string) []string {
	var matched []string
	for name, col := range cfg.Collections {
		colPath := col.Path
		if !filepath.IsAbs(colPath) {
			colPath = filepath.Join(cfg.ProjectRoot, colPath)
		}
		colPath = filepath.Clean(colPath)
		rel, err := filepath.Rel(colPath, cwd)
		if err == nil && !strings.HasPrefix(rel, "..") {
			matched = append(matched, name)
		}
	}
	for name, wc := range cfg.Wikis {
		wikiPath := wc.Path
		if !filepath.IsAbs(wikiPath) {
			wikiPath = filepath.Join(cfg.ProjectRoot, wikiPath)
		}
		wikiPath = filepath.Clean(wikiPath)
		rel, err := filepath.Rel(wikiPath, cwd)
		if err == nil && !strings.HasPrefix(rel, "..") {
			matched = append(matched, name)
		}
	}
	return matched
}

// MatchCollectionsByCWD returns the names of collections whose path encompasses cwd.
func MatchCollectionsByCWD(cfg *Config, cwd string) []string {
	var matched []string
	for name, col := range cfg.Collections {
		colPath := col.Path
		if !filepath.IsAbs(colPath) {
			colPath = filepath.Join(cfg.ProjectRoot, colPath)
		}
		colPath = filepath.Clean(colPath)
		rel, err := filepath.Rel(colPath, cwd)
		if err == nil && !strings.HasPrefix(rel, "..") {
			matched = append(matched, name)
		}
	}
	return matched
}
