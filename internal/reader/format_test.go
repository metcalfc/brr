package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractText(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "brr-test")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("plain text", func(t *testing.T) {
		content := "Hello world this is a test."
		path := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(path, []byte(content), 0644)

		got, err := ExtractText(path)
		if err != nil {
			t.Fatalf("ExtractText: %v", err)
		}
		if got != content {
			t.Errorf("got %q, want %q", got, content)
		}
	})

	t.Run("unknown extension", func(t *testing.T) {
		content := "Some markdown content"
		path := filepath.Join(tmpDir, "test.md")
		os.WriteFile(path, []byte(content), 0644)

		got, err := ExtractText(path)
		if err != nil {
			t.Fatalf("ExtractText: %v", err)
		}
		if got != content {
			t.Errorf("got %q, want %q", got, content)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ExtractText(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestEPUBFormat(t *testing.T) {
	f := &EPUBFormat{}
	if f.Name() != "EPUB" {
		t.Errorf("Name() = %q, want EPUB", f.Name())
	}
	if exts := f.Extensions(); len(exts) != 1 || exts[0] != ".epub" {
		t.Errorf("Extensions() = %v, want [.epub]", exts)
	}
}

func TestSupportedFormats(t *testing.T) {
	formats := SupportedFormats()
	if len(formats) == 0 {
		t.Error("no formats registered")
	}
	for _, f := range formats {
		if f == "EPUB (.epub)" {
			return
		}
	}
	t.Errorf("EPUB not registered: %v", formats)
}
