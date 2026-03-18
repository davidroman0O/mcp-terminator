// Package emulator provides terminal emulation with a VTE parser, cell grid, and PTY lifecycle.
package emulator

import (
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// CursorStyle represents the visual style of the cursor.
type CursorStyle int

const (
	// CursorBlock fills the entire cell.
	CursorBlock CursorStyle = iota
	// CursorUnderline renders at the bottom of the cell.
	CursorUnderline
	// CursorBar renders as a vertical line at the left of the cell.
	CursorBar
)

// Cursor holds the cursor state.
type Cursor struct {
	Position core.Position
	Visible  bool
	Style    CursorStyle
}

// DefaultCursor returns a cursor at origin, visible, block style.
func DefaultCursor() Cursor {
	return Cursor{
		Position: core.Origin(),
		Visible:  true,
		Style:    CursorBlock,
	}
}

// Grid is the terminal grid state buffer.
type Grid struct {
	cells       []core.Cell
	dimensions  core.Dimensions
	cursor      Cursor
	savedCursor *Cursor

	// scrollRegion: top and bottom rows (0-indexed, inclusive). nil means full screen.
	scrollRegion *[2]uint16

	// Current attributes and colors applied to new characters.
	currentAttrs core.CellAttributes
	currentFg    core.Color
	currentBg    core.Color

	// lineWrapped[row] is true if that row is a continuation of the previous row.
	lineWrapped []bool
}

// NewGrid creates a grid filled with empty cells.
func NewGrid(dims core.Dimensions) *Grid {
	count := dims.CellCount()
	cells := make([]core.Cell, count)
	for i := range cells {
		cells[i] = core.DefaultCell()
	}
	return &Grid{
		cells:       cells,
		dimensions:  dims,
		cursor:      DefaultCursor(),
		currentFg:   core.DefaultColor(),
		currentBg:   core.DefaultColor(),
		lineWrapped: make([]bool, dims.Rows),
	}
}

// Dimensions returns the grid dimensions.
func (g *Grid) Dimensions() core.Dimensions {
	return g.dimensions
}

// Cursor returns a pointer to the cursor.
func (g *Grid) Cursor() *Cursor {
	return &g.cursor
}

// CursorVisible reports whether the cursor is visible.
func (g *Grid) CursorVisible() bool {
	return g.cursor.Visible
}

// --- Cell access ---

// cellIndex converts row/col to a flat index. Returns -1 if out of bounds.
func (g *Grid) cellIndex(row, col uint16) int {
	if row >= g.dimensions.Rows || col >= g.dimensions.Cols {
		return -1
	}
	return int(row)*int(g.dimensions.Cols) + int(col)
}

// Cell returns the cell at (row, col), or nil if out of bounds.
func (g *Grid) Cell(row, col uint16) *core.Cell {
	idx := g.cellIndex(row, col)
	if idx < 0 {
		return nil
	}
	return &g.cells[idx]
}

// SetCell sets the cell at (row, col). No-op if out of bounds.
func (g *Grid) SetCell(row, col uint16, cell core.Cell) {
	idx := g.cellIndex(row, col)
	if idx < 0 {
		return
	}
	g.cells[idx] = cell
}

// Row returns the cells for a given row, or nil if out of bounds.
func (g *Grid) Row(row uint16) []core.Cell {
	if row >= g.dimensions.Rows {
		return nil
	}
	start := int(row) * int(g.dimensions.Cols)
	end := start + int(g.dimensions.Cols)
	return g.cells[start:end]
}

// --- Attributes ---

// CurrentAttrs returns the current cell attributes.
func (g *Grid) CurrentAttrs() core.CellAttributes {
	return g.currentAttrs
}

// SetCurrentAttrs sets the current cell attributes.
func (g *Grid) SetCurrentAttrs(attrs core.CellAttributes) {
	g.currentAttrs = attrs
}

// CurrentFg returns the current foreground color.
func (g *Grid) CurrentFg() core.Color {
	return g.currentFg
}

// SetCurrentFg sets the current foreground color.
func (g *Grid) SetCurrentFg(c core.Color) {
	g.currentFg = c
}

// CurrentBg returns the current background color.
func (g *Grid) CurrentBg() core.Color {
	return g.currentBg
}

// SetCurrentBg sets the current background color.
func (g *Grid) SetCurrentBg(c core.Color) {
	g.currentBg = c
}

// --- Cursor movement ---

// MoveCursor sets the cursor to (row, col), clamped to grid bounds.
func (g *Grid) MoveCursor(row, col uint16) {
	if g.dimensions.Rows > 0 {
		if row >= g.dimensions.Rows {
			row = g.dimensions.Rows - 1
		}
	}
	if g.dimensions.Cols > 0 {
		if col >= g.dimensions.Cols {
			col = g.dimensions.Cols - 1
		}
	}
	g.cursor.Position = core.NewPosition(row, col)
}

// CursorUp moves the cursor up by n rows (clamped to row 0).
func (g *Grid) CursorUp(n uint16) {
	if n > g.cursor.Position.Row {
		g.cursor.Position.Row = 0
	} else {
		g.cursor.Position.Row -= n
	}
}

// CursorDown moves the cursor down by n rows (clamped to last row).
func (g *Grid) CursorDown(n uint16) {
	newRow := g.cursor.Position.Row + n
	maxRow := g.dimensions.Rows
	if maxRow > 0 {
		maxRow--
	}
	if newRow > maxRow {
		newRow = maxRow
	}
	g.cursor.Position.Row = newRow
}

// CursorLeft moves the cursor left by n columns (clamped to col 0).
func (g *Grid) CursorLeft(n uint16) {
	if n > g.cursor.Position.Col {
		g.cursor.Position.Col = 0
	} else {
		g.cursor.Position.Col -= n
	}
}

// CursorRight moves the cursor right by n columns (clamped to last column).
func (g *Grid) CursorRight(n uint16) {
	newCol := g.cursor.Position.Col + n
	maxCol := g.dimensions.Cols
	if maxCol > 0 {
		maxCol--
	}
	if newCol > maxCol {
		newCol = maxCol
	}
	g.cursor.Position.Col = newCol
}

// --- Cursor save / restore ---

// SaveCursor saves the current cursor state.
func (g *Grid) SaveCursor() {
	c := g.cursor
	g.savedCursor = &c
}

// RestoreCursor restores the previously saved cursor state.
func (g *Grid) RestoreCursor() {
	if g.savedCursor != nil {
		g.cursor = *g.savedCursor
		g.savedCursor = nil
	}
}

// --- Scroll region ---

// SetScrollRegion sets the scroll region (0-indexed, inclusive top and bottom rows).
func (g *Grid) SetScrollRegion(top, bottom uint16) {
	g.scrollRegion = &[2]uint16{top, bottom}
}

// ClearScrollRegion resets the scroll region to full screen.
func (g *Grid) ClearScrollRegion() {
	g.scrollRegion = nil
}

// scrollTop returns the effective top of the scroll region.
func (g *Grid) scrollTop() uint16 {
	if g.scrollRegion != nil {
		return g.scrollRegion[0]
	}
	return 0
}

// scrollBottom returns the effective bottom of the scroll region.
func (g *Grid) scrollBottom() uint16 {
	if g.scrollRegion != nil {
		return g.scrollRegion[1]
	}
	if g.dimensions.Rows > 0 {
		return g.dimensions.Rows - 1
	}
	return 0
}

// --- Scrolling ---

// ScrollUp scrolls the scroll region up by n lines. The top n lines are discarded
// and blank lines are inserted at the bottom.
func (g *Grid) ScrollUp(n uint16) {
	top := g.scrollTop()
	bottom := g.scrollBottom()
	if top > bottom || n == 0 {
		return
	}
	regionHeight := bottom - top + 1
	if n > regionHeight {
		n = regionHeight
	}
	cols := g.dimensions.Cols

	// Shift rows up within the scroll region.
	for row := top; row <= bottom-n; row++ {
		srcStart := int(row+n) * int(cols)
		dstStart := int(row) * int(cols)
		copy(g.cells[dstStart:dstStart+int(cols)], g.cells[srcStart:srcStart+int(cols)])
		g.lineWrapped[row] = g.lineWrapped[row+n]
	}
	// Clear the bottom n rows.
	for row := bottom - n + 1; row <= bottom; row++ {
		start := int(row) * int(cols)
		for c := start; c < start+int(cols); c++ {
			g.cells[c] = core.DefaultCell()
		}
		g.lineWrapped[row] = false
	}
}

// ScrollDown scrolls the scroll region down by n lines. The bottom n lines are discarded
// and blank lines are inserted at the top.
func (g *Grid) ScrollDown(n uint16) {
	top := g.scrollTop()
	bottom := g.scrollBottom()
	if top > bottom || n == 0 {
		return
	}
	regionHeight := bottom - top + 1
	if n > regionHeight {
		n = regionHeight
	}
	cols := g.dimensions.Cols

	// Shift rows down within the scroll region.
	for row := bottom; row >= top+n; row-- {
		srcStart := int(row-n) * int(cols)
		dstStart := int(row) * int(cols)
		copy(g.cells[dstStart:dstStart+int(cols)], g.cells[srcStart:srcStart+int(cols)])
		g.lineWrapped[row] = g.lineWrapped[row-n]
	}
	// Clear the top n rows.
	for row := top; row < top+n; row++ {
		start := int(row) * int(cols)
		for c := start; c < start+int(cols); c++ {
			g.cells[c] = core.DefaultCell()
		}
		g.lineWrapped[row] = false
	}
}

// --- Line and char insertion/deletion ---

// InsertLines inserts n blank lines at the cursor row, pushing existing lines down.
func (g *Grid) InsertLines(n uint16) {
	curRow := g.cursor.Position.Row
	bottom := g.scrollBottom()
	if curRow > bottom || n == 0 {
		return
	}
	regionHeight := bottom - curRow + 1
	if n > regionHeight {
		n = regionHeight
	}
	cols := g.dimensions.Cols

	// Shift rows down from bottom.
	for row := bottom; row >= curRow+n; row-- {
		srcStart := int(row-n) * int(cols)
		dstStart := int(row) * int(cols)
		copy(g.cells[dstStart:dstStart+int(cols)], g.cells[srcStart:srcStart+int(cols)])
		g.lineWrapped[row] = g.lineWrapped[row-n]
	}
	// Clear the inserted rows.
	for row := curRow; row < curRow+n; row++ {
		start := int(row) * int(cols)
		for c := start; c < start+int(cols); c++ {
			g.cells[c] = core.DefaultCell()
		}
		g.lineWrapped[row] = false
	}
}

// DeleteLines deletes n lines at the cursor row, pulling lines up from below.
func (g *Grid) DeleteLines(n uint16) {
	curRow := g.cursor.Position.Row
	bottom := g.scrollBottom()
	if curRow > bottom || n == 0 {
		return
	}
	regionHeight := bottom - curRow + 1
	if n > regionHeight {
		n = regionHeight
	}
	cols := g.dimensions.Cols

	// Shift rows up.
	for row := curRow; row <= bottom-n; row++ {
		srcStart := int(row+n) * int(cols)
		dstStart := int(row) * int(cols)
		copy(g.cells[dstStart:dstStart+int(cols)], g.cells[srcStart:srcStart+int(cols)])
		g.lineWrapped[row] = g.lineWrapped[row+n]
	}
	// Clear bottom rows.
	for row := bottom - n + 1; row <= bottom; row++ {
		start := int(row) * int(cols)
		for c := start; c < start+int(cols); c++ {
			g.cells[c] = core.DefaultCell()
		}
		g.lineWrapped[row] = false
	}
}

// InsertChars inserts n blank characters at the cursor position, shifting existing
// characters to the right. Characters pushed past the right edge are lost.
func (g *Grid) InsertChars(n uint16) {
	row := g.cursor.Position.Row
	col := g.cursor.Position.Col
	cols := g.dimensions.Cols
	if col >= cols {
		return
	}
	remaining := cols - col
	if n > remaining {
		n = remaining
	}
	rowStart := int(row) * int(cols)

	// Shift right.
	for c := int(cols) - 1; c >= int(col)+int(n); c-- {
		g.cells[rowStart+c] = g.cells[rowStart+c-int(n)]
	}
	// Clear the inserted positions.
	for c := int(col); c < int(col)+int(n); c++ {
		g.cells[rowStart+c] = core.DefaultCell()
	}
}

// DeleteChars deletes n characters at the cursor position, shifting characters left.
// Blank characters are inserted at the right edge.
func (g *Grid) DeleteChars(n uint16) {
	row := g.cursor.Position.Row
	col := g.cursor.Position.Col
	cols := g.dimensions.Cols
	if col >= cols {
		return
	}
	remaining := cols - col
	if n > remaining {
		n = remaining
	}
	rowStart := int(row) * int(cols)

	// Shift left.
	for c := int(col); c < int(cols)-int(n); c++ {
		g.cells[rowStart+c] = g.cells[rowStart+c+int(n)]
	}
	// Clear the right edge.
	for c := int(cols) - int(n); c < int(cols); c++ {
		g.cells[rowStart+c] = core.DefaultCell()
	}
}

// --- Clear operations ---

// ClearLineMode specifies what portion of a line to clear.
type ClearLineMode int

const (
	// ClearLineAfter clears from cursor to end of line.
	ClearLineAfter ClearLineMode = 0
	// ClearLineBefore clears from start of line to cursor (inclusive).
	ClearLineBefore ClearLineMode = 1
	// ClearLineAll clears the entire line.
	ClearLineAll ClearLineMode = 2
)

// ClearLine clears part of a row according to mode.
func (g *Grid) ClearLine(row uint16, mode ClearLineMode) {
	if row >= g.dimensions.Rows {
		return
	}
	cols := g.dimensions.Cols
	curCol := g.cursor.Position.Col

	var startCol, endCol uint16
	switch mode {
	case ClearLineAfter:
		startCol = curCol
		endCol = cols
	case ClearLineBefore:
		startCol = 0
		endCol = curCol + 1
		if endCol > cols {
			endCol = cols
		}
	case ClearLineAll:
		startCol = 0
		endCol = cols
	}

	for c := startCol; c < endCol; c++ {
		idx := g.cellIndex(row, c)
		if idx >= 0 {
			g.cells[idx] = core.DefaultCell()
		}
	}
}

// ClearScreenMode specifies what portion of the screen to clear.
type ClearScreenMode int

const (
	// ClearScreenAfter clears from cursor to end of screen.
	ClearScreenAfter ClearScreenMode = 0
	// ClearScreenBefore clears from start of screen to cursor.
	ClearScreenBefore ClearScreenMode = 1
	// ClearScreenAll clears the entire screen.
	ClearScreenAll ClearScreenMode = 2
	// ClearScreenScrollback clears the entire screen and scrollback.
	ClearScreenScrollback ClearScreenMode = 3
)

// ClearScreen clears the screen according to mode.
func (g *Grid) ClearScreen(mode ClearScreenMode) {
	rows := g.dimensions.Rows
	cols := g.dimensions.Cols
	curRow := g.cursor.Position.Row
	curCol := g.cursor.Position.Col

	switch mode {
	case ClearScreenAfter:
		// Rest of current row.
		for c := curCol; c < cols; c++ {
			if idx := g.cellIndex(curRow, c); idx >= 0 {
				g.cells[idx] = core.DefaultCell()
			}
		}
		// All rows below.
		for r := curRow + 1; r < rows; r++ {
			for c := uint16(0); c < cols; c++ {
				if idx := g.cellIndex(r, c); idx >= 0 {
					g.cells[idx] = core.DefaultCell()
				}
			}
		}
	case ClearScreenBefore:
		// All rows above.
		for r := uint16(0); r < curRow; r++ {
			for c := uint16(0); c < cols; c++ {
				if idx := g.cellIndex(r, c); idx >= 0 {
					g.cells[idx] = core.DefaultCell()
				}
			}
		}
		// Current row up to and including cursor.
		for c := uint16(0); c <= curCol && c < cols; c++ {
			if idx := g.cellIndex(curRow, c); idx >= 0 {
				g.cells[idx] = core.DefaultCell()
			}
		}
	case ClearScreenAll, ClearScreenScrollback:
		g.Clear()
	}
}

// Clear resets all cells to default and clears line-wrap flags.
func (g *Grid) Clear() {
	for i := range g.cells {
		g.cells[i] = core.DefaultCell()
	}
	for i := range g.lineWrapped {
		g.lineWrapped[i] = false
	}
}

// ClearRegion clears cells within the given bounds.
func (g *Grid) ClearRegion(b core.Bounds) {
	for r := b.Row; r < b.Row+b.Height; r++ {
		for c := b.Col; c < b.Col+b.Width; c++ {
			if idx := g.cellIndex(r, c); idx >= 0 {
				g.cells[idx] = core.DefaultCell()
			}
		}
	}
}

// --- Line wrap ---

// IsLineWrapped reports whether the given row is a continuation of the previous row.
func (g *Grid) IsLineWrapped(row uint16) bool {
	if row < uint16(len(g.lineWrapped)) {
		return g.lineWrapped[row]
	}
	return false
}

// SetLineWrapped sets the line-wrap flag for a row.
func (g *Grid) SetLineWrapped(row uint16, wrapped bool) {
	if row < uint16(len(g.lineWrapped)) {
		g.lineWrapped[row] = wrapped
	}
}

// --- Resize ---

// Resize changes the grid dimensions, preserving content from the top-left corner.
func (g *Grid) Resize(newDims core.Dimensions) {
	newCount := newDims.CellCount()
	newCells := make([]core.Cell, newCount)
	for i := range newCells {
		newCells[i] = core.DefaultCell()
	}
	newWrapped := make([]bool, newDims.Rows)

	copyRows := g.dimensions.Rows
	if newDims.Rows < copyRows {
		copyRows = newDims.Rows
	}
	copyCols := g.dimensions.Cols
	if newDims.Cols < copyCols {
		copyCols = newDims.Cols
	}

	for r := uint16(0); r < copyRows; r++ {
		for c := uint16(0); c < copyCols; c++ {
			oldIdx := int(r)*int(g.dimensions.Cols) + int(c)
			newIdx := int(r)*int(newDims.Cols) + int(c)
			newCells[newIdx] = g.cells[oldIdx]
		}
		newWrapped[r] = g.lineWrapped[r]
	}

	g.cells = newCells
	g.lineWrapped = newWrapped
	g.dimensions = newDims

	// Clamp cursor.
	if newDims.Rows > 0 && g.cursor.Position.Row >= newDims.Rows {
		g.cursor.Position.Row = newDims.Rows - 1
	}
	if newDims.Cols > 0 && g.cursor.Position.Col >= newDims.Cols {
		g.cursor.Position.Col = newDims.Cols - 1
	}
}

// --- Text extraction ---

// ExtractText returns the text in the given bounds as a string.
// Trailing whitespace is trimmed per line. Wrapped lines are joined without newlines.
func (g *Grid) ExtractText(b core.Bounds) string {
	var buf strings.Builder
	for r := b.Row; r < b.Row+b.Height; r++ {
		// Add newline if not the first row and this row is not wrapped.
		if r > b.Row && !g.IsLineWrapped(r) {
			buf.WriteByte('\n')
		}
		for c := b.Col; c < b.Col+b.Width; c++ {
			cell := g.Cell(r, c)
			if cell != nil {
				buf.WriteRune(cell.Character)
			}
		}
	}
	// Trim trailing whitespace per line.
	lines := strings.Split(buf.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.Join(lines, "\n")
}

// RawText returns the entire grid as a plain text string.
func (g *Grid) RawText() string {
	return g.ExtractText(core.NewBounds(0, 0, g.dimensions.Cols, g.dimensions.Rows))
}
