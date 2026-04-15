// Package input handles reading user input from various sources (flags, stdin,
// files) and assembling content for the LLM API request.
package input

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Source indicates whether input came from a trusted or untrusted source.
type Source int

const (
	// SourceDirect is trusted input (CLI flag or positional arg).
	SourceDirect Source = iota
	// SourceExternal is untrusted input (file or stdin).
	SourceExternal
)

// Result holds the text read and its source classification.
type Result struct {
	Text   string
	Source Source
}

// ReadUserInput reads the user prompt from the highest-priority available
// source: flag > file > positional args > stdin.
func ReadUserInput(prompt, filePath string, args []string) (Result, error) {
	// Priority 1: --prompt flag
	if prompt != "" {
		return Result{Text: sanitizeUTF8(prompt), Source: SourceDirect}, nil
	}

	// Priority 2: --file flag (supports "-" for stdin)
	if filePath != "" {
		text, err := readFile(filePath)
		if err != nil {
			return Result{}, fmt.Errorf("read input: %w", err)
		}
		return Result{Text: sanitizeUTF8(text), Source: SourceExternal}, nil
	}

	// Priority 3: positional args
	if len(args) > 0 {
		return Result{Text: sanitizeUTF8(strings.Join(args, " ")), Source: SourceDirect}, nil
	}

	// Priority 4: piped stdin
	if isPiped() {
		text, err := readStdin()
		if err != nil {
			return Result{}, fmt.Errorf("read stdin: %w", err)
		}
		return Result{Text: sanitizeUTF8(text), Source: SourceExternal}, nil
	}

	return Result{}, fmt.Errorf("no input provided: use -p, -f, positional args, or pipe stdin")
}

// ReadSystemPrompt reads the system prompt from a flag value or file.
func ReadSystemPrompt(text, filePath string) (string, error) {
	if text != "" {
		return sanitizeUTF8(text), nil
	}
	if filePath != "" {
		content, err := readFile(filePath)
		if err != nil {
			return "", fmt.Errorf("read system prompt: %w", err)
		}
		return sanitizeUTF8(content), nil
	}
	return "", nil
}

// ReadLines reads all non-empty, trimmed lines from a file or stdin.
func ReadLines(filePath string) ([]string, error) {
	var text string
	var err error

	if filePath != "" && filePath != "-" {
		text, err = readFile(filePath)
	} else {
		text, err = readStdin()
	}
	if err != nil {
		return nil, fmt.Errorf("read lines: %w", err)
	}

	raw := strings.Split(text, "\n")
	var lines []string
	for _, line := range raw {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, sanitizeUTF8(line))
		}
	}
	return lines, nil
}

func readFile(path string) (string, error) {
	if path == "-" {
		return readStdin()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func isPiped() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

func sanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "\uFFFD")
}
