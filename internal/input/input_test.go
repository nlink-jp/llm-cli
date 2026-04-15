package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadUserInput_PromptFlag(t *testing.T) {
	r, err := ReadUserInput("hello", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "hello" {
		t.Errorf("Text = %q, want %q", r.Text, "hello")
	}
	if r.Source != SourceDirect {
		t.Errorf("Source = %d, want SourceDirect", r.Source)
	}
}

func TestReadUserInput_FileFlag(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(f, []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := ReadUserInput("", f, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "from file" {
		t.Errorf("Text = %q, want %q", r.Text, "from file")
	}
	if r.Source != SourceExternal {
		t.Errorf("Source = %d, want SourceExternal", r.Source)
	}
}

func TestReadUserInput_PositionalArgs(t *testing.T) {
	r, err := ReadUserInput("", "", []string{"arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "arg1 arg2" {
		t.Errorf("Text = %q, want %q", r.Text, "arg1 arg2")
	}
	if r.Source != SourceDirect {
		t.Errorf("Source = %d, want SourceDirect", r.Source)
	}
}

func TestReadUserInput_NoInput(t *testing.T) {
	_, err := ReadUserInput("", "", nil)
	if err == nil {
		t.Error("expected error for no input")
	}
}

func TestReadUserInput_PromptPriority(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(f, []byte("from file"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := ReadUserInput("from flag", f, []string{"from args"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Text != "from flag" {
		t.Errorf("Text = %q, want flag value", r.Text)
	}
}

func TestReadSystemPrompt_Text(t *testing.T) {
	text, err := ReadSystemPrompt("system text", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "system text" {
		t.Errorf("got %q, want %q", text, "system text")
	}
}

func TestReadSystemPrompt_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "system.txt")
	if err := os.WriteFile(f, []byte("system from file"), 0o644); err != nil {
		t.Fatal(err)
	}

	text, err := ReadSystemPrompt("", f)
	if err != nil {
		t.Fatal(err)
	}
	if text != "system from file" {
		t.Errorf("got %q, want %q", text, "system from file")
	}
}

func TestReadSystemPrompt_Empty(t *testing.T) {
	text, err := ReadSystemPrompt("", "")
	if err != nil {
		t.Fatal(err)
	}
	if text != "" {
		t.Errorf("got %q, want empty", text)
	}
}

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "lines.txt")
	content := "line1\n\nline2\r\n  line3  \n\n"
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, err := ReadLines(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("lines = %v", lines)
	}
}

func TestSanitizeUTF8(t *testing.T) {
	invalid := "hello\x80world"
	got := sanitizeUTF8(invalid)
	want := "hello\uFFFDworld"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
