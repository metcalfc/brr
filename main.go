//go:build !gui

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/metcalfc/brr/internal/reader"
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
)

type model struct {
	*reader.Reader
	quitting bool
	width    int
	height   int
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tick(m.GetDelay())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		case "q", "Q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.Paused {
			return m, nil
		}

		if m.Advance() {
			return m, tick(m.GetDelay())
		}

		// Reached the end
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
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

	word := m.CurrentWord()
	formatted := formatWord(word)

	pause := ""
	if m.Paused {
		pause = pausedStyle.Render(" [PAUSED]")
	}

	current, total := m.Progress()
	status := statusStyle.Render(
		fmt.Sprintf("Word %d/%d | %d WPM%s",
			current,
			total,
			m.WPM,
			pause,
		),
	)

	controls := controlsStyle.Render("SPACE: pause/play  ↑/↓: speed  ←/→: sentence  Q: quit")

	// Reserve 2 lines: 1 for status at top, 1 for controls at bottom
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

	line := anchorORPText(formatted, word, m.width)
	sb.WriteString(line)

	remaining := avail - vPad
	for i := 0; i < remaining; i++ {
		sb.WriteString("\n")
	}

	sb.WriteString(controls)

	return sb.String()
}

func formatWord(word string) string {
	orp := reader.GetORPPosition(word)

	before := word[:orp]
	focus := string(word[orp])
	after := ""
	if orp+1 < len(word) {
		after = word[orp+1:]
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

func newModel(text string, wpm int) model {
	return model{
		Reader:   reader.NewReader(text, wpm),
		quitting: false,
		width:    80,
		height:   24,
	}
}

func main() {
	wpm := flag.Int("w", 300, "Words per minute (default: 300)")
	showVersion := flag.Bool("v", false, "Show version information")
	showVersionLong := flag.Bool("version", false, "Show version information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Brr - Terminal Speed Reading Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  brr [options] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  brr file.txt              Read from file at 300 WPM\n")
		fmt.Fprintf(os.Stderr, "  brr -w 500 file.txt       Read from file at 500 WPM\n")
		fmt.Fprintf(os.Stderr, "  cat file.txt | brr        Read from stdin\n")
		fmt.Fprintf(os.Stderr, "  echo \"Hello world\" | brr  Read from stdin\n")
		fmt.Fprintf(os.Stderr, "\nControls:\n")
		fmt.Fprintf(os.Stderr, "  SPACE    Pause/play\n")
		fmt.Fprintf(os.Stderr, "  +/-      Increase/decrease speed by 50 WPM\n")
		fmt.Fprintf(os.Stderr, "  ↑/↓      Increase/decrease speed by 50 WPM\n")
		fmt.Fprintf(os.Stderr, "  ←/→      Jump to previous/next sentence\n")
		fmt.Fprintf(os.Stderr, "  Q        Quit\n")
	}
	flag.Parse()

	if *showVersion || *showVersionLong {
		fmt.Printf("brr %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	var text string

	if flag.NArg() > 0 {
		filename := flag.Arg(0)
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read file '%s': %v\n", filename, err)
			os.Exit(1)
		}
		text = string(data)
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

	m := newModel(text, *wpm)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
