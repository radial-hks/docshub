package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunSearch_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := RunSearch(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage hint", err)
	}
}

func TestRunSearch_QueryEncoding(t *testing.T) {
	// We can't easily test the full flow without a server, but we can verify
	// that the flag parsing works and the query arg is accepted.
	// A more thorough integration test would use httptest.
	var stdout, stderr bytes.Buffer
	err := RunSearch([]string{"hello world&special=yes"}, &stdout, &stderr)
	// This will fail because there's no server, but it should not fail on
	// flag parsing or query construction.
	if err == nil {
		// If somehow a server is running, that's fine too.
		return
	}
	// The error should be about connecting to the server, not about flags/query.
	if strings.Contains(err.Error(), "usage") {
		t.Errorf("should not return usage error with query arg: %v", err)
	}
}
