package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadEnvFiles loads env files and CLI-provided env/secret entries in precedence order.
// Later values overwrite earlier ones (os.Setenv). Missing files are silently skipped.
//
// Precedence (lowest to highest):
//  1. <UserConfigDir>/gmd/default.env
//  2. <UserConfigDir>/gmd/secret.env
//  3. <project>/.gmd/default.env
//  4. <project>/.gmd/secret.env
//  5. --env flag values (in flag order)
//  6. --secret flag values (in flag order)
func LoadEnvFiles(projectRoot string, envFlags []string, secretFlags []string) error {
	globalDir, err := GlobalConfigDir()
	if err != nil {
		return fmt.Errorf("finding config dir: %w", err)
	}

	// Global env files
	loadEnvFileIfExists(filepath.Join(globalDir, "default.env"))
	loadEnvFileIfExists(filepath.Join(globalDir, "secret.env"))

	// Project env files (if project root is known)
	if projectRoot != "" {
		projectDir := filepath.Join(projectRoot, ".gmd")
		loadEnvFileIfExists(filepath.Join(projectDir, "default.env"))
		loadEnvFileIfExists(filepath.Join(projectDir, "secret.env"))
	}

	// CLI flags (highest precedence)
	for _, entry := range envFlags {
		loadEnvEntry(entry)
	}
	for _, entry := range secretFlags {
		loadEnvEntry(entry)
	}

	return nil
}

func loadEnvFileIfExists(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		loadEnvEntry(line)
	}
}

func loadEnvEntry(entry string) {
	idx := strings.Index(entry, "=")
	if idx < 1 {
		return // skip entries without key=value
	}
	key := strings.TrimSpace(entry[:idx])
	val := strings.TrimSpace(entry[idx+1:])
	val = strings.Trim(val, "'\"")
	if key == "" {
		return
	}
	os.Setenv(key, val)
}
