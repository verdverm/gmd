package main

import (
	"fmt"

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
		fmt.Printf("collection add: %q (not yet implemented, Phase 4)\n", args[0])
		return nil
	},
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("collection list (not yet implemented, Phase 4)")
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a collection and its chunks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection remove: %q (not yet implemented, Phase 4)\n", args[0])
		return nil
	},
}

var collectionRenameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection rename: %q -> %q (not yet implemented, Phase 4)\n", args[0], args[1])
		return nil
	},
}

var collectionShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show collection details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection show: %q (not yet implemented, Phase 4)\n", args[0])
		return nil
	},
}

var collectionIncludeCmd = &cobra.Command{
	Use:   "include <pattern>",
	Short: "Add a file pattern to a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection include: %q %q (not yet implemented, Phase 4)\n", args[0], args[1])
		return nil
	},
}

var collectionExcludeCmd = &cobra.Command{
	Use:   "exclude <pattern>",
	Short: "Remove a file pattern from a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("collection exclude: %q %q (not yet implemented, Phase 4)\n", args[0], args[1])
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
