package reader

import (
	"os"
	"testing"
)

func TestEPUBTOC(t *testing.T) {
	// Skip if SherlockHolmes.epub doesn't exist
	epubPath := "../../SherlockHolmes.epub"
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("SherlockHolmes.epub not found, skipping test")
	}

	f := &EPUBFormat{}
	toc, err := f.TOC(epubPath)
	if err != nil {
		t.Fatalf("TOC extraction failed: %v", err)
	}

	if len(toc) == 0 {
		t.Error("Expected non-empty TOC")
	}

	// Print TOC for manual verification
	t.Logf("Found %d TOC entries:", len(toc))
	for i, entry := range toc {
		indent := ""
		for j := 0; j < entry.Level; j++ {
			indent += "  "
		}
		t.Logf("%d. %s%s (word %d)", i+1, indent, entry.Title, entry.WordIndex)
	}
}

func TestEPUBExtractChapters(t *testing.T) {
	epubPath := "../../SherlockHolmes.epub"
	if _, err := os.Stat(epubPath); os.IsNotExist(err) {
		t.Skip("SherlockHolmes.epub not found, skipping test")
	}

	f := &EPUBFormat{}
	chapters, words, err := f.ExtractChapters(epubPath)
	if err != nil {
		t.Fatalf("ExtractChapters failed: %v", err)
	}

	if len(chapters) == 0 {
		t.Error("Expected non-empty chapters")
	}

	if len(words) == 0 {
		t.Error("Expected non-empty words")
	}

	t.Logf("Found %d chapters, %d total words", len(chapters), len(words))
	for i, ch := range chapters {
		wordCount := ch.WordEnd - ch.WordStart + 1
		t.Logf("%d. %s (words %d-%d, %d words)", i+1, ch.Title, ch.WordStart, ch.WordEnd, wordCount)
	}
}
