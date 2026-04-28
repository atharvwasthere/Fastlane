package output

import (
	"encoding/json"
	"io"
)

// JSONWriter formats and writes test results as JSON
type JSONWriter struct {
	writer io.Writer
}

// NewJSONWriter creates a new JSON output writer
func NewJSONWriter(w io.Writer) *JSONWriter {
	return &JSONWriter{writer: w}
}

// WriteResult marshals a result object to JSON and writes it
func (j *JSONWriter) WriteResult(result interface{}) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	_, err = j.writer.Write(data)
	return err
}

// Result is a generic test result wrapper for JSON output
type Result struct {
	Timestamp string                 `json:"timestamp"`
	Command   string                 `json:"command"`
	Server    string                 `json:"server"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
}

// NewResult creates a new JSON result wrapper
func NewResult(command, server string) *Result {
	return &Result{
		Command: command,
		Server:  server,
		Data:    make(map[string]interface{}),
	}
}
