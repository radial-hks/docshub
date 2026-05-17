package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/radial-hks/docshub/internal/model"
)

func TestGenerateSidebarEmpty(t *testing.T) {
	got := GenerateSidebar(nil)
	want := "- [Home](/)\n"
	if got != want {
		t.Fatalf("empty sidebar mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestGenerateSidebarSingleArticle(t *testing.T) {
	articles := []model.Article{
		{
			Title:    "Agent Patterns",
			File:     "/articles/AI/agent-patterns.md",
			Category: "AI",
			Date:     time.Now(),
		},
	}
	got := GenerateSidebar(articles)
	if !strings.Contains(got, "- [Home](/)") {
		t.Errorf("missing home link in:\n%s", got)
	}
	if !strings.Contains(got, "- AI\n") {
		t.Errorf("missing AI category heading in:\n%s", got)
	}
	if !strings.Contains(got, "  - [Agent Patterns](/articles/AI/agent-patterns.md)") {
		t.Errorf("missing article link in:\n%s", got)
	}
}

func TestGenerateSidebarMultipleCategoriesSorted(t *testing.T) {
	now := time.Now()
	articles := []model.Article{
		{Title: "Voxel", File: "/articles/UE/voxel.md", Category: "UE", Date: now},
		{Title: "Agent", File: "/articles/AI/agent.md", Category: "AI", Date: now},
		{Title: "Build", File: "/articles/Dev/build.md", Category: "Dev", Date: now},
	}
	got := GenerateSidebar(articles)
	aiIdx := strings.Index(got, "- AI\n")
	devIdx := strings.Index(got, "- Dev\n")
	ueIdx := strings.Index(got, "- UE\n")
	if aiIdx == -1 || devIdx == -1 || ueIdx == -1 {
		t.Fatalf("missing categories in:\n%s", got)
	}
	if !(aiIdx < devIdx && devIdx < ueIdx) {
		t.Fatalf("categories not sorted alphabetically:\n%s", got)
	}
}

func TestGenerateSidebarSameCategorySortedByDateDesc(t *testing.T) {
	older := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	middle := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	articles := []model.Article{
		{Title: "Older", File: "/articles/AI/older.md", Category: "AI", Date: older},
		{Title: "Newer", File: "/articles/AI/newer.md", Category: "AI", Date: newer},
		{Title: "Middle", File: "/articles/AI/middle.md", Category: "AI", Date: middle},
	}
	got := GenerateSidebar(articles)
	newerIdx := strings.Index(got, "[Newer]")
	middleIdx := strings.Index(got, "[Middle]")
	olderIdx := strings.Index(got, "[Older]")
	if newerIdx == -1 || middleIdx == -1 || olderIdx == -1 {
		t.Fatalf("missing entries in:\n%s", got)
	}
	if !(newerIdx < middleIdx && middleIdx < olderIdx) {
		t.Fatalf("articles not sorted by date descending:\n%s", got)
	}
}

func TestGenerateSidebarUnclassified(t *testing.T) {
	articles := []model.Article{
		{
			Title: "Draft Notes",
			File:  "/articles/_unclassified/draft-notes.md",
			Date:  time.Now(),
		},
	}
	got := GenerateSidebar(articles)
	if !strings.Contains(got, "- Unclassified\n") {
		t.Errorf("missing Unclassified heading in:\n%s", got)
	}
	if !strings.Contains(got, "  - [Draft Notes](/articles/_unclassified/draft-notes.md)") {
		t.Errorf("missing draft link in:\n%s", got)
	}
}

func TestGenerateSidebarMixClassifiedUnclassified(t *testing.T) {
	now := time.Now()
	articles := []model.Article{
		{Title: "Classified", File: "/articles/AI/c.md", Category: "AI", Date: now},
		{Title: "Loose Note", File: "/articles/_unclassified/u.md", Date: now},
	}
	got := GenerateSidebar(articles)
	aiIdx := strings.Index(got, "- AI\n")
	uIdx := strings.Index(got, "- Unclassified\n")
	if aiIdx == -1 || uIdx == -1 {
		t.Fatalf("missing categories in:\n%s", got)
	}
	if aiIdx > uIdx {
		t.Fatalf("AI should appear before Unclassified (alphabetical):\n%s", got)
	}
	if !strings.Contains(got, "  - [Classified](/articles/AI/c.md)") {
		t.Errorf("missing classified link in:\n%s", got)
	}
	if !strings.Contains(got, "  - [Loose Note](/articles/_unclassified/u.md)") {
		t.Errorf("missing unclassified link in:\n%s", got)
	}
}

func TestRefreshSidebarOnCreate(t *testing.T) {
	s, dir := newTestStore(t)

	if _, err := s.Create(model.PublishRequest{
		Title:    "Patterns",
		Content:  "body",
		Category: "AI",
	}); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "_sidebar.md"))
	if err != nil {
		t.Fatalf("_sidebar.md missing: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "- [Home](/)") {
		t.Errorf("missing home link:\n%s", content)
	}
	if !strings.Contains(content, "- AI\n") {
		t.Errorf("missing AI category:\n%s", content)
	}
	if !strings.Contains(content, "[Patterns]") {
		t.Errorf("missing article link:\n%s", content)
	}
}

func TestRefreshSidebarOnDelete(t *testing.T) {
	s, dir := newTestStore(t)

	resp, err := s.Create(model.PublishRequest{
		Title:    "Goner",
		Content:  "x",
		Category: "AI",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := s.Delete(resp.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "_sidebar.md"))
	if err != nil {
		t.Fatalf("_sidebar.md missing: %v", err)
	}
	if strings.Contains(string(data), "Goner") {
		t.Errorf("deleted article still in sidebar:\n%s", string(data))
	}
}

func TestRefreshSidebarOnNew(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir); err != nil {
		t.Fatalf("New failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "_sidebar.md"))
	if err != nil {
		t.Fatalf("_sidebar.md missing after New: %v", err)
	}
	if string(data) != "- [Home](/)\n" {
		t.Fatalf("unexpected initial sidebar: %q", string(data))
	}
}

func TestGenerateSidebarHTMLArticle(t *testing.T) {
	articles := []model.Article{
		{
			Title:    "HTML Report",
			File:     "/articles/Reports/html-report.html",
			Category: "Reports",
			Format:   "html",
			Date:     time.Now(),
		},
	}
	got := GenerateSidebar(articles)
	if !strings.Contains(got, "- Reports\n") {
		t.Errorf("missing Reports category heading in:\n%s", got)
	}
	if !strings.Contains(got, "  - [HTML Report](/html/Reports/html-report)") {
		t.Errorf("missing HTML article link in:\n%s", got)
	}
	// Should NOT use the raw file path.
	if strings.Contains(got, "/articles/Reports/html-report.html") {
		t.Errorf("HTML article should use /html/ route, not raw file path:\n%s", got)
	}
}

func TestGenerateSidebarHTMLUnclassified(t *testing.T) {
	articles := []model.Article{
		{
			Title:  "Standalone HTML",
			File:   "/articles/_unclassified/standalone-html.html",
			Format: "html",
			Date:   time.Now(),
		},
	}
	got := GenerateSidebar(articles)
	if !strings.Contains(got, "  - [Standalone HTML](/html/_unclassified/standalone-html)") {
		t.Errorf("missing HTML unclassified link in:\n%s", got)
	}
}
