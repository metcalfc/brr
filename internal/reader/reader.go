// Package reader provides core RSVP (Rapid Serial Visual Presentation) speed reading logic.
package reader

import (
	"strings"
	"time"
)

// Reader holds the state for an RSVP speed reading session.
type Reader struct {
	Words          []string
	SentenceStarts []int
	CurrentIndex   int
	WPM            int
	Paused         bool
	LastArrowPress time.Time
}

// NewReader creates a new Reader from the given text and words-per-minute setting.
func NewReader(text string, wpm int) *Reader {
	words := ParseText(text)
	return &Reader{
		Words:          words,
		SentenceStarts: FindSentenceStarts(words),
		CurrentIndex:   0,
		WPM:            wpm,
		Paused:         false,
		LastArrowPress: time.Time{},
	}
}

// ParseText splits text into words.
func ParseText(text string) []string {
	return strings.Fields(text)
}

// FindSentenceStarts returns indices of words that start sentences.
func FindSentenceStarts(words []string) []int {
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

// GetORPPosition returns the Optimal Recognition Point index for a word.
// This is the character position where the eye should focus for fastest recognition.
func GetORPPosition(word string) int {
	length := len(word)
	if length <= 1 {
		return 0
	} else if length <= 5 {
		return 1
	}
	return length / 3
}

// JumpToPrevSentence moves to the start of the previous sentence.
func (r *Reader) JumpToPrevSentence() {
	for i := len(r.SentenceStarts) - 1; i >= 0; i-- {
		if r.SentenceStarts[i] < r.CurrentIndex {
			r.CurrentIndex = r.SentenceStarts[i]
			return
		}
	}
	r.CurrentIndex = 0
}

// JumpToNextSentence moves to the start of the next sentence.
func (r *Reader) JumpToNextSentence() {
	for i := 0; i < len(r.SentenceStarts); i++ {
		if r.SentenceStarts[i] > r.CurrentIndex {
			r.CurrentIndex = r.SentenceStarts[i]
			return
		}
	}
	if len(r.Words) > 0 {
		r.CurrentIndex = len(r.Words) - 1
	}
}

// GetDelay returns the duration to display each word based on WPM.
func (r *Reader) GetDelay() time.Duration {
	return time.Duration(60.0/float64(r.WPM)*1000) * time.Millisecond
}

// CurrentWord returns the word at the current index.
func (r *Reader) CurrentWord() string {
	if r.CurrentIndex >= 0 && r.CurrentIndex < len(r.Words) {
		return r.Words[r.CurrentIndex]
	}
	return ""
}

// Progress returns the current position and total word count.
func (r *Reader) Progress() (current, total int) {
	return r.CurrentIndex + 1, len(r.Words)
}

// Advance moves to the next word. Returns true if there are more words.
func (r *Reader) Advance() bool {
	if r.CurrentIndex < len(r.Words)-1 {
		r.CurrentIndex++
		return true
	}
	return false
}

// AtEnd returns true if the reader is at the last word.
func (r *Reader) AtEnd() bool {
	return r.CurrentIndex >= len(r.Words)-1
}
