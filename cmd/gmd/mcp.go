package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var mcpHTTP bool

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server for AI agent integration",
	Long: `Launches a Model Context Protocol server that lets AI coding assistants
(like Claude, Cursor, VS Code Copilot) interact with GMD directly.

Two transport modes:
  stdio  (default)  for IDE plugins that spawn the process
  HTTP   (--http)   for network-accessible MCP clients

When run without arguments, starts in stdio mode.`,
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
