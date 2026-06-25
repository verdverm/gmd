package llm

import "errors"

var (
	// ErrProviderNotConfigured is returned when a provider is not found in config.
	ErrProviderNotConfigured = errors.New("llm: provider not configured")

	// ErrRoleUnset is returned when a role is not set in the active profile.
	ErrRoleUnset = errors.New("llm: role not set in profile")

	// ErrModelNotFound is returned when a model is not found on the provider.
	ErrModelNotFound = errors.New("llm: model not found on provider")

	// ErrNoChoicesInResponse is returned when the OpenAI API returns no choices.
	ErrNoChoicesInResponse = errors.New("no choices in OpenAI response")
)
