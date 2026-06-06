package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Method string

const (
	AuthNone           Method = "none"
	AuthAPIKey         Method = "apikey"
	AuthServiceAccount Method = "service-account"
)

type Config struct {
	Method          Method
	APIKey          string
	ProjectID       string
	Location        string
	CredentialsFile string
}

func BuildHTTPClient(cfg Config) (*http.Client, error) {
	switch cfg.Method {
	case AuthNone:
		return nil, nil
	case AuthAPIKey:
		return nil, nil
	case AuthServiceAccount:
		return gcpTokenClient(context.Background(), cfg.CredentialsFile)
	default:
		return nil, fmt.Errorf("auth: unknown method %q", cfg.Method)
	}
}

func gcpTokenClient(ctx context.Context, credentialsFile string) (*http.Client, error) {
	var tokenSource oauth2.TokenSource
	if credentialsFile != "" {
		data, err := os.ReadFile(credentialsFile)
		if err != nil {
			return nil, fmt.Errorf("auth: reading credentials file: %w", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("auth: parsing credentials: %w", err)
		}
		tokenSource = creds.TokenSource
	} else {
		var err error
		tokenSource, err = google.DefaultTokenSource(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("auth: getting default token source: %w", err)
		}
	}
	return oauth2.NewClient(ctx, tokenSource), nil
}

func DefaultBaseURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "vertex":
		return ""
	case "opencode":
		return ""
	default:
		return ""
	}
}
