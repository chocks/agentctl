package trace

// Chain verification and completeness checking.
//
// Two independent properties of an append-only chain:
//
//   - Integrity    — VerifyChain: no record was altered, reordered, or deleted.
//   - Completeness — CheckCompleteness: no record is missing (a dropped record
//     leaves a sequence gap).
//
// Both are deliberately mechanical and independently inspectable.

import (
	"fmt"
	"sort"
)

// VerifyResult is the outcome of walking a single chain.
type VerifyResult struct {
	OK       bool   `json:"ok"`
	Records  int    `json:"records"`
	Root     string `json:"root"`                // hash of the first record in the input
	Head     string `json:"head"`                // hash of the last record in the input
	BreakSeq uint64 `json:"break_seq,omitempty"` // seq where verification failed; 0 when OK
	Reason   string `json:"reason,omitempty"`
}

// VerifyChain walks one chain's records and confirms that (1) each record's
// stored hash matches a recomputation of its contents, and (2) each record's
// prev_hash matches the prior record's hash. It reports the exact seq where the
// chain first breaks.
//
// The input is treated as a single chain and sorted by seq; callers with mixed
// chains should GroupByChain first. A slice that starts mid-chain (e.g. a fetch
// over an audit period) still verifies internal consistency — only the first
// record's prev_hash is necessarily unanchored.
func VerifyChain(records []Record) VerifyResult {
	if len(records) == 0 {
		return VerifyResult{OK: true}
	}

	recs := make([]Record, len(records))
	copy(recs, records)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Seq < recs[j].Seq })

	root := recs[0].Hash
	head := recs[len(recs)-1].Hash

	var prevHash string
	for i, rec := range recs {
		want, err := rec.computeHash()
		if err != nil {
			return VerifyResult{OK: false, Records: len(recs), Root: root, BreakSeq: rec.Seq,
				Reason: fmt.Sprintf("hashing record %d: %v", rec.Seq, err)}
		}
		if want != rec.Hash {
			return VerifyResult{OK: false, Records: len(recs), Root: root, BreakSeq: rec.Seq,
				Reason: fmt.Sprintf("record %d content does not match its hash (tampered)", rec.Seq)}
		}
		if i > 0 && rec.PrevHash != prevHash {
			return VerifyResult{OK: false, Records: len(recs), Root: root, BreakSeq: rec.Seq,
				Reason: fmt.Sprintf("record %d prev_hash does not match record %d hash (broken or deleted link)", rec.Seq, recs[i-1].Seq)}
		}
		prevHash = rec.Hash
	}

	return VerifyResult{OK: true, Records: len(recs), Root: root, Head: head}
}

// Gap is a missing sequence number in a chain — a record absent from the
// population.
type Gap struct {
	ChainID    string `json:"chain_id"`
	MissingSeq uint64 `json:"missing_seq"`
}

// CompletenessResult reports whether a record population has sequence gaps.
type CompletenessResult struct {
	Complete bool  `json:"complete"`
	Expected int   `json:"expected"`
	Observed int   `json:"observed"`
	Gaps     []Gap `json:"gaps"`
}

// CheckCompleteness verifies per-chain sequence continuity. For each chain it
// expects a contiguous run from the lowest to the highest observed seq; any
// absent seq in that range is reported as a gap. Expected and Observed are
// summed across chains. Gaps before the lowest observed seq cannot be detected
// by counting alone — chain linkage (VerifyChain) catches deletions there.
func CheckCompleteness(records []Record) CompletenessResult {
	byChain := GroupByChain(records)

	chains := make([]string, 0, len(byChain))
	for id := range byChain {
		chains = append(chains, id)
	}
	sort.Strings(chains)

	res := CompletenessResult{Complete: true, Gaps: []Gap{}}
	for _, id := range chains {
		recs := byChain[id]
		seen := make(map[uint64]bool, len(recs))
		var lo, hi uint64
		for i, r := range recs {
			seen[r.Seq] = true
			if i == 0 || r.Seq < lo {
				lo = r.Seq
			}
			if r.Seq > hi {
				hi = r.Seq
			}
		}
		for s := lo; s <= hi; s++ {
			if !seen[s] {
				res.Gaps = append(res.Gaps, Gap{ChainID: id, MissingSeq: s})
			}
		}
		res.Expected += int(hi - lo + 1)
		res.Observed += len(recs)
	}

	if len(res.Gaps) > 0 {
		res.Complete = false
	}
	return res
}

// GroupByChain partitions records by chain id, preserving input order within
// each chain.
func GroupByChain(records []Record) map[string][]Record {
	out := make(map[string][]Record)
	for _, r := range records {
		out[r.ChainID] = append(out[r.ChainID], r)
	}
	return out
}
