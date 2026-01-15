//go:build !gui

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/metcalfc/brr/internal/reader"
	"github.com/metcalfc/brr/internal/state"
)

// Version info (injected via ldflags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	erpStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF0000"))

	wordBeforeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	wordAfterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	controlsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true)

	pausedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true)

	completeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	tocPanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(0, 1)

	tocTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00")).
			Bold(true)
)

// tocItem implements list.Item for the TOC list
type tocItem struct {
	entry reader.TOCEntry
}

func (i tocItem) Title() string       { return i.entry.Title }
func (i tocItem) Description() string { return i.entry.Preview }
func (i tocItem) FilterValue() string { return i.entry.Title }

type model struct {
	*reader.Reader
	quitting   bool
	width      int
	height     int
	tocVisible bool
	tocList    list.Model
	sourceFile string
	stateStore *state.StateStore
	fileHash   string
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tick(m.GetDelay())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.tocVisible {
		return m.updateTOC(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			m.Paused = !m.Paused
			if !m.Paused {
				return m, tick(m.GetDelay())
			}
			return m, nil

		case "+", "=":
			if m.WPM < 1500 {
				m.WPM += 50
			}
			return m, nil

		case "-":
			if m.WPM > 100 {
				m.WPM -= 50
			}
			return m, nil

		case "up":
			if m.WPM < 1500 {
				m.WPM += 50
			}
			return m, nil

		case "down":
			if m.WPM > 100 {
				m.WPM -= 50
			}
			return m, nil

		case "left":
			now := time.Now()
			if now.Sub(m.LastArrowPress) > 500*time.Millisecond {
				m.Paused = true
			}
			m.LastArrowPress = now
			m.JumpToPrevSentence()
			return m, nil

		case "right":
			now := time.Now()
			if now.Sub(m.LastArrowPress) > 500*time.Millisecond {
				m.Paused = true
			}
			m.LastArrowPress = now
			m.JumpToNextSentence()
			return m, nil

		case "t":
			if len(m.TOC) > 0 {
				m.tocVisible = true
				m.Paused = true
			}
			return m, nil

		case "r":
			m.CurrentIndex = 0
			if m.stateStore != nil && m.fileHash != "" {
				m.stateStore.Clear(m.fileHash)
			}
			return m, nil

		case "q", "Q", "ctrl+c":
			m.savePosition()
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tocList.SetSize(m.width/3-4, m.height-4)
		return m, nil

	case tickMsg:
		if m.Paused {
			return m, nil
		}

		if m.Advance() {
			return m, tick(m.GetDelay())
		}

		m.savePosition()
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) updateTOC(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.tocList.SelectedItem().(tocItem); ok {
				m.JumpToChapter(item.entry.WordIndex)
			}
			m.tocVisible = false
			return m, nil

		case "t", "esc", "q":
			m.tocVisible = false
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tocList.SetSize(m.width/3-4, m.height-4)
		return m, nil
	}

	var cmd tea.Cmd
	m.tocList, cmd = m.tocList.Update(msg)
	return m, cmd
}

func (m *model) savePosition() {
	if m.stateStore != nil && m.fileHash != "" {
		m.stateStore.SetPosition(m.fileHash, m.CurrentIndex)
	}
}

func (m model) View() string {
	if m.quitting {
		if m.AtEnd() {
			return completeStyle.Render("\n  Reading complete!\n")
		}
		return ""
	}

	if len(m.Words) == 0 {
		return "No text to read."
	}

	if m.tocVisible {
		return m.viewWithTOC()
	}

	return m.viewReading(m.width)
}

func (m model) viewReading(width int) string {
	word := m.CurrentWord()
	formatted := formatWord(word)

	pause := ""
	if m.Paused {
		pause = pausedStyle.Render(" [PAUSED]")
	}

	current, total := m.Progress()
	chapterInfo := ""
	if title := m.CurrentChapterTitle(); title != "" {
		chapterInfo = fmt.Sprintf(" | %s", title)
	}
	status := statusStyle.Render(
		fmt.Sprintf("Word %d/%d | %d WPM%s%s",
			current,
			total,
			m.WPM,
			pause,
			chapterInfo,
		),
	)

	tocHint := ""
	if len(m.TOC) > 0 {
		tocHint = "  T: TOC"
	}
	controls := controlsStyle.Render("SPACE: pause  ↑/↓: speed  ←/→: sentence  R: restart" + tocHint + "  Q: quit")

	avail := m.height - 2
	if avail < 1 {
		avail = 1
	}
	vPad := avail / 2
	if vPad < 0 {
		vPad = 0
	}

	var sb strings.Builder

	sb.WriteString(status)
	sb.WriteString("\n")

	for i := 0; i < vPad; i++ {
		sb.WriteString("\n")
	}

	line := anchorORPText(formatted, word, width)
	sb.WriteString(line)

	remaining := avail - vPad
	for i := 0; i < remaining; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString(controls)

	return sb.String()
}

func (m model) viewWithTOC() string {
	tocWidth := m.width / 3
	readingWidth := m.width - tocWidth - 1

	tocPanel := m.renderTOCPanel(tocWidth, m.height)
	readingArea := m.viewReading(readingWidth)

	return lipgloss.JoinHorizontal(lipgloss.Top, tocPanel, readingArea)
}

func (m model) renderTOCPanel(width, height int) string {
	title := tocTitleStyle.Render("Table of Contents")
	instructions := controlsStyle.Render("↑/↓: navigate  Enter: select  T/Esc: close")

	listHeight := height - 4
	if listHeight < 3 {
		listHeight = 3
	}
	m.tocList.SetSize(width-4, listHeight)

	content := fmt.Sprintf("%s\n\n%s\n\n%s", title, m.tocList.View(), instructions)

	return tocPanelStyle.Width(width - 2).Height(height - 2).Render(content)
}

func formatWord(word string) string {
	runes := []rune(word)
	orp := reader.GetORPPosition(word)
	if orp >= len(runes) {
		orp = len(runes) - 1
	}
	if orp < 0 {
		orp = 0
	}

	before := string(runes[:orp])
	focus := string(runes[orp])
	after := ""
	if orp+1 < len(runes) {
		after = string(runes[orp+1:])
	}

	return wordBeforeStyle.Render(before) +
		erpStyle.Render(focus) +
		wordAfterStyle.Render(after)
}

func anchorORPText(text string, word string, width int) string {
	anchor := width / 2
	orp := reader.GetORPPosition(word)
	pad := anchor - orp
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + text
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newModel(text string, wpm int, toc []reader.TOCEntry, chapters []reader.Chapter) model {
	r := reader.NewReader(text, wpm)
	r.SetChapters(chapters, toc)

	items := make([]list.Item, len(toc))
	for i, entry := range toc {
		items[i] = tocItem{entry: entry}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetHeight(2)

	tocList := list.New(items, delegate, 30, 20)
	tocList.Title = ""
	tocList.SetShowTitle(false)
	tocList.SetShowStatusBar(false)
	tocList.SetFilteringEnabled(true)
	tocList.SetShowHelp(false)

	return model{
		Reader:   r,
		quitting: false,
		width:    80,
		height:   24,
		tocList:  tocList,
	}
}

func main() {
	wpm := flag.Int("w", 300, "Words per minute (default: 300)")
	showVersion := flag.Bool("v", false, "Show version information")
	showVersionLong := flag.Bool("version", false, "Show version information")
	showTOC := flag.Bool("toc", false, "Show table of contents at startup")
	freshStart := flag.Bool("fresh", false, "Ignore saved reading position")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Brr - Terminal Speed Reading Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  brr [options] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  brr file.txt              Read from file at 300 WPM\n")
		fmt.Fprintf(os.Stderr, "  brr -w 500 file.txt       Read from file at 500 WPM\n")
		fmt.Fprintf(os.Stderr, "  brr --toc book.epub       Show TOC panel at startup\n")
		fmt.Fprintf(os.Stderr, "  brr --fresh book.epub     Start from beginning\n")
		fmt.Fprintf(os.Stderr, "  cat file.txt | brr        Read from stdin\n")
		fmt.Fprintf(os.Stderr, "\nControls:\n")
		fmt.Fprintf(os.Stderr, "  SPACE    Pause/play\n")
		fmt.Fprintf(os.Stderr, "  +/-      Increase/decrease speed by 50 WPM\n")
		fmt.Fprintf(os.Stderr, "  ↑/↓      Increase/decrease speed by 50 WPM\n")
		fmt.Fprintf(os.Stderr, "  ←/→      Jump to previous/next sentence\n")
		fmt.Fprintf(os.Stderr, "  T        Toggle table of contents\n")
		fmt.Fprintf(os.Stderr, "  R        Restart from beginning\n")
		fmt.Fprintf(os.Stderr, "  Q        Quit\n")
	}
	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Printf("brr %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	var text string
	var toc []reader.TOCEntry
	var chapters []reader.Chapter
	var sourceFile string

	if flag.NArg() > 0 {
		sourceFile = flag.Arg(0)

		if provider, ok := getTOCProvider(sourceFile); ok {
			var err error
			toc, err = provider.TOC(sourceFile)
			if err != nil {
				toc = nil
			}
		}

		if extractor, ok := getChapterExtractor(sourceFile); ok {
			var words []string
			var err error
			chapters, words, err = extractor.ExtractChapters(sourceFile)
			if err == nil && len(words) > 0 {
				text = strings.Join(words, " ")
			}
		}

		if text == "" {
			var err error
			text, err = reader.ExtractText(sourceFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Failed to read file '%s': %v\n", sourceFile, err)
				os.Exit(1)
			}
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Error: No input provided. Provide a file or pipe text to stdin.")
			fmt.Fprintln(os.Stderr, "Try: brr -h")
			os.Exit(1)
		}

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		text = string(data)
	}

	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "Error: No text to read.")
		os.Exit(1)
	}

	m := newModel(text, *wpm, toc, chapters)
	m.sourceFile = sourceFile

	if sourceFile != "" {
		store, err := state.NewStateStore()
		if err == nil {
			m.stateStore = store
			hash, err := state.ComputeHash(sourceFile)
			if err == nil {
				m.fileHash = hash
				if !*freshStart {
					if pos := store.GetPosition(hash); pos > 0 && pos < len(m.Words) {
						m.CurrentIndex = pos
					}
				}
			}
		}
	}

	if *showTOC && len(toc) > 0 {
		m.tocVisible = true
		m.Paused = true
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getTOCProvider(filename string) (reader.TOCProvider, bool) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".epub"):
		return &reader.EPUBFormat{}, true
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"):
		return &reader.MarkdownFormat{}, true
	}
	return nil, false
}

func getChapterExtractor(filename string) (reader.ChapterExtractor, bool) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".epub"):
		return &reader.EPUBFormat{}, true
	case strings.HasSuffix(lower, ".md"), strings.HasSuffix(lower, ".markdown"):
		return &reader.MarkdownFormat{}, true
	}
	return nil, false
}
