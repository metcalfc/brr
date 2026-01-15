package reader

import (
	"fmt"
	"io"
	"strings"

	"github.com/taylorskalyo/goreader/epub"
	"golang.org/x/net/html"
)

// EPUBFormat implements Format for EPUB files.
type EPUBFormat struct{}

func init() {
	Register(&EPUBFormat{})
}

func (f *EPUBFormat) Name() string       { return "EPUB" }
func (f *EPUBFormat) Extensions() []string { return []string{".epub"} }
func (f *EPUBFormat) Extract(filename string) (string, error) {
	return ExtractTextFromEPUB(filename)
}

// ExtractTextFromEPUB extracts all text content from an EPUB file.
func ExtractTextFromEPUB(filename string) (string, error) {
	rc, err := epub.OpenReader(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open epub: %w", err)
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return "", fmt.Errorf("no rootfiles found in epub")
	}

	book := rc.Rootfiles[0]
	var out strings.Builder

	for _, ref := range book.Spine.Itemrefs {
		if ref.Item == nil {
			continue
		}
		r, err := ref.Item.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			continue
		}
		out.WriteString(extractTextFromHTML(string(data)))
		out.WriteString(" ")
	}

	return out.String(), nil
}

func extractTextFromHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return ""
	}

	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			if t := strings.TrimSpace(n.Data); t != "" {
				out.WriteString(t)
				out.WriteString(" ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out.String()
}
