package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	sentinelDir      = ".gmd"
	sentinelFile     = "gmd.cue"
	globalConfigDir  = ".config/gmd"
	globalConfigFile = "config.cue"
)

// FindProjectRoot walks up from start looking for a .gmd/ directory or gmd.cue file.
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
		info, err = os.Stat(filepath.Join(dir, sentinelFile))
		if err == nil && !info.IsDir() {
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
