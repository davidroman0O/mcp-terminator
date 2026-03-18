package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// checkboxPattern defines open/close brackets and valid markers.
type checkboxPattern struct {
	open           rune
	close          rune
	checkedMarkers []rune
	uncheckedMarker rune
}

// CheckboxDetector detects checkbox and radio-button patterns like
// [x], [ ], [X], [*], (*), ( ), etc.
type CheckboxDetector struct {
	patterns       []checkboxPattern
	maxLabelLength int
}

// NewCheckboxDetector creates a new checkbox detector.
func NewCheckboxDetector() *CheckboxDetector {
	return &CheckboxDetector{
		patterns: []checkboxPattern{
			{
				open:           '[',
				close:          ']',
				checkedMarkers: []rune{'x', 'X', '*', '\u2713', '\u2714'}, // x X * ✓ ✔
				uncheckedMarker: ' ',
			},
			{
				open:           '(',
				close:          ')',
				checkedMarkers: []rune{'*', 'o', 'O', '\u25cf', '\u25c9'}, // * o O ● ◉
				uncheckedMarker: ' ',
			},
		},
		maxLabelLength: 60,
	}
}

func (d *CheckboxDetector) Name() string     { return "checkbox" }
func (d *CheckboxDetector) Priority() int    { return 60 }

func (d *CheckboxDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement
	dims := grid.Dimensions()

	for row := 0; row < int(dims.Rows); row++ {
		cbs := d.detectCheckboxesInRow(grid, row)
		for _, cb := range cbs {
			if !ctx.IsRegionClaimed(cb.Bounds) {
				results = append(results, cb)
			}
		}
	}
	return results
}

func (d *CheckboxDetector) detectCheckboxesInRow(grid GridReader, row int) []DetectedElement {
	var checkboxes []DetectedElement
	rowText := extractRowText(grid, row)
	runes := []rune(rowText)

	for _, pat := range d.patterns {
		for i := 0; i < len(runes); i++ {
			if runes[i] == pat.open && i+2 < len(runes) && runes[i+2] == pat.close {
				marker := runes[i+1]
				checked := false
				for _, cm := range pat.checkedMarkers {
					if marker == cm {
						checked = true
						break
					}
				}
				isValid := checked || marker == pat.uncheckedMarker
				if !isValid {
					continue
				}

				// Extract label after the closing bracket.
				labelStart := i + 3
				label := d.extractLabel(rowText, labelStart)

				refID := fmt.Sprintf("checkbox_%d_%d", row, i)

				checkboxWidth := uint16(3)
				totalWidth := checkboxWidth
				if label != "" {
					totalWidth = checkboxWidth + 1 + uint16(len([]rune(label)))
				}

				bounds := core.NewBounds(uint16(row), uint16(i), totalWidth, 1)
				checkboxes = append(checkboxes, DetectedElement{
					Element:    core.NewCheckboxElement(refID, bounds, label, checked),
					Bounds:     bounds,
					Confidence: ConfidenceHigh,
				})
			}
		}
	}
	return checkboxes
}

// extractLabel grabs text after the checkbox, trimming and truncating.
func (d *CheckboxDetector) extractLabel(text string, startPos int) string {
	if startPos >= len(text) {
		return ""
	}
	after := text[startPos:]
	trimmed := strings.TrimLeft(after, " ")
	labelStart := startPos + (len(after) - len(trimmed))

	labelEnd := labelStart + d.maxLabelLength
	if labelEnd > len(text) {
		labelEnd = len(text)
	}

	label := strings.TrimRight(text[labelStart:labelEnd], " \t")

	// Truncate at next checkbox/radio opener.
	if idx := strings.IndexAny(label, "[({\n\r"); idx >= 0 {
		label = strings.TrimRight(label[:idx], " \t")
	}
	return label
}
