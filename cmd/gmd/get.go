package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getPath string

var getCmd = &cobra.Command{
	Use:   "get <path>",
	Short: "Get document content by path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Printf("get: %q (not yet implemented, Phase 3)\n", args[0])
		return nil
	},
}

var multiGetCmd = &cobra.Command{
	Use:   "multi-get",
	Short: "Batch fetch documents by path pattern",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Println("multi-get (not yet implemented, Phase 3)")
		return nil
	},
}
