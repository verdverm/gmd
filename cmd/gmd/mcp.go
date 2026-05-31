package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mcpHTTP bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server (stdio or HTTP)",
	Long: `Starts the MCP server for AI agent integration.
Supports stdio transport (default) and Streamable HTTP.

When run without arguments, starts in stdio mode (for IDE integration).
Use --http to start in HTTP mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Println("MCP server not yet implemented (Phase 6)")
		return nil
	},
}

func init() {
	mcpCmd.Flags().BoolVarP(&mcpHTTP, "http", "", false, "start in HTTP mode")
}
