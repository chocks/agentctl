package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentctl/agentctl/pkg/schema"
)

func TestReadTracesReturnsRecentTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "traces.jsonl")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		payload, err := json.Marshal(schema.Decision{
			TraceID:   fmt.Sprintf("%c", 'a'+i),
			Timestamp: time.Unix(int64(i), 0),
		})
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}
		if _, err := f.Write(append(payload, '\n')); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	got, err := ReadTraces(path, TraceFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ReadTraces() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(got))
	}
	if got[0].TraceID != "b" || got[1].TraceID != "c" {
		t.Fatalf("expected tail traces [b c], got [%s %s]", got[0].TraceID, got[1].TraceID)
	}
}

func TestReadTracesMissingFileReturnsEmpty(t *testing.T) {
	got, err := ReadTraces(filepath.Join(t.TempDir(), "missing.jsonl"), TraceFilter{})
	if err != nil {
		t.Fatalf("ReadTraces() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d entries", len(got))
	}
}
