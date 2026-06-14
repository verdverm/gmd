package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Exchange struct {
	Request struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	} `json:"request"`
	Response struct {
		StatusCode int               `json:"status_code"`
		Headers    map[string]string `json:"headers"`
		Body       string            `json:"body"`
	} `json:"response"`
}

type Mode int

const (
	ModeRecord Mode = iota
	ModeReplay
)

var defaultStripHeaders = map[string]bool{
	"authorization":       true,
	"x-typesense-api-key": true,
	"x-api-key":           true,
	"x-auth-key":          true,
	"cookie":              true,
	"set-cookie":          true,
}

func normalizeStripHeaders(custom []string) map[string]bool {
	if len(custom) == 0 {
		return defaultStripHeaders
	}
	stripped := make(map[string]bool, len(defaultStripHeaders)+len(custom))
	for k, v := range defaultStripHeaders {
		stripped[k] = v
	}
	for _, h := range custom {
		stripped[strings.ToLower(h)] = true
	}
	return stripped
}

func shouldStripHeader(stripped map[string]bool, name string) bool {
	return stripped[strings.ToLower(name)]
}

type Tape struct {
	mu        sync.Mutex
	mode      Mode
	filePath  string
	upstream  *url.URL
	parent    http.RoundTripper
	exchanges []Exchange
	pos       int
	recording bool
	stripSet  map[string]bool
}

func NewTape(filePath string, upstreamURL string, parent http.RoundTripper, mode Mode, stripHeaders ...string) *Tape {
	u, _ := url.Parse(upstreamURL)
	return &Tape{
		mode:     mode,
		filePath: filePath,
		upstream: u,
		parent:   parent,
		stripSet: normalizeStripHeaders(stripHeaders),
	}
}

func NewReplayTape(filePath string) (*Tape, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var exchanges []Exchange
	if err := json.Unmarshal(data, &exchanges); err != nil {
		return nil, fmt.Errorf("invalid tape JSON in %q: %w", filePath, err)
	}
	if exchanges == nil {
		return nil, fmt.Errorf("invalid tape JSON in %q: root is not an array", filePath)
	}
	return &Tape{
		mode:      ModeReplay,
		filePath:  filePath,
		exchanges: exchanges,
		stripSet:  defaultStripHeaders,
	}, nil
}

func (t *Tape) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.recording = true
	t.pos = 0
}

func (t *Tape) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.recording = false
	if t.mode == ModeReplay {
		return nil
	}
	if t.mode == ModeRecord {
		dir := filepath.Dir(t.filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating testdata directory: %w", err)
		}
		data, err := json.MarshalIndent(t.exchanges, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling tape: %w", err)
		}
		if err := os.WriteFile(t.filePath, data, 0644); err != nil {
			return fmt.Errorf("writing tape file: %w", err)
		}
		t.exchanges = nil
	}
	return nil
}

func (t *Tape) Transport() http.RoundTripper {
	return t
}

func (t *Tape) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.mode == ModeReplay {
		return t.replayRoundTrip()
	}
	return t.recordRoundTrip(req)
}

func (t *Tape) recordRoundTrip(req *http.Request) (*http.Response, error) {
	if !t.recording {
		return t.getParent().RoundTrip(req)
	}

	reqBodyBytes, reqErr := readAndRestoreBody(req)

	resp, err := t.getParent().RoundTrip(req)
	if err != nil {
		return nil, err
	}

	respBodyBytes, respErr := readAndRestoreBody(resp)

	exchange := Exchange{}
	exchange.Request.Method = req.Method
	exchange.Request.URL = req.URL.String()
	exchange.Request.Headers = stripHeaders(t.stripSet, copyHeaders(req.Header))
	if reqErr == nil && len(reqBodyBytes) > 0 {
		exchange.Request.Body = string(reqBodyBytes)
	}

	exchange.Response.StatusCode = resp.StatusCode
	exchange.Response.Headers = stripHeaders(t.stripSet, copyHeaders(resp.Header))
	if respErr == nil && len(respBodyBytes) > 0 {
		exchange.Response.Body = string(respBodyBytes)
	}

	t.exchanges = append(t.exchanges, exchange)
	return resp, nil
}

func (t *Tape) replayRoundTrip() (*http.Response, error) {
	if t.pos >= len(t.exchanges) {
		return nil, fmt.Errorf("tape exhausted at position %d", t.pos)
	}
	exchange := t.exchanges[t.pos]
	t.pos++

	resp := &http.Response{
		StatusCode: exchange.Response.StatusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(exchange.Response.Body)),
	}
	for k, v := range exchange.Response.Headers {
		resp.Header.Set(k, v)
	}
	return resp, nil
}

func (t *Tape) getParent() http.RoundTripper {
	if t.parent != nil {
		return t.parent
	}
	return http.DefaultTransport
}

func readAndRestoreBody(reqOrResp interface{}) ([]byte, error) {
	var body io.ReadCloser
	switch r := reqOrResp.(type) {
	case *http.Request:
		if r.Body == nil {
			return nil, nil
		}
		body = r.Body
		defer func() { r.Body = body }()
	case *http.Response:
		if r.Body == nil {
			return nil, nil
		}
		body = r.Body
		defer func() { r.Body = body }()
	default:
		return nil, nil
	}

	data, err := io.ReadAll(body)
	body.Close()
	body = io.NopCloser(bytes.NewReader(data))
	return data, err
}

func copyHeaders(src http.Header) map[string]string {
	dst := make(map[string]string, len(src))
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		dst[k] = src.Get(k)
	}
	return dst
}

func stripHeaders(stripSet map[string]bool, headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if !shouldStripHeader(stripSet, k) {
			result[k] = v
		}
	}
	return result
}
