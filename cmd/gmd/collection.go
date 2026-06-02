package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/verdverm/gmd/pkg/config"
)

var collectionCmd = &cobra.Command{
	Use:   "collection [add|list|remove|rename|show|include|exclude]",
	Short: "Manage collections — add, list, remove, rename, show, include, exclude",
	Long: `Collections define which files to index. Each collection has a root path,
a glob pattern for matching files, optional ignore rules, and optional
context text for AI assistants.

Workflow:
  gmd collection add mydocs --path ./docs --pattern "**/*.md"
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

var collAddPath string
var collAddPatterns []string

var collectionAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new collection to the config",
	Long: `Creates a new collection entry in the project config with the given name,
root path, and file glob patterns.

Example:
  gmd collection add mydocs --path ./docs --patterns "**/*.md"

After adding, run 'gmd update' to index the collection's files.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]
		path := collAddPath
		patterns := collAddPatterns
		if len(patterns) == 0 {
			patterns = []string{"**/*.md"}
		}
		return config.AddCollection(r.Config(), name, path, patterns)
	},
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured collections with paths and patterns",
	Long: `Displays every collection's name, root path, file pattern, and context
description as configured in .gmd/config.cue. Does not query Typesense
— shows config only.

Example:
  gmd collection list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		collections := r.Config().Collections
		names := make([]string, 0, len(collections))
		for name := range collections {
			names = append(names, name)
		}
		sort.Strings(names)

		if len(names) == 0 {
			fmt.Println("No collections configured.")
			return nil
		}

		for _, name := range names {
			col := collections[name]
			fmt.Printf("  %s\n", name)
			fmt.Printf("    path:    %s\n", col.Path)
			fmt.Printf("    patterns: %v\n", col.Patterns)
			if len(col.Ignore) > 0 {
				fmt.Printf("    ignore:  %v\n", col.Ignore)
			}
			if col.Context != "" {
				fmt.Printf("    context: %s\n", col.Context)
			}
		}
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a collection and all its indexed chunks",
	Long: `Removes the collection from the config and deletes all associated chunks
from Typesense. This operation is immediate and cannot be undone.

Example:
  gmd collection remove mydocs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		name := args[0]

		if err := config.RemoveCollection(r.Config(), name); err != nil {
			return err
		}

		if err := r.TSClient().DeleteChunksByCollection(context.Background(), r.Config().CollectionKey(name)); err != nil {
			return fmt.Errorf("deleting chunks for %q: %w", name, err)
		}

		fmt.Printf("Removed collection %q and its chunks.\n", name)
		return nil
	},
}

var collectionRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a collection in the config",
	Long: `Renames a collection without affecting its indexed chunks. The old name
is updated in the config only — existing Typesense data is preserved
under the new collection key.

Example:
  gmd collection rename mydocs docs`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		oldName := args[0]
		newName := args[1]

		if err := config.RenameCollection(r.Config(), oldName, newName); err != nil {
			return err
		}

		fmt.Printf("Renamed collection %q to %q.\n", oldName, newName)
		return nil
	},
}

var collectionShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show collection config details and chunk count",
	Long: `Displays the full configuration for a collection including path, pattern,
ignore rules, context, and includeByDefault — along with the current
chunk count queried from Typesense.

Example:
  gmd collection show mydocs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}

		name := args[0]
		col, ok := r.Config().Collections[name]
		if !ok {
			return fmt.Errorf("collection %q not found", name)
		}

		fmt.Printf("name:    %s\n", name)
		fmt.Printf("path:     %s\n", col.Path)
		fmt.Printf("patterns: %v\n", col.Patterns)
		if len(col.Ignore) > 0 {
			fmt.Printf("ignore:  %v\n", col.Ignore)
		}
		if col.Context != "" {
			fmt.Printf("context: %s\n", col.Context)
		}
		fmt.Printf("includeByDefault: %v\n", col.IncludeByDefault)

		key := r.Config().CollectionKey(name)
		counts, err := r.TSClient().CountByCollection(context.Background(), []string{key})
		if err != nil {
			fmt.Printf("chunks:  (error counting: %v)\n", err)
		} else {
			fmt.Printf("chunks:  %d\n", counts[key])
		}
		return nil
	},
}

var collectionIncludeCmd = &cobra.Command{
	Use:   "include <name> <patterns...>",
	Short: "Set the file glob patterns for a collection",
	Long: `Sets the file-matching patterns for a collection (e.g. "**/*.md").

Example:
  gmd collection include mydocs "**/*.md" "**/*.txt"

Run 'gmd update' after changing patterns to re-index with the new
matching rules.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.SetCollectionPatterns(r.Config(), args[0], args[1:])
	},
}

var collectionExcludeCmd = &cobra.Command{
	Use:   "exclude <name> <pattern>",
	Short: "Add an ignore pattern to exclude files from a collection",
	Long: `Adds a glob pattern to the collection's ignore list. Matching files will
be skipped during indexing. Multiple patterns can be added.

Example:
  gmd collection exclude docs "node_modules/**"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := getRuntime()
		if err != nil {
			return err
		}
		return config.AddIgnorePattern(r.Config(), args[0], args[1])
	},
}

func init() {
	collectionAddCmd.Flags().StringVarP(&collAddPath, "path", "p", ".", "Collection root path")
	collectionAddCmd.Flags().StringSliceVarP(&collAddPatterns, "patterns", "P", []string{"**/*.md"}, "File glob patterns")
	collectionCmd.AddCommand(collectionAddCmd)
	collectionCmd.AddCommand(collectionListCmd)
	collectionCmd.AddCommand(collectionRemoveCmd)
	collectionCmd.AddCommand(collectionRenameCmd)
	collectionCmd.AddCommand(collectionShowCmd)
	collectionCmd.AddCommand(collectionIncludeCmd)
	collectionCmd.AddCommand(collectionExcludeCmd)
}
