package reader

import (
	"testing"
)

func TestExtractTextFromHTML(t *testing.T) {
	htmlContent := `
	<html>
		<head><title>Test</title></head>
		<body>
			<h1>Chapter 1</h1>
			<p>This is the <b>first</b> paragraph.</p>
			<p>
				This is the second paragraph
				with a newline.
			</p>
			<div>Some <span>nested</span> text.</div>
		</body>
	</html>
	`

	expectedWords := []string{"Test", "Chapter", "1", "This", "is", "the", "first", "paragraph.", "This", "is", "the", "second", "paragraph", "with", "a", "newline.", "Some", "nested", "text."}

	text := extractTextFromHTML(htmlContent)
	words := ParseText(text) // Use the existing ParseText to split by whitespace

	if len(words) != len(expectedWords) {
		t.Errorf("Expected %d words, got %d", len(expectedWords), len(words))
	}

	for i, word := range words {
		if i < len(expectedWords) && word != expectedWords[i] {
			t.Errorf("Word %d: expected %q, got %q", i, expectedWords[i], word)
		}
	}
}
