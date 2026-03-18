package emulator

import (
	"testing"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGrid(t *testing.T) {
	g := NewGrid(core.NewDimensions(24, 80))
	assert.Equal(t, uint16(24), g.Dimensions().Rows)
	assert.Equal(t, uint16(80), g.Dimensions().Cols)
	assert.Equal(t, core.Origin(), g.Cursor().Position)
	assert.True(t, g.CursorVisible())
}

func TestGridCellAccess(t *testing.T) {
	g := NewGrid(core.NewDimensions(10, 10))

	// Default cell is space.
	cell := g.Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, ' ', cell.Character)

	// Set cell.
	g.SetCell(5, 5, core.NewCell('X'))
	cell = g.Cell(5, 5)
	require.NotNil(t, cell)
	assert.Equal(t, 'X', cell.Character)

	// Out of bounds.
	assert.Nil(t, g.Cell(10, 10))
}

func TestGridRowAccess(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))

	for c := uint16(0); c < 10; c++ {
		g.SetCell(2, c, core.NewCell(rune('0'+c)))
	}

	row := g.Row(2)
	require.Len(t, row, 10)
	assert.Equal(t, '0', row[0].Character)
	assert.Equal(t, '9', row[9].Character)

	// Out of bounds.
	assert.Nil(t, g.Row(5))
}

func TestCursorMovement(t *testing.T) {
	g := NewGrid(core.NewDimensions(24, 80))

	g.MoveCursor(10, 20)
	assert.Equal(t, uint16(10), g.Cursor().Position.Row)
	assert.Equal(t, uint16(20), g.Cursor().Position.Col)

	// Clamping.
	g.MoveCursor(100, 200)
	assert.Equal(t, uint16(23), g.Cursor().Position.Row)
	assert.Equal(t, uint16(79), g.Cursor().Position.Col)
}

func TestCursorUpDown(t *testing.T) {
	g := NewGrid(core.NewDimensions(24, 80))
	g.MoveCursor(10, 0)

	g.CursorUp(3)
	assert.Equal(t, uint16(7), g.Cursor().Position.Row)

	g.CursorDown(5)
	assert.Equal(t, uint16(12), g.Cursor().Position.Row)

	// Clamp at top.
	g.CursorUp(100)
	assert.Equal(t, uint16(0), g.Cursor().Position.Row)

	// Clamp at bottom.
	g.CursorDown(100)
	assert.Equal(t, uint16(23), g.Cursor().Position.Row)
}

func TestCursorLeftRight(t *testing.T) {
	g := NewGrid(core.NewDimensions(24, 80))
	g.MoveCursor(0, 40)

	g.CursorLeft(10)
	assert.Equal(t, uint16(30), g.Cursor().Position.Col)

	g.CursorRight(5)
	assert.Equal(t, uint16(35), g.Cursor().Position.Col)

	// Clamp at left.
	g.CursorLeft(100)
	assert.Equal(t, uint16(0), g.Cursor().Position.Col)

	// Clamp at right.
	g.CursorRight(200)
	assert.Equal(t, uint16(79), g.Cursor().Position.Col)
}

func TestCursorSaveRestore(t *testing.T) {
	g := NewGrid(core.NewDimensions(24, 80))
	g.MoveCursor(10, 20)
	g.SaveCursor()

	g.MoveCursor(5, 5)
	assert.Equal(t, uint16(5), g.Cursor().Position.Row)

	g.RestoreCursor()
	assert.Equal(t, uint16(10), g.Cursor().Position.Row)
	assert.Equal(t, uint16(20), g.Cursor().Position.Col)
}

func TestScrollUp(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	// Fill rows with distinguishable content.
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	g.ScrollUp(2)

	// Row 0 should now have what was row 2 ('C').
	assert.Equal(t, 'C', g.Cell(0, 0).Character)
	// Row 2 should have what was row 4 ('E').
	assert.Equal(t, 'E', g.Cell(2, 0).Character)
	// Bottom 2 rows should be cleared.
	assert.Equal(t, ' ', g.Cell(3, 0).Character)
	assert.Equal(t, ' ', g.Cell(4, 0).Character)
}

func TestScrollDown(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	g.ScrollDown(2)

	// Top 2 rows should be cleared.
	assert.Equal(t, ' ', g.Cell(0, 0).Character)
	assert.Equal(t, ' ', g.Cell(1, 0).Character)
	// Row 2 should have what was row 0 ('A').
	assert.Equal(t, 'A', g.Cell(2, 0).Character)
	assert.Equal(t, 'B', g.Cell(3, 0).Character)
	assert.Equal(t, 'C', g.Cell(4, 0).Character)
}

func TestScrollRegion(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	// Set scroll region to rows 1-3.
	g.SetScrollRegion(1, 3)
	g.ScrollUp(1)

	// Row 0 should be untouched ('A').
	assert.Equal(t, 'A', g.Cell(0, 0).Character)
	// Row 1 should have what was row 2 ('C').
	assert.Equal(t, 'C', g.Cell(1, 0).Character)
	// Row 2 should have what was row 3 ('D').
	assert.Equal(t, 'D', g.Cell(2, 0).Character)
	// Row 3 should be cleared.
	assert.Equal(t, ' ', g.Cell(3, 0).Character)
	// Row 4 should be untouched ('E').
	assert.Equal(t, 'E', g.Cell(4, 0).Character)
}

func TestClearLine(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))
	for c := uint16(0); c < 10; c++ {
		g.SetCell(2, c, core.NewCell('X'))
	}
	g.MoveCursor(2, 5)

	// Clear after cursor.
	g.ClearLine(2, ClearLineAfter)
	assert.Equal(t, 'X', g.Cell(2, 4).Character)
	assert.Equal(t, ' ', g.Cell(2, 5).Character)
	assert.Equal(t, ' ', g.Cell(2, 9).Character)
}

func TestClearLineBefore(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))
	for c := uint16(0); c < 10; c++ {
		g.SetCell(2, c, core.NewCell('X'))
	}
	g.MoveCursor(2, 5)

	g.ClearLine(2, ClearLineBefore)
	assert.Equal(t, ' ', g.Cell(2, 0).Character)
	assert.Equal(t, ' ', g.Cell(2, 5).Character)
	assert.Equal(t, 'X', g.Cell(2, 6).Character)
}

func TestClearLineAll(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))
	for c := uint16(0); c < 10; c++ {
		g.SetCell(2, c, core.NewCell('X'))
	}

	g.ClearLine(2, ClearLineAll)
	for c := uint16(0); c < 10; c++ {
		assert.Equal(t, ' ', g.Cell(2, c).Character)
	}
}

func TestClearScreen(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 10; c++ {
			g.SetCell(r, c, core.NewCell('X'))
		}
	}
	g.MoveCursor(2, 5)

	g.ClearScreen(ClearScreenAfter)
	// Before cursor: still 'X'.
	assert.Equal(t, 'X', g.Cell(0, 0).Character)
	assert.Equal(t, 'X', g.Cell(2, 4).Character)
	// From cursor onward: cleared.
	assert.Equal(t, ' ', g.Cell(2, 5).Character)
	assert.Equal(t, ' ', g.Cell(4, 9).Character)
}

func TestClearScreenAll(t *testing.T) {
	g := NewGrid(core.NewDimensions(3, 3))
	for r := uint16(0); r < 3; r++ {
		for c := uint16(0); c < 3; c++ {
			g.SetCell(r, c, core.NewCell('X'))
		}
	}

	g.ClearScreen(ClearScreenAll)
	for r := uint16(0); r < 3; r++ {
		for c := uint16(0); c < 3; c++ {
			assert.Equal(t, ' ', g.Cell(r, c).Character)
		}
	}
}

func TestInsertLines(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	g.MoveCursor(1, 0)
	g.InsertLines(2)

	// Row 0 untouched.
	assert.Equal(t, 'A', g.Cell(0, 0).Character)
	// Rows 1-2 should be blank (inserted).
	assert.Equal(t, ' ', g.Cell(1, 0).Character)
	assert.Equal(t, ' ', g.Cell(2, 0).Character)
	// Row 3 should have what was row 1 ('B').
	assert.Equal(t, 'B', g.Cell(3, 0).Character)
	// Row 4 should have what was row 2 ('C').
	assert.Equal(t, 'C', g.Cell(4, 0).Character)
}

func TestDeleteLines(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	g.MoveCursor(1, 0)
	g.DeleteLines(2)

	assert.Equal(t, 'A', g.Cell(0, 0).Character)
	// Row 1 should have what was row 3 ('D').
	assert.Equal(t, 'D', g.Cell(1, 0).Character)
	assert.Equal(t, 'E', g.Cell(2, 0).Character)
	// Bottom rows should be blank.
	assert.Equal(t, ' ', g.Cell(3, 0).Character)
	assert.Equal(t, ' ', g.Cell(4, 0).Character)
}

func TestInsertChars(t *testing.T) {
	g := NewGrid(core.NewDimensions(1, 10))
	for c := uint16(0); c < 10; c++ {
		g.SetCell(0, c, core.NewCell(rune('0'+c)))
	}

	g.MoveCursor(0, 3)
	g.InsertChars(2)

	assert.Equal(t, '0', g.Cell(0, 0).Character)
	assert.Equal(t, '2', g.Cell(0, 2).Character)
	assert.Equal(t, ' ', g.Cell(0, 3).Character) // inserted
	assert.Equal(t, ' ', g.Cell(0, 4).Character) // inserted
	assert.Equal(t, '3', g.Cell(0, 5).Character)
	assert.Equal(t, '7', g.Cell(0, 9).Character) // '8' and '9' pushed off
}

func TestDeleteChars(t *testing.T) {
	g := NewGrid(core.NewDimensions(1, 10))
	for c := uint16(0); c < 10; c++ {
		g.SetCell(0, c, core.NewCell(rune('0'+c)))
	}

	g.MoveCursor(0, 3)
	g.DeleteChars(2)

	assert.Equal(t, '0', g.Cell(0, 0).Character)
	assert.Equal(t, '2', g.Cell(0, 2).Character)
	assert.Equal(t, '5', g.Cell(0, 3).Character) // shifted left
	assert.Equal(t, '9', g.Cell(0, 7).Character)
	assert.Equal(t, ' ', g.Cell(0, 8).Character) // blank fill
	assert.Equal(t, ' ', g.Cell(0, 9).Character)
}

func TestResize(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 5))
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			g.SetCell(r, c, core.NewCell('A'))
		}
	}

	// Grow.
	g.Resize(core.NewDimensions(10, 10))
	assert.Equal(t, uint16(10), g.Dimensions().Rows)
	assert.Equal(t, uint16(10), g.Dimensions().Cols)
	assert.Equal(t, 'A', g.Cell(0, 0).Character)
	assert.Equal(t, 'A', g.Cell(4, 4).Character)
	assert.Equal(t, ' ', g.Cell(9, 9).Character) // new cell

	// Shrink.
	g.Resize(core.NewDimensions(3, 3))
	assert.Equal(t, uint16(3), g.Dimensions().Rows)
	assert.Equal(t, 'A', g.Cell(2, 2).Character)
}

func TestResizeClamsCursor(t *testing.T) {
	g := NewGrid(core.NewDimensions(10, 10))
	g.MoveCursor(8, 8)

	g.Resize(core.NewDimensions(5, 5))
	assert.Equal(t, uint16(4), g.Cursor().Position.Row)
	assert.Equal(t, uint16(4), g.Cursor().Position.Col)
}

func TestExtractText(t *testing.T) {
	g := NewGrid(core.NewDimensions(5, 10))
	text := "HELLO"
	for i, ch := range text {
		g.SetCell(1, uint16(i), core.NewCell(ch))
	}

	bounds := core.NewBounds(1, 0, 10, 1)
	extracted := g.ExtractText(bounds)
	assert.Equal(t, "HELLO", extracted)
}

func TestExtractTextTrimsTrailingSpaces(t *testing.T) {
	g := NewGrid(core.NewDimensions(3, 10))
	// Row 0: "ABC       " -> trimmed to "ABC"
	g.SetCell(0, 0, core.NewCell('A'))
	g.SetCell(0, 1, core.NewCell('B'))
	g.SetCell(0, 2, core.NewCell('C'))

	text := g.ExtractText(core.NewBounds(0, 0, 10, 1))
	assert.Equal(t, "ABC", text)
}

func TestRawText(t *testing.T) {
	g := NewGrid(core.NewDimensions(3, 5))
	for r := uint16(0); r < 3; r++ {
		for c := uint16(0); c < 5; c++ {
			if (r+c)%2 == 0 {
				g.SetCell(r, c, core.NewCell('X'))
			} else {
				g.SetCell(r, c, core.NewCell('O'))
			}
		}
	}

	text := g.RawText()
	lines := splitLines(text)
	assert.Len(t, lines, 3)
	assert.Equal(t, "XOXOX", lines[0])
	assert.Equal(t, "OXOXO", lines[1])
	assert.Equal(t, "XOXOX", lines[2])
}

func TestLineWrapExtractText(t *testing.T) {
	g := NewGrid(core.NewDimensions(3, 5))
	g.SetCell(0, 0, core.NewCell('A'))
	g.SetCell(1, 0, core.NewCell('B'))
	g.SetLineWrapped(1, true) // row 1 is continuation of row 0

	text := g.ExtractText(core.NewBounds(0, 0, 5, 2))
	// Wrapped lines should be joined without newline.
	assert.Equal(t, "A    B", text)
}

// splitLines splits a string by newline.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
