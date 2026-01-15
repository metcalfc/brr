package reader

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// MarkdownFormat implements Format for Markdown files.
type MarkdownFormat struct{}

func init() {
	Register(&MarkdownFormat{})
}

func (f *MarkdownFormat) Name() string         { return "Markdown" }
func (f *MarkdownFormat) Extensions() []string { return []string{".md", ".markdown"} }

func (f *MarkdownFormat) Extract(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// headerRegex matches markdown headers (# to ######)
var headerRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// TOC extracts the table of contents from a Markdown file by parsing headers.
func (f *MarkdownFormat) TOC(filename string) ([]TOCEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []TOCEntry
	var wordCount int

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if line is a header
		if match := headerRegex.FindStringSubmatch(line); match != nil {
			level := len(match[1]) - 1 // h1 = level 0, h2 = level 1, etc.
			title := strings.TrimSpace(match[2])

			// Get preview from the next non-empty, non-header line
			preview := ""
			// We'll leave preview empty for now since we can't peek ahead easily

			entries = append(entries, TOCEntry{
				Title:     title,
				Preview:   preview,
				WordIndex: wordCount,
				Level:     level,
			})
		}

		// Count words in this line
		words := strings.Fields(line)
		wordCount += len(words)
	}

	return entries, scanner.Err()
}

// ExtractChapters extracts text with chapter boundaries from headers.
func (f *MarkdownFormat) ExtractChapters(filename string) ([]Chapter, []string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	var allWords []string
	var chapters []Chapter
	var currentChapter *Chapter
	var currentWords []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if line is a header
		if match := headerRegex.FindStringSubmatch(line); match != nil {
			// Save previous chapter
			if currentChapter != nil && len(currentWords) > 0 {
				currentChapter.WordEnd = len(allWords) - 1
				chapters = append(chapters, *currentChapter)
			}

			title := strings.TrimSpace(match[2])
			currentChapter = &Chapter{
				Title:     title,
				WordStart: len(allWords),
			}
			currentWords = nil
		}

		// Add words from this line
		words := strings.Fields(line)
		allWords = append(allWords, words...)
		currentWords = append(currentWords, words...)
	}

	// Save last chapter
	if currentChapter != nil && len(currentWords) > 0 {
		currentChapter.WordEnd = len(allWords) - 1
		chapters = append(chapters, *currentChapter)
	}

	// If no chapters found, create a single chapter with all content
	if len(chapters) == 0 && len(allWords) > 0 {
		chapters = append(chapters, Chapter{
			Title:     "Document",
			WordStart: 0,
			WordEnd:   len(allWords) - 1,
		})
	}

	return chapters, allWords, scanner.Err()
}
