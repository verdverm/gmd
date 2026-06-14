package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordThenReplayRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"echo": "%s"}`, string(body))
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	reqBody := `{"message":"hello"}`
	req, _ := http.NewRequestWithContext(t.Context(), "POST", srv.URL+"/test", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-key")

	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(respBody), "hello") {
		t.Fatalf("unexpected response: %s", string(respBody))
	}

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	replay, err := NewReplayTape(tapeFile)
	if err != nil {
		t.Fatalf("NewReplayTape failed: %v", err)
	}
	replay.Start()

	req2, _ := http.NewRequestWithContext(t.Context(), "POST", "http://unused/test", strings.NewReader("ignored"))
	resp2, err := replay.Transport().RoundTrip(req2)
	if err != nil {
		t.Fatalf("replay RoundTrip failed: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	respBody2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if !strings.Contains(string(respBody2), "hello") {
		t.Fatalf("unexpected replay response: %s", string(respBody2))
	}
}

func TestHeaderStripping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("x-api-key", "test-key")
	req.Header.Set("X-TYPESENSE-API-KEY", "ts-key")
	req.Header.Set("X-Auth-Key", "auth-key")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("Accept", "application/json")

	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	data, err := os.ReadFile(tapeFile)
	if err != nil {
		t.Fatalf("reading tape file: %v", err)
	}
	var exchanges []Exchange
	if err := json.Unmarshal(data, &exchanges); err != nil {
		t.Fatalf("unmarshaling tape: %v", err)
	}
	if len(exchanges) != 1 {
		t.Fatalf("expected 1 exchange, got %d", len(exchanges))
	}

	h := exchanges[0].Request.Headers
	if _, ok := h["Authorization"]; ok {
		t.Error("Authorization should be stripped")
	}
	if _, ok := h["X-Api-Key"]; ok {
		t.Error("X-Api-Key should be stripped")
	}
	if _, ok := h["X-Typesense-Api-Key"]; ok {
		t.Error("X-Typesense-Api-Key should be stripped")
	}
	if _, ok := h["X-Auth-Key"]; ok {
		t.Error("X-Auth-Key should be stripped")
	}
	if _, ok := h["Cookie"]; ok {
		t.Error("Cookie should be stripped")
	}
	if _, ok := h["Accept"]; !ok {
		t.Error("Accept should not be stripped")
	}
}

func TestResponseHeaderStripping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=xyz")
		w.Header().Set("X-Typesense-Api-Key", "echo-key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	data, err := os.ReadFile(tapeFile)
	if err != nil {
		t.Fatalf("reading tape file: %v", err)
	}
	var exchanges []Exchange
	if err := json.Unmarshal(data, &exchanges); err != nil {
		t.Fatalf("unmarshaling tape: %v", err)
	}

	h := exchanges[0].Response.Headers
	if _, ok := h["Set-Cookie"]; ok {
		t.Error("Set-Cookie should be stripped from response")
	}
	if _, ok := h["X-Typesense-Api-Key"]; ok {
		t.Error("X-Typesense-Api-Key should be stripped from response")
	}
	if _, ok := h["Content-Type"]; !ok {
		t.Error("Content-Type should not be stripped from response")
	}
}

func TestParentDirectoryCreation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	tapeFile := filepath.Join(tmpDir, "subdir", "nested", "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if _, err := os.Stat(tapeFile); os.IsNotExist(err) {
		t.Fatal("tape file was not created")
	}
}

func TestTapeExhaustion(t *testing.T) {
	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	exchanges := []Exchange{{}}
	exchanges[0].Request.Method = "GET"
	exchanges[0].Response.StatusCode = 200
	exchanges[0].Response.Body = "ok"
	data, _ := json.MarshalIndent(exchanges, "", "  ")
	if err := os.WriteFile(tapeFile, data, 0644); err != nil {
		t.Fatalf("writing tape file: %v", err)
	}

	tape, err := NewReplayTape(tapeFile)
	if err != nil {
		t.Fatalf("NewReplayTape failed: %v", err)
	}
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", "http://unused/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}
	resp.Body.Close()

	resp, err = tape.Transport().RoundTrip(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected tape exhausted error")
	}
	if !strings.Contains(err.Error(), "tape exhausted") {
		t.Fatalf("expected 'tape exhausted' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "position 1") {
		t.Fatalf("expected 'position 1' in error, got: %v", err)
	}
}

func TestEmptyTape(t *testing.T) {
	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	data, _ := json.MarshalIndent([]Exchange{}, "", "  ")
	if err := os.WriteFile(tapeFile, data, 0644); err != nil {
		t.Fatalf("writing tape file: %v", err)
	}

	tape, err := NewReplayTape(tapeFile)
	if err != nil {
		t.Fatalf("NewReplayTape failed: %v", err)
	}
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", "http://unused/test", nil)
	resp, rerr := tape.Transport().RoundTrip(req)
	if rerr == nil {
		resp.Body.Close()
		t.Fatal("expected tape exhausted error for empty tape")
	}
}

func TestStartStopGate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/before", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("pre-start RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	tape.Start()

	req2, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/during", nil)
	resp2, err := tape.Transport().RoundTrip(req2)
	if err != nil {
		t.Fatalf("post-start RoundTrip failed: %v", err)
	}
	resp2.Body.Close()

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	data, err := os.ReadFile(tapeFile)
	if err != nil {
		t.Fatalf("reading tape file: %v", err)
	}
	var exchanges []Exchange
	if err := json.Unmarshal(data, &exchanges); err != nil {
		t.Fatalf("unmarshaling tape: %v", err)
	}

	if len(exchanges) != 1 {
		t.Fatalf("expected 1 exchange, got %d", len(exchanges))
	}
	if !strings.Contains(exchanges[0].Request.URL, "/during") {
		t.Error("recorded request should be /during")
	}
}

func TestResponseBodyReReadability(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "response body content")
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if !strings.Contains(string(body), "response body content") {
		t.Fatalf("unexpected body: %s", string(body))
	}

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	replay, err := NewReplayTape(tapeFile)
	if err != nil {
		t.Fatalf("NewReplayTape failed: %v", err)
	}
	replay.Start()

	req2, _ := http.NewRequestWithContext(t.Context(), "GET", "http://unused/test", nil)
	resp2, err := replay.Transport().RoundTrip(req2)
	if err != nil {
		t.Fatalf("replay RoundTrip failed: %v", err)
	}

	body2, err := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		t.Fatalf("reading replay body failed: %v", err)
	}
	if string(body2) != "response body content" {
		t.Fatalf("unexpected replay body: %s", string(body2))
	}
}

func TestLargeResponseBodyRoundTrip(t *testing.T) {
	largeBody := make([]byte, 1*1024*1024)
	for i := range largeBody {
		largeBody[i] = byte('a' + (i % 26))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(largeBody)
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("reading body failed: %v", err)
	}
	if len(body) != 1*1024*1024 {
		t.Fatalf("expected %d bytes, got %d", 1*1024*1024, len(body))
	}

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	replay, err := NewReplayTape(tapeFile)
	if err != nil {
		t.Fatalf("NewReplayTape failed: %v", err)
	}
	replay.Start()

	req2, _ := http.NewRequestWithContext(t.Context(), "GET", "http://unused/test", nil)
	resp2, err := replay.Transport().RoundTrip(req2)
	if err != nil {
		t.Fatalf("replay RoundTrip failed: %v", err)
	}

	body2, err := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if err != nil {
		t.Fatalf("reading replay body failed: %v", err)
	}
	if !bytes.Equal(body2, largeBody) {
		t.Fatal("replay body does not match original")
	}
}

func TestInvalidTapeJSON(t *testing.T) {
	tapeFile := filepath.Join(t.TempDir(), "tape.json")

	if err := os.WriteFile(tapeFile, []byte(`not json`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err := NewReplayTape(tapeFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	if err := os.WriteFile(tapeFile, []byte(`"not array"`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err = NewReplayTape(tapeFile)
	if err == nil {
		t.Fatal("expected error for non-array root")
	}

	if err := os.WriteFile(tapeFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	_, err = NewReplayTape(tapeFile)
	if err == nil {
		t.Fatal("expected error for object root")
	}
}

func TestFileWriteFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	readOnlyDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnlyDir, 0755) })

	tapeFile := filepath.Join(readOnlyDir, "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord)
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	err = tape.Stop()
	if err == nil {
		t.Fatal("expected error writing to read-only directory")
	}
}

func TestNonexistentTapeFile(t *testing.T) {
	tapeFile := filepath.Join(t.TempDir(), "nonexistent.json")
	_, err := NewReplayTape(tapeFile)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCustomStripHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	tapeFile := filepath.Join(t.TempDir(), "tape.json")
	tape := NewTape(tapeFile, srv.URL, nil, ModeRecord, "X-Custom-Secret")
	tape.Start()

	req, _ := http.NewRequestWithContext(t.Context(), "GET", srv.URL+"/test", nil)
	req.Header.Set("X-Custom-Secret", "secret-value")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	resp, err := tape.Transport().RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
	resp.Body.Close()

	if err := tape.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	data, err := os.ReadFile(tapeFile)
	if err != nil {
		t.Fatalf("reading tape file: %v", err)
	}
	var exchanges []Exchange
	if err := json.Unmarshal(data, &exchanges); err != nil {
		t.Fatalf("unmarshaling tape: %v", err)
	}

	h := exchanges[0].Request.Headers
	if _, ok := h["X-Custom-Secret"]; ok {
		t.Error("X-Custom-Secret should be stripped")
	}
	if _, ok := h["Authorization"]; ok {
		t.Error("Authorization should be stripped (default)")
	}
	if _, ok := h["Accept"]; !ok {
		t.Error("Accept should not be stripped")
	}
}
