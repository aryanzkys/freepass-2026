package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	apiKey  string
	model   string
	http    *http.Client
	baseURL string
}

type InvalidResponseError struct {
	Message string
}

func (e InvalidResponseError) Error() string {
	return e.Message
}

func NewClient(apiKey string, model string, timeout time.Duration) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: timeout},
		baseURL: "https://generativelanguage.googleapis.com",
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.apiKey != ""
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

func (c *Client) GenerateJSON(ctx context.Context, systemPrompt string, userPrompt string, schema map[string]interface{}) ([]byte, error) {
	if !c.Configured() {
		return nil, errors.New("ai_not_configured")
	}
	payload := request{
		SystemInstruction: &content{
			Role:  "system",
			Parts: []part{{Text: systemPrompt}},
		},
		Contents:         []content{{Role: "user", Parts: []part{{Text: userPrompt}}}},
		GenerationConfig: generationConfig{ResponseMimeType: "application/json", ResponseSchema: schema},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	var respBody []byte
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		respBody, err = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			if attempt == 2 {
				return nil, fmt.Errorf("ai_upstream_error")
			}
			backoff := 200 * time.Millisecond
			if attempt == 1 {
				backoff = 500 * time.Millisecond
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			continue
		}
		return nil, fmt.Errorf("ai_upstream_error")
	}
	var parsed response
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, InvalidResponseError{Message: err.Error()}
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, InvalidResponseError{Message: "missing_content"}
	}
	text := parsed.Candidates[0].Content.Parts[0].Text
	if text == "" {
		return nil, InvalidResponseError{Message: "empty_content"}
	}
	var check map[string]interface{}
	if err := json.Unmarshal([]byte(text), &check); err != nil {
		return nil, InvalidResponseError{Message: err.Error()}
	}
	return []byte(text), nil
}

type request struct {
	SystemInstruction *content         `json:"systemInstruction,omitempty"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type generationConfig struct {
	ResponseMimeType string                 `json:"responseMimeType,omitempty"`
	ResponseSchema   map[string]interface{} `json:"responseSchema,omitempty"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type response struct {
	Candidates []candidate `json:"candidates"`
}

type candidate struct {
	Content content `json:"content"`
}
