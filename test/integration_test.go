package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/radial-hks/docshub/internal/model"
	"github.com/radial-hks/docshub/internal/server"
)

func TestEndToEnd(t *testing.T) {
	dataDir := t.TempDir()

	htmlSrc, err := os.ReadFile(filepath.Join("..", "web", "index.html"))
	if err != nil {
		t.Fatalf("read web/index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "index.html"), htmlSrc, 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	store, err := server.New(dataDir)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	mdFile := filepath.Join(t.TempDir(), "sample.md")
	content := "# Hello World\n\nThis is a test article for the integration suite."
	if err := os.WriteFile(mdFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write sample md: %v", err)
	}
	mdContent, err := os.ReadFile(mdFile)
	if err != nil {
		t.Fatalf("read sample md: %v", err)
	}

	pubReq := model.PublishRequest{
		Title:    "Integration Test Article",
		Content:  string(mdContent),
		Category: "Test",
		Tags:     []string{"test", "integration"},
		Author:   "tester",
	}
	body, err := json.Marshal(pubReq)
	if err != nil {
		t.Fatalf("marshal publish request: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/articles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/articles: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d, body=%s", resp.StatusCode, buf)
	}

	var pub model.PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&pub); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	if pub.ID == "" {
		t.Errorf("PublishResponse.ID is empty")
	}
	if pub.Status != "ok" {
		t.Errorf("PublishResponse.Status = %q, want %q", pub.Status, "ok")
	}
	if pub.Version != 1 {
		t.Errorf("PublishResponse.Version = %d, want 1", pub.Version)
	}
	if !strings.HasPrefix(pub.Path, "/articles/Test/") || !strings.HasSuffix(pub.Path, ".md") {
		t.Errorf("PublishResponse.Path = %q, want /articles/Test/<slug>.md", pub.Path)
	}

	articleDiskPath := filepath.Join(dataDir, filepath.FromSlash(strings.TrimPrefix(pub.Path, "/")))
	if _, err := os.Stat(articleDiskPath); err != nil {
		t.Fatalf("article file missing on disk at %q: %v", articleDiskPath, err)
	}

	indexData, err := os.ReadFile(filepath.Join(dataDir, "index.json"))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	var indexed []model.Article
	if err := json.Unmarshal(indexData, &indexed); err != nil {
		t.Fatalf("parse index.json: %v", err)
	}
	if !containsID(indexed, pub.ID) {
		t.Errorf("index.json does not contain article %q; got %+v", pub.ID, indexed)
	}

	sidebar, err := os.ReadFile(filepath.Join(dataDir, "_sidebar.md"))
	if err != nil {
		t.Fatalf("read _sidebar.md: %v", err)
	}
	if !strings.Contains(string(sidebar), pubReq.Title) {
		t.Errorf("_sidebar.md missing title %q:\n%s", pubReq.Title, sidebar)
	}
	if !strings.Contains(string(sidebar), pub.Path) {
		t.Errorf("_sidebar.md missing path %q:\n%s", pub.Path, sidebar)
	}

	listResp, err := http.Get(ts.URL + "/api/articles")
	if err != nil {
		t.Fatalf("GET /api/articles: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/articles status %d", listResp.StatusCode)
	}
	var list []model.Article
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !containsID(list, pub.ID) {
		t.Errorf("article list does not contain %q; got %+v", pub.ID, list)
	}

	getResp, err := http.Get(ts.URL + "/api/articles/" + pub.ID)
	if err != nil {
		t.Fatalf("GET /api/articles/{id}: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/articles/{id} status %d", getResp.StatusCode)
	}
	var got model.Article
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode article: %v", err)
	}
	if got.Title != pubReq.Title {
		t.Errorf("article Title = %q, want %q", got.Title, pubReq.Title)
	}
	if got.Category != pubReq.Category {
		t.Errorf("article Category = %q, want %q", got.Category, pubReq.Category)
	}
	if got.Author != pubReq.Author {
		t.Errorf("article Author = %q, want %q", got.Author, pubReq.Author)
	}

	rootResp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer rootResp.Body.Close()
	if rootResp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status %d", rootResp.StatusCode)
	}
	rootBody, err := io.ReadAll(rootResp.Body)
	if err != nil {
		t.Fatalf("read /: %v", err)
	}
	if !strings.Contains(string(rootBody), "DocsHub") {
		t.Errorf("root HTML does not contain \"DocsHub\":\n%s", rootBody)
	}

	mdResp, err := http.Get(ts.URL + pub.Path)
	if err != nil {
		t.Fatalf("GET %s: %v", pub.Path, err)
	}
	defer mdResp.Body.Close()
	if mdResp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status %d", pub.Path, mdResp.StatusCode)
	}
	mdBody, err := io.ReadAll(mdResp.Body)
	if err != nil {
		t.Fatalf("read article md: %v", err)
	}
	if string(mdBody) != content {
		t.Errorf("article markdown body mismatch.\nwant:\n%s\ngot:\n%s", content, mdBody)
	}
}

func TestVersioningFlow(t *testing.T) {
	dataDir := t.TempDir()

	store, err := server.New(dataDir)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	v1Req := model.PublishRequest{
		Title:    "Versioned Article",
		Content:  "Version 1 content for the versioning test.",
		Category: "Docs",
		Author:   "tester",
	}
	v1 := postArticle(t, ts.URL, v1Req)
	if v1.Version != 1 {
		t.Fatalf("v1 Version = %d, want 1", v1.Version)
	}

	v1DiskPath := filepath.Join(dataDir, filepath.FromSlash(strings.TrimPrefix(v1.Path, "/")))
	if _, err := os.Stat(v1DiskPath); err != nil {
		t.Fatalf("v1 file missing: %v", err)
	}

	v2Req := model.PublishRequest{
		Title:     "Versioned Article",
		Content:   "Version 2 content — significantly updated.",
		Category:  "Docs",
		Author:    "tester",
		VersionOf: v1.ID,
	}
	v2 := postArticle(t, ts.URL, v2Req)
	if v2.Version != 2 {
		t.Errorf("v2 Version = %d, want 2", v2.Version)
	}
	if len(v2.PreviousVersions) == 0 {
		t.Errorf("v2 PreviousVersions should not be empty")
	}
	for _, prev := range v2.PreviousVersions {
		archivedPath := filepath.Join(dataDir, filepath.FromSlash(strings.TrimPrefix(prev, "/")))
		if _, err := os.Stat(archivedPath); err != nil {
			t.Errorf("archived version missing at %q: %v", archivedPath, err)
		}
	}

	listResp, err := http.Get(ts.URL + "/api/articles")
	if err != nil {
		t.Fatalf("GET /api/articles: %v", err)
	}
	defer listResp.Body.Close()
	var list []model.Article
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 current article after versioning, got %d: %+v", len(list), list)
	}
	if list[0].Version != 2 {
		t.Errorf("current article Version = %d, want 2", list[0].Version)
	}

	mdResp, err := http.Get(ts.URL + list[0].File)
	if err != nil {
		t.Fatalf("GET current md: %v", err)
	}
	defer mdResp.Body.Close()
	mdBody, err := io.ReadAll(mdResp.Body)
	if err != nil {
		t.Fatalf("read current md: %v", err)
	}
	if !strings.Contains(string(mdBody), "Version 2 content") {
		t.Errorf("current markdown does not reflect v2:\n%s", mdBody)
	}
}

func postArticle(t *testing.T, baseURL string, req model.PublishRequest) model.PublishResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(baseURL+"/api/articles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/articles: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /api/articles status %d: %s", resp.StatusCode, buf)
	}
	var out model.PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	return out
}

func containsID(articles []model.Article, id string) bool {
	for _, a := range articles {
		if a.ID == id {
			return true
		}
	}
	return false
}

func TestHTMLArticleFlow(t *testing.T) {
	dataDir := t.TempDir()

	htmlSrc, err := os.ReadFile(filepath.Join("..", "web", "index.html"))
	if err != nil {
		t.Fatalf("read web/index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "index.html"), htmlSrc, 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	store, err := server.New(dataDir)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	ts := httptest.NewServer(store.Handler())
	defer ts.Close()

	htmlContent := "<!DOCTYPE html><html><head><title>Report</title></head><body><h1>Quarterly Report</h1><p>Revenue is up.</p></body></html>"
	pubReq := model.PublishRequest{
		Title:    "Quarterly Report",
		Content:  htmlContent,
		Category: "Finance",
		Tags:     []string{"quarterly", "report"},
		Author:   "cfo",
		Format:   "html",
	}
	body, err := json.Marshal(pubReq)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/articles", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/articles: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		buf, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d, body=%s", resp.StatusCode, buf)
	}

	var pub model.PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&pub); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pub.Version != 1 {
		t.Errorf("Version = %d, want 1", pub.Version)
	}
	if !strings.HasSuffix(pub.Path, ".html") {
		t.Errorf("Path = %q, want .html suffix", pub.Path)
	}

	// Verify file exists on disk.
	articleDiskPath := filepath.Join(dataDir, filepath.FromSlash(strings.TrimPrefix(pub.Path, "/")))
	if _, err := os.Stat(articleDiskPath); err != nil {
		t.Fatalf("HTML file missing on disk at %q: %v", articleDiskPath, err)
	}

	// GET via the /html/ route.
	htmlResp, err := http.Get(ts.URL + "/html/Finance/quarterly-report")
	if err != nil {
		t.Fatalf("GET /html/Finance/quarterly-report: %v", err)
	}
	defer htmlResp.Body.Close()
	if htmlResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /html/ status = %d, want 200", htmlResp.StatusCode)
	}
	ct := htmlResp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	htmlBody, err := io.ReadAll(htmlResp.Body)
	if err != nil {
		t.Fatalf("read HTML body: %v", err)
	}
	if !strings.Contains(string(htmlBody), "Revenue is up.") {
		t.Errorf("HTML body should contain article content, got: %s", htmlBody)
	}

	// Verify sidebar does NOT contain HTML articles (they live under /html/ index page).
	sidebar, err := os.ReadFile(filepath.Join(dataDir, "_sidebar.md"))
	if err != nil {
		t.Fatalf("read _sidebar.md: %v", err)
	}
	if strings.Contains(string(sidebar), "Quarterly Report") {
		t.Errorf("_sidebar.md should NOT contain HTML article entries:\n%s", sidebar)
	}

	// Verify /html/ index page lists the HTML article.
	idxResp, err := http.Get(ts.URL + "/html/")
	if err != nil {
		t.Fatalf("GET /html/: %v", err)
	}
	defer idxResp.Body.Close()
	if idxResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /html/ status = %d, want 200", idxResp.StatusCode)
	}
	idxBody, err := io.ReadAll(idxResp.Body)
	if err != nil {
		t.Fatalf("read /html/ body: %v", err)
	}
	if !strings.Contains(string(idxBody), "quarterly-report") {
		t.Errorf("/html/ index should link to the HTML article, got: %s", idxBody)
	}

	// Verify article metadata via API.
	getResp, err := http.Get(ts.URL + "/api/articles/" + pub.ID)
	if err != nil {
		t.Fatalf("GET /api/articles/{id}: %v", err)
	}
	defer getResp.Body.Close()
	var got model.Article
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Format != "html" {
		t.Errorf("Format = %q, want html", got.Format)
	}
	if strings.Contains(got.Summary, "<") {
		t.Errorf("Summary should have HTML tags stripped, got %q", got.Summary)
	}
	if !strings.Contains(got.Summary, "Quarterly Report") {
		t.Errorf("Summary should contain text content, got %q", got.Summary)
	}

	// Verify list API returns the article.
	listResp, err := http.Get(ts.URL + "/api/articles")
	if err != nil {
		t.Fatalf("GET /api/articles: %v", err)
	}
	defer listResp.Body.Close()
	var list []model.Article
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !containsID(list, pub.ID) {
		t.Errorf("article list missing %q", pub.ID)
	}
}
