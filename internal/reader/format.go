package reader

import (
	"os"
	"path/filepath"
	"strings"
)

// Format defines a file format reader for extracting text.
type Format interface {
	Name() string
	Extensions() []string
	Extract(filename string) (string, error)
}

var registry []Format

// Register adds a format reader to the registry.
func Register(f Format) {
	registry = append(registry, f)
}

// ExtractText extracts text from a file, using a registered format or plain text fallback.
func ExtractText(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, f := range registry {
		for _, e := range f.Extensions() {
			if ext == e {
				return f.Extract(filename)
			}
		}
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SupportedFormats returns registered format names with their extensions.
func SupportedFormats() []string {
	var out []string
	for _, f := range registry {
		out = append(out, f.Name()+" ("+strings.Join(f.Extensions(), ", ")+")")
	}
	return out
}
