package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	serveHost string
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long: `Starts an HTTP server exposing all GMD operations as REST endpoints.

Example:
  gmd serve                     # http://localhost:8181
  gmd serve --port 9000 --host 0.0.0.0

Endpoints:
  GET  /health           liveness check
  GET  /status           index and collection health
  POST /search           full-text search
  POST /vsearch          vector search
  POST /query            full hybrid pipeline
  GET  /documents/{path} get document by path
  GET  /collections      list collections
  POST /update           trigger reindex`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := getRuntime()
		if err != nil {
			return err
		}
		fmt.Printf("serve on %s:%d (not yet implemented, Phase 5)\n", serveHost, servePort)
		return nil
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveHost, "host", "", "localhost", "server host")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8181, "server port")
}
