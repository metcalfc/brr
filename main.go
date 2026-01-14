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

// Styles using lipgloss
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

// Model holds the state of the application
type model struct {
	words        []string
	currentIndex int
	wpm          int
	paused       bool
	quitting     bool
	width        int
	height       int
}

// tickMsg is sent on each tick to advance to the next word
type tickMsg time.Time

// Init initializes the model
func (m model) Init() tea.Cmd {
	return tick(m.getDelay())
}

// Update handles messages
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

// View renders the UI
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

	// Get current word
	word := m.words[m.currentIndex]

	// Format the word with ORP highlighting
	formattedWord := formatWord(word)

	// Create status line
	pausedText := ""
	if m.paused {
		pausedText = pausedStyle.Render(" [PAUSED]")
	}

	status := statusStyle.Render(
		fmt.Sprintf("Word %d/%d | %d WPM%s",
			m.currentIndex+1,
			len(m.words),
			m.wpm,
			pausedText,
		),
	)

	// Create controls line
	controls := controlsStyle.Render("SPACE: pause/play  +/-: speed  Q: quit")

	// Calculate vertical centering for the word only
	// Reserve 2 lines: 1 for status at top, 1 for controls at bottom
	availableHeight := m.height - 2
	if availableHeight < 1 {
		availableHeight = 1
	}
	vPadding := availableHeight / 2
	if vPadding < 0 {
		vPadding = 0
	}

	// Build the complete view
	var sb strings.Builder

	// Status at very top
	sb.WriteString(status)
	sb.WriteString("\n")

	// Padding before word
	for i := 0; i < vPadding; i++ {
		sb.WriteString("\n")
	}

	// Anchor the ORP character at a fixed position
	wordLine := anchorORPText(formattedWord, word, m.width)
	sb.WriteString(wordLine)

	// Padding after word (to push controls to bottom)
	remainingLines := availableHeight - vPadding
	for i := 0; i < remainingLines; i++ {
		sb.WriteString("\n")
	}

	// Controls at very bottom
	sb.WriteString(controls)

	return sb.String()
}

// formatWord formats a word with ORP highlighting using lipgloss
func formatWord(word string) string {
	orp := getORPPosition(word)

	if orp >= len(word) {
		orp = len(word) - 1
	}

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

// getORPPosition calculates the Optimal Recognition Point position
func getORPPosition(word string) int {
	length := len(word)
	if length <= 1 {
		return 0
	} else if length <= 5 {
		return 1
	}
	return length / 3
}

// anchorORPText positions text so the ORP character is at a fixed anchor point
func anchorORPText(formattedText string, originalWord string, width int) string {
	// Calculate the fixed anchor point (center of screen)
	anchorPoint := width / 2

	// Find the ORP position in the original word
	orpPos := getORPPosition(originalWord)

	// Calculate padding needed to position ORP at anchor point
	// The ORP should be at position: padding + orpPos = anchorPoint
	padding := anchorPoint - orpPos
	if padding < 0 {
		padding = 0
	}

	return strings.Repeat(" ", padding) + formattedText
}

// getDelay calculates delay between words based on WPM
func (m model) getDelay() time.Duration {
	return time.Duration(60.0/float64(m.wpm)*1000) * time.Millisecond
}

// tick creates a tick command with the given delay
func tick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// parseText splits text into words
func parseText(text string) []string {
	return strings.Fields(text)
}

// newModel creates a new model
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
	// Command line flags
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

	// Read text from file or stdin
	if flag.NArg() > 0 {
		// Read from file
		filename := flag.Arg(0)
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read file '%s': %v\n", filename, err)
			os.Exit(1)
		}
		text = string(data)
	} else {
		// Check if stdin is from terminal
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Fprintln(os.Stderr, "Error: No input provided. Provide a file or pipe text to stdin.")
			fmt.Fprintln(os.Stderr, "Try: brr -h")
			os.Exit(1)
		}

		// Read from stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		text = string(data)
	}

	// Validate text
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "Error: No text to read.")
		os.Exit(1)
	}

	// Create and run the bubbletea program
	m := newModel(text, *wpm)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
