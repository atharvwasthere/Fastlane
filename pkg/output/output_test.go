package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONWriter(&buf)

	result := NewResult("ping", "example.com")
	result.Data["latency_ms"] = 42.5
	result.Data["jitter_ms"] = 2.1

	err := writer.WriteResult(result)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected JSON output, got empty string")
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	err = json.Unmarshal([]byte(output), &decoded)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	if decoded["command"] != "ping" {
		t.Errorf("Expected command 'ping', got %v", decoded["command"])
	}
}

func TestTextWriterPingResult(t *testing.T) {
	var buf bytes.Buffer
	writer := NewTextWriter(&buf, false)

	err := writer.WritePingResult(42.5, 2.1, 40.0, 45.0)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected text output, got empty string")
	}

	// Verify key values are present
	if !bytes.Contains([]byte(output), []byte("42.50")) {
		t.Error("Expected latency value in output")
	}
}

func TestTextWriterVerbose(t *testing.T) {
	var buf bytes.Buffer
	writer := NewTextWriter(&buf, true)

	err := writer.WriteVerbose("test message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("test message")) {
		t.Error("Expected verbose message in output")
	}
}

func TestTextWriterVerboseDisabled(t *testing.T) {
	var buf bytes.Buffer
	writer := NewTextWriter(&buf, false)

	err := writer.WriteVerbose("test message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Error("Expected no output when verbose is disabled")
	}
}

func TestNewResult(t *testing.T) {
	result := NewResult("test", "server.com")

	if result.Command != "test" {
		t.Errorf("Expected command 'test', got %v", result.Command)
	}

	if result.Server != "server.com" {
		t.Errorf("Expected server 'server.com', got %v", result.Server)
	}

	if result.Data == nil {
		t.Error("Expected data map to be initialized")
	}
}
