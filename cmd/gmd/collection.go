package main

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var collectionCmd = &cobra.Command{
	Use:   "collection [add|list|remove|rename|show|include|exclude]",
	Short: "Manage collections",
	Long:  `Manage document collections: add, list, remove, rename, show, include, exclude.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var collectionAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new collection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection add: %q (not yet implemented, needs config file editing in pkg)\n", args[0])
		return nil
	},
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
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
			fmt.Printf("    pattern: %s\n", col.Pattern)
			if col.Context != "" {
				fmt.Printf("    context: %s\n", col.Context)
			}
		}
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a collection and its chunks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection remove: %q (not yet implemented, needs config file editing in pkg)\n", args[0])
		return nil
	},
}

var collectionRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection rename: %q -> %q (not yet implemented, needs config file editing in pkg)\n", args[0], args[1])
		return nil
	},
}

var collectionShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show collection details",
	Args:  cobra.ExactArgs(1),
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
		fmt.Printf("path:    %s\n", col.Path)
		fmt.Printf("pattern: %s\n", col.Pattern)
		if len(col.Ignore) > 0 {
			fmt.Printf("ignore:  %v\n", col.Ignore)
		}
		if col.Context != "" {
			fmt.Printf("context: %s\n", col.Context)
		}
		fmt.Printf("includeByDefault: %v\n", col.IncludeByDefault)

		counts, err := r.TSClient().CountByCollection(context.Background(), []string{name})
		if err != nil {
			fmt.Printf("chunks:  (error counting: %v)\n", err)
		} else {
			fmt.Printf("chunks:  %d\n", counts[name])
		}
		return nil
	},
}

var collectionIncludeCmd = &cobra.Command{
	Use:   "include <name> <pattern>",
	Short: "Add a file pattern to a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection include: %q %q (not yet implemented, needs config file editing in pkg)\n", args[0], args[1])
		return nil
	},
}

var collectionExcludeCmd = &cobra.Command{
	Use:   "exclude <name> <pattern>",
	Short: "Remove a file pattern from a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection exclude: %q %q (not yet implemented, needs config file editing in pkg)\n", args[0], args[1])
		return nil
	},
}

func init() {
	collectionCmd.AddCommand(collectionAddCmd)
	collectionCmd.AddCommand(collectionListCmd)
	collectionCmd.AddCommand(collectionRemoveCmd)
	collectionCmd.AddCommand(collectionRenameCmd)
	collectionCmd.AddCommand(collectionShowCmd)
	collectionCmd.AddCommand(collectionIncludeCmd)
	collectionCmd.AddCommand(collectionExcludeCmd)
}
