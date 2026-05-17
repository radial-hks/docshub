package server

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
)

func newTestServer(t *testing.T) (*httptest.Server, *Store, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "docshub-handler-test-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	store, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ts := httptest.NewServer(store.Handler())
	t.Cleanup(ts.Close)
	return ts, store, dir
}

func doJSON(t *testing.T, method, url string, body any) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	return resp
}

func TestHandleCreateArticle(t *testing.T) {
	ts, _, _ := newTestServer(t)

	req := model.PublishRequest{
		Title:    "Hello World",
		Content:  "# Hello\n\nThis is content.",
		Category: "AI",
		Tags:     []string{"intro", "test"},
		Author:   "alice",
	}
	resp := doJSON(t, http.MethodPost, ts.URL+"/api/articles", req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 201; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var pubResp model.PublishResponse
	if err := json.NewDecoder(resp.Body).Decode(&pubResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pubResp.ID == "" {
		t.Errorf("ID empty")
	}
	if pubResp.Status != "ok" {
		t.Errorf("Status = %q, want ok", pubResp.Status)
	}
	if pubResp.Version != 1 {
		t.Errorf("Version = %d, want 1", pubResp.Version)
	}
	if !strings.HasPrefix(pubResp.Path, "/articles/") {
		t.Errorf("Path = %q, want prefix /articles/", pubResp.Path)
	}
}

func TestHandleCreateArticleInvalidJSON(t *testing.T) {
	ts, _, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/articles", strings.NewReader("{not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var errBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errBody["error"] == "" {
		t.Errorf("error message missing")
	}
}

func TestHandleListArticles(t *testing.T) {
	ts, store, _ := newTestServer(t)

	_, err := store.Create(model.PublishRequest{Title: "AI Article", Content: "x", Category: "AI", Author: "a"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = store.Create(model.PublishRequest{Title: "UE Article", Content: "y", Category: "UE", Author: "b"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := http.Get(ts.URL + "/api/articles")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got []model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
}

func TestHandleListArticlesWithCategoryFilter(t *testing.T) {
	ts, store, _ := newTestServer(t)

	_, _ = store.Create(model.PublishRequest{Title: "AI One", Content: "x", Category: "AI", Author: "a"})
	_, _ = store.Create(model.PublishRequest{Title: "AI Two", Content: "x", Category: "AI", Author: "a"})
	_, _ = store.Create(model.PublishRequest{Title: "UE One", Content: "y", Category: "UE", Author: "b"})

	resp, err := http.Get(ts.URL + "/api/articles?category=AI")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var got []model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2", len(got))
	}
	for _, a := range got {
		if a.Category != "AI" {
			t.Errorf("got Category = %q, want AI", a.Category)
		}
	}
}

func TestHandleGetArticle(t *testing.T) {
	ts, store, _ := newTestServer(t)
	pub, err := store.Create(model.PublishRequest{Title: "Find Me", Content: "x", Category: "AI", Author: "a"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := http.Get(ts.URL + "/api/articles/" + pub.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got model.Article
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != pub.ID {
		t.Errorf("ID = %q, want %q", got.ID, pub.ID)
	}
	if got.Title != "Find Me" {
		t.Errorf("Title = %q, want Find Me", got.Title)
	}
}

func TestHandleGetArticleNotFound(t *testing.T) {
	ts, _, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/articles/does-not-exist")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	var errBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errBody["error"] == "" {
		t.Errorf("error message missing")
	}
}

func TestHandleDeleteArticle(t *testing.T) {
	ts, store, _ := newTestServer(t)
	pub, err := store.Create(model.PublishRequest{Title: "Delete Me", Content: "x", Category: "AI", Author: "a"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/articles/"+pub.ID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	if _, err := store.Get(pub.ID); err == nil {
		t.Errorf("expected article to be removed")
	}
}

func TestHandleDeleteArticleNotFound(t *testing.T) {
	ts, _, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/articles/missing", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestStaticFileServing(t *testing.T) {
	ts, _, dataDir := newTestServer(t)

	htmlContent := "<!DOCTYPE html><html><body>DocsHub</body></html>"
	if err := os.WriteFile(filepath.Join(dataDir, "index.html"), []byte(htmlContent), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	resp, err := http.Get(ts.URL + "/index.html")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "DocsHub") {
		t.Errorf("body = %q, want to contain DocsHub", body)
	}
}

func TestCORSHeaders(t *testing.T) {
	ts, _, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/api/articles")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("Access-Control-Allow-Methods = %q, missing POST", got)
	}
}

func TestCORSPreflight(t *testing.T) {
	ts, _, _ := newTestServer(t)

	req, _ := http.NewRequest(http.MethodOptions, ts.URL+"/api/articles", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
}

func TestHandleServeHTML(t *testing.T) {
	ts, store, dataDir := newTestServer(t)

	// Create an HTML article via the store directly.
	_, err := store.Create(model.PublishRequest{
		Title:    "HTML Doc",
		Content:  "<!DOCTYPE html><html><body><h1>Hello HTML</h1></body></html>",
		Category: "Docs",
		Author:   "alice",
		Format:   "html",
	})
	if err != nil {
		t.Fatalf("Create HTML article: %v", err)
	}

	// Verify the file was created on disk.
	expectedPath := filepath.Join(dataDir, "articles", "Docs", "html-doc.html")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("HTML file missing on disk: %v", err)
	}

	resp, err := http.Get(ts.URL + "/html/Docs/html-doc")
	if err != nil {
		t.Fatalf("GET /html/Docs/html-doc: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "<h1>Hello HTML</h1>") {
		t.Errorf("body = %q, want to contain HTML content", body)
	}
}

func TestHandleServeHTMLNotFound(t *testing.T) {
	ts, _, _ := newTestServer(t)

	resp, err := http.Get(ts.URL + "/html/Nope/missing")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
