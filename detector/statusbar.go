package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// StatusBarDetector detects status/help bars, typically on the last row
// of the terminal. It matches common patterns (Press, ESC, Ctrl+, F1-F10)
// and distinct background colors.
type StatusBarDetector struct{}

// NewStatusBarDetector creates a new status bar detector.
func NewStatusBarDetector() *StatusBarDetector {
	return &StatusBarDetector{}
}

func (d *StatusBarDetector) Name() string     { return "status_bar" }
func (d *StatusBarDetector) Priority() int    { return 50 }

func (d *StatusBarDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement
	dims := grid.Dimensions()

	lastRow := int(dims.Rows) - 1
	if lastRow < 0 {
		return results
	}

	bounds := core.NewBounds(uint16(lastRow), 0, dims.Cols, 1)
	if ctx.IsRegionClaimed(bounds) {
		return results
	}

	text := extractRowText(grid, lastRow)

	if d.looksLikeStatusBar(text, grid, lastRow) {
		refID := fmt.Sprintf("status_bar_%d", lastRow)
		content := strings.TrimSpace(text)
		results = append(results, DetectedElement{
			Element:    core.NewStatusBarElement(refID, bounds, content),
			Bounds:     bounds,
			Confidence: ConfidenceMedium,
		})
	}
	return results
}

func (d *StatusBarDetector) looksLikeStatusBar(text string, grid GridReader, row int) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	patterns := []string{
		"Press", "press", "ESC", "Esc",
		"q to quit", "Q to quit",
		"Help:", "Status:",
		"\u2502", "|",
		"Ctrl+", "Alt+",
		"F1", "F2", "F3", "F4", "F5",
		"F6", "F7", "F8", "F9", "F10",
	}

	for _, p := range patterns {
		if strings.Contains(trimmed, p) {
			return true
		}
	}

	return d.rowHasDistinctBackground(grid, row)
}

func (d *StatusBarDetector) rowHasDistinctBackground(grid GridReader, row int) bool {
	dims := grid.Dimensions()
	nonDefaultCount := 0
	totalNonSpace := 0

	for col := 0; col < int(dims.Cols); col++ {
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' {
				totalNonSpace++
				if cell.Bg.Type != core.ColorDefault {
					nonDefaultCount++
				}
			}
		}
	}

	return totalNonSpace > 0 && nonDefaultCount > totalNonSpace/2
}
