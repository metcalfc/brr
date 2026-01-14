package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple sentence",
			input:    "Hello world this is a test",
			expected: []string{"Hello", "world", "this", "is", "a", "test"},
		},
		{
			name:     "multiple spaces",
			input:    "Hello    world     test",
			expected: []string{"Hello", "world", "test"},
		},
		{
			name:     "newlines and tabs",
			input:    "Hello\nworld\ttest",
			expected: []string{"Hello", "world", "test"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single word",
			input:    "Hello",
			expected: []string{"Hello"},
		},
		{
			name:     "punctuation",
			input:    "Hello, world! How are you?",
			expected: []string{"Hello,", "world!", "How", "are", "you?"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseText(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseText() length = %v, want %v", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseText()[%d] = %v, want %v", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetORPPosition(t *testing.T) {
	tests := []struct {
		name     string
		word     string
		expected int
	}{
		{"single char", "a", 0},
		{"two chars", "ab", 1},
		{"three chars", "abc", 1},
		{"four chars", "abcd", 1},
		{"five chars", "abcde", 1},
		{"six chars", "abcdef", 2},
		{"nine chars", "abcdefghi", 3},
		{"twelve chars", "abcdefghijkl", 4},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getORPPosition(tt.word)
			if result != tt.expected {
				t.Errorf("getORPPosition(%q) = %v, want %v", tt.word, result, tt.expected)
			}
		})
	}
}

func TestFormatWord(t *testing.T) {
	tests := []struct {
		name string
		word string
	}{
		{"simple word", "hello"},
		{"single char", "a"},
		{"with punctuation", "hello,"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWord(tt.word)
			// Just check that we get a non-empty result
			if result == "" {
				t.Errorf("formatWord(%q) returned empty string", tt.word)
			}
			// Result should contain the original word (possibly with styling)
			if !strings.Contains(result, tt.word[0:1]) {
				t.Errorf("formatWord(%q) should contain first character", tt.word)
			}
		})
	}
}

func TestNewModel(t *testing.T) {
	text := "Hello world test"
	wpm := 500

	m := newModel(text, wpm)

	if m.wpm != wpm {
		t.Errorf("newModel() wpm = %v, want %v", m.wpm, wpm)
	}

	if len(m.words) != 3 {
		t.Errorf("newModel() words length = %v, want %v", len(m.words), 3)
	}

	if m.currentIndex != 0 {
		t.Errorf("newModel() currentIndex = %v, want %v", m.currentIndex, 0)
	}

	if m.paused != false {
		t.Errorf("newModel() paused = %v, want %v", m.paused, false)
	}

	if m.quitting != false {
		t.Errorf("newModel() quitting = %v, want %v", m.quitting, false)
	}
}

func TestModelGetDelay(t *testing.T) {
	tests := []struct {
		name     string
		wpm      int
		expected time.Duration
	}{
		{"300 wpm", 300, 200 * time.Millisecond},
		{"600 wpm", 600, 100 * time.Millisecond},
		{"100 wpm", 100, 600 * time.Millisecond},
		{"900 wpm", 900, 66666667 * time.Nanosecond}, // ~66.67ms
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModel("test", tt.wpm)
			result := m.getDelay()
			// Allow for small floating point differences
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Millisecond {
				t.Errorf("getDelay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelUpdate(t *testing.T) {
	t.Run("space pauses", func(t *testing.T) {
		m := newModel("hello world", 300)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if !updated.paused {
			t.Error("space should pause the model")
		}
	})

	t.Run("space unpauses", func(t *testing.T) {
		m := newModel("hello world", 300)
		m.paused = true
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.paused {
			t.Error("space should unpause the model")
		}
	})

	t.Run("plus increases speed", func(t *testing.T) {
		m := newModel("hello world", 300)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.wpm != 350 {
			t.Errorf("plus should increase wpm to 350, got %d", updated.wpm)
		}
	})

	t.Run("minus decreases speed", func(t *testing.T) {
		m := newModel("hello world", 300)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.wpm != 250 {
			t.Errorf("minus should decrease wpm to 250, got %d", updated.wpm)
		}
	})

	t.Run("speed caps at 1500", func(t *testing.T) {
		m := newModel("hello world", 1500)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.wpm != 1500 {
			t.Errorf("wpm should cap at 1500, got %d", updated.wpm)
		}
	})

	t.Run("speed floors at 100", func(t *testing.T) {
		m := newModel("hello world", 100)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.wpm != 100 {
			t.Errorf("wpm should floor at 100, got %d", updated.wpm)
		}
	})

	t.Run("q quits", func(t *testing.T) {
		m := newModel("hello world", 300)
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

		updatedModel, cmd := m.Update(msg)
		updated := updatedModel.(model)

		if !updated.quitting {
			t.Error("q should set quitting to true")
		}
		if cmd == nil {
			t.Error("q should return tea.Quit command")
		}
	})

	t.Run("tick advances word", func(t *testing.T) {
		m := newModel("hello world test", 300)
		msg := tickMsg(time.Now())

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.currentIndex != 1 {
			t.Errorf("tick should advance index to 1, got %d", updated.currentIndex)
		}
	})

	t.Run("tick doesn't advance when paused", func(t *testing.T) {
		m := newModel("hello world test", 300)
		m.paused = true
		msg := tickMsg(time.Now())

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.currentIndex != 0 {
			t.Errorf("tick should not advance when paused, got index %d", updated.currentIndex)
		}
	})

	t.Run("window size updates dimensions", func(t *testing.T) {
		m := newModel("hello world", 300)
		msg := tea.WindowSizeMsg{Width: 120, Height: 40}

		updatedModel, _ := m.Update(msg)
		updated := updatedModel.(model)

		if updated.width != 120 {
			t.Errorf("width should be 120, got %d", updated.width)
		}
		if updated.height != 40 {
			t.Errorf("height should be 40, got %d", updated.height)
		}
	})
}

func TestModelView(t *testing.T) {
	t.Run("shows word", func(t *testing.T) {
		m := newModel("hello world", 300)
		view := m.View()

		// Should contain word tracking info
		if !strings.Contains(view, "Word 1/2") {
			t.Error("view should contain word count")
		}
		if !strings.Contains(view, "300 WPM") {
			t.Error("view should contain WPM")
		}
	})

	t.Run("shows paused state", func(t *testing.T) {
		m := newModel("hello world", 300)
		m.paused = true
		view := m.View()

		if !strings.Contains(view, "PAUSED") {
			t.Error("view should show paused state")
		}
	})

	t.Run("shows completion", func(t *testing.T) {
		m := newModel("hello", 300)
		m.currentIndex = 0
		m.quitting = true
		view := m.View()

		if !strings.Contains(view, "Reading complete") {
			t.Error("view should show completion message")
		}
	})
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
	}{
		{"short text", "hello", 20},
		{"exact width", "hello", 5},
		{"longer than width", "hello world", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := centerText(tt.text, tt.width)
			// Just check we get a result
			if result == "" && tt.text != "" {
				t.Error("centerText should return non-empty result")
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseText(b *testing.B) {
	text := strings.Repeat("Hello world this is a test sentence with multiple words. ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parseText(text)
	}
}

func BenchmarkGetORPPosition(b *testing.B) {
	words := []string{"a", "hello", "testing", "extraordinary", "supercalifragilisticexpialidocious"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, word := range words {
			getORPPosition(word)
		}
	}
}

func BenchmarkFormatWord(b *testing.B) {
	words := []string{"a", "hello", "testing", "extraordinary"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, word := range words {
			formatWord(word)
		}
	}
}

func BenchmarkModelView(b *testing.B) {
	m := newModel("hello world this is a test", 300)
	m.width = 80
	m.height = 24
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.View()
	}
}
