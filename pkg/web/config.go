package web

import "net/http"

type ProviderConfig struct {
	Name       string
	Extra      map[string]any
	HTTPClient *http.Client
}
