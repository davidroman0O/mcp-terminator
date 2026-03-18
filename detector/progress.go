package detector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// ProgressDetector detects visual progress bars (block characters like
// ████░░░░, bracket bars like [====    ]), and percentage text (75%).
type ProgressDetector struct {
	minLength  int
	filledChars map[rune]bool
	emptyChars  map[rune]bool
	// blockChars are the subset of filled/empty that prove it's a real bar.
	blockChars map[rune]bool
}

// NewProgressDetector creates a new progress detector.
func NewProgressDetector() *ProgressDetector {
	return &ProgressDetector{
		minLength: 5,
		filledChars: map[rune]bool{
			'\u2588': true, '\u2593': true, '\u2592': true, '#': true, '=': true, '*': true,
		},
		emptyChars: map[rune]bool{
			'\u2591': true, '\u00b7': true, ' ': true, '-': true, '.': true, '\u2581': true,
		},
		blockChars: map[rune]bool{
			'\u2588': true, '\u2593': true, '\u2592': true, '#': true, // filled
			'\u2591': true, '\u2581': true, // empty
		},
	}
}

func (d *ProgressDetector) Name() string     { return "progress" }
func (d *ProgressDetector) Priority() int    { return 60 }

func (d *ProgressDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement
	dims := grid.Dimensions()

	for row := 0; row < int(dims.Rows); row++ {
		rowText := extractRowText(grid, row)
		rowHasProgress := false

		// Block progress (highest priority).
		if elem := d.detectBlockProgress(rowText, row); elem != nil {
			if !ctx.IsRegionClaimed(elem.Bounds) {
				results = append(results, *elem)
				rowHasProgress = true
			}
		}

		// Bracket progress.
		if !rowHasProgress {
			for _, elem := range d.detectBracketProgress(rowText, row) {
				if !ctx.IsRegionClaimed(elem.Bounds) {
					results = append(results, elem)
					rowHasProgress = true
				}
			}
		}

		// Percentage text (lowest priority).
		if !rowHasProgress {
			for _, elem := range d.detectPercentageText(rowText, row) {
				if !ctx.IsRegionClaimed(elem.Bounds) {
					results = append(results, elem)
				}
			}
		}
	}
	return results
}

// detectBlockProgress finds sequences of filled/empty block characters.
func (d *ProgressDetector) detectBlockProgress(text string, row int) *DetectedElement {
	runes := []rune(strings.TrimRight(text, " \t"))

	startIdx := -1
	filledCount := 0
	emptyCount := 0
	hasBlockChars := false

	for i, ch := range runes {
		isFilled := d.filledChars[ch]
		isEmpty := d.emptyChars[ch]

		if isFilled || isEmpty {
			if startIdx < 0 {
				startIdx = i
			}
			if isFilled {
				filledCount++
				if d.blockChars[ch] {
					hasBlockChars = true
				}
			} else {
				emptyCount++
				if d.blockChars[ch] {
					hasBlockChars = true
				}
			}
		} else if startIdx >= 0 {
			total := filledCount + emptyCount
			if total >= d.minLength && hasBlockChars {
				pct := uint8(float64(filledCount) / float64(total) * 100.0)
				return d.makeBlockElement(row, startIdx, total, pct)
			}
			startIdx = -1
			filledCount = 0
			emptyCount = 0
			hasBlockChars = false
		}
	}

	// Bar extends to end of line.
	if startIdx >= 0 {
		total := filledCount + emptyCount
		if total >= d.minLength && hasBlockChars {
			pct := uint8(float64(filledCount) / float64(total) * 100.0)
			return d.makeBlockElement(row, startIdx, total, pct)
		}
	}
	return nil
}

func (d *ProgressDetector) makeBlockElement(row, startIdx, total int, pct uint8) *DetectedElement {
	refID := fmt.Sprintf("progress_%d_%d", row, startIdx)
	bounds := core.NewBounds(uint16(row), uint16(startIdx), uint16(total), 1)
	return &DetectedElement{
		Element:    core.NewProgressBarElement(refID, bounds, pct),
		Bounds:     bounds,
		Confidence: ConfidenceHigh,
	}
}

// detectBracketProgress finds [====    ] style bars.
func (d *ProgressDetector) detectBracketProgress(text string, row int) []DetectedElement {
	var results []DetectedElement
	runes := []rune(text)

	for i, ch := range runes {
		if ch != '[' {
			continue
		}
		// Find matching close bracket.
		closeIdx := -1
		for j := i + 1; j < len(runes); j++ {
			if runes[j] == ']' {
				closeIdx = j
				break
			}
		}
		if closeIdx < 0 {
			continue
		}

		inner := runes[i+1 : closeIdx]
		if len(inner) < d.minLength {
			continue
		}

		filled := 0
		empty := 0
		for _, c := range inner {
			switch c {
			case '=', '#', '*':
				filled++
			case ' ', '-', '.':
				empty++
			}
		}

		total := filled + empty
		if total >= d.minLength && total == len(inner) {
			pct := uint8(float64(filled) / float64(total) * 100.0)
			width := closeIdx - i + 1
			refID := fmt.Sprintf("progress_%d_%d", row, i)
			bounds := core.NewBounds(uint16(row), uint16(i), uint16(width), 1)
			results = append(results, DetectedElement{
				Element:    core.NewProgressBarElement(refID, bounds, pct),
				Bounds:     bounds,
				Confidence: ConfidenceHigh,
			})
		}
	}
	return results
}

// detectPercentageText finds "N%" patterns and creates progress elements.
func (d *ProgressDetector) detectPercentageText(text string, row int) []DetectedElement {
	var results []DetectedElement
	searchStart := 0

	for searchStart < len(text) {
		remaining := text[searchStart:]
		pctPos := strings.Index(remaining, "%")
		if pctPos < 0 {
			break
		}
		absPos := searchStart + pctPos

		before := text[:absPos]
		if len(before) == 0 {
			searchStart = absPos + 1
			continue
		}

		// Find the start of the number.
		numStart := -1
		for j := len(before) - 1; j >= 0; j-- {
			ch := before[j]
			if (ch >= '0' && ch <= '9') || ch == '.' {
				numStart = j
			} else {
				break
			}
		}

		if numStart >= 0 {
			numStr := before[numStart:]
			val, err := strconv.ParseFloat(numStr, 64)
			if err == nil {
				pct := uint8(val)
				if val > 100 {
					pct = 100
				}
				width := absPos - numStart + 1
				refID := fmt.Sprintf("progress_%d_%d", row, numStart)
				bounds := core.NewBounds(uint16(row), uint16(numStart), uint16(width), 1)
				results = append(results, DetectedElement{
					Element:    core.NewProgressBarElement(refID, bounds, pct),
					Bounds:     bounds,
					Confidence: ConfidenceMedium,
				})
			}
		}

		searchStart = absPos + 1
	}
	return results
}
