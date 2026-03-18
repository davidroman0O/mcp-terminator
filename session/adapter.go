package session

import (
	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/davidroman0O/mcp-terminator/emulator"
)

// GridAdapter wraps an emulator.Grid to satisfy the detector.GridReader
// interface. The emulator Grid uses uint16 for cell access while GridReader
// uses int; this adapter bridges that gap.
type GridAdapter struct {
	grid *emulator.Grid
}

// NewGridAdapter creates a GridAdapter for the given grid.
func NewGridAdapter(g *emulator.Grid) *GridAdapter {
	return &GridAdapter{grid: g}
}

// Cell returns the cell at (row, col) and true, or a zero-value Cell and
// false when out of bounds.
func (a *GridAdapter) Cell(row, col int) (core.Cell, bool) {
	if row < 0 || col < 0 {
		return core.Cell{}, false
	}
	c := a.grid.Cell(uint16(row), uint16(col))
	if c == nil {
		return core.Cell{}, false
	}
	return *c, true
}

// Dimensions returns the grid dimensions.
func (a *GridAdapter) Dimensions() core.Dimensions {
	return a.grid.Dimensions()
}

// CursorPosition returns the cursor position.
func (a *GridAdapter) CursorPosition() core.Position {
	return a.grid.Cursor().Position
}

// CursorVisible reports whether the cursor is visible.
func (a *GridAdapter) CursorVisible() bool {
	return a.grid.CursorVisible()
}

// ExtractText extracts text within the given bounds.
func (a *GridAdapter) ExtractText(bounds core.Bounds) string {
	return a.grid.ExtractText(bounds)
}
