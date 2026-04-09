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

// writeTraces writes a slice of decisions as JSON lines to path.
func writeTraces(t *testing.T, path string, decisions []schema.Decision) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, d := range decisions {
		line, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestReadTracesLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	decisions := []schema.Decision{
		{TraceID: "a", Timestamp: time.Unix(1, 0)},
		{TraceID: "b", Timestamp: time.Unix(2, 0)},
		{TraceID: "c", Timestamp: time.Unix(3, 0)},
	}
	writeTraces(t, path, decisions)

	got, err := ReadTraces(path, TraceFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ReadTraces: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(got))
	}
	if got[0].TraceID != "b" || got[1].TraceID != "c" {
		t.Fatalf("expected tail [b c], got [%s %s]", got[0].TraceID, got[1].TraceID)
	}
}

func TestReadTracesMissingFileReturnsEmpty(t *testing.T) {
	got, err := ReadTraces(filepath.Join(t.TempDir(), "missing.jsonl"), TraceFilter{})
	if err != nil {
		t.Fatalf("ReadTraces: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d entries", len(got))
	}
}

func TestReadTracesFilter(t *testing.T) {
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	installParams, _ := json.Marshal(schema.InstallPackageParams{Manager: "pip", Package: "requests"})
	otherParams, _ := json.Marshal(schema.InstallPackageParams{Manager: "pip", Package: "numpy"})

	decisions := []schema.Decision{
		{
			TraceID:   "t1",
			Verdict:   schema.VerdictAllow,
			Timestamp: base,
			Request: schema.ActionRequest{
				Action:  schema.ActionInstallPackage,
				Params:  installParams,
				Context: schema.RequestContext{SessionID: "sess-a"},
			},
		},
		{
			TraceID:   "t2",
			Verdict:   schema.VerdictDeny,
			Timestamp: base.Add(time.Hour),
			Request: schema.ActionRequest{
				Action:  schema.ActionRunCode,
				Context: schema.RequestContext{SessionID: "sess-b"},
			},
		},
		{
			TraceID:   "t3",
			Verdict:   schema.VerdictAllow,
			Timestamp: base.Add(2 * time.Hour),
			Request: schema.ActionRequest{
				Action:  schema.ActionInstallPackage,
				Params:  otherParams,
				Context: schema.RequestContext{SessionID: "sess-a"},
			},
		},
	}

	path := filepath.Join(t.TempDir(), "traces.jsonl")
	writeTraces(t, path, decisions)

	tests := []struct {
		name   string
		filter TraceFilter
		wantID []string
	}{
		{
			name:   "no filter returns all",
			filter: TraceFilter{},
			wantID: []string{"t1", "t2", "t3"},
		},
		{
			name:   "filter by session",
			filter: TraceFilter{SessionID: "sess-a"},
			wantID: []string{"t1", "t3"},
		},
		{
			name:   "filter by verdict deny",
			filter: TraceFilter{Verdict: schema.VerdictDeny},
			wantID: []string{"t2"},
		},
		{
			name:   "filter by action run_code",
			filter: TraceFilter{Action: schema.ActionRunCode},
			wantID: []string{"t2"},
		},
		{
			name:   "filter by since excludes earlier entries",
			filter: TraceFilter{Since: base.Add(30 * time.Minute)},
			wantID: []string{"t2", "t3"},
		},
		{
			name:   "filter by until excludes later entries",
			filter: TraceFilter{Until: base.Add(90 * time.Minute)},
			wantID: []string{"t1", "t2"},
		},
		{
			name:   "filter by package name",
			filter: TraceFilter{Package: "requests"},
			wantID: []string{"t1"},
		},
		{
			name:   "limit 1 returns only last entry",
			filter: TraceFilter{Limit: 1},
			wantID: []string{"t3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadTraces(path, tc.filter)
			if err != nil {
				t.Fatalf("ReadTraces: %v", err)
			}
			if len(got) != len(tc.wantID) {
				t.Fatalf("expected %d traces, got %d", len(tc.wantID), len(got))
			}
			for i, id := range tc.wantID {
				if got[i].TraceID != id {
					t.Errorf("result[%d]: want TraceID %q, got %q", i, id, got[i].TraceID)
				}
			}
		})
	}
}
