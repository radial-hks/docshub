package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ClassifyResult holds the metadata suggested by the LLM.
type ClassifyResult struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
	Author   string   `json:"author"`
}

// chatCompletionRequest is the payload sent to an OpenAI-compatible API.
type chatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []chatMessage       `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionResponse is the subset of the OpenAI response we need.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

const classifySystemPrompt = `You are a document classifier. Analyze the given article and return a JSON object with fields: title (concise Chinese or English title), category (one of: AI, UE, DevOps, Research, Note, Other), tags (array of 2-5 relevant tags), author (leave empty if unknown). Return ONLY the JSON, no explanation.`

// ClassifyWithLLM sends article content to an OpenAI-compatible chat
// completions API and returns suggested metadata. Returns nil, nil if
// classify_url is not configured (classification disabled).
func ClassifyWithLLM(content string, cfg *Config) (*ClassifyResult, error) {
	if cfg.ClassifyURL == "" {
		return nil, nil
	}

	// Truncate content to first 3000 characters.
	if len(content) > 3000 {
		content = content[:3000]
	}

	model := cfg.ClassifyModel
	if model == "" {
		model = "qwen2.5:7b"
	}

	reqBody := chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: classifySystemPrompt},
			{Role: "user", Content: content},
		},
		Temperature: 0.3,
		MaxTokens:   200,
	}

	buf, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal classify request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, cfg.ClassifyURL, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build classify request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("classify request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read classify response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("classify API returned HTTP %d: %s", resp.StatusCode, string(data))
	}

	var ccResp chatCompletionResponse
	if err := json.Unmarshal(data, &ccResp); err != nil {
		return nil, fmt.Errorf("parse classify response: %w", err)
	}

	if len(ccResp.Choices) == 0 {
		return nil, fmt.Errorf("classify API returned no choices")
	}

	result, err := parseClassifyResponse(ccResp.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("parse classify result: %w", err)
	}

	return result, nil
}

// parseClassifyResponse extracts a ClassifyResult from the LLM's response
// content. The content may be plain JSON, wrapped in markdown code fences,
// or have extra text before/after the JSON.
func parseClassifyResponse(content string) (*ClassifyResult, error) {
	jsonStr := extractJSON(content)

	var result ClassifyResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON in classify response: %w", err)
	}
	return &result, nil
}

// jsonRegex matches a JSON object (with some nesting support).
var jsonRegex = regexp.MustCompile(`(?s)\{.*\}`)

// extractJSON pulls the first JSON object out of a string that may contain
// markdown code fences or surrounding text.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	// Try to strip markdown code fences: ```json ... ``` or ``` ... ```
	if fenced := stripCodeFence(s); fenced != "" {
		return fenced
	}

	// Find first { ... } pair.
	m := jsonRegex.FindString(s)
	if m != "" {
		return m
	}

	return s
}

// codeFenceRegex matches ```json or ``` at start, and ``` at end.
var codeFenceRegex = regexp.MustCompile("(?s)^```(?:json)?\\s*\\n(.*?)\\n\\s*```$")

func stripCodeFence(s string) string {
	m := codeFenceRegex.FindStringSubmatch(s)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}
