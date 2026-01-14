# Brr - Terminal Speed Reading Tool

A fast, lightweight CLI tool for speed reading using the RSVP (Rapid Serial Visual Presentation) technique. Displays text one word at a time with the optimal recognition point (ORP) highlighted in red.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)

## Features

- 🚀 Adjustable reading speed (100-1500 WPM)
- 🎯 Optimal Recognition Point highlighting
- ⏯️  Pause/resume controls
- 📊 Real-time progress tracking
- 📄 Read from files or stdin
- ⚡ Lightweight and fast
- 🎨 Clean terminal UI with ANSI colors

## Installation

### Homebrew (macOS/Linux)

```bash
brew install metcalfc/tap/brr
```

### From Source

```bash
go install github.com/metcalfc/brr@latest
```

### Manual Build

```bash
git clone https://github.com/metcalfc/brr.git
cd brr
go build -o brr
sudo mv brr /usr/local/bin/
sudo cp brr.1 /usr/local/share/man/man1/
```

## Usage

### Basic Usage

```bash
# Read a file
brr article.txt

# Read from stdin
cat book.txt | brr
echo "Speed reading is awesome" | brr

# Specify reading speed (words per minute)
brr -w 500 article.txt
```

### Interactive Controls

While reading:

- **SPACE** - Pause/play
- **+ or =** - Increase speed by 50 WPM
- **-** - Decrease speed by 50 WPM
- **Q** - Quit

## Examples

Start at 300 WPM (default):
```bash
brr sample.txt
```

Start at 900 WPM for experienced readers:
```bash
brr -w 900 sample.txt
```

Read from a pipeline:
```bash
curl -s https://example.com/article.txt | brr -w 400
```

## How It Works

The RSVP (Rapid Serial Visual Presentation) technique works by:

1. **Displaying words one at a time** at a fixed position
2. **Highlighting the Optimal Recognition Point** (typically 1/3 into the word)
3. **Eliminating eye movement** across lines and pages
4. **Allowing focus** purely on comprehension rather than tracking

This technique can significantly increase reading speed while maintaining comprehension. Most users can comfortably read at 400-600 WPM with practice, and experienced speed readers can exceed 900 WPM.

## Tips for Speed Reading

- Start at 300 WPM and gradually increase as you become comfortable
- Keep your eyes focused on the center where words appear
- Don't try to "read ahead" or look around the screen
- Practice regularly to build speed and maintain comprehension
- Take breaks during long reading sessions to avoid eye strain

## Development

### Running Tests

```bash
go test -v
```

### Running Benchmarks

```bash
go test -bench=.
```

### Viewing the Manpage

```bash
man ./brr.1
```

## Technical Details

- **Language**: Go 1.21+
- **Dependencies**: `golang.org/x/term` for terminal control
- **Platforms**: Linux, macOS, BSD (any UNIX with terminal support)
- **Terminal Requirements**: ANSI color support, raw mode capability

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

## Author

Chad Metcalf

## Inspiration

This project was inspired by [this video demonstration](https://www.youtube.com/watch?v=NdKcDPBQ-Lw) of RSVP speed reading, which shows words presented one at a time with the optimal recognition point highlighted, starting at 300 WPM and scaling up to 900 WPM.
