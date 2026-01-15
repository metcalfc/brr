package reader

// TOCEntry represents a single entry in a table of contents
type TOCEntry struct {
	Title     string
	Preview   string
	WordIndex int
	Level     int
}

// Chapter represents extracted chapter content with boundaries
type Chapter struct {
	Title     string
	WordStart int
	WordEnd   int
}

// TOCProvider is an optional interface for formats that support TOC extraction
type TOCProvider interface {
	TOC(filename string) ([]TOCEntry, error)
}

// ChapterExtractor is an optional interface for chapter-aware extraction
type ChapterExtractor interface {
	ExtractChapters(filename string) ([]Chapter, []string, error)
}
