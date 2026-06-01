package trace

// Hash-chained, append-only record envelope.
//
// Each trace line is a Record: the canonical Decision payload plus a small
// envelope (chain id, sequence number, prev_hash, hash) that binds it to its
// predecessor. Tampering with, reordering, or deleting any record invalidates
// every hash that follows, which `agentctl trace verify` detects and reports.
//
// The integrity claim is deliberately simple and independently checkable: the
// hash is SHA-256 over the JSON encoding of the record with its Hash field
// cleared. Because Decision is a struct and Params is json.RawMessage,
// json.Marshal is deterministic, so any third party can recompute the chain.

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chocks/agentctl/pkg/schema"
	"github.com/google/uuid"
)

// memoryChainID labels chains from non-file stores (replay, discard sinks).
const memoryChainID = "memory"

// Record is one line in the append-only, hash-chained trace store.
type Record struct {
	ChainID  string          `json:"chain_id"`
	Seq      uint64          `json:"seq"`
	PrevHash string          `json:"prev_hash"`
	Hash     string          `json:"hash"`
	Decision schema.Decision `json:"decision"`
}

// computeHash returns "sha256:<hex>" over the record's canonical contents: the
// JSON encoding of the record with the Hash field cleared. r is a value copy,
// so clearing Hash here does not affect the caller.
func (r Record) computeHash() (string, error) {
	r.Hash = ""
	data, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("canonicalizing record: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// resolveChainID returns a stable identifier for the chain, persisted next to
// the trace file at <dir>/store.id and created on first use. The id ties every
// record written by this install to a single chain.
func resolveChainID(tracePath string) (string, error) {
	idPath := filepath.Join(filepath.Dir(tracePath), "store.id")
	data, err := os.ReadFile(idPath)
	if err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("reading chain id %s: %w", idPath, err)
	}
	id := uuid.New().String()
	if err := os.WriteFile(idPath, []byte(id+"\n"), 0644); err != nil {
		return "", fmt.Errorf("writing chain id %s: %w", idPath, err)
	}
	return id, nil
}

// readChainHead scans an existing trace file and returns the seq and hash of
// the last well-formed record, so a newly opened store resumes the chain
// across process restarts (the hook runs once per tool call). Legacy
// bare-decision lines and malformed lines are skipped.
func readChainHead(path string) (uint64, string, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("opening trace file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var lastSeq uint64
	var lastHash string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil || rec.Hash == "" {
			continue
		}
		lastSeq = rec.Seq
		lastHash = rec.Hash
	}
	if err := scanner.Err(); err != nil {
		return 0, "", fmt.Errorf("scanning trace file: %w", err)
	}
	return lastSeq, lastHash, nil
}

// ReadRecords reads hash-chained records from a JSON lines file, applying the
// same filter as ReadTraces against each record's embedded decision. Lines
// that are not well-formed records (legacy bare decisions, malformed) are
// skipped. Records are returned in file (append) order.
func ReadRecords(path string, filter TraceFilter) ([]Record, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Record{}, nil
		}
		return nil, fmt.Errorf("reading trace file: %w", err)
	}
	defer func() { _ = f.Close() }()

	var all []Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil || rec.Hash == "" {
			continue
		}
		if !matchesFilter(rec.Decision, filter) {
			continue
		}
		all = append(all, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning trace file: %w", err)
	}

	if filter.Limit > 0 && len(all) > filter.Limit {
		return all[len(all)-filter.Limit:], nil
	}
	return all, nil
}
