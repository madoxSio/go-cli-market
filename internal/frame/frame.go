// Package frame dessine le cadre commun du client et de l'administration.
package frame

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Theme rassemble les couleurs d'une interface.
type Theme struct {
	Badge   string
	Canvas  lipgloss.Color
	Surface lipgloss.Color
	Bar     lipgloss.Color
	Accent  lipgloss.Color
	Ink     lipgloss.Color
	Text    lipgloss.Color
	Muted   lipgloss.Color
	Line    lipgloss.Color
	Ok      lipgloss.Color
	Err     lipgloss.Color
}

// Client est la palette violette de l'espace acheteur.
var Client = Theme{
	Badge:   "CLIENT",
	Canvas:  lipgloss.Color("#100818"),
	Surface: lipgloss.Color("#1C1230"),
	Bar:     lipgloss.Color("#160D24"),
	Accent:  lipgloss.Color("#C4B5FD"),
	Ink:     lipgloss.Color("#1A1028"),
	Text:    lipgloss.Color("#F4F0FF"),
	Muted:   lipgloss.Color("#A89BC4"),
	Line:    lipgloss.Color("#3A2D55"),
	Ok:      lipgloss.Color("#86EFAC"),
	Err:     lipgloss.Color("#FCA5A5"),
}

// Admin est la palette ambre de la gestion des comptes.
var Admin = Theme{
	Badge:   "ADMIN",
	Canvas:  lipgloss.Color("#120E08"),
	Surface: lipgloss.Color("#241C10"),
	Bar:     lipgloss.Color("#1A140C"),
	Accent:  lipgloss.Color("#FBBF24"),
	Ink:     lipgloss.Color("#1A140C"),
	Text:    lipgloss.Color("#FFF8EB"),
	Muted:   lipgloss.Color("#CDB89A"),
	Line:    lipgloss.Color("#4A3C28"),
	Ok:      lipgloss.Color("#86EFAC"),
	Err:     lipgloss.Color("#FCA5A5"),
}

// Screen compose l'en-tête, la colonne centrée et le pied de page.
func (t Theme) Screen(width, height int, heading, tagline, status string, ok bool, body, hint string) string {
	if width < 40 {
		width = 40
	}
	if height < 12 {
		height = 12
	}

	header := t.header(width)
	footer := t.footer(width, hint)
	content := t.column(heading, tagline, status, ok, body)

	middleH := height - lipgloss.Height(header) - lipgloss.Height(footer)
	if middleH < 1 {
		middleH = 1
	}
	middle := lipgloss.Place(width, middleH, lipgloss.Center, lipgloss.Top, content,
		lipgloss.WithWhitespaceBackground(t.Canvas),
	)
	return header + "\n" + middle + "\n" + footer
}

// PanelWidth est la largeur de la colonne de contenu.
func (t Theme) PanelWidth(term int) int {
	if term <= 0 {
		term = 80
	}
	width := term - 8
	if width > 72 {
		width = 72
	}
	if width < 36 {
		width = 36
	}
	if term > 24 && width > term-2 {
		width = term - 2
	}
	return width
}

// Card encadre un texte simple.
func (t Theme) Card(body string, width int) string {
	box := width - 2
	if box < 20 {
		box = width
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Foreground(t.Text).
		Background(t.Surface).
		Padding(1, 2).
		Width(box).
		Render(body)
}

// Code affiche un code de confirmation dans un badge.
func (t Theme) Code(code string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Ink).
		Background(t.Accent).
		Padding(0, 1).
		Render(code)
}

// Faint rend un libellé secondaire.
func (t Theme) Faint(text string) string {
	return lipgloss.NewStyle().Foreground(t.Muted).Background(t.Canvas).Render(text)
}

// TextLine rend une ligne de texte sur le fond de l'écran.
func (t Theme) TextLine(text string) string {
	return lipgloss.NewStyle().Foreground(t.Text).Background(t.Canvas).Render(text)
}

// Keys assemble les raccourcis du pied de page.
func (t Theme) Keys(pairs ...string) string {
	keyStyle := lipgloss.NewStyle().Foreground(t.Accent).Background(t.Canvas).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Muted).Background(t.Canvas)
	sep := lipgloss.NewStyle().Foreground(t.Line).Background(t.Canvas).Render("   ")
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, keyStyle.Render(pairs[i])+" "+labelStyle.Render(pairs[i+1]))
	}
	return strings.Join(parts, sep)
}

// NewSpinner renvoie un indicateur aux couleurs du thème.
func (t Theme) NewSpinner() spinner.Model {
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = lipgloss.NewStyle().Foreground(t.Accent)
	return spin
}

// FormTheme aligne Huh sur la couleur d'accent.
func (t Theme) FormTheme() *huh.Theme {
	theme := huh.ThemeCatppuccin()
	accent := t.Accent
	theme.Focused.Base = theme.Focused.Base.BorderForeground(accent)
	theme.Focused.Title = theme.Focused.Title.Foreground(accent)
	theme.Focused.NoteTitle = theme.Focused.NoteTitle.Foreground(accent)
	theme.Focused.Description = theme.Focused.Description.Foreground(t.Muted)
	theme.Focused.SelectSelector = theme.Focused.SelectSelector.Foreground(accent)
	theme.Focused.NextIndicator = theme.Focused.NextIndicator.Foreground(accent)
	theme.Focused.PrevIndicator = theme.Focused.PrevIndicator.Foreground(accent)
	theme.Focused.Option = theme.Focused.Option.Foreground(t.Text)
	theme.Focused.TextInput.Cursor = theme.Focused.TextInput.Cursor.Foreground(accent)
	theme.Focused.TextInput.Prompt = theme.Focused.TextInput.Prompt.Foreground(accent)
	theme.Focused.TextInput.Text = theme.Focused.TextInput.Text.Foreground(t.Text)
	theme.Focused.TextInput.Placeholder = theme.Focused.TextInput.Placeholder.Foreground(t.Line)
	theme.Focused.FocusedButton = theme.Focused.FocusedButton.Foreground(t.Ink).Background(accent)
	theme.Focused.BlurredButton = theme.Focused.BlurredButton.Foreground(t.Text).Background(t.Surface)
	theme.Focused.ErrorIndicator = theme.Focused.ErrorIndicator.Foreground(t.Err)
	theme.Focused.ErrorMessage = theme.Focused.ErrorMessage.Foreground(t.Err)

	theme.Blurred = theme.Focused
	theme.Blurred.Base = theme.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
	theme.Blurred.Card = theme.Blurred.Base
	theme.Group.Title = theme.Focused.Title
	theme.Group.Description = theme.Focused.Description

	theme.Help.ShortKey = theme.Help.ShortKey.Foreground(t.Muted)
	theme.Help.ShortDesc = theme.Help.ShortDesc.Foreground(t.Muted)
	theme.Help.ShortSeparator = theme.Help.ShortSeparator.Foreground(t.Line)
	theme.Help.FullKey = theme.Help.FullKey.Foreground(t.Muted)
	theme.Help.FullDesc = theme.Help.FullDesc.Foreground(t.Muted)
	theme.Help.FullSeparator = theme.Help.FullSeparator.Foreground(t.Line)
	return theme
}

func (t Theme) column(heading, tagline, status string, ok bool, body string) string {
	var blocks []string
	if heading != "" {
		mark := lipgloss.NewStyle().Foreground(t.Accent).Background(t.Canvas).Bold(true).Render("▸ ")
		title := lipgloss.NewStyle().Foreground(t.Text).Background(t.Canvas).Bold(true).Render(heading)
		blocks = append(blocks, mark+title)
	}
	if tagline != "" {
		blocks = append(blocks, t.Faint(tagline))
	}
	if heading != "" || tagline != "" {
		blocks = append(blocks, "")
	}
	if status != "" {
		blocks = append(blocks, t.pill(status, ok), "")
	}
	if body != "" {
		blocks = append(blocks, body)
	}
	return strings.Join(blocks, "\n")
}

func (t Theme) pill(text string, ok bool) string {
	color := t.Ok
	if !ok {
		color = t.Err
	}
	return lipgloss.NewStyle().
		Foreground(color).
		Background(t.Surface).
		Bold(true).
		Padding(0, 1).
		Render("● " + text)
}

func (t Theme) header(width int) string {
	badge := lipgloss.NewStyle().
		Foreground(t.Ink).
		Background(t.Accent).
		Bold(true).
		Padding(0, 1).
		Render(t.Badge)
	left := lipgloss.NewStyle().Foreground(t.Accent).Background(t.Bar).Bold(true).Render("◆") +
		lipgloss.NewStyle().Foreground(t.Text).Background(t.Bar).Bold(true).Render("  go-cli-market")

	gap := width - lipgloss.Width(left) - lipgloss.Width(badge)
	var line string
	if gap < 1 {
		line = lipgloss.NewStyle().Background(t.Bar).Foreground(t.Text).Bold(true).Width(width).MaxWidth(width).Render("◆  go-cli-market")
	} else {
		line = left + lipgloss.NewStyle().Background(t.Bar).Render(strings.Repeat(" ", gap)) + badge
	}
	rule := lipgloss.NewStyle().Foreground(t.Accent).Render(strings.Repeat("━", width))
	return line + "\n" + rule
}

func (t Theme) footer(width int, hint string) string {
	rule := lipgloss.NewStyle().Foreground(t.Line).Render(strings.Repeat("─", width))
	line := lipgloss.PlaceHorizontal(width, lipgloss.Left, "  "+hint,
		lipgloss.WithWhitespaceBackground(t.Canvas),
	)
	return rule + "\n" + line
}
