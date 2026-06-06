package web

import (
	"errors"
	"fmt"
)

var (
	ErrNotSupported          = errors.New("gmd/web: operation not supported by provider")
	ErrProviderNotFound      = errors.New("gmd/web: provider not found in registry")
	ErrProviderNotConfigured = errors.New("gmd/web: provider referenced by group but not configured")
	ErrBrowserNotAvailable   = errors.New("gmd/web: browser not available on this machine")
	ErrAuthMissing           = errors.New("gmd/web: required credentials not set — check env vars")
	ErrAuthFailed            = errors.New("gmd/web: authentication failed")
	ErrRateLimited           = errors.New("gmd/web: rate limited by provider")
	ErrTimeout               = errors.New("gmd/web: request timed out")
	ErrSSRFBlocked           = errors.New("gmd/web: request blocked — private/internal IP")
)

type ProviderError struct {
	Provider string
	Err      error
	Detail   string
}

func (e *ProviderError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("gmd/web: %s: %s: %v", e.Provider, e.Detail, e.Err)
	}
	return fmt.Sprintf("gmd/web: %s: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func WrapProviderError(provider, detail string, err error) error {
	if err == nil {
		return nil
	}
	return &ProviderError{
		Provider: provider,
		Err:      err,
		Detail:   detail,
	}
}
