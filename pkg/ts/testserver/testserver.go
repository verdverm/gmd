package testserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultPort   = "18108"
	DefaultAPIKey = "test-integration-key-12345"
	DefaultImage  = "typesense/typesense:30.2"
)

type Server struct {
	ContainerID string
	Port        string
	APIKey      string
	DataDir     string
	Image       string
}

type Options struct {
	Port    string
	APIKey  string
	Image   string
	DataDir string
}

const containerName = "gmd-ts-integration"

func Start(ctx context.Context, opts Options) (*Server, error) {
	port := opts.Port
	if port == "" {
		port = DefaultPort
	}
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	image := opts.Image
	if image == "" {
		image = DefaultImage
	}

	// Clean up any leftover container from a previous interrupted run.
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", containerName).Run()

	var dataDir string
	ownTempDir := false
	if opts.DataDir != "" {
		dataDir = opts.DataDir
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
		}
	} else {
		tmpDir, err := os.MkdirTemp("", "typesense-test-*")
		if err != nil {
			return nil, fmt.Errorf("create temp data dir: %w", err)
		}
		dataDir = tmpDir
		ownTempDir = true
	}
	defer func() {
		if ownTempDir && dataDir != "" {
			os.RemoveAll(dataDir)
		}
	}()

	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, fmt.Errorf("abs data dir: %w", err)
	}
	volMount := fmt.Sprintf("%s:/data", absDataDir)

	//nolint:gosec // containerName, port, image are programmatically generated
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-d",
		"--name", containerName,
		"-p", port+":8108",
		"-v", volMount,
		image,
		"--data-dir", "/data",
		"--api-key", apiKey,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run: %w\n%s", err, string(out))
	}
	containerID := strings.TrimSpace(string(out))

	srvDataDir := dataDir
	dataDir = "" // owned by Server now; defer will no-op

	s := &Server{
		ContainerID: containerID,
		Port:        port,
		APIKey:      apiKey,
		DataDir:     srvDataDir,
		Image:       image,
	}
	return s, nil
}

func (s *Server) URL() string {
	return fmt.Sprintf("http://localhost:%s", s.Port)
}

func (s *Server) Stop(ctx context.Context) error {
	//nolint:gosec // ContainerID is programmatically generated
	err := exec.CommandContext(ctx, "docker", "kill", s.ContainerID).Run()
	if err != nil {
		return fmt.Errorf("docker kill %s: %w", s.ContainerID, err)
	}
	if strings.HasPrefix(s.DataDir, os.TempDir()) {
		os.RemoveAll(s.DataDir)
	}
	return nil
}

func (s *Server) WaitForHealth(ctx context.Context, timeout time.Duration) error {
	healthURL := s.URL() + "/health"
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	err := s.logs(context.Background())
	if err != nil {
		return fmt.Errorf("health check timed out (log fetch: %v)", err)
	}
	return fmt.Errorf("health check timed out")
}

func (s *Server) logs(ctx context.Context) error {
	//nolint:gosec // ContainerID is programmatically generated
	out, err := exec.CommandContext(ctx, "docker", "logs", s.ContainerID).Output()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "--- typesense container logs ---\n%s\n--- end ---\n", string(out))
	return nil
}
