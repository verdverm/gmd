package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	sentinelDir      = ".gmd"
	globalConfigDir  = ".config/gmd"
	globalConfigFile = "config.cue"
)

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

// GlobalConfigPath returns the path to the global config file (~/.config/gmd/config.cue).
func GlobalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, globalConfigDir, globalConfigFile), nil
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
