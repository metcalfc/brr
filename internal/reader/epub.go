package reader

import (
	"fmt"
	"io"
	"strings"

	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"
)

// ExtractTextFromEPUB opens an EPUB file and extracts all text content from it.
func ExtractTextFromEPUB(filename string) (string, error) {
	rc, err := epub.OpenReader(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open epub: %w", err)
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return "", fmt.Errorf("no rootfiles found in epub")
	}

	// Typically the first rootfile is the main content
	book := rc.Rootfiles[0]
	var fullText strings.Builder

	for _, itemref := range book.Spine.Itemrefs {
		item := itemref.Item
		if item == nil {
			continue
		}

		// Open the item (usually XHTML)
		r, err := item.Open()
		if err != nil {
			continue // Skip items we can't open
		}

		content, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			continue
		}

		text := extractTextFromHTML(string(content))
		fullText.WriteString(text)
		fullText.WriteString(" ") // Add space between chapters/sections
	}

	return fullText.String(), nil
}

// extractTextFromHTML parses HTML content and returns plain text.
func extractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return sb.String()
}
