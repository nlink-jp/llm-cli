package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		input   string
		want    Mode
		wantErr bool
	}{
		{"text", ModeText, false},
		{"", ModeText, false},
		{"json", ModeJSON, false},
		{"jsonl", ModeJSONL, false},
		{"xml", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseMode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseMode(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestWriteText(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeText)

	if err := f.Write("Hello world"); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "Hello world\n" {
		t.Errorf("got %q, want %q", got, "Hello world\n")
	}
}

func TestWriteJSON_ValidJSON(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeJSON)

	if err := f.Write(`{"key": "value"}`); err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want %q", result["key"], "value")
	}
}

func TestWriteJSON_WithThinkTags(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeJSON)

	input := `<think>Let me think about this...</think>{"result": "clean"}`
	if err := f.Write(input); err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["result"] != "clean" {
		t.Errorf("result = %q, want %q", result["result"], "clean")
	}
}

func TestWriteJSON_NoJSON(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeJSON)

	if err := f.Write("just plain text"); err != nil {
		t.Fatal(err)
	}

	// Should be encoded as JSON string
	got := strings.TrimSpace(buf.String())
	if got != `"just plain text"` {
		t.Errorf("got %s, want JSON-encoded string", got)
	}
}

func TestWriteJSONL(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeJSONL)

	// Success case
	if err := f.WriteJSONL("input1", "output1", nil); err != nil {
		t.Fatal(err)
	}

	var entry struct {
		Input  string  `json:"input"`
		Output *string `json:"output"`
		Error  *string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if entry.Input != "input1" {
		t.Errorf("Input = %q, want %q", entry.Input, "input1")
	}
	if entry.Output == nil || *entry.Output != "output1" {
		t.Errorf("Output = %v, want %q", entry.Output, "output1")
	}
	if entry.Error != nil {
		t.Errorf("Error = %v, want nil", entry.Error)
	}
}

func TestWriteJSONL_Error(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeJSONL)

	if err := f.WriteJSONL("input1", "", errors.New("test error")); err != nil {
		t.Fatal(err)
	}

	var entry struct {
		Input  string  `json:"input"`
		Output *string `json:"output"`
		Error  *string `json:"error"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Output != nil {
		t.Errorf("Output = %v, want nil", entry.Output)
	}
	if entry.Error == nil || *entry.Error != "test error" {
		t.Errorf("Error = %v, want %q", entry.Error, "test error")
	}
}

func TestStreamingOutput(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf, ModeText)

	tokens := []string{"Hello", " ", "world", "!"}
	for _, tok := range tokens {
		if err := f.WriteText(tok); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Newline(); err != nil {
		t.Fatal(err)
	}

	if got := buf.String(); got != "Hello world!\n" {
		t.Errorf("got %q, want %q", got, "Hello world!\n")
	}
}
