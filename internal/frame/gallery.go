package frame

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	cardBody  = 3
	cardLines = cardBody + 2
	rowGap    = 1
)

// Choice est une carte de menu.
type Choice struct {
	Title  string
	Detail string
}

// GridShape indique le nombre de colonnes et la taille d'une page.
func (t Theme) GridShape(count, width, maxHeight int) (cols, pageSize int) {
	cols, rows := t.grid(count, width, maxHeight)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, cols * rows
}

// Gallery dessine les cartes de la page qui contient cursor.
func (t Theme) Gallery(choices []Choice, cursor, width, maxHeight int) string {
	if len(choices) == 0 {
		return t.Faint("Aucun choix")
	}
	if width < 24 {
		width = 24
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(choices) {
		cursor = len(choices) - 1
	}

	cols, rows := t.grid(len(choices), width, maxHeight)
	pageSize := cols * rows
	if pageSize < 1 {
		pageSize = 1
		cols = 1
		rows = 1
	}
	pages := (len(choices) + pageSize - 1) / pageSize
	page := cursor / pageSize
	start := page * pageSize
	end := start + pageSize
	if end > len(choices) {
		end = len(choices)
	}

	cardWidth := width
	if cols > 1 {
		cardWidth = (width - 2) / 2
	}

	var rowsView []string
	for row := 0; row < rows; row++ {
		var cells []string
		for col := 0; col < cols; col++ {
			index := start + row*cols + col
			if index >= end {
				break
			}
			number := row*cols + col + 1
			cells = append(cells, t.choiceCard(number, choices[index], index == cursor, cardWidth))
		}
		if len(cells) == 0 {
			break
		}
		if len(cells) == 1 {
			rowsView = append(rowsView, cells[0])
			continue
		}
		rowsView = append(rowsView, lipgloss.JoinHorizontal(lipgloss.Top, cells[0], "  ", cells[1]))
	}

	view := strings.Join(rowsView, "\n\n")
	if pages > 1 {
		view += "\n\n" + t.dots(pages, page)
	}
	return view
}

func (t Theme) grid(count, width, maxHeight int) (cols, rows int) {
	cols = 1
	if count > 1 && width >= 68 {
		cols = 2
	}
	if maxHeight < cardLines {
		return cols, 1
	}
	rows = 1
	for rows < 6 {
		if rows*cols >= count {
			break
		}
		if t.galleryHeight(rows+1, true) > maxHeight {
			break
		}
		rows++
	}
	if count > cols*rows && t.galleryHeight(rows, true) > maxHeight && rows > 1 {
		rows--
	}
	return cols, rows
}

func (t Theme) galleryHeight(rows int, withDots bool) int {
	if rows < 1 {
		rows = 1
	}
	height := rows*cardLines + (rows-1)*rowGap
	if withDots {
		height += 2
	}
	return height
}

func (t Theme) choiceCard(number int, choice Choice, selected bool, width int) string {
	inner := width - 2
	if inner < 12 {
		inner = 12
	}
	background := t.Surface
	border := t.Line
	titleColor := t.Text
	detailColor := t.Muted
	if selected {
		border = t.Accent
		titleColor = t.Accent
	}

	textWidth := inner - 2
	if textWidth < 4 {
		textWidth = 4
	}
	title := ansi.Truncate(choice.Title, textWidth, "…")
	detail := choice.Detail
	if strings.TrimSpace(detail) == "" {
		detail = " "
	}
	detail = ansi.Truncate(detail, textWidth, "…")

	body := strings.Join([]string{
		t.cardLine(fmt.Sprintf("%02d", number), t.Accent, background, inner, true),
		t.cardLine(title, titleColor, background, inner, true),
		t.cardLine(detail, detailColor, background, inner, false),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		BorderBackground(t.Canvas).
		Render(body)
}

func (t Theme) cardLine(text string, foreground, background lipgloss.Color, width int, bold bool) string {
	style := lipgloss.NewStyle().
		Foreground(foreground).
		Background(background).
		Width(width).
		Padding(0, 1)
	if bold {
		style = style.Bold(true)
	}
	return style.Render(text)
}

func (t Theme) dots(pages, page int) string {
	var parts []string
	for i := 0; i < pages; i++ {
		glyph := "○"
		color := t.Line
		if i == page {
			glyph = "●"
			color = t.Accent
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(color).Background(t.Canvas).Render(glyph))
	}
	return strings.Join(parts, " ")
}
