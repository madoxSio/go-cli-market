package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#6D28D9")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#C4B5FD"))

	formTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EDE9FE"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2)

	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ADE80"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171"))
	codeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FDE68A")).
			Background(lipgloss.Color("#3F3F46")).
			Padding(0, 1)
	hintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A1A1AA"))
)
