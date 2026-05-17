package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/radial-hks/docshub/internal/model"
)

func TestFormatList(t *testing.T) {
	articles := []model.Article{
		{
			ID:       "intro-20260517",
			Title:    "Intro to docshub",
			Category: "AI",
			Tags:     []string{"llm", "rag"},
			Author:   "radial",
			Date:     time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:    "draft-20260101",
			Title: "Draft Notes",
			Date:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	var buf bytes.Buffer
	formatList(&buf, articles)
	out := buf.String()

	wants := []string{
		"ID", "TITLE", "CATEGORY", "TAGS", "AUTHOR", "DATE",
		"intro-20260517", "Intro to docshub", "AI", "llm,rag", "radial", "2026-05-17",
		"draft-20260101", "Draft Notes", "2026-01-01",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("output missing %q\n--- output ---\n%s", w, out)
		}
	}
	// The second article has no category/tags/author — those columns should show '-'.
	if !strings.Contains(out, "-") {
		t.Errorf("expected `-` placeholder for missing fields, got:\n%s", out)
	}
}

func TestFormatListEmpty(t *testing.T) {
	var buf bytes.Buffer
	formatList(&buf, nil)
	out := buf.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "TITLE") {
		t.Errorf("header missing from empty output:\n%s", out)
	}
}
