package table

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type TableConfig struct {
	Headers []string
	Rows    [][]string
}

func ShowTable(config TableConfig) (int, []string) {
	m := model{
		config:      config,
		maxRowLines: 999,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok && m.selected {
		return m.cursor, m.config.Rows[m.cursor]
	}

	return -1, nil
}

// ShowInteractive shows a two-level table hierarchy within a single bubbletea program,
// avoiding alt screen flicker between levels. Returns the final selected row and its index.
func ShowInteractive(buckets TableConfig, loadKV func(string) TableConfig) (int, []string) {
	m := model{
		config:      buckets,
		maxRowLines: 999,
		level:       0,
		buckets:     &buckets,
		loadKV:      loadKV,
		kvCursors:   make(map[string]int),
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(model); ok && m.selected && m.level == 1 {
		if m.cursor >= 0 && m.cursor < len(m.config.Rows) {
			return m.cursor, m.config.Rows[m.cursor]
		}
	}

	return -1, nil
}

type model struct {
	config      TableConfig
	cursor      int
	offset      int
	colOff      int
	selected    bool
	height      int
	width       int
	colWidths   []int
	maxRowLines int

	// interactive two-level fields
	level        int           // 0: bucket list, 1: kv table
	buckets      *TableConfig
	loadKV       func(string) TableConfig
	prevBucket   int           // cursor in bucket list, restored when going back
	kvCursors    map[string]int // per-bucket KV cursor memory
	currentBucket string        // bucket being viewed in level 1
}

func (m *model) loadKVTable() TableConfig {
	if m.buckets == nil || m.cursor < 0 || m.cursor >= len(m.buckets.Rows) {
		return TableConfig{}
	}
	bucketName := m.buckets.Rows[m.cursor][0]
	return m.loadKV(bucketName)
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.calcColWidths()
		m.calcMaxRowLines()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.level == 0 || m.buckets == nil {
				return m, tea.Quit
			}
			// save KV cursor for current bucket, go back to bucket list
			if m.currentBucket != "" {
				m.kvCursors[m.currentBucket] = m.cursor
			}
			m.level = 0
			m.config = *m.buckets
			m.cursor = m.prevBucket
			m.offset = 0
			m.colOff = 0
			m.calcColWidths()
			m.calcMaxRowLines()
			m.ensureVisible()
			return m, nil
		case "enter":
			if m.level == 0 && m.loadKV != nil {
				// save bucket cursor
				m.prevBucket = m.cursor

				// transition to KV table for selected bucket
				bucketName := m.buckets.Rows[m.cursor][0]
				m.currentBucket = bucketName
				kvConfig := m.loadKVTable()
				if len(kvConfig.Rows) == 0 {
					return m, nil
				}
				m.level = 1
				m.config = kvConfig
				// restore KV cursor for this bucket if previously visited
				if m.kvCursors != nil {
					m.cursor = m.kvCursors[bucketName]
				} else {
					m.cursor = 0
				}
				m.offset = 0
				m.colOff = 0
				m.calcColWidths()
				m.calcMaxRowLines()
				m.ensureVisible()
				return m, nil
			}
			m.selected = true
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.config.Rows)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "pgdown", "f", " ":
			_, end := m.visibleRowRange()
			n := max(end-m.offset, 1)
			m.cursor += n
			if m.cursor >= len(m.config.Rows) {
				m.cursor = len(m.config.Rows) - 1
			}
			m.ensureVisible()
		case "pgup", "b":
			_, end := m.visibleRowRange()
			n := max(end-m.offset, 1)
			m.cursor -= n
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case "g":
			m.cursor = 0
			m.offset = 0
		case "G":
			m.cursor = len(m.config.Rows) - 1
			m.ensureVisible()
		case "right", "l":
			if _, colEnd := m.visibleCols(); colEnd < len(m.colWidths) {
				m.colOff++
			}
		case "left", "h":
			if colStart, _ := m.visibleCols(); colStart > 0 {
				m.colOff--
			}
		}
	}

	return m, nil
}

func (m *model) calcColWidths() {
	m.colWidths = make([]int, len(m.config.Headers))

	const maxColWidth = 30

	for i, h := range m.config.Headers {
		w := lipgloss.Width(h)
		for _, row := range m.config.Rows {
			if i < len(row) {
				cw := lipgloss.Width(row[i])
				if cw > w {
					w = cw
				}
			}
		}
		if w > maxColWidth {
			w = maxColWidth
		}
		m.colWidths[i] = w
	}
}

// availDataLines returns the number of terminal lines available for data rows.
func (m *model) availDataLines() int {
	return max(1, m.height-6)
}

// calcMaxRowLines calculates the maximum number of lines a single row can
// occupy, ensuring at least 2 logical rows are always visible.
func (m *model) calcMaxRowLines() {
	if m.height <= 0 {
		m.maxRowLines = 999
		return
	}
	availLines := m.availDataLines()
	m.maxRowLines = max(1, (availLines-1)/2)
}

// rowHeight returns the number of terminal lines a logical row occupies
// due to text wrapping.
func (m *model) rowHeight(ri int) int {
	if m.colWidths == nil || ri < 0 || ri >= len(m.config.Rows) {
		return 1
	}
	maxLines := 1
	for ci := range m.config.Headers {
		if ci < len(m.config.Rows[ri]) {
			w := lipgloss.Width(m.config.Rows[ri][ci])
			if w > m.colWidths[ci] && m.colWidths[ci] > 0 {
				lines := (w + m.colWidths[ci] - 1) / m.colWidths[ci]
				if lines > maxLines {
					maxLines = lines
				}
			}
		}
	}
	if maxLines > m.maxRowLines {
		maxLines = m.maxRowLines
	}
	return maxLines
}

// visibleRowRange returns the start (inclusive) and end (exclusive) indices
// of logical rows that fit in the visible area, respecting per-row heights.
// At least one row is always included, even if it overflows the screen.
func (m *model) visibleRowRange() (start, end int) {
	start = m.offset
	if start >= len(m.config.Rows) {
		return
	}
	availLines := m.availDataLines()
	used := 0
	end = start
	for end < len(m.config.Rows) {
		rh := m.rowHeight(end)
		sep := 0
		if end > start {
			sep = 1 // row separator line
		}
		if used+sep+rh > availLines {
			if end == start {
				end++
			}
			break
		}
		used += sep + rh
		end++
	}
	return
}

func (m *model) ensureVisible() {
	if m.cursor < m.offset {
		m.offset = m.cursor
		return
	}

	for {
		_, end := m.visibleRowRange()
		if m.cursor < end {
			return
		}
		if m.offset >= m.cursor {
			m.offset = m.cursor
			return
		}
		m.offset++
	}
}

// visibleCols returns the start (inclusive) and end (exclusive) indices
// of columns that fit in the terminal width.
func (m model) visibleCols() (start, end int) {
	if m.width <= 0 || len(m.colWidths) == 0 {
		return 0, len(m.colWidths)
	}

	colContent := func(i int) int { return m.colWidths[i] + 2 }

	totalWidth := func(l, r int) int {
		w := 2 // left + right border
		for i := l; i < r; i++ {
			w += colContent(i)
			if i < r-1 {
				w++ // separator between columns
			}
		}
		return w
	}

	start = m.colOff
	end = m.colOff + 1

	// expand right
	for end < len(m.colWidths) {
		if totalWidth(start, end+1) > m.width {
			break
		}
		end++
	}

	// expand left
	for start > 0 {
		if totalWidth(start-1, end) > m.width {
			break
		}
		start--
	}

	return start, end
}

// wrapText wraps a string into lines of at most width runes.
func wrapText(s string, width int, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return []string{s}
	}
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		runes := []rune(line)
		for len(runes) > 0 {
			if len(lines) >= maxLines {
				// Truncate the last rendered line with "..."
				lastIdx := len(lines) - 1
				lastLine := []rune(lines[lastIdx])
				if len(lastLine) > 3 {
					lines[lastIdx] = string(lastLine[:len(lastLine)-3]) + "..."
				} else {
					lines[lastIdx] = "..."
				}
				return lines
			}
			if len(runes) <= width {
				lines = append(lines, string(runes))
				break
			}
			lines = append(lines, string(runes[:width]))
			runes = runes[width:]
		}
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func (m model) View() tea.View {
	headerStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0"))

	if m.colWidths == nil {
		return tea.NewView("")
	}

	colStart, colEnd := m.visibleCols()
	visCols := m.colWidths[colStart:colEnd]

	pad := func(s string, w int) string {
		return lipgloss.NewStyle().Width(w).MaxWidth(w).Render(s)
	}

	var b strings.Builder

	// top border
	b.WriteString(borderStyle.Render("┌"))
	for i, w := range visCols {
		if i > 0 {
			b.WriteString(borderStyle.Render("┬"))
		}
		b.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
	}
	b.WriteString(borderStyle.Render("┐"))
	b.WriteByte('\n')

	// header row
	b.WriteString(borderStyle.Render("│"))
	for ci := colStart; ci < colEnd; ci++ {
		if ci > colStart {
			b.WriteString(borderStyle.Render("│"))
		}
		b.WriteString(headerStyle.Render(pad(m.config.Headers[ci], m.colWidths[ci])))
	}
	b.WriteString(borderStyle.Render("│"))
	b.WriteByte('\n')

	// header separator
	b.WriteString(borderStyle.Render("├"))
	for i, w := range visCols {
		if i > 0 {
			b.WriteString(borderStyle.Render("┼"))
		}
		b.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
	}
	b.WriteString(borderStyle.Render("┤"))

	// data rows (with text wrapping)
	startRow, endRow := m.visibleRowRange()
	availLines := m.availDataLines()
	renderedLines := 0
	for ri := startRow; ri < endRow; ri++ {
		// Wrap each cell's content to fit column width
		cellLines := make([][]string, len(m.colWidths))
		maxLines := 1
		for ci := colStart; ci < colEnd; ci++ {
			val := ""
			if ci < len(m.config.Rows[ri]) {
				val = m.config.Rows[ri][ci]
			}
			lines := wrapText(val, m.colWidths[ci], m.maxRowLines)
			cellLines[ci] = lines
			if len(lines) > maxLines {
				maxLines = len(lines)
			}
		}

		// Render each wrapped line, capped to available screen lines
		for li := 0; li < maxLines && renderedLines < availLines; li++ {
			b.WriteByte('\n')
			b.WriteString(borderStyle.Render("│"))
			for ci := colStart; ci < colEnd; ci++ {
				if ci > colStart {
					b.WriteString(borderStyle.Render("│"))
				}
				cell := ""
				if li < len(cellLines[ci]) {
					cell = pad(cellLines[ci][li], m.colWidths[ci])
				} else {
					cell = pad("", m.colWidths[ci])
				}
				if ri == m.cursor {
					b.WriteString(selectedStyle.Render(cell))
				} else {
					b.WriteString(cellStyle.Render(cell))
				}
			}
			b.WriteString(borderStyle.Render("│"))
			renderedLines++
		}

		// row separator
		if ri < endRow-1 && renderedLines < availLines {
			b.WriteByte('\n')
			b.WriteString(borderStyle.Render("├"))
			for i, w := range visCols {
				if i > 0 {
					b.WriteString(borderStyle.Render("┼"))
				}
				b.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
			}
			b.WriteString(borderStyle.Render("┤"))
			renderedLines++
		}
	}

	// bottom border
	b.WriteByte('\n')
	b.WriteString(borderStyle.Render("└"))
	for i, w := range visCols {
		if i > 0 {
			b.WriteString(borderStyle.Render("┴"))
		}
		b.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
	}
	b.WriteString(borderStyle.Render("┘"))
	b.WriteByte('\n')

	// status line
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statusStr := fmt.Sprintf(" Rows %d-%d / %d  Cols %d-%d / %d",
		startRow+1, endRow, len(m.config.Rows),
		colStart+1, colEnd, len(m.config.Headers))
	b.WriteString(statusStyle.Render(statusStr))
	b.WriteByte('\n')

	tableStr := b.String()
	if m.width > 0 {
		tableWidth := 2
		for i, w := range visCols {
			tableWidth += w + 2
			if i > 0 {
				tableWidth++
			}
		}
		if tableWidth <= m.width {
			leftPad := (m.width - tableWidth) / 2
			if leftPad > 0 {
				lines := strings.Split(tableStr, "\n")
				for i, line := range lines {
					lines[i] = strings.Repeat(" ", leftPad) + line
				}
				tableStr = strings.Join(lines, "\n")
			}
		}
	}

	// vertical centering: pad top when table is smaller than terminal height
	if m.height > 0 {
		tableLines := strings.Count(tableStr, "\n") + 1
		if tableLines < m.height {
			topPad := (m.height - tableLines) / 2
			if topPad > 0 {
				tableStr = strings.Repeat("\n", topPad) + tableStr
			}
		}
	}

	v := tea.NewView(tableStr)
	v.AltScreen = true
	return v
}

// 教程
// package main

// import (
// 	"fmt"
// 	"main/table"
// )

// func main() {
// 	headers := []string{"NAME", "TYPE"}
// 	rows := [][]string{
// 		{"Bulbasaur", "Grass / Poison"},
// 		{"Charmander", "Fire"},
// 		{"Squirtle", "Water"},
// 		{"Pikachu", "Electric"},
// 	}

// 	config := table.TableConfig{
// 		Headers: headers,
// 		Rows:    rows,
// 	}

// 	rowIdx, rowData := table.ShowTable(config)

// 	if rowIdx >= 0 {
// 		fmt.Printf("\n--- Row %d ---\n", rowIdx)
// 		fmt.Printf("%s (%s)\n", rowData[0], rowData[1])
// 	} else {
// 		fmt.Println("\nOperation cancelled.")
// 	}
// }
