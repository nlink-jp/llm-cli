// Package client provides an HTTP client for OpenAI-compatible chat/completions
// API endpoints, with streaming (SSE) and structured output support.
//
// Uses nlk/backoff for retry on transient errors.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nlink-jp/nlk/backoff"
)

// Client is an OpenAI-compatible API client.
type Client struct {
	endpoint   string
	apiKey     string
	model      string
	timeout    time.Duration
	strategy   string // response_format_strategy: auto, native, prompt
	httpClient *http.Client
	stderr     io.Writer
	debug      io.Writer
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithModel sets the default model name.
func WithModel(model string) Option {
	return func(c *Client) { c.model = model }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
		c.httpClient.Timeout = d
	}
}

// WithStrategy sets the response_format_strategy.
func WithStrategy(s string) Option {
	return func(c *Client) { c.strategy = s }
}

// WithStderr sets the writer for warnings.
func WithStderr(w io.Writer) Option {
	return func(c *Client) { c.stderr = w }
}

// WithDebug enables debug logging to the given writer.
func WithDebug(w io.Writer) Option {
	return func(c *Client) { c.debug = w }
}

// New creates a new Client with the given endpoint and options.
func New(endpoint string, opts ...Option) *Client {
	c := &Client{
		endpoint:   normalizeEndpoint(endpoint),
		timeout:    120 * time.Second,
		strategy:   "auto",
		httpClient: &http.Client{Timeout: 120 * time.Second},
		stderr:     io.Discard,
		debug:      nil,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ResponseFormat describes the desired structured output format.
type ResponseFormat struct {
	Type       string          // "json_object" or "json_schema"
	SchemaName string          // identifier for json_schema
	Schema     json.RawMessage // raw JSON Schema document
}

// ImageData holds a base64-encoded image with its MIME type.
type ImageData struct {
	MIMEType string // e.g. "image/jpeg", "image/png"
	Base64   string // base64-encoded image data
}

// ChatInput holds the parameters for a chat request.
type ChatInput struct {
	Model          string
	SystemPrompt   string
	UserPrompt     string
	Images         []ImageData
	ResponseFormat *ResponseFormat
}

// Chat sends a blocking chat request and returns the response text.
func (c *Client) Chat(ctx context.Context, in ChatInput) (string, error) {
	sendFormat := c.strategy != "prompt"
	return c.chatWithRetry(ctx, in, sendFormat)
}

func (c *Client) chatWithRetry(ctx context.Context, in ChatInput, sendFormat bool) (string, error) {
	bo := backoff.New()

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			d := bo.Duration(attempt)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(d):
			}
		}

		result, err := c.doChat(ctx, in, sendFormat)
		if err == nil {
			return result, nil
		}

		// Check for response_format fallback
		if sendFormat && in.ResponseFormat != nil && c.strategy == "auto" {
			if isFormatUnsupportedError(err) {
				fmt.Fprintf(c.stderr, "Warning: response_format not supported by this API, falling back to prompt injection\n")
				return c.chatWithRetry(ctx, in, false)
			}
		}

		// Don't retry client errors (4xx) except rate limits (429)
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 && apiErr.StatusCode != 429 {
				return "", err
			}
		}

		lastErr = err
	}
	return "", fmt.Errorf("after 3 attempts: %w", lastErr)
}

func (c *Client) doChat(ctx context.Context, in ChatInput, sendFormat bool) (string, error) {
	body := c.buildRequest(in, false, sendFormat)

	if c.debug != nil {
		debugJSON(c.debug, "Request", body)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doHTTP(ctx, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if c.debug != nil {
		debugRaw(c.debug, "Response", respBody)
	}

	if resp.StatusCode != http.StatusOK {
		return "", &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return cr.Choices[0].Message.Content, nil
}

// ChatStream sends a streaming chat request. Tokens are sent to the returned
// channel. The channel is closed when the stream ends or an error occurs.
// Errors are returned via the error channel.
func (c *Client) ChatStream(ctx context.Context, in ChatInput) (<-chan string, <-chan error) {
	tokens := make(chan string, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(tokens)
		defer close(errs)

		sendFormat := c.strategy != "prompt"
		body := c.buildRequest(in, true, sendFormat && in.ResponseFormat != nil)

		if c.debug != nil {
			debugJSON(c.debug, "Stream Request", body)
		}

		data, err := json.Marshal(body)
		if err != nil {
			errs <- fmt.Errorf("marshal request: %w", err)
			return
		}

		resp, err := c.doHTTP(ctx, data)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			errs <- &APIError{StatusCode: resp.StatusCode, Body: string(body)}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue // skip malformed chunks
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content == "" {
					continue
				}
				select {
				case tokens <- choice.Delta.Content:
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("stream read: %w", err)
		}
	}()

	return tokens, errs
}

func (c *Client) buildRequest(in ChatInput, stream, sendFormat bool) chatRequest {
	model := in.Model
	if model == "" {
		model = c.model
	}

	var msgs []message
	if in.SystemPrompt != "" {
		msgs = append(msgs, message{Role: "system", Text: in.SystemPrompt})
	}

	// Build user message: text-only or multimodal (text + images)
	if len(in.Images) > 0 {
		parts := []contentPart{{Type: "text", Text: in.UserPrompt}}
		for _, img := range in.Images {
			parts = append(parts, contentPart{
				Type: "image_url",
				ImageURL: &imageURL{
					URL: "data:" + img.MIMEType + ";base64," + img.Base64,
				},
			})
		}
		msgs = append(msgs, message{Role: "user", Parts: parts})
	} else {
		msgs = append(msgs, message{Role: "user", Text: in.UserPrompt})
	}

	req := chatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   stream,
	}

	if sendFormat && in.ResponseFormat != nil {
		rf := &responseFormat{Type: in.ResponseFormat.Type}
		if in.ResponseFormat.Type == "json_schema" {
			rf.JSONSchema = &jsonSchema{
				Name:   in.ResponseFormat.SchemaName,
				Strict: true,
				Schema: in.ResponseFormat.Schema,
			}
		}
		req.ResponseFormat = rf
	}

	return req
}

func (c *Client) doHTTP(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	return resp, nil
}

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimRight(endpoint, "/")
	if !strings.HasSuffix(endpoint, "/v1") {
		endpoint += "/v1"
	}
	return endpoint + "/chat/completions"
}

func isFormatUnsupportedError(err error) bool {
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	if apiErr.StatusCode != 400 && apiErr.StatusCode != 422 {
		return false
	}
	body := strings.ToLower(apiErr.Body)
	keywords := []string{"response_format", "not supported", "unsupported", "unknown field"}
	for _, kw := range keywords {
		if strings.Contains(body, kw) {
			return true
		}
	}
	return false
}

func debugJSON(w io.Writer, label string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "[DEBUG] %s: marshal error: %v\n", label, err)
		return
	}
	fmt.Fprintf(w, "[DEBUG] %s:\n%s\n", label, data)
}

func debugRaw(w io.Writer, label string, data []byte) {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		fmt.Fprintf(w, "[DEBUG] %s:\n%s\n", label, buf.String())
	} else {
		fmt.Fprintf(w, "[DEBUG] %s:\n%s\n", label, data)
	}
}
