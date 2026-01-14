# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Brr is a terminal-based speed reading tool that uses the RSVP (Rapid Serial Visual Presentation) technique. It displays text one word at a time with the Optimal Recognition Point (ORP) highlighted in red to enable faster reading speeds (100-1500 WPM).

## Core Architecture

### Bubbletea TUI Framework
The application is built using the Elm Architecture pattern via [Charmbracelet's Bubbletea](https://github.com/charmbracelet/bubbletea):

- **Model** (`model` struct): Holds application state (words array, current index, WPM, pause state, dimensions)
- **Update** (`Update` method): Handles messages (keyboard input, ticks, window resizing)
- **View** (`View` method): Renders the current state to the terminal
- **Commands**: Async operations (primarily `tick()` for timing word display)

### Key Components

**ORP (Optimal Recognition Point) Calculation** (`getORPPosition`):
- Single char: position 0
- 2-5 chars: position 1
- 6+ chars: position at length/3
- The ORP character is highlighted in red using lipgloss styles

**Word Timing** (`getDelay`, `tick`):
- Delay calculated as `60.0/WPM * 1000` milliseconds
- `tickMsg` advances to next word on each tick
- Timing pauses when `paused` flag is true

**Layout** (`View` method):
- Status line at top showing progress and WPM
- Word vertically centered with horizontal centering
- Controls displayed at bottom
- Uses `lipgloss` for styling and layout calculations

## Development Commands

### Build and Run
```bash
go build -o brr
./brr sample.txt
./brr -w 500 sample.txt
```

### Testing
```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestParseText

# Run benchmarks
go test -bench=.

# Run specific benchmark
go test -bench=BenchmarkFormatWord
```

### View Documentation
```bash
man ./brr.1
```

## Testing Strategy

Tests are organized by component:
- **Text parsing**: `TestParseText` - validates word splitting logic
- **ORP calculation**: `TestGetORPPosition` - ensures correct highlight position for different word lengths
- **Model lifecycle**: `TestNewModel`, `TestModelUpdate` - verifies state management and message handling
- **UI rendering**: `TestModelView`, `TestFormatWord` - checks display output
- **Performance**: Benchmark suite for parsing, ORP calculation, formatting, and rendering

## Input Handling

The application accepts text from:
1. **File argument**: `brr file.txt`
2. **Stdin**: `cat file.txt | brr` or `echo "text" | brr`

Main checks for stdin vs terminal input using `os.Stdin.Stat()` and `os.ModeCharDevice`.

## Interactive Controls

Keyboard handling in `Update` method:
- **SPACE**: Toggle pause/play
- **+/=**: Increase WPM by 50 (capped at 1500)
- **-**: Decrease WPM by 50 (floored at 100)
- **Q/Ctrl+C**: Quit application

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling and layout
- `golang.org/x/term` - Terminal control (used by bubbletea)

## Known UI Issue

There is a reported issue where "the red letter moves" when it should be anchored. The ORP character should remain at a fixed position on screen, but text may be shifting horizontally. Check `centerText` function and the horizontal positioning logic in the `View` method.
