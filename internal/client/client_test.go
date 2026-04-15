package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	srv := httptest.NewServer(handler)
	c := New(srv.URL, WithModel("test-model"))
	// Override the endpoint to use test server URL directly
	c.endpoint = srv.URL + "/v1/chat/completions"
	return srv, c
}

func TestChat_BasicResponse(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		resp := `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	result, err := c.Chat(context.Background(), ChatInput{
		UserPrompt: "Hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello!" {
		t.Errorf("got %q, want %q", result, "Hello!")
	}
}

func TestChat_SystemPrompt(t *testing.T) {
	var rawMessages []json.RawMessage

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Messages []json.RawMessage `json:"messages"`
		}
		json.Unmarshal(body, &req)
		rawMessages = req.Messages

		resp := `{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	_, err := c.Chat(context.Background(), ChatInput{
		SystemPrompt: "You are helpful",
		UserPrompt:   "Hi",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(rawMessages) != 2 {
		t.Fatalf("got %d messages, want 2", len(rawMessages))
	}

	// Parse as simple {role, content} structs
	type simpleMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var sys, usr simpleMsg
	json.Unmarshal(rawMessages[0], &sys)
	json.Unmarshal(rawMessages[1], &usr)

	if sys.Role != "system" || sys.Content != "You are helpful" {
		t.Errorf("system message = %+v", sys)
	}
	if usr.Role != "user" || usr.Content != "Hi" {
		t.Errorf("user message = %+v", usr)
	}
}

func TestChat_APIError(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error":"internal"}`)
	})
	defer srv.Close()

	_, err := c.Chat(context.Background(), ChatInput{UserPrompt: "Hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error may be wrapped (e.g. "after 3 attempts: API error 500: ...")
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want to contain 500", err.Error())
	}
}

func TestChat_ResponseFormat(t *testing.T) {
	var receivedFormat *responseFormat

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)
		receivedFormat = req.ResponseFormat

		resp := `{"choices":[{"message":{"role":"assistant","content":"{\"key\":\"value\"}"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	schema := json.RawMessage(`{"type":"object","properties":{"key":{"type":"string"}}}`)
	_, err := c.Chat(context.Background(), ChatInput{
		UserPrompt: "test",
		ResponseFormat: &ResponseFormat{
			Type:       "json_schema",
			SchemaName: "test_schema",
			Schema:     schema,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if receivedFormat == nil {
		t.Fatal("response_format not sent")
	}
	if receivedFormat.Type != "json_schema" {
		t.Errorf("Type = %q, want json_schema", receivedFormat.Type)
	}
	if receivedFormat.JSONSchema == nil {
		t.Fatal("json_schema not set")
	}
	if receivedFormat.JSONSchema.Name != "test_schema" {
		t.Errorf("Name = %q, want test_schema", receivedFormat.JSONSchema.Name)
	}
}

func TestChat_PromptStrategy(t *testing.T) {
	var receivedFormat *responseFormat

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)
		receivedFormat = req.ResponseFormat

		resp := `{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	c.strategy = "prompt"

	_, err := c.Chat(context.Background(), ChatInput{
		UserPrompt: "test",
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if receivedFormat != nil {
		t.Error("response_format should not be sent with prompt strategy")
	}
}

func TestChat_AutoFallback(t *testing.T) {
	callCount := 0

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		json.Unmarshal(body, &req)

		if req.ResponseFormat != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"response_format not supported"}`)
			return
		}

		resp := `{"choices":[{"message":{"role":"assistant","content":"fallback OK"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	c.strategy = "auto"
	c.stderr = io.Discard

	result, err := c.Chat(context.Background(), ChatInput{
		UserPrompt: "test",
		ResponseFormat: &ResponseFormat{
			Type: "json_object",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "fallback OK" {
		t.Errorf("got %q, want %q", result, "fallback OK")
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (first attempt + fallback)", callCount)
	}
}

func TestChatStream(t *testing.T) {
	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			fmt.Fprintln(w, chunk)
			fmt.Fprintln(w)
			if flusher != nil {
				flusher.Flush()
			}
		}
	})
	defer srv.Close()

	tokens, errs := c.ChatStream(context.Background(), ChatInput{
		UserPrompt: "Hi",
	})

	var result strings.Builder
	for tok := range tokens {
		result.WriteString(tok)
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}

	if result.String() != "Hello world" {
		t.Errorf("got %q, want %q", result.String(), "Hello world")
	}
}

func TestChat_AuthorizationHeader(t *testing.T) {
	var receivedAuth string

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		resp := `{"choices":[{"message":{"role":"assistant","content":"OK"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	c.apiKey = "test-key-123"

	_, err := c.Chat(context.Background(), ChatInput{UserPrompt: "Hi"})
	if err != nil {
		t.Fatal(err)
	}

	if receivedAuth != "Bearer test-key-123" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer test-key-123")
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:1234", "http://localhost:1234/v1/chat/completions"},
		{"http://localhost:1234/", "http://localhost:1234/v1/chat/completions"},
		{"http://localhost:1234/v1", "http://localhost:1234/v1/chat/completions"},
		{"http://localhost:1234/v1/", "http://localhost:1234/v1/chat/completions"},
	}

	for _, tt := range tests {
		got := normalizeEndpoint(tt.input)
		if got != tt.want {
			t.Errorf("normalizeEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestChat_MultimodalImages(t *testing.T) {
	var receivedBody []byte

	srv, c := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		resp := `{"choices":[{"message":{"role":"assistant","content":"I see an image"}}]}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, resp)
	})
	defer srv.Close()

	result, err := c.Chat(context.Background(), ChatInput{
		UserPrompt: "What is in this image?",
		Images: []ImageData{
			{MIMEType: "image/jpeg", Base64: "dGVzdA=="},
			{MIMEType: "image/png", Base64: "cG5n"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "I see an image" {
		t.Errorf("got %q", result)
	}

	// Verify the request body has multimodal content array
	var req struct {
		Messages []struct {
			Role    string            `json:"role"`
			Content json.RawMessage   `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(receivedBody, &req); err != nil {
		t.Fatal(err)
	}

	// User message should have content as array (multimodal)
	userMsg := req.Messages[0] // only user message (no system prompt)
	if userMsg.Role != "user" {
		t.Fatalf("role = %q, want user", userMsg.Role)
	}

	var parts []struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		ImageURL *struct {
			URL string `json:"url"`
		} `json:"image_url,omitempty"`
	}
	if err := json.Unmarshal(userMsg.Content, &parts); err != nil {
		t.Fatalf("content is not array: %v", err)
	}

	if len(parts) != 3 {
		t.Fatalf("got %d parts, want 3 (text + 2 images)", len(parts))
	}
	if parts[0].Type != "text" || parts[0].Text != "What is in this image?" {
		t.Errorf("part[0] = %+v", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL == nil {
		t.Errorf("part[1] = %+v", parts[1])
	} else if !strings.HasPrefix(parts[1].ImageURL.URL, "data:image/jpeg;base64,") {
		t.Errorf("part[1] URL = %q", parts[1].ImageURL.URL)
	}
	if parts[2].Type != "image_url" || parts[2].ImageURL == nil {
		t.Errorf("part[2] = %+v", parts[2])
	} else if !strings.HasPrefix(parts[2].ImageURL.URL, "data:image/png;base64,") {
		t.Errorf("part[2] URL = %q", parts[2].ImageURL.URL)
	}
}
