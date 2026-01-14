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
	words        []string
	currentIndex int
	wpm          int
	paused       bool
	quitting     bool
	width        int
	height       int
}

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tick(m.getDelay())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ":
			m.paused = !m.paused
			if !m.paused {
				return m, tick(m.getDelay())
			}
			return m, nil

		case "+", "=":
			if m.wpm < 1500 {
				m.wpm += 50
			}
			return m, nil

		case "-":
			if m.wpm > 100 {
				m.wpm -= 50
			}
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
		if m.paused {
			return m, nil
		}

		if m.currentIndex < len(m.words)-1 {
			m.currentIndex++
			return m, tick(m.getDelay())
		}

		// Reached the end
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		if m.currentIndex >= len(m.words)-1 {
			return completeStyle.Render("\n  Reading complete!\n")
		}
		return ""
	}

	if len(m.words) == 0 {
		return "No text to read."
	}

	word := m.words[m.currentIndex]
	formatted := formatWord(word)

	pause := ""
	if m.paused {
		pause = pausedStyle.Render(" [PAUSED]")
	}

	status := statusStyle.Render(
		fmt.Sprintf("Word %d/%d | %d WPM%s",
			m.currentIndex+1,
			len(m.words),
			m.wpm,
			pause,
		),
	)

	controls := controlsStyle.Render("SPACE: pause/play  +/-: speed  Q: quit")

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
	orp := getORPPosition(word)

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

func getORPPosition(word string) int {
	length := len(word)
	if length <= 1 {
		return 0
	} else if length <= 5 {
		return 1
	}
	return length / 3
}

func anchorORPText(text string, word string, width int) string {
	anchor := width / 2
	orp := getORPPosition(word)
	pad := anchor - orp
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + text
}

func (m model) getDelay() time.Duration {
	return time.Duration(60.0/float64(m.wpm)*1000) * time.Millisecond
}

func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func parseText(text string) []string {
	return strings.Fields(text)
}

func newModel(text string, wpm int) model {
	return model{
		words:        parseText(text),
		currentIndex: 0,
		wpm:          wpm,
		paused:       false,
		quitting:     false,
		width:        80,
		height:       24,
	}
}

func main() {
	wpm := flag.Int("w", 300, "Words per minute (default: 300)")
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
		fmt.Fprintf(os.Stderr, "  Q        Quit\n")
	}
	flag.Parse()

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
