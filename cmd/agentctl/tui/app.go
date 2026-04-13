package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const refreshInterval = 3 * time.Second

type PolicyChecker func() error

type tickMsg struct{}

type policyCheckedMsg struct {
	err error
}

type Model struct {
	styles        Styles
	traces        tracesModel
	approvals     approvalsModel
	activeTab     int
	width         int
	height        int
	policyChecker PolicyChecker
	policyErr     error
}

func New(tracePath string, approvalLoader ApprovalLoader, approvalResolver ApprovalResolver, policyChecker PolicyChecker) Model {
	return Model{
		styles:        DefaultStyles(),
		traces:        newTracesModel(tracePath),
		approvals:     newApprovalsModel(approvalLoader, approvalResolver),
		policyChecker: policyChecker,
		width:         100,
		height:        30,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(checkPolicyCmd(m.policyChecker), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeChildren()
		return m, nil

	case tickMsg:
		return m, tea.Batch(checkPolicyCmd(m.policyChecker), tickCmd())

	case policyCheckedMsg:
		m.policyErr = msg.err
		if msg.err != nil {
			return m, nil
		}
		return m, tea.Batch(m.traces.reloadCmd(), m.approvals.reloadCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			return m, nil
		}

		if m.policyErr != nil {
			return m, nil
		}

		if m.activeTab == 0 {
			var cmd tea.Cmd
			m.traces, cmd = m.traces.Update(msg)
			return m, cmd
		}

		var cmd tea.Cmd
		m.approvals, cmd = m.approvals.Update(msg)
		return m, cmd
	}

	if m.policyErr != nil {
		return m, nil
	}

	var traceCmd tea.Cmd
	var approvalCmd tea.Cmd
	m.traces, traceCmd = m.traces.Update(msg)
	m.approvals, approvalCmd = m.approvals.Update(msg)
	return m, tea.Batch(traceCmd, approvalCmd)
}

func (m Model) View() string {
	var body string
	if m.policyErr != nil {
		body = m.styles.ErrorBanner.Render(fmt.Sprintf(
			"Policy error: %v\nFix ~/.agentctl/policy.yaml to re-enable the UI.",
			m.policyErr,
		))
	} else if m.activeTab == 0 {
		body = m.traces.View(m.styles)
	} else {
		body = m.approvals.View(m.styles)
	}

	status := "[q] quit"
	if m.policyErr == nil {
		if m.activeTab == 0 {
			status = m.traces.HelpText()
		} else {
			status = m.approvals.HelpText()
		}
	}

	return m.styles.App.Render(joinVertical([]string{
		m.styles.Tabs(m.activeTab),
		body,
		m.styles.status(m.width-2, status),
	}))
}

func (m *Model) resizeChildren() {
	bodyHeight := maxInt(10, m.height-5)
	bodyWidth := maxInt(40, m.width-4)
	m.traces.SetSize(bodyWidth, bodyHeight)
	m.approvals.SetSize(bodyWidth, bodyHeight)
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func checkPolicyCmd(checker PolicyChecker) tea.Cmd {
	return func() tea.Msg {
		if checker == nil {
			return policyCheckedMsg{}
		}
		return policyCheckedMsg{err: checker()}
	}
}
