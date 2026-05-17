package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/radial-hks/docshub/internal/model"
)

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	return s, dir
}

func TestNewCreatesArticlesDir(t *testing.T) {
	_, dir := newTestStore(t)
	info, err := os.Stat(filepath.Join(dir, "articles"))
	if err != nil {
		t.Fatalf("articles dir missing: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("articles is not a directory")
	}
}

func TestCreateArticleNoCategory(t *testing.T) {
	s, dir := newTestStore(t)

	resp, err := s.Create(model.PublishRequest{
		Title:   "Hello World",
		Content: "# Hello World\n\nThis is the body.",
		Author:  "alice",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if resp.ID == "" {
		t.Fatalf("expected non-empty ID")
	}
	if resp.Version != 1 {
		t.Fatalf("expected version 1, got %d", resp.Version)
	}

	expectedPath := filepath.Join(dir, "articles", "_unclassified", "hello-world.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("article file missing at %s: %v", expectedPath, err)
	}
	if !strings.Contains(string(data), "Hello World") {
		t.Fatalf("article content mismatch: %q", string(data))
	}

	indexPath := filepath.Join(dir, "index.json")
	idxData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.json missing: %v", err)
	}
	var arts []model.Article
	if err := json.Unmarshal(idxData, &arts); err != nil {
		t.Fatalf("index.json invalid JSON: %v", err)
	}
	if len(arts) != 1 {
		t.Fatalf("expected 1 article in index, got %d", len(arts))
	}
	if arts[0].ID != resp.ID {
		t.Fatalf("index id mismatch: %s vs %s", arts[0].ID, resp.ID)
	}
}

func TestCreateArticleWithCategory(t *testing.T) {
	s, dir := newTestStore(t)

	resp, err := s.Create(model.PublishRequest{
		Title:    "Agent Patterns",
		Content:  "Body text.",
		Category: "AI",
		Tags:     []string{"llm", "agents"},
		Author:   "alice",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	expectedPath := filepath.Join(dir, "articles", "AI", "agent-patterns.md")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected file at %s: %v", expectedPath, err)
	}
	if resp.Path != "/articles/AI/agent-patterns.md" {
		t.Fatalf("unexpected response path: %s", resp.Path)
	}
}

func TestGetArticle(t *testing.T) {
	s, _ := newTestStore(t)

	resp, err := s.Create(model.PublishRequest{Title: "Foo", Content: "body"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := s.Get(resp.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Title != "Foo" {
		t.Fatalf("expected title Foo, got %s", got.Title)
	}

	if _, err := s.Get("nope"); err == nil {
		t.Fatalf("expected error for unknown id")
	}
}

func TestListAll(t *testing.T) {
	s, _ := newTestStore(t)

	_, _ = s.Create(model.PublishRequest{Title: "A", Content: "x", Category: "AI"})
	_, _ = s.Create(model.PublishRequest{Title: "B", Content: "y", Category: "UE"})
	_, _ = s.Create(model.PublishRequest{Title: "C", Content: "z"})

	all, err := s.List(ListFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 articles, got %d", len(all))
	}
}

func TestListByCategory(t *testing.T) {
	s, _ := newTestStore(t)

	_, _ = s.Create(model.PublishRequest{Title: "A", Content: "x", Category: "AI"})
	_, _ = s.Create(model.PublishRequest{Title: "B", Content: "y", Category: "UE"})
	_, _ = s.Create(model.PublishRequest{Title: "C", Content: "z", Category: "AI"})

	ai, err := s.List(ListFilter{Category: "AI"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ai) != 2 {
		t.Fatalf("expected 2 AI articles, got %d", len(ai))
	}
	for _, a := range ai {
		if a.Category != "AI" {
			t.Fatalf("unexpected category in filtered result: %s", a.Category)
		}
	}
}

func TestListByTag(t *testing.T) {
	s, _ := newTestStore(t)

	_, _ = s.Create(model.PublishRequest{Title: "A", Content: "x", Tags: []string{"llm", "rag"}})
	_, _ = s.Create(model.PublishRequest{Title: "B", Content: "y", Tags: []string{"ue", "render"}})
	_, _ = s.Create(model.PublishRequest{Title: "C", Content: "z", Tags: []string{"llm"}})

	llm, err := s.List(ListFilter{Tag: "llm"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(llm) != 2 {
		t.Fatalf("expected 2 llm articles, got %d", len(llm))
	}
}

func TestListByAuthor(t *testing.T) {
	s, _ := newTestStore(t)

	_, _ = s.Create(model.PublishRequest{Title: "A", Content: "x", Author: "alice"})
	_, _ = s.Create(model.PublishRequest{Title: "B", Content: "y", Author: "bob"})

	alice, err := s.List(ListFilter{Author: "alice"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(alice) != 1 || alice[0].Author != "alice" {
		t.Fatalf("expected single alice article, got %+v", alice)
	}
}

func TestListByQuery(t *testing.T) {
	s, _ := newTestStore(t)

	_, _ = s.Create(model.PublishRequest{Title: "LLM Patterns", Content: "Agent loops and tools."})
	_, _ = s.Create(model.PublishRequest{Title: "Voxel Render", Content: "Marching cubes."})
	_, _ = s.Create(model.PublishRequest{Title: "Random", Content: "About LLM and agents."})

	// Matches title.
	got, err := s.List(ListFilter{Query: "patterns"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 1 || got[0].Title != "LLM Patterns" {
		t.Fatalf("expected 1 title match, got %+v", got)
	}

	// Matches summary.
	got, err = s.List(ListFilter{Query: "marching"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 1 || got[0].Title != "Voxel Render" {
		t.Fatalf("expected 1 summary match, got %+v", got)
	}

	// Case-insensitive.
	got, err = s.List(ListFilter{Query: "LLM"})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matches for LLM, got %d", len(got))
	}
}

func TestDelete(t *testing.T) {
	s, dir := newTestStore(t)

	resp, err := s.Create(model.PublishRequest{Title: "Doomed", Content: "bye", Category: "Misc"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	filePath := filepath.Join(dir, "articles", "Misc", "doomed.md")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file missing before delete: %v", err)
	}

	if err := s.Delete(resp.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file should be gone, err=%v", err)
	}
	list, _ := s.List(ListFilter{})
	if len(list) != 0 {
		t.Fatalf("expected empty index, got %d", len(list))
	}

	if err := s.Delete("nope"); err == nil {
		t.Fatalf("expected error deleting unknown id")
	}
}

func TestVersioning(t *testing.T) {
	s, dir := newTestStore(t)

	orig, err := s.Create(model.PublishRequest{
		Title:    "Spec",
		Content:  "version one",
		Category: "Docs",
		Author:   "alice",
	})
	if err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if orig.Version != 1 {
		t.Fatalf("expected v1, got %d", orig.Version)
	}

	updated, err := s.Create(model.PublishRequest{
		Title:     "Spec",
		Content:   "version two",
		Category:  "Docs",
		Author:    "alice",
		VersionOf: orig.ID,
	})
	if err != nil {
		t.Fatalf("versioned Create failed: %v", err)
	}
	if updated.ID != orig.ID {
		t.Fatalf("expected same ID across versions, got %s vs %s", updated.ID, orig.ID)
	}
	if updated.Version != 2 {
		t.Fatalf("expected v2, got %d", updated.Version)
	}
	if len(updated.PreviousVersions) != 1 {
		t.Fatalf("expected 1 previous version, got %d", len(updated.PreviousVersions))
	}

	// New file holds new content.
	newPath := filepath.Join(dir, "articles", "Docs", "spec.md")
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new file missing: %v", err)
	}
	if string(data) != "version two" {
		t.Fatalf("new file has wrong content: %q", string(data))
	}

	// Archived version exists somewhere under .versions/spec/.
	versionsDir := filepath.Join(dir, "articles", "Docs", ".versions", "spec")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		t.Fatalf("versions dir missing: %v", err)
	}
	var foundV1 bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "v1-") && strings.HasSuffix(e.Name(), ".md") {
			foundV1 = true
			archivedData, err := os.ReadFile(filepath.Join(versionsDir, e.Name()))
			if err != nil {
				t.Fatalf("archived v1 unreadable: %v", err)
			}
			if string(archivedData) != "version one" {
				t.Fatalf("archived content mismatch: %q", string(archivedData))
			}
		}
	}
	if !foundV1 {
		t.Fatalf("v1 archive not found in %s; got %v", versionsDir, entries)
	}

	// meta.json records v1.
	metaPath := filepath.Join(versionsDir, "meta.json")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("meta.json missing: %v", err)
	}
	var meta versionMeta
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("meta.json invalid: %v", err)
	}
	if len(meta.Versions) != 1 || meta.Versions[0].Version != 1 {
		t.Fatalf("meta.json should record v1, got %+v", meta.Versions)
	}

	// Index should hold only the latest entry.
	all, _ := s.List(ListFilter{})
	if len(all) != 1 {
		t.Fatalf("expected 1 article in index, got %d", len(all))
	}
	if all[0].Version != 2 {
		t.Fatalf("indexed version should be 2, got %d", all[0].Version)
	}
}

func TestVersioningUnknownID(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.Create(model.PublishRequest{Title: "X", Content: "y", VersionOf: "missing"})
	if err == nil {
		t.Fatalf("expected error when VersionOf references unknown id")
	}
}

func TestGenerateIDDeterministicAndURLSafe(t *testing.T) {
	s := &Store{}
	id1 := s.generateID("Hello, World!")
	id2 := s.generateID("Hello, World!")
	if id1 != id2 {
		t.Fatalf("expected deterministic ID, got %s vs %s", id1, id2)
	}
	safe := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !safe.MatchString(id1) {
		t.Fatalf("ID is not URL-safe: %s", id1)
	}
	if !strings.HasPrefix(id1, "hello-world-") {
		t.Fatalf("expected slug prefix, got %s", id1)
	}
}

func TestSlugifySpecialChars(t *testing.T) {
	s := &Store{}
	cases := map[string]string{
		"Hello World":            "hello-world",
		"  Trim me!  ":           "trim-me",
		"AI & ML/NLP":            "ai-ml-nlp",
		"Already-Sluggy":         "already-sluggy",
		"~~~":                    "article",
	}
	for in, want := range cases {
		got := s.slugify(in)
		if got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExtractSummary(t *testing.T) {
	s := &Store{}

	content := "# Title\n\nThis is **bold** and *italic* and `code`. See [link](https://example.com) for more."
	got := s.extractSummary(content)
	want := "Title This is bold and italic and code. See link for more."
	if got != want {
		t.Fatalf("extractSummary mismatch:\n got: %q\nwant: %q", got, want)
	}

	// Truncation to 200 chars.
	long := strings.Repeat("a", 500)
	got = s.extractSummary(long)
	if len([]rune(got)) != 200 {
		t.Fatalf("expected summary length 200, got %d", len([]rune(got)))
	}

	// Code fences stripped.
	fenced := "Intro text.\n\n```go\nfunc x() {}\n```\n\nMore."
	got = s.extractSummary(fenced)
	if strings.Contains(got, "func x()") {
		t.Fatalf("expected fenced code stripped, got %q", got)
	}
	if !strings.Contains(got, "Intro text.") || !strings.Contains(got, "More.") {
		t.Fatalf("expected surrounding text preserved, got %q", got)
	}
}

func TestLoadIndexPersists(t *testing.T) {
	dir := t.TempDir()
	s1, err := New(dir)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	resp, err := s1.Create(model.PublishRequest{Title: "Persist", Content: "data"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	s2, err := New(dir)
	if err != nil {
		t.Fatalf("re-open New failed: %v", err)
	}
	got, err := s2.Get(resp.ID)
	if err != nil {
		t.Fatalf("Get after reopen failed: %v", err)
	}
	if got.Title != "Persist" {
		t.Fatalf("expected Persist, got %s", got.Title)
	}
}
