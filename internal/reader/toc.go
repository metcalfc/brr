package reader

// TOCEntry represents a single entry in a table of contents
type TOCEntry struct {
	Title     string // Display title
	Preview   string // First line preview (for UI)
	WordIndex int    // Starting word index in Reader.Words
	Level     int    // Nesting level (0 = top level)
}

// Chapter represents extracted chapter content with boundaries
type Chapter struct {
	Title     string
	WordStart int // Starting index in combined word array
	WordEnd   int // Ending index in combined word array
}

// TOCProvider is an optional interface for formats that support TOC extraction
type TOCProvider interface {
	// TOC extracts the table of contents from the given file
	TOC(filename string) ([]TOCEntry, error)
}

// ChapterExtractor is an optional interface for chapter-aware extraction
type ChapterExtractor interface {
	// ExtractChapters extracts text with chapter boundaries preserved
	ExtractChapters(filename string) ([]Chapter, []string, error)
}
