// Package trace implements the append-only trace store for agentctl.
// Every gate decision is recorded here for debugging, replay, and audit.
package trace

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/chocks/agentctl/pkg/schema"
)

// Store is the append-only, hash-chained trace storage backend. It is the
// single authority for the chain: it assigns each record a sequence number and
// links it to its predecessor by hash.
type Store struct {
	writer   io.Writer
	mu       sync.Mutex
	chainID  string
	lastSeq  uint64
	lastHash string
}

// NewFileStore creates a trace store that appends hash-chained records to a
// file. It resumes any existing chain in the file so records written across
// separate process invocations (the hook runs once per tool call) stay linked.
func NewFileStore(path string) (*Store, error) {
	chainID, err := resolveChainID(path)
	if err != nil {
		return nil, err
	}
	lastSeq, lastHash, err := readChainHead(path)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening trace file: %w", err)
	}
	return &Store{writer: f, chainID: chainID, lastSeq: lastSeq, lastHash: lastHash}, nil
}

// NewWriterStore creates a trace store that writes to any io.Writer.
// Useful for stdout, testing, or discarding (replay). Its chain starts fresh.
func NewWriterStore(w io.Writer) *Store {
	return &Store{writer: w, chainID: memoryChainID}
}

// Record appends a decision to the trace store as the next link in the chain.
// Trace failures are logged but never block the gate. The chain head is only
// advanced after a successful write, so a failed write cannot leave a dangling
// prev_hash on disk.
func (s *Store) Record(d *schema.Decision) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec := Record{
		ChainID:  s.chainID,
		Seq:      s.lastSeq + 1,
		PrevHash: s.lastHash,
		Decision: *d,
	}
	hash, err := rec.computeHash()
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl: trace hash error: %v\n", err)
		return
	}
	rec.Hash = hash

	data, err := json.Marshal(rec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentctl: trace marshal error: %v\n", err)
		return
	}

	data = append(data, '\n')
	if _, err := s.writer.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "agentctl: trace write error: %v\n", err)
		return
	}

	s.lastSeq = rec.Seq
	s.lastHash = rec.Hash
}

// ── Query support (for replay and audit) ────────────────────────────────────

// TraceFilter defines search criteria for traces.
type TraceFilter struct {
	SessionID string
	Action    schema.Action
	Verdict   schema.Verdict
	Since     time.Time
	Until     time.Time
	Package   string // for install_package queries
	Limit     int
}

// ReadTraces reads and filters traces from a JSON lines file.
func ReadTraces(path string, filter TraceFilter) ([]schema.Decision, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []schema.Decision{}, nil
		}
		return nil, fmt.Errorf("reading trace file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	var all []schema.Decision
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024) // allow lines up to 4 MB
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Lines are hash-chained records; unwrap to the embedded decision.
		// Fall back to legacy bare-decision lines for backward compatibility.
		var d schema.Decision
		var rec Record
		if err := json.Unmarshal(line, &rec); err == nil && rec.Hash != "" {
			d = rec.Decision
		} else if err := json.Unmarshal(line, &d); err != nil {
			continue // skip malformed lines
		}

		if !matchesFilter(d, filter) {
			continue
		}

		all = append(all, d)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning trace file: %w", err)
	}

	if filter.Limit > 0 && len(all) > filter.Limit {
		return all[len(all)-filter.Limit:], nil
	}

	return all, nil
}

func matchesFilter(d schema.Decision, f TraceFilter) bool {
	if f.SessionID != "" && d.Request.Context.SessionID != f.SessionID {
		return false
	}
	if f.Action != "" && d.Request.Action != f.Action {
		return false
	}
	if f.Verdict != "" && d.Verdict != f.Verdict {
		return false
	}
	if !f.Since.IsZero() && d.Timestamp.Before(f.Since) {
		return false
	}
	if !f.Until.IsZero() && d.Timestamp.After(f.Until) {
		return false
	}
	if f.Package != "" {
		if d.Request.Action != schema.ActionInstallPackage {
			return false
		}
		var params schema.InstallPackageParams
		if err := json.Unmarshal(d.Request.Params, &params); err == nil && params.Package != f.Package {
			return false
		}
	}
	return true
}
