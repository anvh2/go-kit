package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestNewLogrWritesJSONLines(t *testing.T) {
	var out bytes.Buffer
	logger, err := NewLogr(LogrConfig{Output: &out, MessageKey: "event"})
	if err != nil {
		t.Fatal(err)
	}
	logger = logger.WithName("Service")
	logger.V(1).Info("hidden.at.info.level")
	logger.Info("tool.call", "requestId", "r-1", "durationMs", 3)
	logger.Error(errors.New("boom"), "Store.Get")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d: %q", len(lines), out.String())
	}
	var info, failure map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &info); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &failure); err != nil {
		t.Fatal(err)
	}
	if info["level"] != "info" || info["event"] != "tool.call" || info["logger"] != "Service" || info["requestId"] != "r-1" {
		t.Errorf("unexpected info line: %v", info)
	}
	ts, _ := info["ts"].(string)
	if len(ts) != len("2026-11-20T00:00:00.000Z") || !strings.HasSuffix(ts, "Z") {
		t.Errorf("ts is not UTC with milliseconds: %q", ts)
	}
	if failure["level"] != "error" || failure["event"] != "Store.Get" || failure["error"] != "boom" {
		t.Errorf("unexpected error line: %v", failure)
	}
}

// The guard settings keep sensitive data out of logs; if they break, customer data can end up in logs.
func TestNewLogrGuardKeepsSensitiveDataOut(t *testing.T) {
	var out bytes.Buffer
	logger, err := NewLogr(LogrConfig{
		Output:         &out,
		AllowedKeys:    []string{"requestId"},
		OmitErrorText:  true,
		MessagePattern: `^[A-Za-z][A-Za-z0-9_.]*$`,
	})
	if err != nil {
		t.Fatal(err)
	}
	logger.WithValues("session", "secret-session").Info("tool.call", "requestId", "r-1",
		"record", "Customer A owes 1.200.000", "rows", []int{1}, "nested", map[string]string{"k": "secret"})
	logger.Error(errors.New("row 4711: Customer A"), "Source.Status")
	logger.Info("customer Customer A paid")

	text := out.String()
	for _, leaked := range []string{"Customer A", "1.200.000", "secret", "4711"} {
		if strings.Contains(text, leaked) {
			t.Errorf("log carries %q: %s", leaked, text)
		}
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	var first, last map[string]any
	_ = json.Unmarshal([]byte(lines[0]), &first)
	_ = json.Unmarshal([]byte(lines[2]), &last)
	if first["requestId"] != "r-1" || fmt.Sprint(first["droppedKeys"]) != "[record rows nested]" {
		t.Errorf("unexpected first line: %v", first)
	}
	if last["msg"] != "invalid_message" {
		t.Errorf("a formatted message was not replaced: %v", last)
	}
}

func TestNewLogrRejectsBadConfig(t *testing.T) {
	if _, err := NewLogr(LogrConfig{Level: "loud"}); err == nil {
		t.Error("want an error for an unknown level")
	}
	if _, err := NewLogr(LogrConfig{Encoding: "xml"}); err == nil {
		t.Error("want an error for an unknown encoding")
	}
	if _, err := NewLogr(LogrConfig{MessagePattern: "("}); err == nil {
		t.Error("want an error for an invalid message pattern")
	}
}
