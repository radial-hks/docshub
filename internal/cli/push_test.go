package cli

import (
	"flag"
	"reflect"
	"strings"
	"testing"
)

func TestParseFrontmatter_TagsArray(t *testing.T) {
	content := `---
title: Article Title
category: AI
tags: [llm, rag, workflow]
author: radial
---
# Body

Hello.
`
	fm, body := parseFrontmatter(content)
	if fm.Title != "Article Title" {
		t.Errorf("Title = %q", fm.Title)
	}
	if fm.Category != "AI" {
		t.Errorf("Category = %q", fm.Category)
	}
	if fm.Author != "radial" {
		t.Errorf("Author = %q", fm.Author)
	}
	wantTags := []string{"llm", "rag", "workflow"}
	if !reflect.DeepEqual(fm.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", fm.Tags, wantTags)
	}
	if !strings.HasPrefix(body, "# Body") {
		t.Errorf("body should start with `# Body`, got %q", body)
	}
}

func TestParseFrontmatter_TagsCommaString(t *testing.T) {
	content := `---
title: Two
tags: a, b , c
---
body
`
	fm, _ := parseFrontmatter(content)
	wantTags := []string{"a", "b", "c"}
	if !reflect.DeepEqual(fm.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", fm.Tags, wantTags)
	}
}

func TestParseFrontmatter_None(t *testing.T) {
	content := "# Just a header\n\nbody text"
	fm, body := parseFrontmatter(content)
	if fm.Title != "" || fm.Category != "" || fm.Author != "" || len(fm.Tags) != 0 {
		t.Errorf("expected zero Frontmatter, got %+v", fm)
	}
	if body != content {
		t.Errorf("body should equal original content")
	}
}

func TestBuildPublishRequest_NoFrontmatterDefaults(t *testing.T) {
	cfg := &Config{Author: "default-author"}
	req, err := buildPublishRequest("/tmp/my-note.md", "hello world", PushOptions{}, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Title != "my-note" {
		t.Errorf("Title = %q, want filename fallback %q", req.Title, "my-note")
	}
	if req.Author != "default-author" {
		t.Errorf("Author = %q, want from cfg", req.Author)
	}
	if req.Category != "" {
		t.Errorf("Category = %q, want empty", req.Category)
	}
}

func TestBuildPublishRequest_FlagsOverrideFrontmatter(t *testing.T) {
	content := `---
title: FM Title
category: FM-Cat
tags: [a, b]
author: fm-author
---
body
`
	cfg := &Config{Author: "cfg-author"}
	opts := PushOptions{Category: "Override", Tags: "x,y"}
	req, err := buildPublishRequest("/tmp/note.md", content, opts, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Title != "FM Title" {
		t.Errorf("Title = %q, want frontmatter value", req.Title)
	}
	if req.Category != "Override" {
		t.Errorf("Category = %q, want CLI override", req.Category)
	}
	wantTags := []string{"x", "y"}
	if !reflect.DeepEqual(req.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", req.Tags, wantTags)
	}
	if req.Author != "fm-author" {
		t.Errorf("Author = %q, want frontmatter value (fm beats cfg)", req.Author)
	}
}

func TestBuildPublishRequest_ClassifyJSONWins(t *testing.T) {
	content := `---
title: FM Title
category: FM-Cat
tags: [a, b]
---
body
`
	cfg := &Config{Author: "cfg-author"}
	opts := PushOptions{
		Category:     "FlagCat",
		Tags:         "f1,f2",
		ClassifyJSON: `{"title":"AI Title","category":"AI","tags":["llm","rag"],"author":"ai"}`,
	}
	req, err := buildPublishRequest("/tmp/note.md", content, opts, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Title != "AI Title" {
		t.Errorf("Title = %q, want classify override", req.Title)
	}
	if req.Category != "AI" {
		t.Errorf("Category = %q, want classify override", req.Category)
	}
	if req.Author != "ai" {
		t.Errorf("Author = %q, want classify override", req.Author)
	}
	wantTags := []string{"llm", "rag"}
	if !reflect.DeepEqual(req.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", req.Tags, wantTags)
	}
}

func TestBuildPublishRequest_FormatAutoDetectMD(t *testing.T) {
	cfg := &Config{Author: "default"}
	req, err := buildPublishRequest("/tmp/note.md", "hello", PushOptions{}, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Format != "md" {
		t.Errorf("Format = %q, want md for .md file", req.Format)
	}
}

func TestBuildPublishRequest_FormatAutoDetectHTML(t *testing.T) {
	cfg := &Config{Author: "default"}
	req, err := buildPublishRequest("/tmp/report.html", "<h1>Hi</h1>", PushOptions{}, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Format != "html" {
		t.Errorf("Format = %q, want html for .html file", req.Format)
	}
}

func TestBuildPublishRequest_FormatFlagOverrides(t *testing.T) {
	cfg := &Config{Author: "default"}
	opts := PushOptions{Format: "html"}
	req, err := buildPublishRequest("/tmp/note.md", "hello", opts, cfg)
	if err != nil {
		t.Fatalf("buildPublishRequest: %v", err)
	}
	if req.Format != "html" {
		t.Errorf("Format = %q, want html from --format flag override", req.Format)
	}
}

func TestReorderFlags_FileBeforeFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cat := fs.String("category", "", "")
	tags := fs.String("tags", "", "")
	yes := fs.Bool("yes", false, "")
	args := []string{"file.md", "--category", "Note", "--tags", "a,b", "--yes"}
	reordered := reorderFlags(fs, args)
	if err := fs.Parse(reordered); err != nil {
		t.Fatalf("parse reordered: %v", err)
	}
	if *cat != "Note" {
		t.Errorf("category = %q, want Note", *cat)
	}
	if *tags != "a,b" {
		t.Errorf("tags = %q, want a,b", *tags)
	}
	if !*yes {
		t.Errorf("yes = false, want true")
	}
	if len(fs.Args()) != 1 || fs.Args()[0] != "file.md" {
		t.Errorf("positional args = %v, want [file.md]", fs.Args())
	}
}

func TestReorderFlags_EqualsForm(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cat := fs.String("category", "", "")
	args := []string{"file.md", "--category=AI"}
	reordered := reorderFlags(fs, args)
	if err := fs.Parse(reordered); err != nil {
		t.Fatalf("parse reordered: %v", err)
	}
	if *cat != "AI" {
		t.Errorf("category = %q, want AI", *cat)
	}
}
