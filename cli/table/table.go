package table

import (
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
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

// InteractiveCallbacks provides database operations for ShowInteractive.
type InteractiveCallbacks struct {
	CreateBucket  func(name, keyType string) error
	RenameBucket  func(oldName, newName string) error
	DropBucket    func(name string) error
	PutKV         func(bucket, key, value string) error
	DeleteKV      func(bucket, key string) error
	ReloadBuckets func() TableConfig // refresh after bucket changes
}

// ShowInteractive shows a two-level table hierarchy within a single bubbletea program,
// avoiding alt screen flicker between levels. Supports inline actions via callbacks.
func ShowInteractive(buckets TableConfig, loadKV func(string) TableConfig,
	cb InteractiveCallbacks) (int, []string) {

	m := model{
		config:      buckets,
		maxRowLines: 999,
		level:       0,
		buckets:     &buckets,
		loadKV:      loadKV,
		cb:          cb,
		kvCursors:   make(map[string]int),
		kvOffsets:   make(map[string]int),
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

// promptState tracks what kind of inline input is active.
type promptState int

const (
	promptNone promptState = iota
	promptCreateBucketName
	promptCreateBucketType
	promptRenameBucket
	promptDropBucket
	promptPutKey
	promptPutValue
	promptDeleteKV
)

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
	level        int            // 0: bucket list, 1: kv table
	buckets      *TableConfig
	loadKV       func(string) TableConfig
	cb           InteractiveCallbacks
	prevBucket   int            // cursor in bucket list, restored when going back
	kvCursors    map[string]int // per-bucket KV cursor memory
	kvOffsets    map[string]int // per-bucket KV scroll offset memory
	currentBucket string         // bucket being viewed in level 1
	currentType  string         // key type of current bucket (string/seq/time)

	// inline prompt fields
	prompt      promptState
	promptInput textinput.Model
	promptBuf   string    // stores first input value while waiting for second
	actionErr   string    // error message to display after an action
	errUntil    time.Time // show actionErr until this time
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
		// If in prompt mode, handle prompt input first
		if m.prompt != promptNone {
			return m.handlePromptKey(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.level == 0 || m.buckets == nil {
				return m, tea.Quit
			}
			// save KV cursor and offset for current bucket, go back to bucket list
			if m.currentBucket != "" {
				m.kvCursors[m.currentBucket] = m.cursor
				m.kvOffsets[m.currentBucket] = m.offset
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
				m.prevBucket = m.cursor
				bucketName := m.buckets.Rows[m.cursor][0]
				m.currentBucket = bucketName
				m.currentType = m.buckets.Rows[m.cursor][1]
				kvConfig := m.loadKVTable()
				m.level = 1
				m.config = kvConfig
				if m.kvCursors != nil {
					m.cursor = m.kvCursors[bucketName]
					m.offset = m.kvOffsets[bucketName]
				} else {
					m.cursor = 0
					m.offset = 0
				}
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
			if len(m.config.Rows) > 0 {
				m.cursor = len(m.config.Rows) - 1
			}
			m.ensureVisible()
		case "right", "l":
			if _, colEnd := m.visibleCols(); colEnd < len(m.colWidths) {
				m.colOff++
			}
		case "left", "h":
			if colStart, _ := m.visibleCols(); colStart > 0 {
				m.colOff--
			}

		// bucket-level actions
		case "c":
			if m.level == 0 && m.cb.CreateBucket != nil {
				m.startPrompt(promptCreateBucketName, "Bucket name:")
			}
		case "r":
			if m.level == 0 && m.cb.RenameBucket != nil && len(m.config.Rows) > 0 {
				m.startPrompt(promptRenameBucket, "New name:")
			}
		case "d":
			if m.level == 0 && m.cb.DropBucket != nil && len(m.config.Rows) > 0 {
				m.startPrompt(promptDropBucket, "Delete '"+m.config.Rows[m.cursor][0]+"'? (y/n):")
			}

		// KV-level actions
		case "p":
			if m.level == 1 && m.cb.PutKV != nil {
				if m.currentType == "seq" || m.currentType == "time" {
					// auto-generated key, skip to value prompt
					m.startPrompt(promptPutValue, "Value:")
				} else {
					m.startPrompt(promptPutKey, "Key:")
				}
			}
		case "x":
			if m.level == 1 && m.cb.DeleteKV != nil && len(m.config.Rows) > 0 {
				m.startPrompt(promptDeleteKV, "Delete '"+m.config.Rows[m.cursor][0]+"'? (y/n):")
			}
		}
	}

	return m, nil
}

// startPrompt initializes the inline text input for a given action.
func (m *model) startPrompt(state promptState, placeholder string) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetWidth(40)
	ti.CharLimit = 256
	ti.Focus()
	m.prompt = state
	m.promptInput = ti
	m.promptBuf = ""
}

// handlePromptKey processes keypresses when in prompt mode.
func (m *model) handlePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "esc":
		m.prompt = promptNone
		return m, nil
	case "enter":
		val := m.promptInput.Value()
		return m.handlePromptSubmit(val)
	default:
		var cmd tea.Cmd
		m.promptInput, cmd = m.promptInput.Update(msg)
		return m, cmd
	}
}

// handlePromptSubmit processes the collected input after Enter is pressed.
func (m *model) handlePromptSubmit(val string) (tea.Model, tea.Cmd) {
	switch m.prompt {
	case promptCreateBucketName:
		if val == "" {
			m.prompt = promptNone
			return m, nil
		}
		m.promptBuf = val
		m.promptInput.SetValue("")
		m.promptInput.Placeholder = "Key type (string/seq/time):"
		m.prompt = promptCreateBucketType
		return m, nil

	case promptCreateBucketType:
		if val != "string" && val != "seq" && val != "time" {
			m.actionErr = "invalid key type, must be string/seq/time"
			m.errUntil = time.Now().Add(3 * time.Second)
			m.prompt = promptNone
			m.refreshBuckets()
			return m, nil
		}
		name := m.promptBuf
		if err := m.cb.CreateBucket(name, val); err != nil {
			m.actionErr = err.Error()
			m.errUntil = time.Now().Add(3 * time.Second)
		}
		m.prompt = promptNone
		m.refreshBuckets()
		return m, nil

	case promptRenameBucket:
		if val == "" {
			m.prompt = promptNone
			return m, nil
		}
		oldName := m.config.Rows[m.cursor][0]
		if err := m.cb.RenameBucket(oldName, val); err != nil {
			m.actionErr = err.Error()
			m.errUntil = time.Now().Add(3 * time.Second)
		}
		m.prompt = promptNone
		m.refreshBuckets()
		return m, nil

	case promptDropBucket:
		if val == "y" || val == "Y" {
			name := m.config.Rows[m.cursor][0]
			if err := m.cb.DropBucket(name); err != nil {
				m.actionErr = err.Error()
				m.errUntil = time.Now().Add(3 * time.Second)
			}
		}
		m.prompt = promptNone
		m.refreshBuckets()
		return m, nil

	case promptPutKey:
		m.promptBuf = val
		m.promptInput.SetValue("")
		m.promptInput.Placeholder = "Value:"
		m.prompt = promptPutValue
		return m, nil

	case promptPutValue:
		key, value := m.promptBuf, val
		bucket := m.currentBucket
		if err := m.cb.PutKV(bucket, key, value); err != nil {
			m.actionErr = err.Error()
			m.errUntil = time.Now().Add(3 * time.Second)
		}
		m.prompt = promptNone
		// refresh KV table
		if m.currentBucket != "" {
			kvConfig := m.loadKV(m.currentBucket)
			m.config = kvConfig
			m.cursor = 0
			m.offset = 0
			m.calcColWidths()
			m.calcMaxRowLines()
		}
		return m, nil

	case promptDeleteKV:
		if val == "y" || val == "Y" {
			key := m.config.Rows[m.cursor][0]
			bucket := m.currentBucket
			if err := m.cb.DeleteKV(bucket, key); err != nil {
				m.actionErr = err.Error()
				m.errUntil = time.Now().Add(3 * time.Second)
			}
		}
		m.prompt = promptNone
		// refresh KV table
		if m.currentBucket != "" {
			kvConfig := m.loadKV(m.currentBucket)
			m.config = kvConfig
			if len(kvConfig.Rows) > 0 && m.cursor >= len(kvConfig.Rows) {
				m.cursor = len(kvConfig.Rows) - 1
			} else if len(kvConfig.Rows) == 0 {
				m.cursor = 0
			}
			m.offset = 0
			m.calcColWidths()
			m.calcMaxRowLines()
		}
		return m, nil
	}

	m.prompt = promptNone
	return m, nil
}

// refreshBuckets reloads the bucket list from the loadKV callback (using the first bucket as signal).
func (m *model) refreshBuckets() {
	if m.cb.ReloadBuckets == nil {
		return
	}
	newBuckets := m.cb.ReloadBuckets()
	m.buckets = &newBuckets
	m.config = newBuckets
	if m.cursor >= len(m.config.Rows) && len(m.config.Rows) > 0 {
		m.cursor = len(m.config.Rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.offset = 0
	m.colOff = 0
	m.calcColWidths()
	m.calcMaxRowLines()
	m.ensureVisible()
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

	tableBody := b.String()

	// horizontal centering (table body only)
	leftPad := 0
	if m.width > 0 {
		tableWidth := 2
		for i, w := range visCols {
			tableWidth += w + 2
			if i > 0 {
				tableWidth++
			}
		}
		if tableWidth <= m.width {
			leftPad = (m.width - tableWidth) / 2
		}
	}

	// center the table body
	if leftPad > 0 {
		lines := strings.Split(tableBody, "\n")
		for i, line := range lines {
			lines[i] = strings.Repeat(" ", leftPad) + line
		}
		tableBody = strings.Join(lines, "\n")
	}

	// build footer (info + hints + error + prompt) with same left padding
	var footer strings.Builder
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	if leftPad > 0 {
		footer.WriteString(strings.Repeat(" ", leftPad))
	}
	// line 1: row/col info
	footer.WriteString(infoStyle.Render(fmt.Sprintf("Rows %d-%d / %d  Cols %d-%d / %d",
		startRow+1, endRow, len(m.config.Rows),
		colStart+1, colEnd, len(m.config.Headers))))
	footer.WriteByte('\n')

	if leftPad > 0 {
		footer.WriteString(strings.Repeat(" ", leftPad))
	}
	// line 2: action hints
	if m.level == 0 {
		footer.WriteString(infoStyle.Render("[c]create  [r]rename  [d]drop"))
	} else {
		footer.WriteString(infoStyle.Render("[p]put  [x]delete"))
	}
	footer.WriteByte('\n')

	// action error message (auto-expires)
	if m.actionErr != "" && time.Now().Before(m.errUntil) {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
		if leftPad > 0 {
			footer.WriteString(strings.Repeat(" ", leftPad))
		}
		footer.WriteString(errStyle.Render(m.actionErr))
		footer.WriteByte('\n')
	}

	// inline prompt
	if m.prompt != promptNone {
		footer.WriteByte('\n')
		if leftPad > 0 {
			footer.WriteString(strings.Repeat(" ", leftPad))
		}
		footer.WriteString(m.promptInput.View())
		footer.WriteByte('\n')
	}

	fullStr := tableBody + "\n" + footer.String()

	// vertical centering: pad top when table is smaller than terminal height
	if m.height > 0 {
		tableLines := strings.Count(fullStr, "\n") + 1
		if tableLines < m.height {
			topPad := (m.height - tableLines) / 2
			if topPad > 0 {
				fullStr = strings.Repeat("\n", topPad) + fullStr
			}
		}
	}

	v := tea.NewView(fullStr)
	v.AltScreen = true
	return v
}

// (the rest of the functions - calcColWidths, availDataLines, calcMaxRowLines,
//  rowHeight, visibleRowRange, ensureVisible, visibleCols, wrapText, scrollHint -
//  remain unchanged from the original)

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

func (m *model) availDataLines() int {
	return max(1, m.height-6)
}

func (m *model) calcMaxRowLines() {
	if m.height <= 0 {
		m.maxRowLines = 999
		return
	}
	availLines := m.availDataLines()
	m.maxRowLines = max(1, (availLines-1)/2)
}

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
			sep = 1
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

func (m model) visibleCols() (start, end int) {
	if m.width <= 0 || len(m.colWidths) == 0 {
		return 0, len(m.colWidths)
	}

	colContent := func(i int) int { return m.colWidths[i] + 2 }

	totalWidth := func(l, r int) int {
		w := 2
		for i := l; i < r; i++ {
			w += colContent(i)
			if i < r-1 {
				w++
			}
		}
		return w
	}

	start = m.colOff
	end = m.colOff + 1
	for end < len(m.colWidths) {
		if totalWidth(start, end+1) > m.width {
			break
		}
		end++
	}
	for start > 0 {
		if totalWidth(start-1, end) > m.width {
			break
		}
		start--
	}
	return start, end
}

func wrapText(s string, width int, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return []string{s}
	}
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		runes := []rune(line)
		for len(runes) > 0 {
			if len(lines) >= maxLines {
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
