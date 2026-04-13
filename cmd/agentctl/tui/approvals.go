package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type Approval struct {
	ID          string
	Action      string
	Reason      string
	Status      string
	RequestedAt time.Time
}

type ApprovalLoader func() ([]Approval, error)
type ApprovalResolver func(approvalID, status string) error

type approvalsLoadedMsg struct {
	approvals []Approval
	err       error
}

type approvalResolvedMsg struct {
	status string
	id     string
	err    error
}

type approvalsModel struct {
	loader   ApprovalLoader
	resolver ApprovalResolver
	table    table.Model
	rows     []Approval
	err      error
	info     string
	width    int
	height   int
}

func newApprovalsModel(loader ApprovalLoader, resolver ApprovalResolver) approvalsModel {
	columns := []table.Column{
		{Title: "ID", Width: 10},
		{Title: "ACTION", Width: 18},
		{Title: "REASON", Width: 34},
		{Title: "STATUS", Width: 12},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(12),
	)

	return approvalsModel{
		loader:   loader,
		resolver: resolver,
		table:    tbl,
		info:     "Loading approvals…",
	}
}

func (m approvalsModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		approvals, err := m.loader()
		if err != nil {
			return approvalsLoadedMsg{err: err}
		}

		sort.Slice(approvals, func(i, j int) bool {
			return approvals[i].RequestedAt.After(approvals[j].RequestedAt)
		})

		return approvalsLoadedMsg{approvals: approvals}
	}
}

func (m approvalsModel) Update(msg tea.Msg) (approvalsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case approvalsLoadedMsg:
		m.err = msg.err
		if msg.err != nil {
			m.rows = nil
			m.table.SetRows([]table.Row{})
			return m, nil
		}

		m.rows = msg.approvals
		rows := make([]table.Row, 0, len(msg.approvals))
		for _, approval := range msg.approvals {
			rows = append(rows, table.Row{
				truncateApprovalID(approval.ID),
				approval.Action,
				truncateText(approval.Reason, 34),
				approval.Status,
			})
		}
		m.table.SetRows(rows)
		if len(rows) == 0 {
			m.info = "No pending approvals."
		} else {
			m.info = fmt.Sprintf("%d approvals loaded", len(rows))
		}
		return m, nil

	case approvalResolvedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		m.err = nil
		m.info = fmt.Sprintf("%s %s", msg.id, msg.status)
		return m, m.reloadCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "j":
			m.table.MoveDown(1)
			return m, nil
		case "k":
			m.table.MoveUp(1)
			return m, nil
		case "a":
			if selected, ok := m.selected(); ok {
				return m, m.resolveCmd(selected.ID, "approved")
			}
			return m, nil
		case "d":
			if selected, ok := m.selected(); ok {
				return m, m.resolveCmd(selected.ID, "denied")
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *approvalsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if width > 0 {
		m.table.SetWidth(width)
	}
	if height > 0 {
		m.table.SetHeight(maxInt(5, height-2))
	}
}

func (m approvalsModel) HelpText() string {
	return "[tab] switch  [q] quit  [a] approve  [d] deny  [j/k] move"
}

func (m approvalsModel) View(styles Styles) string {
	if m.err != nil {
		return styles.ErrorBanner.Render("Approval error: " + m.err.Error())
	}
	if len(m.rows) == 0 {
		return styles.Info.Render(m.info)
	}
	return m.table.View()
}

func (m approvalsModel) resolveCmd(approvalID, status string) tea.Cmd {
	return func() tea.Msg {
		err := m.resolver(approvalID, status)
		return approvalResolvedMsg{
			status: status,
			id:     truncateApprovalID(approvalID),
			err:    err,
		}
	}
}

func (m approvalsModel) selected() (Approval, bool) {
	if len(m.rows) == 0 {
		return Approval{}, false
	}
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.rows) {
		return Approval{}, false
	}
	return m.rows[cursor], true
}

func truncateApprovalID(id string) string {
	return truncateText(id, 10)
}

func truncateText(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
