package ts

import (
	"net/http"
	"testing"

	"github.com/verdverm/gmd/pkg/testutil"
)

func TestReplayDemo(t *testing.T) {
	tape, err := testutil.NewReplayTape("testdata/replay_demo.json")
	if err != nil {
		t.Fatal(err)
	}
	tape.Start()
	defer func() {
		if err := tape.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	client := New(Config{
		Host:       "http://unused",
		APIKey:     "test-key",
		HTTPClient: &http.Client{Transport: tape.Transport()},
	})

	count, err := client.CountByPath(t.Context(), "chunk-crud.md")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}

	_, err = client.CountByPath(t.Context(), "another.md")
	if err == nil {
		t.Fatal("expected tape exhausted error")
	}
}
