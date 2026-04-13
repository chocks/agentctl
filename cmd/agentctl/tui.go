package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	tuiapp "github.com/chocks/agentctl/cmd/agentctl/tui"
	"github.com/chocks/agentctl/pkg/config"
)

func cmdTUI(paths config.Paths) {
	loader := func() ([]tuiapp.Approval, error) {
		records, err := readApprovals(paths.Approvals, approvalStatusPending)
		if err != nil {
			return nil, err
		}

		approvals := make([]tuiapp.Approval, 0, len(records))
		for _, record := range records {
			approvals = append(approvals, tuiapp.Approval{
				ID:          record.ApprovalID,
				Action:      string(record.Action),
				Reason:      record.Reason,
				Status:      string(record.Status),
				RequestedAt: record.RequestedAt,
			})
		}
		return approvals, nil
	}

	resolver := func(approvalID, status string) error {
		_, err := resolveApproval(paths.Approvals, approvalID, approvalStatus(status), "tui")
		return err
	}

	policyChecker := func() error {
		_, err := paths.LoadPolicy()
		return err
	}

	program := tea.NewProgram(
		tuiapp.New(paths.Traces, loader, resolver, policyChecker),
		tea.WithAltScreen(),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
