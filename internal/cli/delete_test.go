package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDelete_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := RunDelete(nil, nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for no args")
	}
	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("error = %q, want usage hint", err)
	}
}

func TestDeleteConfirm_Yes(t *testing.T) {
	stdin := strings.NewReader("y\n")
	var stdout bytes.Buffer
	ok, err := deleteConfirm(stdin, &stdout)
	if err != nil {
		t.Fatalf("deleteConfirm: %v", err)
	}
	if !ok {
		t.Error("expected ok=true for 'y' input")
	}
}

func TestDeleteConfirm_YesFull(t *testing.T) {
	stdin := strings.NewReader("yes\n")
	var stdout bytes.Buffer
	ok, err := deleteConfirm(stdin, &stdout)
	if err != nil {
		t.Fatalf("deleteConfirm: %v", err)
	}
	if !ok {
		t.Error("expected ok=true for 'yes' input")
	}
}

func TestDeleteConfirm_No(t *testing.T) {
	stdin := strings.NewReader("n\n")
	var stdout bytes.Buffer
	ok, err := deleteConfirm(stdin, &stdout)
	if err != nil {
		t.Fatalf("deleteConfirm: %v", err)
	}
	if ok {
		t.Error("expected ok=false for 'n' input")
	}
}

func TestDeleteConfirm_Empty(t *testing.T) {
	stdin := strings.NewReader("\n")
	var stdout bytes.Buffer
	ok, err := deleteConfirm(stdin, &stdout)
	if err != nil {
		t.Fatalf("deleteConfirm: %v", err)
	}
	if ok {
		t.Error("expected ok=false for empty input")
	}
}

func TestRunDelete_FlagParsing(t *testing.T) {
	// Test that --yes flag is parsed correctly; the rest requires a server.
	var stdout, stderr bytes.Buffer
	err := RunDelete([]string{"--yes", "some-id"}, nil, &stdout, &stderr)
	// Will fail on server connection, but should not fail on flag parsing.
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "usage") {
		t.Errorf("should not return usage error with id arg: %v", err)
	}
}

func TestRunDelete_YesSkipsConfirm(t *testing.T) {
	// With --yes, the command should attempt DELETE directly without GET.
	// We verify the flag is accepted; server connection will fail.
	var stdout, stderr bytes.Buffer
	err := RunDelete([]string{"test-id", "--yes"}, nil, &stdout, &stderr)
	if err == nil {
		return
	}
	// Should not be a usage/confirm error — just server connection.
	if strings.Contains(err.Error(), "usage") {
		t.Errorf("should not return usage error: %v", err)
	}
}
