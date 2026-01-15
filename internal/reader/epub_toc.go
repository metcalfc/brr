package reader

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/taylorskalyo/goreader/epub"
)

// NCX XML structures for parsing toc.ncx
type ncx struct {
	NavMap navMap `xml:"navMap"`
}

type navMap struct {
	NavPoints []navPoint `xml:"navPoint"`
}

type navPoint struct {
	ID        string     `xml:"id,attr"`
	PlayOrder int        `xml:"playOrder,attr"`
	Label     navLabel   `xml:"navLabel"`
	Content   navContent `xml:"content"`
	Children  []navPoint `xml:"navPoint"`
}

type navLabel struct {
	Text string `xml:"text"`
}

type navContent struct {
	Src string `xml:"src,attr"`
}

// TOC extracts the table of contents from an EPUB file.
func (f *EPUBFormat) TOC(filename string) ([]TOCEntry, error) {
	rc, err := epub.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open epub: %w", err)
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return nil, fmt.Errorf("no rootfiles found in epub")
	}

	book := rc.Rootfiles[0]

	ncxData, err := findAndReadNCX(filename, book)
	if err != nil {
		return nil, err
	}

	var toc ncx
	if err := xml.Unmarshal(ncxData, &toc); err != nil {
		return nil, fmt.Errorf("failed to parse NCX: %w", err)
	}

	spineMap := buildAccurateSpineMap(filename, book)
	entries := flattenNavPoints(toc.NavMap.NavPoints, spineMap, 0)

	return entries, nil
}

// ExtractChapters extracts text with chapter boundaries preserved.
func (f *EPUBFormat) ExtractChapters(filename string) ([]Chapter, []string, error) {
	rc, err := epub.OpenReader(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open epub: %w", err)
	}
	defer rc.Close()

	if len(rc.Rootfiles) == 0 {
		return nil, nil, fmt.Errorf("no rootfiles found in epub")
	}

	book := rc.Rootfiles[0]

	tocByHref := buildTOCHrefMap(filename, book)

	var allWords []string
	var chapters []Chapter

	for i, ref := range book.Spine.Itemrefs {
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

		text := extractTextFromHTML(string(data))
		words := strings.Fields(text)

		if len(words) == 0 {
			continue
		}

		wordStart := len(allWords)
		allWords = append(allWords, words...)
		wordEnd := len(allWords) - 1

		title := fmt.Sprintf("Section %d", i+1)
		if ref.Item.HREF != "" {
			if t, ok := tocByHref[ref.Item.HREF]; ok {
				title = t
			} else if t, ok := tocByHref[path.Base(ref.Item.HREF)]; ok {
				title = t
			}
		}

		chapters = append(chapters, Chapter{
			Title:     title,
			WordStart: wordStart,
			WordEnd:   wordEnd,
		})
	}

	return chapters, allWords, nil
}

// buildTOCHrefMap parses the NCX and returns a map of href to title
func buildTOCHrefMap(filename string, book *epub.Rootfile) map[string]string {
	result := make(map[string]string)

	ncxData, err := findAndReadNCX(filename, book)
	if err != nil {
		return result
	}

	var toc ncx
	if err := xml.Unmarshal(ncxData, &toc); err != nil {
		return result
	}

	var extract func(points []navPoint)
	extract = func(points []navPoint) {
		for _, np := range points {
			href := np.Content.Src
			title := strings.TrimSpace(np.Label.Text)

			if _, exists := result[href]; !exists {
				result[href] = title
			}
			if idx := strings.Index(href, "#"); idx != -1 {
				baseHref := href[:idx]
				if _, exists := result[baseHref]; !exists {
					result[baseHref] = title
				}
			}
			baseHref := path.Base(href)
			if idx := strings.Index(baseHref, "#"); idx != -1 {
				baseHref = baseHref[:idx]
			}
			if _, exists := result[baseHref]; !exists {
				result[baseHref] = title
			}

			extract(np.Children)
		}
	}
	extract(toc.NavMap.NavPoints)

	return result
}

func findAndReadNCX(filename string, book *epub.Rootfile) ([]byte, error) {
	zr, err := zip.OpenReader(filename)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	var ncxPath string
	for _, item := range book.Manifest.Items {
		if item.MediaType == "application/x-dtbncx+xml" {
			ncxPath = item.HREF
			break
		}
	}
	if ncxPath == "" {
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".ncx") {
				ncxPath = f.Name
				break
			}
		}
	}

	if ncxPath == "" {
		return nil, fmt.Errorf("no NCX file found in EPUB")
	}

	for _, f := range zr.File {
		if f.Name == ncxPath || strings.HasSuffix(f.Name, "/"+ncxPath) || path.Base(f.Name) == path.Base(ncxPath) {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}

	return nil, fmt.Errorf("NCX file %s not found in archive", ncxPath)
}

type spineInfo struct {
	wordIndex int
	preview   string
}

func buildAccurateSpineMap(filename string, book *epub.Rootfile) map[string]spineInfo {
	m := make(map[string]spineInfo)
	wordCount := 0

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

		text := extractTextFromHTML(string(data))
		words := strings.Fields(text)

		preview := ""
		if len(words) > 0 {
			previewWords := words
			if len(previewWords) > 10 {
				previewWords = previewWords[:10]
			}
			preview = strings.Join(previewWords, " ") + "..."
		}

		if ref.Item.HREF != "" {
			m[ref.Item.HREF] = spineInfo{wordIndex: wordCount, preview: preview}
			m[path.Base(ref.Item.HREF)] = spineInfo{wordIndex: wordCount, preview: preview}
		}

		wordCount += len(words)
	}

	return m
}

func flattenNavPoints(points []navPoint, spineMap map[string]spineInfo, level int) []TOCEntry {
	var entries []TOCEntry

	for _, np := range points {
		href := np.Content.Src
		baseHref := href
		if idx := strings.Index(href, "#"); idx != -1 {
			baseHref = href[:idx]
		}

		wordIndex := 0
		preview := ""
		if info, ok := spineMap[baseHref]; ok {
			wordIndex = info.wordIndex
			preview = info.preview
		} else if info, ok := spineMap[path.Base(baseHref)]; ok {
			wordIndex = info.wordIndex
			preview = info.preview
		}

		entry := TOCEntry{
			Title:     strings.TrimSpace(np.Label.Text),
			Preview:   preview,
			WordIndex: wordIndex,
			Level:     level,
		}
		entries = append(entries, entry)
		if len(np.Children) > 0 {
			children := flattenNavPoints(np.Children, spineMap, level+1)
			entries = append(entries, children...)
		}
	}

	return entries
}
