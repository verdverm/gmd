//go:build integration

package wiki

import (
	"strings"
	"testing"

	"github.com/verdverm/gmd/pkg/llm"
)

func TestIntegrationFormatDoctorResult_FullyConnected(t *testing.T) {
	result := &DoctorResult{
		WikiName:    "test-wiki",
		TSConnected: true,
		LLMStatus: []llm.EndpointStatus{
			{Label: "embed", URL: "http://localhost:8000/v1", OK: true, Model: "nomic-embed-text-v1.5"},
			{Label: "chat", URL: "http://localhost:8001/v1", OK: true, Model: "llama-3.2-3b"},
		},
		Agents: []AgentStatus{
			{Name: "claude", Installed: true, SkillInst: true},
			{Name: "codex", Installed: false, SkillInst: false},
		},
	}
	output := FormatDoctorResult(result)
	if !strings.Contains(output, "test-wiki") {
		t.Error("expected wiki name in output")
	}
	if !strings.Contains(output, "connected") {
		t.Error("expected connected status")
	}
	if !strings.Contains(output, "embed") {
		t.Error("expected embed LLM label")
	}
	if !strings.Contains(output, "claude") {
		t.Error("expected claude agent status")
	}
	if !strings.Contains(output, "not detected") {
		t.Error("expected codex not detected")
	}
}

func TestIntegrationFormatDoctorResult_Disconnected(t *testing.T) {
	result := &DoctorResult{
		WikiName:    "test-wiki",
		TSConnected: false,
		Errors:      []string{"Typesense: connection refused"},
	}
	output := FormatDoctorResult(result)
	if !strings.Contains(output, "not connected") {
		t.Error("expected not connected status")
	}
	if !strings.Contains(output, "connection refused") {
		t.Error("expected error message in output")
	}
}

func TestIntegrationFormatDoctorResult_LLMErrors(t *testing.T) {
	result := &DoctorResult{
		WikiName:    "test-wiki",
		TSConnected: true,
		LLMStatus: []llm.EndpointStatus{
			{Label: "embed", URL: "http://localhost:8000/v1", OK: false, Model: "", Err: "connection timeout"},
		},
	}
	output := FormatDoctorResult(result)
	if !strings.Contains(output, "connection timeout") {
		t.Error("expected connection timeout in output")
	}
}

func TestIntegrationFormatDoctorResult_Empty(t *testing.T) {
	result := &DoctorResult{}
	output := FormatDoctorResult(result)
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestIntegrationDoctorFix(t *testing.T) {
	_, agent := newTestWikiAgent(t)
	fixes, err := DoctorFix(agent.wiki)
	if err != nil {
		t.Fatalf("DoctorFix error: %v", err)
	}
	// May return fixes if any agents are installed on the dev machine
	t.Logf("DoctorFix returned %d fixes", len(fixes))
	for _, f := range fixes {
		t.Logf("  fix: %s", f)
	}
}
