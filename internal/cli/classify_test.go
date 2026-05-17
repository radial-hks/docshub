package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseClassifyResponse_PlainJSON(t *testing.T) {
	input := `{"title":"Test Article","category":"AI","tags":["ml","go"],"author":""}`
	result, err := parseClassifyResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Test Article" {
		t.Errorf("title = %q, want %q", result.Title, "Test Article")
	}
	if result.Category != "AI" {
		t.Errorf("category = %q, want %q", result.Category, "AI")
	}
	if len(result.Tags) != 2 || result.Tags[0] != "ml" || result.Tags[1] != "go" {
		t.Errorf("tags = %v, want [ml go]", result.Tags)
	}
}

func TestParseClassifyResponse_CodeFence(t *testing.T) {
	input := "```json\n{\"title\":\"Fenced\",\"category\":\"DevOps\",\"tags\":[\"ci\",\"cd\"],\"author\":\"bob\"}\n```"
	result, err := parseClassifyResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Fenced" {
		t.Errorf("title = %q, want %q", result.Title, "Fenced")
	}
	if result.Category != "DevOps" {
		t.Errorf("category = %q, want %q", result.Category, "DevOps")
	}
	if result.Author != "bob" {
		t.Errorf("author = %q, want %q", result.Author, "bob")
	}
}

func TestParseClassifyResponse_CodeFenceNoLang(t *testing.T) {
	input := "```\n{\"title\":\"NoLang\",\"category\":\"Note\",\"tags\":[\"test\"],\"author\":\"\"}\n```"
	result, err := parseClassifyResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "NoLang" {
		t.Errorf("title = %q, want %q", result.Title, "NoLang")
	}
}

func TestParseClassifyResponse_ExtraText(t *testing.T) {
	input := "Here is the result:\n{\"title\":\"Extra\",\"category\":\"Research\",\"tags\":[\"physics\"],\"author\":\"alice\"}\nHope this helps!"
	result, err := parseClassifyResponse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Extra" {
		t.Errorf("title = %q, want %q", result.Title, "Extra")
	}
}

func TestClassifyWithLLM_Disabled(t *testing.T) {
	cfg := &Config{ClassifyURL: ""}
	result, err := ClassifyWithLLM("some content", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result when ClassifyURL is empty, got %+v", result)
	}
}

func TestClassifyWithLLM_Success(t *testing.T) {
	// Set up a fake OpenAI-compatible server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Read and verify the request body.
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Return a valid classification response.
		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: `{"title":"LLM Article","category":"AI","tags":["llm","nlp"],"author":"testbot"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &Config{
		ClassifyURL:   srv.URL,
		ClassifyModel: "test-model",
	}

	result, err := ClassifyWithLLM("This is an article about LLMs.", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Title != "LLM Article" {
		t.Errorf("title = %q, want %q", result.Title, "LLM Article")
	}
	if result.Category != "AI" {
		t.Errorf("category = %q, want %q", result.Category, "AI")
	}
	if len(result.Tags) != 2 || result.Tags[0] != "llm" || result.Tags[1] != "nlp" {
		t.Errorf("tags = %v, want [llm nlp]", result.Tags)
	}
	if result.Author != "testbot" {
		t.Errorf("author = %q, want %q", result.Author, "testbot")
	}
}

func TestClassifyWithLLM_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer srv.Close()

	cfg := &Config{
		ClassifyURL:   srv.URL,
		ClassifyModel: "bad-model",
	}

	result, err := ClassifyWithLLM("content", cfg)
	if err == nil {
		t.Fatal("expected error for server 500, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result on error, got %+v", result)
	}
}

func TestClassifyWithLLM_DefaultModel(t *testing.T) {
	var capturedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedModel = req.Model

		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: `{"title":"T","category":"Other","tags":[],"author":""}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &Config{
		ClassifyURL:   srv.URL,
		ClassifyModel: "", // empty — should default to qwen2.5:7b
	}

	_, err := ClassifyWithLLM("content", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedModel != "qwen2.5:7b" {
		t.Errorf("model = %q, want %q", capturedModel, "qwen2.5:7b")
	}
}

func TestClassifyWithLLM_TruncatesContent(t *testing.T) {
	var capturedContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		capturedContent = req.Messages[1].Content

		resp := chatCompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: `{"title":"T","category":"Other","tags":[],"author":""}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	cfg := &Config{
		ClassifyURL:   srv.URL,
		ClassifyModel: "test",
	}

	longContent := stringsRepeat("x", 5000)
	_, err := ClassifyWithLLM(longContent, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedContent) > 3000 {
		t.Errorf("content length = %d, want <= 3000", len(capturedContent))
	}
}

// helper to avoid importing strings just for repeat in tests.
func stringsRepeat(s string, n int) string {
	out := make([]byte, len(s)*n)
	for i := 0; i < n; i++ {
		copy(out[i*len(s):], s)
	}
	return string(out)
}
