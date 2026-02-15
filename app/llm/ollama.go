package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	BaseURL string
	Model   string
}

func (c *OllamaClient) baseURL() string {
	u := strings.TrimSpace(c.BaseURL)
	u = strings.TrimRight(u, "/")
	if u == "" {
		u = "http://localhost:11434"
	}
	return u
}

func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ollama returned http %d", resp.StatusCode)
	}

	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	names := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		names = append(names, m.Name)
	}
	return names, nil
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	model := strings.TrimSpace(c.Model)
	if model == "" {
		return "", fmt.Errorf("ollama model is required")
	}

	body := map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned http %d", resp.StatusCode)
	}

	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return out.Response, nil
}

type CaptionTags struct {
	Label string   `json:"label"`
	Tags  []string `json:"tags"`
}

func ParseCaptionTags(s string) (CaptionTags, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return CaptionTags{}, false
	}

	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start >= 0 && end > start {
		s = s[start : end+1]
	}

	var out CaptionTags
	if err := json.Unmarshal([]byte(s), &out); err == nil {
		out.Label = strings.TrimSpace(out.Label)
		if out.Label == "" && len(out.Tags) == 0 {
			return CaptionTags{}, false
		}
		cleanTags := make([]string, 0, len(out.Tags))
		seen := map[string]bool{}
		for _, t := range out.Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			key := strings.ToLower(t)
			if seen[key] {
				continue
			}
			seen[key] = true
			cleanTags = append(cleanTags, t)
		}
		out.Tags = cleanTags
		return out, true
	}

	return CaptionTags{Label: s}, true
}
