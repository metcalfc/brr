//go:build gui

package main

import (
	"flag"
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type model struct {
	words          []string
	sentenceStarts []int
	currentIndex   int
	wpm            int
	fontSize       float32
	paused         bool
	lastArrowPress time.Time
}

func parseText(text string) []string {
	return strings.Fields(text)
}

func findSentenceStarts(words []string) []int {
	starts := []int{0}
	for i, word := range words {
		if len(word) > 0 {
			last := word[len(word)-1]
			if last == '.' || last == '!' || last == '?' {
				if i+1 < len(words) {
					starts = append(starts, i+1)
				}
			}
		}
	}
	return starts
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

func (m *model) jumpToPrevSentence() {
	for i := len(m.sentenceStarts) - 1; i >= 0; i-- {
		if m.sentenceStarts[i] < m.currentIndex {
			m.currentIndex = m.sentenceStarts[i]
			return
		}
	}
	m.currentIndex = 0
}

func (m *model) jumpToNextSentence() {
	for i := 0; i < len(m.sentenceStarts); i++ {
		if m.sentenceStarts[i] > m.currentIndex {
			m.currentIndex = m.sentenceStarts[i]
			return
		}
	}
	if len(m.words) > 0 {
		m.currentIndex = len(m.words) - 1
	}
}

func newModel(text string, wpm int) *model {
	words := parseText(text)
	return &model{
		words:          words,
		sentenceStarts: findSentenceStarts(words),
		currentIndex:   0,
		wpm:            wpm,
		fontSize:       72,
		paused:         true,
		lastArrowPress: time.Time{},
	}
}

func (m *model) getDelay() time.Duration {
	return time.Duration(60.0/float64(m.wpm)*1000) * time.Millisecond
}

func createWordDisplay(word string, fontSize float32, windowWidth float32) *fyne.Container {
	orp := getORPPosition(word)

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
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: grr [options] <file>")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	filename := flag.Arg(0)
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to read file '%s': %v\n", filename, err)
		os.Exit(1)
	}

	text := string(data)
	if strings.TrimSpace(text) == "" {
		fmt.Fprintln(os.Stderr, "Error: No text to read.")
		os.Exit(1)
	}

	m := newModel(text, *wpm)

	a := app.New()
	w := a.NewWindow("grr - Speed Reader")

	statusLabel := widget.NewLabel(fmt.Sprintf("Word %d/%d | %d WPM | Font: %.0f [PAUSED]",
		m.currentIndex+1, len(m.words), m.wpm, m.fontSize))
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

	ticker := time.NewTicker(m.getDelay())
	done := make(chan bool)
	var closeOnce sync.Once

	updateDisplay := func() {
		if m.currentIndex >= len(m.words) {
			m.currentIndex = len(m.words) - 1
		}

		canvasWidth := w.Canvas().Size().Width
		if canvasWidth <= 0 {
			canvasWidth = 800
		}

		newWordDisplay := createWordDisplay(m.words[m.currentIndex], m.fontSize, canvasWidth)

		// Replace all objects in wordContainer
		wordContainer.Objects = []fyne.CanvasObject{newWordDisplay}
		wordContainer.Refresh()

		pauseText := ""
		if m.paused {
			pauseText = " [PAUSED]"
		}
		statusLabel.SetText(fmt.Sprintf("Word %d/%d | %d WPM | Font: %.0f%s",
			m.currentIndex+1, len(m.words), m.wpm, m.fontSize, pauseText))
	}

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if !m.paused && m.currentIndex < len(m.words)-1 {
					m.currentIndex++
					fyne.Do(updateDisplay)
				} else if m.currentIndex >= len(m.words)-1 && !m.paused {
					m.paused = true
					fyne.Do(updateDisplay)
				}
			}
		}
	}()

	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeySpace:
			m.paused = !m.paused
			updateDisplay()

		case fyne.KeyUp:
			if m.wpm < 1500 {
				m.wpm += 50
				ticker.Reset(m.getDelay())
				updateDisplay()
			}

		case fyne.KeyDown:
			if m.wpm > 100 {
				m.wpm -= 50
				ticker.Reset(m.getDelay())
				updateDisplay()
			}

		case fyne.KeyLeft:
			now := time.Now()
			if now.Sub(m.lastArrowPress) > 500*time.Millisecond {
				m.paused = true
			}
			m.lastArrowPress = now
			m.jumpToPrevSentence()
			updateDisplay()

		case fyne.KeyRight:
			now := time.Now()
			if now.Sub(m.lastArrowPress) > 500*time.Millisecond {
				m.paused = true
			}
			m.lastArrowPress = now
			m.jumpToNextSentence()
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
					m.paused = true
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
