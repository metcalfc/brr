//go:build gui

package main

import (
	"flag"
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/metcalfc/brr/internal/reader"
)

// Version info (injected via ldflags)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type model struct {
	*reader.Reader
	fontSize float32
}

func newModel(text string, wpm int) *model {
	r := reader.NewReader(text, wpm)
	r.Paused = true // GUI starts paused
	return &model{
		Reader:   r,
		fontSize: 72,
	}
}

func createWordDisplay(word string, fontSize float32, windowWidth float32) *fyne.Container {
	orp := reader.GetORPPosition(word)

	before := word[:orp]
	focus := string(word[orp])
	after := ""
	if orp+1 < len(word) {
		after = word[orp+1:]
	}

	beforeText := canvas.NewText(before, color.White)
	beforeText.TextSize = fontSize
	beforeText.TextStyle.Bold = true

	focusText := canvas.NewText(focus, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	focusText.TextSize = fontSize
	focusText.TextStyle.Bold = true

	afterText := canvas.NewText(after, color.White)
	afterText.TextSize = fontSize
	afterText.TextStyle.Bold = true

	// Measure text
	beforeSize := beforeText.MinSize()
	focusSize := focusText.MinSize()

	// Horizontal: anchor ORP at center
	centerX := windowWidth / 2
	beforeX := centerX - beforeSize.Width
	focusX := centerX
	afterX := centerX + focusSize.Width

	if beforeX < 0 {
		beforeX = 0
	}

	// Create a container that will be positioned by border layout
	// We'll use a custom layout to center vertically
	c := &fyne.Container{
		Layout: &centerVerticalLayout{},
		Objects: []fyne.CanvasObject{
			beforeText,
			focusText,
			afterText,
		},
	}

	// Position horizontally
	beforeText.Move(fyne.NewPos(beforeX, 0))
	focusText.Move(fyne.NewPos(focusX, 0))
	afterText.Move(fyne.NewPos(afterX, 0))

	return c
}

type centerVerticalLayout struct{}

func (l *centerVerticalLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var maxH float32
	for _, o := range objects {
		size := o.MinSize()
		if size.Height > maxH {
			maxH = size.Height
		}
	}
	return fyne.NewSize(0, maxH)
}

func (l *centerVerticalLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Find max text height
	var maxH float32
	for _, o := range objects {
		objSize := o.MinSize()
		if objSize.Height > maxH {
			maxH = objSize.Height
		}
	}

	// Center vertically
	y := (size.Height - maxH) / 2
	if y < 0 {
		y = 0
	}

	// Position each object at the correct Y (X already set)
	for _, o := range objects {
		pos := o.Position()
		o.Move(fyne.NewPos(pos.X, y))
		o.Resize(o.MinSize())
	}
}

func main() {
	wpm := flag.Int("w", 300, "Words per minute")
	showVersion := flag.Bool("v", false, "Show version information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Grr - GUI Speed Reading Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  grr [options] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  grr file.txt              Read from file at 300 WPM\n")
		fmt.Fprintf(os.Stderr, "  grr -w 500 file.txt       Read from file at 500 WPM\n")
		fmt.Fprintf(os.Stderr, "  cat file.txt | grr        Read from stdin\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("grr %s (commit: %s, built: %s)\n", version, commit, date)
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
			fmt.Fprintln(os.Stderr, "Try: grr -h")
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

	a := app.New()
	w := a.NewWindow("grr - Speed Reader")

	current, total := m.Progress()
	statusLabel := widget.NewLabel(fmt.Sprintf("Word %d/%d | %d WPM | Font: %.0f [PAUSED]",
		current, total, m.WPM, m.fontSize))
	statusLabel.Alignment = fyne.TextAlignCenter

	controlsLabel := widget.NewLabel("SPACE: pause/play  ↑/↓: speed  +/-: font size  ←/→: sentence  F: fullscreen  Q: quit")
	controlsLabel.Alignment = fyne.TextAlignCenter

	// Create placeholder for word display
	wordContainer := container.NewMax()

	content := container.NewBorder(
		statusLabel,
		controlsLabel,
		nil, nil,
		wordContainer,
	)

	ticker := time.NewTicker(m.GetDelay())
	done := make(chan bool)
	var closeOnce sync.Once

	updateDisplay := func() {
		if m.CurrentIndex >= len(m.Words) {
			m.CurrentIndex = len(m.Words) - 1
		}

		canvasWidth := w.Canvas().Size().Width
		if canvasWidth <= 0 {
			canvasWidth = 800
		}

		newWordDisplay := createWordDisplay(m.CurrentWord(), m.fontSize, canvasWidth)

		// Replace all objects in wordContainer
		wordContainer.Objects = []fyne.CanvasObject{newWordDisplay}
		wordContainer.Refresh()

		pauseText := ""
		if m.Paused {
			pauseText = " [PAUSED]"
		}
		current, total := m.Progress()
		statusLabel.SetText(fmt.Sprintf("Word %d/%d | %d WPM | Font: %.0f%s",
			current, total, m.WPM, m.fontSize, pauseText))
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if !m.Paused && !m.AtEnd() {
					m.Advance()
					fyne.Do(updateDisplay)
				} else if m.AtEnd() && !m.Paused {
					m.Paused = true
					fyne.Do(updateDisplay)
				}
			}
		}
	}()

	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeySpace:
			m.Paused = !m.Paused
			updateDisplay()

		case fyne.KeyUp:
			if m.WPM < 1500 {
				m.WPM += 50
				ticker.Reset(m.GetDelay())
				updateDisplay()
			}

		case fyne.KeyDown:
			if m.WPM > 100 {
				m.WPM -= 50
				ticker.Reset(m.GetDelay())
				updateDisplay()
			}

		case fyne.KeyLeft:
			now := time.Now()
			if now.Sub(m.LastArrowPress) > 500*time.Millisecond {
				m.Paused = true
			}
			m.LastArrowPress = now
			m.JumpToPrevSentence()
			updateDisplay()

		case fyne.KeyRight:
			now := time.Now()
			if now.Sub(m.LastArrowPress) > 500*time.Millisecond {
				m.Paused = true
			}
			m.LastArrowPress = now
			m.JumpToNextSentence()
			updateDisplay()

		case fyne.KeyF:
			w.SetFullScreen(!w.FullScreen())

		case fyne.KeyQ:
			closeOnce.Do(func() {
				close(done)
			})
			a.Quit()
		}
	})

	// Handle +/- keys
	w.Canvas().SetOnTypedRune(func(r rune) {
		switch r {
		case '+', '=':
			if m.fontSize < 200 {
				m.fontSize += 5
				updateDisplay()
			}
		case '-':
			if m.fontSize > 20 {
				m.fontSize -= 5
				updateDisplay()
			}
		}
	})

	w.Resize(fyne.NewSize(800, 600))
	w.SetContent(content)

	// Handle window resize - pause and redraw
	var lastWidth float32
	lastWidth = 800
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				time.Sleep(100 * time.Millisecond)
				currentWidth := w.Canvas().Size().Width
				if currentWidth > 0 && currentWidth != lastWidth {
					lastWidth = currentWidth
					m.Paused = true
					fyne.Do(updateDisplay)
				}
			}
		}
	}()

	w.SetOnClosed(func() {
		closeOnce.Do(func() {
			close(done)
		})
	})

	// Initialize first word after window shows
	go func() {
		time.Sleep(100 * time.Millisecond)
		fyne.Do(updateDisplay)
	}()

	w.ShowAndRun()
}
