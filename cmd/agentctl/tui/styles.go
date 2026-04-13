package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	App         lipgloss.Style
	ActiveTab   lipgloss.Style
	InactiveTab lipgloss.Style
	StatusBar   lipgloss.Style
	ErrorBanner lipgloss.Style
	Info        lipgloss.Style
	DetailTitle lipgloss.Style
}

func DefaultStyles() Styles {
	return Styles{
		App: lipgloss.NewStyle().
			Padding(0, 1),
		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("31")).
			Padding(0, 2),
		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 2),
		StatusBar: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236")).
			Padding(0, 1),
		ErrorBanner: lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("160")).
			Bold(true).
			Padding(0, 1),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),
		DetailTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true),
	}
}

func (s Styles) Tabs(active int) string {
	tabs := []string{
		s.renderTab("Traces", active == 0),
		s.renderTab("Approvals", active == 1),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (s Styles) renderTab(label string, active bool) string {
	if active {
		return s.ActiveTab.Render(label)
	}
	return s.InactiveTab.Render(label)
}

func (s Styles) status(width int, text string) string {
	style := s.StatusBar
	if width > 0 {
		style = style.Width(width)
	}
	return style.Render(text)
}

func formatVerdict(verdict string) string {
	color := "245"
	switch verdict {
	case "allow":
		color = "42"
	case "deny":
		color = "160"
	case "escalate":
		color = "214"
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true).
		Render(strings.ToUpper(verdict))
}
