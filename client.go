// Package stt provides a speech-to-text client for OpenAI-compatible STT APIs
// (ox-whisper, OpenAI Whisper, etc.) via multipart upload.
package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultTimeout = 60 * time.Second

// Client sends audio files to an OpenAI-compatible /v1/audio/transcriptions endpoint.
type Client struct {
	baseURL  string
	language string
	model    string
	format   string
	http     *http.Client
}

// Option configures the Client.
type Option func(*Client)

// WithLanguage sets the transcription language (default: "ru").
func WithLanguage(lang string) Option {
	return func(c *Client) { c.language = lang }
}

// WithModel sets the model name (default: "moonshine-v2").
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithFormat sets the response format: "json" or "verbose_json" (default: "json").
func WithFormat(format string) Option {
	return func(c *Client) { c.format = format }
}

// WithTimeout sets the HTTP request timeout (default: 60s).
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.http.Timeout = timeout }
}

// Response holds the transcription result.
type Response struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// New creates an STT client for the given base URL (e.g. "http://127.0.0.1:8092").
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:  baseURL,
		language: "ru",
		model:    "moonshine-v2",
		format:   "json",
		http:     &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Transcribe sends the audio file to the STT service and returns the transcription.
func (c *Client) Transcribe(ctx context.Context, audioPath string) (*Response, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("open audio: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy audio: %w", err)
	}
	_ = w.WriteField("model", c.model)
	_ = w.WriteField("language", c.language)
	_ = w.WriteField("response_format", c.format)
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/audio/transcriptions", &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("STT error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Language == "" {
		result.Language = c.language
	}
	return &result, nil
}

// IsAvailable checks if the STT service is reachable via /health.
func (c *Client) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) //nolint:mnd
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
