package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownTOC(t *testing.T) {
	// Create a temp markdown file
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")

	content := `# Introduction
This is the introduction.

## Getting Started
Here's how to get started with the project.

### Prerequisites
You'll need these things installed.

## Usage
Here's how to use it.

# Advanced Topics
More complex stuff here.

## Configuration
Configure everything.
`
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	f := &MarkdownFormat{}
	toc, err := f.TOC(mdFile)
	if err != nil {
		t.Fatalf("TOC extraction failed: %v", err)
	}

	if len(toc) != 6 {
		t.Errorf("Expected 6 TOC entries, got %d", len(toc))
	}

	// Check levels
	expectedLevels := []int{0, 1, 2, 1, 0, 1} // h1=0, h2=1, h3=2
	for i, entry := range toc {
		if entry.Level != expectedLevels[i] {
			t.Errorf("Entry %d (%s): expected level %d, got %d", i, entry.Title, expectedLevels[i], entry.Level)
		}
	}

	// Check titles
	expectedTitles := []string{"Introduction", "Getting Started", "Prerequisites", "Usage", "Advanced Topics", "Configuration"}
	for i, entry := range toc {
		if entry.Title != expectedTitles[i] {
			t.Errorf("Entry %d: expected title %q, got %q", i, expectedTitles[i], entry.Title)
		}
	}

	// Word indices should be monotonically increasing
	lastIdx := -1
	for i, entry := range toc {
		if entry.WordIndex < lastIdx {
			t.Errorf("Entry %d: word index %d is less than previous %d", i, entry.WordIndex, lastIdx)
		}
		lastIdx = entry.WordIndex
	}
}

func TestMarkdownExtractChapters(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "test.md")

	content := `# Chapter 1
First chapter content with some words.

# Chapter 2
Second chapter has more content here.

# Chapter 3
Third and final chapter.
`
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	f := &MarkdownFormat{}
	chapters, words, err := f.ExtractChapters(mdFile)
	if err != nil {
		t.Fatalf("ExtractChapters failed: %v", err)
	}

	if len(chapters) != 3 {
		t.Errorf("Expected 3 chapters, got %d", len(chapters))
	}

	if len(words) == 0 {
		t.Error("Expected non-empty words")
	}

	// Check chapter titles
	expectedTitles := []string{"Chapter 1", "Chapter 2", "Chapter 3"}
	for i, ch := range chapters {
		if ch.Title != expectedTitles[i] {
			t.Errorf("Chapter %d: expected title %q, got %q", i, expectedTitles[i], ch.Title)
		}
	}

	// Word boundaries should be continuous
	for i := 1; i < len(chapters); i++ {
		if chapters[i].WordStart != chapters[i-1].WordEnd+1 {
			t.Errorf("Gap between chapter %d and %d", i-1, i)
		}
	}
}

func TestMarkdownNoHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	mdFile := filepath.Join(tmpDir, "plain.md")

	content := `This is just plain text.
No headers at all.
Just paragraphs.
`
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	f := &MarkdownFormat{}
	toc, err := f.TOC(mdFile)
	if err != nil {
		t.Fatalf("TOC extraction failed: %v", err)
	}

	if len(toc) != 0 {
		t.Errorf("Expected empty TOC for file without headers, got %d entries", len(toc))
	}

	// ExtractChapters should still work
	chapters, words, err := f.ExtractChapters(mdFile)
	if err != nil {
		t.Fatalf("ExtractChapters failed: %v", err)
	}

	// Should have created a default chapter
	if len(chapters) != 1 {
		t.Errorf("Expected 1 default chapter, got %d", len(chapters))
	}

	if chapters[0].Title != "Document" {
		t.Errorf("Expected default title 'Document', got %q", chapters[0].Title)
	}

	if len(words) == 0 {
		t.Error("Expected non-empty words")
	}
}
