package exa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.exa.ai"
const defaultTimeout = 30 * time.Second
const maxRetries = 3

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	var resp SearchResponse
	err := c.do(ctx, "POST", "/search", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetContents(ctx context.Context, req ContentsRequest) (*ContentsResponse, error) {
	var resp ContentsResponse
	err := c.do(ctx, "POST", "/contents", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) FindSimilar(ctx context.Context, req FindSimilarRequest) (*FindSimilarResponse, error) {
	var resp FindSimilarResponse
	err := c.do(ctx, "POST", "/findSimilar", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Answer(ctx context.Context, req AnswerRequest) (*AnswerResponse, error) {
	var resp AnswerResponse
	err := c.do(ctx, "POST", "/answer", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = &APIError{
				StatusCode: resp.StatusCode,
				Message:    "rate limited",
				Body:       string(respBody),
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Body:       string(respBody),
			}
			if attempt == 0 {
				continue
			}
			return lastErr
		}

		if resp.StatusCode >= 400 {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    http.StatusText(resp.StatusCode),
				Body:       string(respBody),
			}
		}

		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
		return nil
	}

	return lastErr
}

type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("exa API error: %d %s — %s", e.StatusCode, e.Message, e.Body)
}

func IsRateLimit(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

func IsAuthError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden
	}
	return false
}
