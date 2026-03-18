package detector

import (
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// testGrid implements GridReader by parsing a multi-line ASCII string
// into a 2D cell grid. Each character becomes a Cell with default
// attributes and colors.
type testGrid struct {
	cells      [][]core.Cell
	rows, cols int
	cursorPos  core.Position
	cursorVis  bool
}

// gridFromString parses a multi-line string into a testGrid.
// Lines are split on \n. The grid dimensions are determined by the
// number of lines (rows) and the longest line (cols). Shorter lines
// are padded with spaces.
func gridFromString(s string) *testGrid {
	lines := strings.Split(s, "\n")
	// Remove trailing empty line if present.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	maxCols := 0
	for _, line := range lines {
		r := []rune(line)
		if len(r) > maxCols {
			maxCols = len(r)
		}
	}

	rows := len(lines)
	cells := make([][]core.Cell, rows)
	for i, line := range lines {
		runes := []rune(line)
		row := make([]core.Cell, maxCols)
		for j := 0; j < maxCols; j++ {
			if j < len(runes) {
				row[j] = core.NewCell(runes[j])
			} else {
				row[j] = core.DefaultCell()
			}
		}
		cells[i] = row
	}

	return &testGrid{
		cells:     cells,
		rows:      rows,
		cols:      maxCols,
		cursorPos: core.Origin(),
		cursorVis: true,
	}
}

// gridFromStringWithSize creates a grid with explicit dimensions, padding
// as needed.
func gridFromStringWithSize(s string, numRows, numCols int) *testGrid {
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	cells := make([][]core.Cell, numRows)
	for i := 0; i < numRows; i++ {
		row := make([]core.Cell, numCols)
		for j := 0; j < numCols; j++ {
			row[j] = core.DefaultCell()
		}
		if i < len(lines) {
			runes := []rune(lines[i])
			for j := 0; j < numCols && j < len(runes); j++ {
				row[j] = core.NewCell(runes[j])
			}
		}
		cells[i] = row
	}

	return &testGrid{
		cells:     cells,
		rows:      numRows,
		cols:      numCols,
		cursorPos: core.Origin(),
		cursorVis: true,
	}
}

func (g *testGrid) Cell(row, col int) (core.Cell, bool) {
	if row < 0 || row >= g.rows || col < 0 || col >= g.cols {
		return core.Cell{}, false
	}
	return g.cells[row][col], true
}

func (g *testGrid) Dimensions() core.Dimensions {
	return core.NewDimensions(uint16(g.rows), uint16(g.cols))
}

func (g *testGrid) CursorPosition() core.Position {
	return g.cursorPos
}

func (g *testGrid) CursorVisible() bool {
	return g.cursorVis
}

func (g *testGrid) ExtractText(bounds core.Bounds) string {
	var b strings.Builder
	for rowOff := 0; rowOff < int(bounds.Height); rowOff++ {
		row := int(bounds.Row) + rowOff
		if rowOff > 0 {
			b.WriteByte('\n')
		}
		for colOff := 0; colOff < int(bounds.Width); colOff++ {
			col := int(bounds.Col) + colOff
			if cell, ok := g.Cell(row, col); ok {
				b.WriteRune(cell.Character)
			}
		}
	}
	return b.String()
}

// withCursor returns the grid with a specific cursor position.
func (g *testGrid) withCursor(row, col int) *testGrid {
	g.cursorPos = core.NewPosition(uint16(row), uint16(col))
	return g
}

// setCellAttr sets attributes on a cell for testing attribute-based detectors.
func (g *testGrid) setCellAttr(row, col int, attrs core.CellAttributes) {
	if row >= 0 && row < g.rows && col >= 0 && col < g.cols {
		g.cells[row][col].Attrs = attrs
	}
}

// setCellBg sets the background color on a cell.
func (g *testGrid) setCellBg(row, col int, bg core.Color) {
	if row >= 0 && row < g.rows && col >= 0 && col < g.cols {
		g.cells[row][col].Bg = bg
	}
}
