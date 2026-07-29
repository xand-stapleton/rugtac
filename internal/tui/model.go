// Package tui contains rugtac's small Bubble Tea user interface.
package tui

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"rugtac/internal/catalog"
	"rugtac/internal/search"
)

const (
	scopeAll = iota
	scopeCore
	scopeLibrary
)

const (
	pickerTactics = iota
	pickerLibraries
)

// Model is the complete application state. It intentionally avoids component
// libraries: text input, selection, and rendering are small enough to be clear
// here, and Bubble Tea only supplies the event loop.
type Model struct {
	entries        []catalog.Entry
	results        []search.Result
	query          string
	cursor         int
	detail         int
	width          int
	height         int
	scope          int
	library        int
	libraries      []string
	color          bool
	picker         int
	libraryQuery   string
	libraryResults []string
	libraryCursor  int
}

// New creates a ready-to-run search model.
func New(entries []catalog.Entry, initialQuery, initialScope string) Model {
	model := Model{
		entries:   entries,
		query:     initialQuery,
		width:     80,
		height:    24,
		libraries: catalog.Libraries(entries),
		color:     os.Getenv("NO_COLOR") == "",
	}
	model.selectInitialScope(initialScope)
	model.refresh()
	return model
}

// Init performs no I/O; all documentation is already in memory.
func (Model) Init() tea.Cmd { return nil }

// Update handles typing, selection, resizing, and quitting.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		if m.picker == pickerLibraries {
			return m.updateLibraryPicker(msg)
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.scope == scopeLibrary && len(m.libraries) > 0 {
				m.openLibraryPicker()
			}
		case "tab":
			m.cycleScope(1)
		case "shift+tab":
			m.cycleScope(-1)
		case "left":
			m.cycleLibrary(-1)
		case "right":
			m.cycleLibrary(1)
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
				m.detail = 0
			}
		case "down", "ctrl+n":
			if m.cursor+1 < len(m.results) {
				m.cursor++
				m.detail = 0
			}
		case "pgup", "ctrl+b":
			m.detail -= 5
			if m.detail < 0 {
				m.detail = 0
			}
		case "pgdown", "ctrl+f":
			m.detail += 5
			if maximum := m.maxDetailOffset(); m.detail > maximum {
				m.detail = maximum
			}
		case "home":
			m.cursor = 0
			m.detail = 0
		case "end":
			if len(m.results) > 0 {
				m.cursor = len(m.results) - 1
				m.detail = 0
			}
		case "backspace":
			_, size := utf8.DecodeLastRuneInString(m.query)
			if size > 0 {
				m.setQuery(m.query[:len(m.query)-size])
			}
		case "ctrl+u":
			m.setQuery("")
		default:
			if text := msg.Key().Text; text != "" {
				m.setQuery(m.query + text)
			}
		}
	}
	return m, nil
}

func (m Model) updateLibraryPicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.closeLibraryPicker(false)
	case "enter":
		m.closeLibraryPicker(true)
	case "up", "ctrl+p":
		if m.libraryCursor > 0 {
			m.libraryCursor--
		}
	case "down", "ctrl+n":
		if m.libraryCursor+1 < len(m.libraryResults) {
			m.libraryCursor++
		}
	case "home":
		m.libraryCursor = 0
	case "end":
		if len(m.libraryResults) > 0 {
			m.libraryCursor = len(m.libraryResults) - 1
		}
	case "backspace":
		_, size := utf8.DecodeLastRuneInString(m.libraryQuery)
		if size > 0 {
			m.setLibraryQuery(m.libraryQuery[:len(m.libraryQuery)-size])
		}
	case "ctrl+u":
		m.setLibraryQuery("")
	default:
		if text := msg.Key().Text; text != "" {
			m.setLibraryQuery(m.libraryQuery + text)
		}
	}
	return m, nil
}

func (m *Model) openLibraryPicker() {
	m.picker = pickerLibraries
	m.libraryQuery = ""
	m.libraryResults = search.FindStrings(m.libraries, "")
	m.libraryCursor = 0
	current := m.currentScope()
	for i, library := range m.libraryResults {
		if library == current {
			m.libraryCursor = i
			break
		}
	}
}

func (m *Model) closeLibraryPicker(confirm bool) {
	if confirm && len(m.libraryResults) > 0 {
		selected := m.libraryResults[m.libraryCursor]
		for i, library := range m.libraries {
			if library == selected {
				m.library = i
				break
			}
		}
		m.refresh()
	}
	m.picker = pickerTactics
	m.libraryQuery = ""
	m.libraryResults = nil
	m.libraryCursor = 0
}

func (m *Model) setLibraryQuery(query string) {
	m.libraryQuery = query
	m.libraryResults = search.FindStrings(m.libraries, query)
	m.libraryCursor = 0
}

func (m *Model) setQuery(query string) {
	m.query = query
	m.refresh()
}

func (m *Model) refresh() {
	m.results = search.Find(catalog.Filter(m.entries, m.currentScope()), m.query)
	m.cursor = 0
	m.detail = 0
}

func (m *Model) selectInitialScope(scope string) {
	switch strings.ToLower(scope) {
	case "", "all":
		m.scope = scopeAll
	case "core":
		m.scope = scopeCore
	default:
		m.scope = scopeLibrary
		for i, library := range m.libraries {
			if strings.EqualFold(library, scope) {
				m.library = i
				return
			}
		}
	}
	if m.scope != scopeLibrary {
		for i, library := range m.libraries {
			if library == "data" {
				m.library = i
				break
			}
		}
	}
}

func (m *Model) cycleScope(step int) {
	for {
		m.scope = (m.scope + step + 3) % 3
		if m.scope != scopeLibrary || len(m.libraries) > 0 {
			break
		}
	}
	m.refresh()
}

func (m *Model) cycleLibrary(step int) {
	if m.scope != scopeLibrary || len(m.libraries) == 0 {
		return
	}
	m.library = (m.library + step + len(m.libraries)) % len(m.libraries)
	m.refresh()
}

func (m Model) currentScope() string {
	switch m.scope {
	case scopeCore:
		return "core"
	case scopeLibrary:
		if len(m.libraries) > 0 {
			return m.libraries[m.library]
		}
	}
	return "all"
}

// View renders a compact result list and the selected entry's documentation.
func (m Model) View() tea.View {
	if m.picker == pickerLibraries {
		return m.libraryPickerView()
	}

	var b strings.Builder
	b.WriteString(m.paint("1;36", "rugtac — local tactic search"))
	b.WriteString("\n\n")
	b.WriteString(m.scopeView())
	b.WriteString("\n")
	query := truncate(m.query, m.contentWidth()-10)
	fmt.Fprintf(&b, "%s %s\n\n", m.paint("1;32", "search>"), query+m.paint("36", "█"))

	if len(m.results) == 0 {
		b.WriteString(m.paint("33", "No matches. Try fewer words or another scope."))
		b.WriteString("\n")
	} else {
		fmt.Fprintf(&b, "%s\n", m.paint("2", fmt.Sprintf("%d matches", len(m.results))))
		start, end := m.visibleRange()
		for i := start; i < end; i++ {
			marker := "  "
			if i == m.cursor {
				marker = "› "
			}
			entry := m.results[i].Entry
			line := fmt.Sprintf("%s%s [%s · %s] — %s", marker, entry.Name, entry.Source, entry.Library, entry.Summary)
			line = truncate(line, m.contentWidth())
			if i == m.cursor {
				line = m.paint("1;35", line)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}

		entry := m.results[m.cursor].Entry
		b.WriteString("\n")
		fmt.Fprintf(&b, "%s %s\n", m.paint("1;33", entry.Name), m.paint("34", "["+entry.Source+" · "+entry.Library+"]"))
		if entry.Module != "" {
			module := truncate(entry.Module, m.contentWidth()-8)
			fmt.Fprintf(&b, "%s %s\n", m.paint("2", "Module:"), m.paint("34", module))
		}
		b.WriteString(m.detailView(entry))
	}

	b.WriteString("\n")
	b.WriteString(m.paint("2", "tab scope • enter choose library • ↑/↓ select • pgup/C-b pgdn/C-f docs • esc quit"))
	return tea.NewView(b.String())
}

func (m Model) scopeView() string {
	all := " all "
	core := " core "
	library := " library"
	if len(m.libraries) > 0 {
		library += ": " + m.libraries[m.library] + " ↵ "
	}
	switch m.scope {
	case scopeAll:
		all = m.paint("1;7;36", all)
	case scopeCore:
		core = m.paint("1;7;36", core)
	case scopeLibrary:
		library = m.paint("1;7;36", library)
	}
	return m.paint("1;32", "scope>") + all + core + library
}

func (m Model) libraryPickerView() tea.View {
	var b strings.Builder
	b.WriteString(m.paint("1;36", "rugtac — choose library"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s %s\n\n", m.paint("1;32", "filter>"), truncate(m.libraryQuery, m.contentWidth()-10)+m.paint("36", "█"))

	if len(m.libraryResults) == 0 {
		b.WriteString(m.paint("33", "No matching libraries."))
		b.WriteByte('\n')
	} else {
		fmt.Fprintf(&b, "%s\n", m.paint("2", fmt.Sprintf("%d libraries", len(m.libraryResults))))
		start, end := m.libraryVisibleRange()
		for i := start; i < end; i++ {
			marker := "  "
			if i == m.libraryCursor {
				marker = "› "
			}
			line := marker + m.libraryResults[i]
			if i == m.libraryCursor {
				line = m.paint("1;35", line)
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	b.WriteString("\n")
	b.WriteString(m.paint("2", "type to filter • ↑/↓ select • enter confirm • esc cancel"))
	return tea.NewView(b.String())
}

func (m Model) libraryVisibleRange() (int, int) {
	limit := m.height - 8
	if limit < 3 {
		limit = 3
	}
	if limit > 12 {
		limit = 12
	}
	start := m.libraryCursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > len(m.libraryResults) {
		start = len(m.libraryResults) - limit
	}
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(m.libraryResults) {
		end = len(m.libraryResults)
	}
	return start, end
}

func (m Model) paint(code, text string) string {
	if !m.color {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (m Model) detailView(entry catalog.Entry) string {
	lines := m.detailLines(entry)
	capacity := m.detailCapacity()
	maximum := len(lines) - capacity
	if maximum < 0 {
		maximum = 0
	}
	start := m.detail
	if start > maximum {
		start = maximum
	}
	end := start + capacity
	if end > len(lines) {
		end = len(lines)
	}

	var b strings.Builder
	for _, line := range lines[start:end] {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if maximum > 0 {
		position := fmt.Sprintf("docs %d–%d/%d", start+1, end, len(lines))
		b.WriteString(m.paint("2", position))
		b.WriteByte('\n')
	}
	return b.String()
}

func (m Model) detailLines(entry catalog.Entry) []string {
	lines := strings.Split(wrap(entry.Description, m.contentWidth(), ""), "\n")
	if entry.Usage != "" {
		lines = append(lines, "", m.paint("1;33", "Usage:"))
		for _, line := range strings.Split(entry.Usage, "\n") {
			lines = append(lines, truncate("  "+line, m.contentWidth()))
		}
	}
	if entry.URL != "" {
		lines = append(lines, "", m.paint("2", "Docs:")+" "+m.paint("4;34", truncate(entry.URL, m.contentWidth()-6)))
	}
	return lines
}

func (m Model) detailCapacity() int {
	start, end := m.visibleRange()
	fixed := 12 + (end - start)
	if capacity := m.height - fixed; capacity > 3 {
		return capacity
	}
	return 3
}

func (m Model) maxDetailOffset() int {
	if len(m.results) == 0 {
		return 0
	}
	maximum := len(m.detailLines(m.results[m.cursor].Entry)) - m.detailCapacity()
	if maximum < 0 {
		return 0
	}
	return maximum
}

func (m Model) visibleRange() (int, int) {
	limit := 7
	if available := (m.height - 18) / 2; available > 2 && available < limit {
		limit = available
	}
	start := m.cursor - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > len(m.results) {
		start = len(m.results) - limit
	}
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(m.results) {
		end = len(m.results)
	}
	return start, end
}

func (m Model) contentWidth() int {
	width := m.width
	if width < 30 {
		return 30
	}
	if width > 100 {
		return 100
	}
	return width
}

func wrap(text string, width int, prefix string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	lineLength := 0
	for _, word := range words {
		wordLength := utf8.RuneCountInString(word)
		if lineLength > 0 && lineLength+1+wordLength > width {
			b.WriteByte('\n')
			b.WriteString(prefix)
			lineLength = utf8.RuneCountInString(prefix)
		} else if lineLength > 0 {
			b.WriteByte(' ')
			lineLength++
		}
		b.WriteString(word)
		lineLength += wordLength
	}
	return b.String()
}

func truncate(text string, width int) string {
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}
