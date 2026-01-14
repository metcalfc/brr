package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

// ANSI color codes
const (
	colorRed   = "\033[91m"
	colorBold  = "\033[1m"
	colorReset = "\033[0m"
)

// SpeedReader manages the speed reading session
type SpeedReader struct {
	words        []string
	wpm          int
	currentIndex int
	paused       bool
	running      bool
	oldState     *term.State
}

// NewSpeedReader creates a new speed reader instance
func NewSpeedReader(text string, wpm int) *SpeedReader {
	return &SpeedReader{
		words:        parseText(text),
		wpm:          wpm,
		currentIndex: 0,
		paused:       false,
		running:      true,
	}
}

// parseText splits text into words
func parseText(text string) []string {
	return strings.Fields(text)
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

// formatWord formats a word with ORP character highlighted in red
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

	return fmt.Sprintf("%s%s%s%s%s%s", before, colorRed, colorBold, focus, colorReset, after)
}

// clearScreen clears the terminal screen
func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// getTerminalSize returns terminal dimensions
func getTerminalSize() (width, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return width, height
}

// displayWord displays a single word centered on screen
func (sr *SpeedReader) displayWord(word string, info string) {
	clearScreen()

	cols, rows := getTerminalSize()

	// Display info at top
	fmt.Printf("\033[1;1H%s", info)

	// Display word at center
	vCenter := rows / 2
	visualLength := len(word) // Visual length of the word without ANSI codes
	hCenter := (cols - visualLength) / 2

	formattedWord := formatWord(word)
	fmt.Printf("\033[%d;%dH%s", vCenter, hCenter, formattedWord)

	// Display controls at bottom
	controls := "SPACE: pause/play  +/-: speed  Q: quit"
	fmt.Printf("\033[%d;1H%s", rows, controls)
}

// getDelay calculates delay between words based on WPM
func (sr *SpeedReader) getDelay() time.Duration {
	return time.Duration(60.0/float64(sr.wpm)*1000) * time.Millisecond
}

// handleInput processes keyboard input
func (sr *SpeedReader) handleInput(key byte) {
	switch key {
	case ' ':
		sr.paused = !sr.paused
	case '+', '=':
		if sr.wpm < 1500 {
			sr.wpm += 50
		}
	case '-':
		if sr.wpm > 100 {
			sr.wpm -= 50
		}
	case 'q', 'Q', 3: // q, Q, or Ctrl+C
		sr.running = false
	}
}

// startInputReader starts a goroutine that continuously reads from stdin
func startInputReader() chan byte {
	keyCh := make(chan byte, 10) // Buffered channel to avoid missing keys

	go func() {
		var buf [1]byte
		for {
			n, err := os.Stdin.Read(buf[:])
			if err != nil {
				close(keyCh)
				return
			}
			if n > 0 {
				keyCh <- buf[0]
			}
		}
	}()

	return keyCh
}

// Read starts the main reading loop
func (sr *SpeedReader) Read() error {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	sr.oldState = oldState

	defer func() {
		// Restore terminal
		term.Restore(int(os.Stdin.Fd()), sr.oldState)
		clearScreen()
	}()

	// Start input reader goroutine
	keyCh := startInputReader()

	// Create ticker for timing
	checkInterval := 10 * time.Millisecond
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for sr.running && sr.currentIndex < len(sr.words) {
		word := sr.words[sr.currentIndex]

		// Display status info
		pausedText := ""
		if sr.paused {
			pausedText = " | PAUSED"
		}
		progress := fmt.Sprintf("Word %d/%d | %d WPM%s",
			sr.currentIndex+1, len(sr.words), sr.wpm, pausedText)

		sr.displayWord(word, progress)

		// Wait for delay or process input
		delay := sr.getDelay()
		elapsed := time.Duration(0)

		for elapsed < delay && sr.running {
			select {
			case key, ok := <-keyCh:
				if !ok {
					// Input channel closed
					sr.running = false
					break
				}
				sr.handleInput(key)
				if !sr.running {
					break
				}
				// Redisplay with updated state
				pausedText = ""
				if sr.paused {
					pausedText = " | PAUSED"
				}
				progress = fmt.Sprintf("Word %d/%d | %d WPM%s",
					sr.currentIndex+1, len(sr.words), sr.wpm, pausedText)
				sr.displayWord(word, progress)

			case <-ticker.C:
				if !sr.paused {
					elapsed += checkInterval
				}
			}
		}

		if !sr.paused && sr.running {
			sr.currentIndex++
		}
	}

	// Show completion message
	if sr.running {
		clearScreen()
		fmt.Println("\n\n  Reading complete!")
	}

	return nil
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
		if term.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Fprintln(os.Stderr, "Error: No input provided. Provide a file or pipe text to stdin.")
			fmt.Fprintln(os.Stderr, "Try: brr -h")
			os.Exit(1)
		}

		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		var sb strings.Builder
		for {
			line, err := reader.ReadString('\n')
			sb.WriteString(line)
			if err != nil {
				if err == io.EOF {
					break
				}
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
		}
		text = sb.String()
	}

	// Validate text
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "Error: No text to read.")
		os.Exit(1)
	}

	// Start speed reading
	reader := NewSpeedReader(text, *wpm)
	if err := reader.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
