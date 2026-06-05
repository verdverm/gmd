package main

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionCmd = &cobra.Command{
	Use:   "collection [create|list|remove|rename|show|include|exclude]",
	Short: "Manage collections — create, list, remove, rename, show, include, exclude",
	Long: `Collections define which files to index. Each collection has a root path,
a glob pattern for matching files, optional ignore rules, and optional
context text for AI assistants.

Workflow:
  gmd collection create mydocs --path ./docs --pattern "**/*.md"
  gmd collection list
  gmd collection show mydocs
  gmd collection include mydocs "**/*.{md,txt}"
  gmd collection exclude mydocs node_modules/**
  gmd collection rename mydocs docs
  gmd collection remove docs

After adding or modifying collections, run 'gmd update' to index files.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// sortedKeys returns the keys of m in sorted order.
func sortedKeys(m map[string]config.FrontmatterField) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
