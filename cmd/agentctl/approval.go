package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agentctl/agentctl/pkg/schema"
)

type approvalStatus string

const (
	approvalStatusPending  approvalStatus = "pending"
	approvalStatusApproved approvalStatus = "approved"
	approvalStatusDenied   approvalStatus = "denied"
)

type approvalRecord struct {
	ApprovalID  string         `json:"approval_id"`
	TraceID     string         `json:"trace_id"`
	SessionID   string         `json:"session_id"`
	Action      schema.Action  `json:"action"`
	Status      approvalStatus `json:"status"`
	Reason      string         `json:"reason"`
	RequestedAt time.Time      `json:"requested_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy  string         `json:"resolved_by,omitempty"`
}

func approvalFilePath() string {
	return agentctlDataPath("AGENTCTL_APPROVAL_FILE", "approvals.jsonl")
}

func recordApprovalForDecision(path string, decision *schema.Decision) error {
	if decision == nil || !decision.ApprovalRequired {
		return nil
	}

	return appendApproval(path, approvalRecord{
		ApprovalID:  decision.TraceID,
		TraceID:     decision.TraceID,
		SessionID:   decision.Request.Context.SessionID,
		Action:      decision.Request.Action,
		Status:      approvalStatusPending,
		Reason:      decision.Reason,
		RequestedAt: decision.Timestamp,
	})
}

func appendApproval(path string, record approvalRecord) error {
	ensureDir(filepath.Dir(path))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening approval file: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encoding approval record: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing approval record: %w", err)
	}

	return nil
}

func readApprovals(path string, status approvalStatus) ([]approvalRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []approvalRecord{}, nil
		}
		return nil, fmt.Errorf("opening approval file: %w", err)
	}
	defer func() { _ = f.Close() }()

	latest := map[string]approvalRecord{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024) // allow lines up to 4 MB
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 {
			continue
		}
		var record approvalRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		latest[record.ApprovalID] = record
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning approvals: %w", err)
	}

	records := make([]approvalRecord, 0, len(latest))
	for _, record := range latest {
		if status != "" && record.Status != status {
			continue
		}
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].RequestedAt.After(records[j].RequestedAt)
	})

	return records, nil
}

func resolveApproval(path, approvalID string, status approvalStatus, resolvedBy string) (approvalRecord, error) {
	records, err := readApprovals(path, "")
	if err != nil {
		return approvalRecord{}, err
	}

	for _, record := range records {
		if record.ApprovalID != approvalID {
			continue
		}
		if record.Status != approvalStatusPending {
			return approvalRecord{}, fmt.Errorf("approval %q is already %s", approvalID, record.Status)
		}

		now := time.Now()
		record.Status = status
		record.ResolvedBy = resolvedBy
		record.ResolvedAt = &now
		if err := appendApproval(path, record); err != nil {
			return approvalRecord{}, err
		}
		return record, nil
	}

	return approvalRecord{}, fmt.Errorf("approval %q not found", approvalID)
}

func cmdApproval() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: agentctl approval [list|approve|deny]")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "list":
		cmdApprovalList()
	case "approve":
		cmdApprovalResolve(approvalStatusApproved)
	case "deny":
		cmdApprovalResolve(approvalStatusDenied)
	default:
		fmt.Fprintf(os.Stderr, "unknown approval command: %s\n", os.Args[2])
		os.Exit(1)
	}
}

func cmdApprovalList() {
	status := approvalStatus(stringFlagValue("--status", ""))
	records, err := readApprovals(approvalFilePath(), status)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-36s %-18s %-18s %-10s %s\n", "APPROVAL ID", "SESSION", "ACTION", "STATUS", "REASON")
	fmt.Println(strings.Repeat("-", 100))
	for _, record := range records {
		fmt.Printf("%-36s %-18s %-18s %-10s %s\n",
			record.ApprovalID,
			truncate(record.SessionID, 18),
			record.Action,
			record.Status,
			truncate(record.Reason, 40),
		)
	}
	fmt.Printf("\n%d approvals shown\n", len(records))
}

func cmdApprovalResolve(status approvalStatus) {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "usage: agentctl approval %s <approval_id> [--by name]\n", status)
		os.Exit(1)
	}

	record, err := resolveApproval(approvalFilePath(), os.Args[3], status, stringFlagValue("--by", "local-operator"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error encoding approval: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}
