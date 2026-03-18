package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// InputDetector detects text input fields by looking for labeled fields
// (e.g. "Username: john"), bracketed inputs, and reverse-video regions
// near the cursor position.
type InputDetector struct {
	minWidth int
}

// NewInputDetector creates a new input detector.
func NewInputDetector() *InputDetector {
	return &InputDetector{minWidth: 3}
}

func (d *InputDetector) Name() string     { return "input" }
func (d *InputDetector) Priority() int    { return 70 }

func (d *InputDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement
	cursorRow := int(ctx.Cursor.Row)
	cursorCol := int(ctx.Cursor.Col)

	// Strategy 1: Detect input at cursor position.
	if elem := d.detectInputAtCursor(grid, cursorRow, cursorCol); elem != nil {
		if !ctx.IsRegionClaimed(elem.Bounds) {
			results = append(results, *elem)
			return results
		}
	}

	// Strategy 2: Reverse video detection (for focused inputs).
	if elem := d.detectReverseVideoInput(grid, cursorRow); elem != nil {
		if !ctx.IsRegionClaimed(elem.Bounds) {
			results = append(results, *elem)
		}
	}

	return results
}

// detectInputAtCursor looks for a labeled or bracketed input on the cursor row.
func (d *InputDetector) detectInputAtCursor(grid GridReader, cursorRow, cursorCol int) *DetectedElement {
	dims := grid.Dimensions()
	rowText := extractRowTextRegion(grid, cursorRow, 0, int(dims.Cols))

	// Try labeled input: "Label: value"
	if valueStart, valueEnd, ok := d.looksLikeLabeledInput(rowText); ok {
		if valueEnd > len(rowText) {
			valueEnd = len(rowText)
		}
		value := strings.TrimRight(rowText[valueStart:valueEnd], " \t")
		cursorPos := 0
		if cursorCol >= valueStart {
			cursorPos = cursorCol - valueStart
			if cursorPos > len([]rune(value)) {
				cursorPos = len([]rune(value))
			}
		}

		refID := fmt.Sprintf("input_%d_%d", cursorRow, valueStart)
		w := valueEnd - valueStart
		bounds := core.NewBounds(uint16(cursorRow), uint16(valueStart), uint16(w), 1)

		return &DetectedElement{
			Element:    core.NewInputElement(refID, bounds, value, cursorPos),
			Bounds:     bounds,
			Confidence: ConfidenceHigh,
		}
	}

	// Try bracketed input: "[  hello  ]"
	if valueStart, valueEnd, ok := d.looksLikeBracketedInput(rowText); ok {
		value := strings.TrimSpace(rowText[valueStart:valueEnd])
		cursorPos := 0
		if cursorCol >= valueStart && cursorCol < valueEnd {
			cursorPos = cursorCol - valueStart
			if cursorPos > len([]rune(value)) {
				cursorPos = len([]rune(value))
			}
		}

		refID := fmt.Sprintf("input_%d_%d", cursorRow, valueStart)
		w := valueEnd - valueStart
		bounds := core.NewBounds(uint16(cursorRow), uint16(valueStart), uint16(w), 1)

		return &DetectedElement{
			Element:    core.NewInputElement(refID, bounds, value, cursorPos),
			Bounds:     bounds,
			Confidence: ConfidenceMedium,
		}
	}

	return nil
}

// looksLikeLabeledInput checks for "label: value" pattern.
func (d *InputDetector) looksLikeLabeledInput(text string) (valueStart, valueEnd int, ok bool) {
	colonPos := strings.Index(text, ":")
	if colonPos < 0 {
		return 0, 0, false
	}

	afterColon := text[colonPos+1:]
	trimmed := strings.TrimLeft(afterColon, " ")
	if len(trimmed) == 0 && len(afterColon) <= 1 {
		return 0, 0, false
	}

	vs := colonPos + 1 + (len(afterColon) - len(trimmed))
	return vs, len(text), true
}

// looksLikeBracketedInput checks for "[   text   ]" pattern.
func (d *InputDetector) looksLikeBracketedInput(text string) (valueStart, valueEnd int, ok bool) {
	trimmed := strings.TrimSpace(text)
	type pair struct{ open, close byte }
	brackets := []pair{
		{'[', ']'}, {'(', ')'}, {'{', '}'}, {'\xe2', '\xe2'}, // │ handled below
	}

	// Explicit check for │...│
	if len(trimmed) > 2 {
		runes := []rune(trimmed)
		if runes[0] == '\u2502' && runes[len(runes)-1] == '\u2502' {
			start := strings.IndexRune(text, '\u2502')
			end := strings.LastIndex(text, "\u2502")
			if end > start+1 && (end-start-1) >= d.minWidth {
				// byte offsets: start is the position of first │
				return start + len("\u2502"), end, true
			}
		}
	}

	for _, br := range brackets[:3] {
		if len(trimmed) > 2 && trimmed[0] == br.open && trimmed[len(trimmed)-1] == br.close {
			start := strings.IndexByte(text, br.open)
			end := strings.LastIndexByte(text, br.close)
			if end > start+1 && (end-start-1) >= d.minWidth {
				return start + 1, end, true
			}
		}
	}

	return 0, 0, false
}

// detectReverseVideoInput finds a consecutive run of reverse-video cells.
func (d *InputDetector) detectReverseVideoInput(grid GridReader, cursorRow int) *DetectedElement {
	dims := grid.Dimensions()
	startCol := -1
	endCol := 0
	var value strings.Builder

	for col := 0; col < int(dims.Cols); col++ {
		if cell, ok := grid.Cell(cursorRow, col); ok {
			if cell.Attrs.Reverse && cell.Character != ' ' {
				if startCol < 0 {
					startCol = col
				}
				endCol = col + 1
				value.WriteRune(cell.Character)
			}
		}
	}

	if startCol >= 0 {
		w := endCol - startCol
		if w >= d.minWidth {
			refID := fmt.Sprintf("input_%d_%d", cursorRow, startCol)
			cursorPos := len([]rune(strings.TrimSpace(value.String())))
			bounds := core.NewBounds(uint16(cursorRow), uint16(startCol), uint16(w), 1)

			return &DetectedElement{
				Element:    core.NewInputElement(refID, bounds, strings.TrimSpace(value.String()), cursorPos),
				Bounds:     bounds,
				Confidence: ConfidenceHigh,
			}
		}
	}
	return nil
}
