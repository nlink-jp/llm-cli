// Package output handles formatting and writing LLM responses in text, JSON,
// and JSONL formats, using nlk/jsonfix for JSON repair and nlk/strip for
// thinking tag removal.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nlink-jp/nlk/jsonfix"
	"github.com/nlink-jp/nlk/strip"
)

// Mode represents the output format.
type Mode int

const (
	ModeText Mode = iota
	ModeJSON
	ModeJSONL
)

// ParseMode converts a string format name to a Mode.
func ParseMode(s string) (Mode, error) {
	switch s {
	case "text", "":
		return ModeText, nil
	case "json":
		return ModeJSON, nil
	case "jsonl":
		return ModeJSONL, nil
	default:
		return 0, fmt.Errorf("unknown format: %q (use text, json, or jsonl)", s)
	}
}

// Formatter writes formatted output to a writer.
type Formatter struct {
	w    io.Writer
	mode Mode
}

// NewFormatter creates a new Formatter.
func NewFormatter(w io.Writer, mode Mode) *Formatter {
	return &Formatter{w: w, mode: mode}
}

// Write writes the full response in the configured format.
func (f *Formatter) Write(response string) error {
	switch f.mode {
	case ModeText:
		return f.writeText(response)
	case ModeJSON:
		return f.writeJSON(response)
	default:
		return fmt.Errorf("use WriteJSONL for jsonl format")
	}
}

// WriteText writes a single token to the output (for streaming).
func (f *Formatter) WriteText(token string) error {
	_, err := fmt.Fprint(f.w, token)
	return err
}

// Newline writes a trailing newline (after streaming completes).
func (f *Formatter) Newline() error {
	_, err := fmt.Fprintln(f.w)
	return err
}

// WriteJSONL writes a single batch result in JSONL format.
func (f *Formatter) WriteJSONL(input, output string, outputErr error) error {
	entry := jsonlEntry{Input: input}
	if outputErr != nil {
		errStr := outputErr.Error()
		entry.Error = &errStr
	} else {
		entry.Output = &output
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal jsonl: %w", err)
	}
	_, err = fmt.Fprintln(f.w, string(data))
	return err
}

func (f *Formatter) writeText(response string) error {
	_, err := fmt.Fprintln(f.w, response)
	return err
}

func (f *Formatter) writeJSON(response string) error {
	// Strip thinking tags before JSON extraction
	cleaned := strip.ThinkTags(response)

	// Extract and repair JSON using nlk/jsonfix
	extracted, err := jsonfix.Extract(cleaned)
	if err != nil {
		// If no JSON found, encode the full response as a JSON string
		data, _ := json.Marshal(response)
		_, writeErr := fmt.Fprintln(f.w, string(data))
		return writeErr
	}

	// Pretty-print the extracted JSON
	var pretty json.RawMessage
	if err := json.Unmarshal([]byte(extracted), &pretty); err != nil {
		_, writeErr := fmt.Fprintln(f.w, extracted)
		return writeErr
	}

	indented, err := json.MarshalIndent(pretty, "", "  ")
	if err != nil {
		_, writeErr := fmt.Fprintln(f.w, extracted)
		return writeErr
	}
	_, writeErr := fmt.Fprintln(f.w, string(indented))
	return writeErr
}

type jsonlEntry struct {
	Input  string  `json:"input"`
	Output *string `json:"output"`
	Error  *string `json:"error"`
}
