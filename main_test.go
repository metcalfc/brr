package main

import (
	"strings"
	"testing"
	"time"
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
		name         string
		word         string
		shouldContain []string // strings the result should contain
	}{
		{
			name:         "simple word",
			word:         "hello",
			shouldContain: []string{"h", "e", "llo", colorRed, colorBold, colorReset},
		},
		{
			name:         "single char",
			word:         "a",
			shouldContain: []string{"a", colorRed, colorBold, colorReset},
		},
		{
			name:         "with punctuation",
			word:         "hello,",
			shouldContain: []string{"he", "l", "lo,", colorRed, colorBold, colorReset},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWord(tt.word)
			// Check that result contains ANSI color codes
			if !strings.Contains(result, colorRed) {
				t.Errorf("formatWord(%q) should contain red color code", tt.word)
			}
			if !strings.Contains(result, colorBold) {
				t.Errorf("formatWord(%q) should contain bold code", tt.word)
			}
			if !strings.Contains(result, colorReset) {
				t.Errorf("formatWord(%q) should contain reset code", tt.word)
			}
			// Check that the result has reasonable length (original + ANSI codes)
			if len(result) < len(tt.word) {
				t.Errorf("formatWord(%q) result too short: got %q", tt.word, result)
			}
		})
	}
}

func TestNewSpeedReader(t *testing.T) {
	text := "Hello world test"
	wpm := 500

	sr := NewSpeedReader(text, wpm)

	if sr.wpm != wpm {
		t.Errorf("NewSpeedReader() wpm = %v, want %v", sr.wpm, wpm)
	}

	if len(sr.words) != 3 {
		t.Errorf("NewSpeedReader() words length = %v, want %v", len(sr.words), 3)
	}

	if sr.currentIndex != 0 {
		t.Errorf("NewSpeedReader() currentIndex = %v, want %v", sr.currentIndex, 0)
	}

	if sr.paused != false {
		t.Errorf("NewSpeedReader() paused = %v, want %v", sr.paused, false)
	}

	if sr.running != true {
		t.Errorf("NewSpeedReader() running = %v, want %v", sr.running, true)
	}
}

func TestSpeedReaderGetDelay(t *testing.T) {
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
			sr := NewSpeedReader("test", tt.wpm)
			result := sr.getDelay()
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

func TestSpeedReaderHandleInput(t *testing.T) {
	tests := []struct {
		name          string
		initialWPM    int
		initialPaused bool
		input         byte
		expectedWPM   int
		expectedPaused bool
		expectRunning  bool
	}{
		{"space pauses", 300, false, ' ', 300, true, true},
		{"space unpauses", 300, true, ' ', 300, false, true},
		{"plus increases speed", 300, false, '+', 350, false, true},
		{"equals increases speed", 300, false, '=', 350, false, true},
		{"minus decreases speed", 300, false, '-', 250, false, true},
		{"plus caps at 1500", 1500, false, '+', 1500, false, true},
		{"minus floors at 100", 100, false, '-', 100, false, true},
		{"q quits", 300, false, 'q', 300, false, false},
		{"Q quits", 300, false, 'Q', 300, false, false},
		{"ctrl-c quits", 300, false, 3, 300, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := NewSpeedReader("test", tt.initialWPM)
			sr.paused = tt.initialPaused

			sr.handleInput(tt.input)

			if sr.wpm != tt.expectedWPM {
				t.Errorf("handleInput() wpm = %v, want %v", sr.wpm, tt.expectedWPM)
			}
			if sr.paused != tt.expectedPaused {
				t.Errorf("handleInput() paused = %v, want %v", sr.paused, tt.expectedPaused)
			}
			if sr.running != tt.expectRunning {
				t.Errorf("handleInput() running = %v, want %v", sr.running, tt.expectRunning)
			}
		})
	}
}

func TestGetTerminalSize(t *testing.T) {
	// This is a simple test that just checks the function returns reasonable values
	width, height := getTerminalSize()

	if width < 1 || width > 1000 {
		t.Errorf("getTerminalSize() width = %v, expected reasonable value", width)
	}

	if height < 1 || height > 1000 {
		t.Errorf("getTerminalSize() height = %v, expected reasonable value", height)
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
