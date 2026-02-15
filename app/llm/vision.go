package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type VisionClient struct {
	BaseURL string
	APIKey  string
	Model   string
}

func (c *VisionClient) baseURL() string {
	u := strings.TrimSpace(c.BaseURL)
	u = strings.TrimRight(u, "/")
	if u == "" {
		u = "http://localhost:11434/v1"
	}
	return u
}

// CaptionFromImages sends base64-encoded images to an OpenAI-compatible
// /v1/chat/completions endpoint and returns the assistant's text response.
func (c *VisionClient) CaptionFromImages(ctx context.Context, base64Images []string, prompt string) (string, error) {
	model := strings.TrimSpace(c.Model)
	if model == "" {
		return "", fmt.Errorf("vision model is required")
	}
	if len(base64Images) == 0 {
		return "", fmt.Errorf("at least one image is required")
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = "Describe what is happening in these video frames."
	}

	// Build content array: text prompt + image_url blocks
	content := []map[string]any{
		{"type": "text", "text": prompt},
	}
	for _, img := range base64Images {
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]string{
				"url": "data:image/png;base64," + img,
			},
		})
	}

	body := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": content,
			},
		},
		"max_tokens": 1024,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL() + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(c.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call vision API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("vision API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(out.Choices) == 0 {
		return "", fmt.Errorf("vision API returned no choices")
	}

	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
