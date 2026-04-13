package tui

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/chocks/agentctl/pkg/schema"
	"github.com/chocks/agentctl/pkg/trace"
)

type tracesLoadedMsg struct {
	traces []schema.Decision
	err    error
}

type tracesModel struct {
	path       string
	table      table.Model
	detail     viewport.Model
	traces     []schema.Decision
	err        error
	info       string
	detailOpen bool
	width      int
	height     int
}

func newTracesModel(path string) tracesModel {
	columns := []table.Column{
		{Title: "TIME", Width: 8},
		{Title: "ACTION", Width: 18},
		{Title: "VERDICT", Width: 10},
		{Title: "RISK", Width: 6},
		{Title: "AGENT", Width: 20},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	tblStyles := table.DefaultStyles()
	tblStyles.Header = tblStyles.Header.Bold(true)
	tblStyles.Selected = tblStyles.Selected.
		Foreground(tblStyles.Selected.GetForeground()).
		Background(tblStyles.Selected.GetBackground()).
		Bold(true)
	tbl.SetStyles(tblStyles)

	return tracesModel{
		path:  path,
		table: tbl,
		detail: viewport.New(
			80,
			8,
		),
		info: "Loading traces…",
	}
}

func (m tracesModel) reloadCmd() tea.Cmd {
	return func() tea.Msg {
		traces, err := trace.ReadTraces(m.path, trace.TraceFilter{Limit: 200})
		if err != nil {
			return tracesLoadedMsg{err: err}
		}

		sort.Slice(traces, func(i, j int) bool {
			return traces[i].Timestamp.After(traces[j].Timestamp)
		})

		return tracesLoadedMsg{traces: traces}
	}
}

func (m tracesModel) Update(msg tea.Msg) (tracesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tracesLoadedMsg:
		m.err = msg.err
		if msg.err != nil {
			m.info = ""
			m.traces = nil
			m.table.SetRows([]table.Row{})
			return m, nil
		}

		m.traces = msg.traces
		rows := make([]table.Row, 0, len(msg.traces))
		for _, decision := range msg.traces {
			agent := decision.Request.Context.Agent
			if agent == "" {
				agent = "-"
			}
			rows = append(rows, table.Row{
				decision.Timestamp.Format("15:04:05"),
				string(decision.Request.Action),
				formatVerdict(string(decision.Verdict)),
				fmt.Sprintf("%d", decision.RiskScore),
				agent,
			})
		}
		m.table.SetRows(rows)
		if len(rows) == 0 {
			m.info = "No traces recorded yet."
		} else {
			m.info = fmt.Sprintf("%d traces loaded", len(rows))
		}
		m.syncDetail()
		return m, nil

	case tea.KeyMsg:
		if m.detailOpen {
			switch msg.String() {
			case "enter", "esc":
				m.detailOpen = false
				return m, nil
			}

			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "j":
			m.table.MoveDown(1)
			m.syncDetail()
			return m, nil
		case "k":
			m.table.MoveUp(1)
			m.syncDetail()
			return m, nil
		case "enter":
			if len(m.traces) == 0 {
				return m, nil
			}
			m.detailOpen = true
			m.syncDetail()
			return m, nil
		}

		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		m.syncDetail()
		return m, cmd
	}

	return m, nil
}

func (m *tracesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.updateLayout()
}

func (m tracesModel) HelpText() string {
	if m.detailOpen {
		return "[tab] switch  [q] quit  [enter/esc] close detail  [j/k] scroll"
	}
	return "[tab] switch  [q] quit  [enter] detail  [j/k] move"
}

func (m tracesModel) View(styles Styles) string {
	if m.err != nil {
		return styles.ErrorBanner.Render("Trace load error: " + m.err.Error())
	}
	if len(m.traces) == 0 {
		return styles.Info.Render(m.info)
	}

	if !m.detailOpen {
		return m.table.View()
	}

	return lipglossJoinVertical(
		m.table.View(),
		styles.DetailTitle.Render("Selected Trace"),
		m.detail.View(),
	)
}

func (m *tracesModel) updateLayout() {
	width := m.width
	if width <= 0 {
		width = 100
	}
	height := m.height
	if height <= 0 {
		height = 20
	}

	m.table.SetWidth(width)

	detailHeight := 0
	tableHeight := height - 2
	if m.detailOpen {
		detailHeight = maxInt(6, height/3)
		tableHeight = maxInt(5, height-detailHeight-2)
		m.detail.Width = width
		m.detail.Height = detailHeight
	}
	m.table.SetHeight(maxInt(5, tableHeight))
}

func (m *tracesModel) syncDetail() {
	m.updateLayout()

	if len(m.traces) == 0 {
		m.detail.SetContent("No trace selected.")
		return
	}

	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(m.traces) {
		cursor = 0
	}

	data, err := json.MarshalIndent(m.traces[cursor], "", "  ")
	if err != nil {
		m.detail.SetContent("error rendering trace detail: " + err.Error())
		return
	}
	m.detail.SetContent(string(data))
	m.detail.GotoTop()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func lipglossJoinVertical(parts ...string) string {
	return tableStyleJoin(parts...)
}

func tableStyleJoin(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return joinVertical(filtered)
}

func joinVertical(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		out := parts[0]
		for i := 1; i < len(parts); i++ {
			out += "\n" + parts[i]
		}
		return out
	}
}
