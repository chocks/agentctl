package trace

import (
	"path/filepath"
	"testing"

	"github.com/chocks/agentctl/pkg/schema"
)

// recordAll writes a sequence of decisions to a fresh file store and returns
// the resulting on-disk records.
func recordAll(t *testing.T, path string, decisions ...*schema.Decision) []Record {
	t.Helper()
	s, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	for _, d := range decisions {
		s.Record(d)
	}
	recs, err := ReadRecords(path, TraceFilter{})
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	return recs
}

func TestRecordBuildsLinkedChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recs := recordAll(t, path,
		&schema.Decision{TraceID: "a", Verdict: schema.VerdictAllow},
		&schema.Decision{TraceID: "b", Verdict: schema.VerdictDeny},
		&schema.Decision{TraceID: "c", Verdict: schema.VerdictEscalate},
	)

	if len(recs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(recs))
	}
	for i, r := range recs {
		if r.Seq != uint64(i+1) {
			t.Errorf("record %d: expected seq %d, got %d", i, i+1, r.Seq)
		}
		if r.Hash == "" {
			t.Errorf("record %d: empty hash", i)
		}
		if r.ChainID == "" {
			t.Errorf("record %d: empty chain_id", i)
		}
		if r.ChainID != recs[0].ChainID {
			t.Errorf("record %d: chain_id %q != %q", i, r.ChainID, recs[0].ChainID)
		}
	}
	if recs[0].PrevHash != "" {
		t.Errorf("genesis record prev_hash should be empty, got %q", recs[0].PrevHash)
	}
	if recs[1].PrevHash != recs[0].Hash {
		t.Errorf("record 1 prev_hash %q != record 0 hash %q", recs[1].PrevHash, recs[0].Hash)
	}
	if recs[2].PrevHash != recs[1].Hash {
		t.Errorf("record 2 prev_hash %q != record 1 hash %q", recs[2].PrevHash, recs[1].Hash)
	}
}

func TestChainContinuesAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")

	s1, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	s1.Record(&schema.Decision{TraceID: "a"})

	// A second process opening the same file must resume the chain, not restart it.
	s2, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore (reopen): %v", err)
	}
	s2.Record(&schema.Decision{TraceID: "b"})

	recs, err := ReadRecords(path, TraceFilter{})
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	if recs[1].Seq != 2 {
		t.Errorf("expected resumed seq 2, got %d", recs[1].Seq)
	}
	if recs[1].PrevHash != recs[0].Hash {
		t.Errorf("chain not linked across reopen: prev %q != %q", recs[1].PrevHash, recs[0].Hash)
	}
	if res := VerifyChain(recs); !res.OK {
		t.Errorf("reopened chain should verify, broke at seq %d: %s", res.BreakSeq, res.Reason)
	}
}

func TestVerifyChainOK(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recs := recordAll(t, path,
		&schema.Decision{TraceID: "a"},
		&schema.Decision{TraceID: "b"},
	)
	res := VerifyChain(recs)
	if !res.OK {
		t.Fatalf("expected OK, broke at seq %d: %s", res.BreakSeq, res.Reason)
	}
	if res.Records != 2 {
		t.Errorf("expected 2 records counted, got %d", res.Records)
	}
	if res.Root != recs[0].Hash {
		t.Errorf("root %q != first hash %q", res.Root, recs[0].Hash)
	}
	if res.Head != recs[1].Hash {
		t.Errorf("head %q != last hash %q", res.Head, recs[1].Hash)
	}
}

func TestVerifyChainDetectsTamperedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recs := recordAll(t, path,
		&schema.Decision{TraceID: "a", Verdict: schema.VerdictDeny},
		&schema.Decision{TraceID: "b", Verdict: schema.VerdictDeny},
	)

	// Flip a recorded verdict but leave the stored hash untouched.
	// Recomputing the hash must expose the edit.
	recs[1].Decision.Verdict = schema.VerdictAllow

	res := VerifyChain(recs)
	if res.OK {
		t.Fatal("expected chain to break on tampered content")
	}
	if res.BreakSeq != 2 {
		t.Errorf("expected break at seq 2, got %d", res.BreakSeq)
	}
}

func TestVerifyChainDetectsDeletedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	recs := recordAll(t, path,
		&schema.Decision{TraceID: "a"},
		&schema.Decision{TraceID: "b"},
		&schema.Decision{TraceID: "c"},
	)

	// Drop the middle record. The survivor's prev_hash no longer matches.
	pruned := []Record{recs[0], recs[2]}

	res := VerifyChain(pruned)
	if res.OK {
		t.Fatal("expected chain to break when a record is deleted")
	}
	if res.BreakSeq != 3 {
		t.Errorf("expected break at seq 3 (broken linkage), got %d", res.BreakSeq)
	}
}

func TestCheckCompletenessDetectsGap(t *testing.T) {
	chain := "chain-1"
	recs := []Record{
		{ChainID: chain, Seq: 1},
		{ChainID: chain, Seq: 2},
		{ChainID: chain, Seq: 4}, // seq 3 never arrived (dropped shipment)
	}
	comp := CheckCompleteness(recs)
	if comp.Complete {
		t.Fatal("expected completeness check to fail on a sequence gap")
	}
	if comp.Expected != 4 || comp.Observed != 3 {
		t.Errorf("expected 4 expected / 3 observed, got %d / %d", comp.Expected, comp.Observed)
	}
	if len(comp.Gaps) != 1 || comp.Gaps[0].MissingSeq != 3 || comp.Gaps[0].ChainID != chain {
		t.Errorf("expected one gap at seq 3 on %q, got %+v", chain, comp.Gaps)
	}
}

func TestCheckCompletenessClean(t *testing.T) {
	recs := []Record{
		{ChainID: "c", Seq: 1},
		{ChainID: "c", Seq: 2},
		{ChainID: "c", Seq: 3},
	}
	comp := CheckCompleteness(recs)
	if !comp.Complete {
		t.Fatalf("expected complete, got gaps %+v", comp.Gaps)
	}
}
