// Package detector provides UI element detection from a terminal cell grid.
//
// It implements 8 pattern-matching detectors that scan a grid of cells and
// produce a list of detected elements (buttons, menus, tables, etc.) with
// confidence scoring and bounding-box region claiming.
//
// This is a faithful port of the Rust terminal-mcp-detector crate.
package detector

import (
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// Confidence represents the detector's certainty about a detection.
type Confidence int

const (
	// ConfidenceLow means less than 60% certain.
	ConfidenceLow Confidence = iota
	// ConfidenceMedium means 60-90% certain.
	ConfidenceMedium
	// ConfidenceHigh means greater than 90% certain.
	ConfidenceHigh
)

// String returns a human-readable representation.
func (c Confidence) String() string {
	switch c {
	case ConfidenceLow:
		return "Low"
	case ConfidenceMedium:
		return "Medium"
	case ConfidenceHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// DetectedElement is a raw detection result before assembly into a TST.
type DetectedElement struct {
	Element    core.Element
	Bounds     core.Bounds
	Confidence Confidence
}

// DetectionContext carries state shared across detectors within a single
// detection pass: claimed regions, cursor info, and the ref-ID generator.
type DetectionContext struct {
	ClaimedRegions []core.Bounds
	Cursor         core.Position
	CursorVisible  bool
	RefCounter     *core.RefIDGenerator
}

// NewDetectionContext creates a fresh context with the given cursor position.
func NewDetectionContext(cursor core.Position) *DetectionContext {
	return &DetectionContext{
		Cursor:     cursor,
		RefCounter: core.NewRefIDGenerator(),
	}
}

// IsRegionClaimed reports whether bounds overlaps any already-claimed region.
func (ctx *DetectionContext) IsRegionClaimed(bounds core.Bounds) bool {
	for _, claimed := range ctx.ClaimedRegions {
		if bounds.Intersects(claimed) {
			return true
		}
	}
	return false
}

// ClaimRegion marks a region as claimed so later detectors skip it.
func (ctx *DetectionContext) ClaimRegion(bounds core.Bounds) {
	ctx.ClaimedRegions = append(ctx.ClaimedRegions, bounds)
}

// Detector is the interface that each element detector implements.
type Detector interface {
	// Name returns the detector's name for debugging/logging.
	Name() string
	// Priority returns the detection priority. Higher values run first.
	Priority() int
	// Detect scans the grid and returns any elements found.
	Detect(grid GridReader, ctx *DetectionContext) []DetectedElement
}

// GridReader abstracts access to the terminal cell grid, allowing the
// detector package to read cells without a circular dependency on the
// emulator package.
type GridReader interface {
	// Cell returns the cell at (row, col). Returns a zero-value Cell
	// and false when out of bounds.
	Cell(row, col int) (core.Cell, bool)
	// Dimensions returns the grid size.
	Dimensions() core.Dimensions
	// CursorPosition returns the current cursor location.
	CursorPosition() core.Position
	// CursorVisible reports whether the cursor is visible.
	CursorVisible() bool
	// ExtractText extracts all text within the given bounds as a string.
	ExtractText(bounds core.Bounds) string
}

// --- helpers used by multiple detectors ---

// extractRowText reads an entire row from the grid into a string.
func extractRowText(grid GridReader, row int) string {
	dims := grid.Dimensions()
	var b strings.Builder
	b.Grow(int(dims.Cols))
	for col := 0; col < int(dims.Cols); col++ {
		if cell, ok := grid.Cell(row, col); ok {
			b.WriteRune(cell.Character)
		}
	}
	return b.String()
}

// extractRowTextRegion reads a portion of a row.
func extractRowTextRegion(grid GridReader, row, startCol, width int) string {
	var b strings.Builder
	b.Grow(width)
	for colOff := 0; colOff < width; colOff++ {
		col := startCol + colOff
		if cell, ok := grid.Cell(row, col); ok {
			b.WriteRune(cell.Character)
		}
	}
	return b.String()
}
