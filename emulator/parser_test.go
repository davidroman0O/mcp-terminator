package emulator

import (
	"testing"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newParser(rows, cols uint16) *Parser {
	return NewParser(NewGrid(core.NewDimensions(rows, cols)))
}

// --- Print and basic control ---

func TestParserPrintBasic(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("Hello"))

	assert.Equal(t, 'H', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'e', p.Grid().Cell(0, 1).Character)
	assert.Equal(t, 'l', p.Grid().Cell(0, 2).Character)
	assert.Equal(t, 'l', p.Grid().Cell(0, 3).Character)
	assert.Equal(t, 'o', p.Grid().Cell(0, 4).Character)
	assert.Equal(t, uint16(5), p.Grid().Cursor().Position.Col)
}

func TestParserLinefeed(t *testing.T) {
	p := newParser(24, 80)
	assert.Equal(t, uint16(0), p.Grid().Cursor().Position.Row)

	p.Process([]byte("\n"))
	assert.Equal(t, uint16(1), p.Grid().Cursor().Position.Row)

	p.Process([]byte("\n"))
	assert.Equal(t, uint16(2), p.Grid().Cursor().Position.Row)
}

func TestParserCarriageReturn(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("Hello"))
	assert.Equal(t, uint16(5), p.Grid().Cursor().Position.Col)

	p.Process([]byte("\r"))
	assert.Equal(t, uint16(0), p.Grid().Cursor().Position.Col)
}

func TestParserBackspace(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("Hello"))
	p.Process([]byte("\b"))
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Col)
}

func TestParserTab(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\t"))
	assert.Equal(t, uint16(8), p.Grid().Cursor().Position.Col)

	p.Process([]byte("\t"))
	assert.Equal(t, uint16(16), p.Grid().Cursor().Position.Col)
}

func TestParserNewlineAtBottom(t *testing.T) {
	p := newParser(5, 10)
	// Move to last row.
	for i := 0; i < 4; i++ {
		p.Process([]byte("\n"))
	}
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Row)

	// Write something on last row, then LF should scroll.
	p.Process([]byte("LAST"))
	p.Process([]byte("\n"))
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Row)
}

// --- Cursor movement CSI ---

func TestParserCursorUp(t *testing.T) {
	p := newParser(24, 80)
	// Move down first.
	for i := 0; i < 10; i++ {
		p.Process([]byte("\n"))
	}
	assert.Equal(t, uint16(10), p.Grid().Cursor().Position.Row)

	p.Process([]byte("\x1b[5A"))
	assert.Equal(t, uint16(5), p.Grid().Cursor().Position.Row)
}

func TestParserCursorDown(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[10B"))
	assert.Equal(t, uint16(10), p.Grid().Cursor().Position.Row)
}

func TestParserCursorForward(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[15C"))
	assert.Equal(t, uint16(15), p.Grid().Cursor().Position.Col)
}

func TestParserCursorBackward(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[20C")) // Move right 20
	p.Process([]byte("\x1b[5D"))  // Back 5
	assert.Equal(t, uint16(15), p.Grid().Cursor().Position.Col)
}

func TestParserCursorPosition(t *testing.T) {
	p := newParser(24, 80)
	// CSI 11;21 H -> row 10, col 20 (1-based in sequence)
	p.Process([]byte("\x1b[11;21H"))
	assert.Equal(t, uint16(10), p.Grid().Cursor().Position.Row)
	assert.Equal(t, uint16(20), p.Grid().Cursor().Position.Col)
}

func TestParserHVP(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[5;10f"))
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Row)
	assert.Equal(t, uint16(9), p.Grid().Cursor().Position.Col)
}

func TestParserCursorSaveRestore(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[5;10H")) // Move to (4,9)
	p.Process([]byte("\x1b[s"))     // Save
	p.Process([]byte("\x1b[1;1H"))  // Move to (0,0)
	assert.Equal(t, uint16(0), p.Grid().Cursor().Position.Row)

	p.Process([]byte("\x1b[u")) // Restore
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Row)
	assert.Equal(t, uint16(9), p.Grid().Cursor().Position.Col)
}

func TestParserDECSCDECRC(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[5;10H")) // Move to (4,9)
	p.Process([]byte("\x1b7"))      // DECSC
	p.Process([]byte("\x1b[1;1H"))  // Move to (0,0)
	p.Process([]byte("\x1b8"))      // DECRC
	assert.Equal(t, uint16(4), p.Grid().Cursor().Position.Row)
	assert.Equal(t, uint16(9), p.Grid().Cursor().Position.Col)
}

// --- Erase sequences ---

func TestParserEraseDisplay(t *testing.T) {
	p := newParser(5, 10)
	// Fill with X.
	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			p.Grid().SetCell(uint16(r), uint16(c), core.NewCell('X'))
		}
	}

	p.Process([]byte("\x1b[3;6H")) // Move to (2,5)
	p.Process([]byte("\x1b[J"))    // Erase from cursor to end

	// Before cursor still has X.
	assert.Equal(t, 'X', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'X', p.Grid().Cell(2, 4).Character)

	// From cursor onward cleared.
	assert.Equal(t, ' ', p.Grid().Cell(2, 5).Character)
	assert.Equal(t, ' ', p.Grid().Cell(4, 9).Character)
}

func TestParserEraseDisplayAbove(t *testing.T) {
	p := newParser(5, 10)
	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			p.Grid().SetCell(uint16(r), uint16(c), core.NewCell('X'))
		}
	}

	p.Process([]byte("\x1b[3;6H")) // Move to (2,5)
	p.Process([]byte("\x1b[1J"))   // Erase from start to cursor

	assert.Equal(t, ' ', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, ' ', p.Grid().Cell(2, 5).Character)
	assert.Equal(t, 'X', p.Grid().Cell(2, 6).Character)
	assert.Equal(t, 'X', p.Grid().Cell(4, 9).Character)
}

func TestParserEraseDisplayAll(t *testing.T) {
	p := newParser(5, 10)
	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			p.Grid().SetCell(uint16(r), uint16(c), core.NewCell('X'))
		}
	}

	p.Process([]byte("\x1b[2J"))
	for r := 0; r < 5; r++ {
		for c := 0; c < 10; c++ {
			assert.Equal(t, ' ', p.Grid().Cell(uint16(r), uint16(c)).Character)
		}
	}
}

func TestParserEraseLine(t *testing.T) {
	p := newParser(5, 10)
	for c := uint16(0); c < 10; c++ {
		p.Grid().SetCell(0, c, core.NewCell('X'))
	}

	p.Process([]byte("\x1b[1;6H")) // (0,5)
	p.Process([]byte("\x1b[K"))    // Erase to end of line

	assert.Equal(t, 'X', p.Grid().Cell(0, 4).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 5).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 9).Character)
}

func TestParserEraseLineFromStart(t *testing.T) {
	p := newParser(5, 10)
	for c := uint16(0); c < 10; c++ {
		p.Grid().SetCell(0, c, core.NewCell('X'))
	}

	p.Process([]byte("\x1b[1;6H")) // (0,5)
	p.Process([]byte("\x1b[1K"))   // Erase from start to cursor

	assert.Equal(t, ' ', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 5).Character)
	assert.Equal(t, 'X', p.Grid().Cell(0, 6).Character)
}

func TestParserEraseLineAll(t *testing.T) {
	p := newParser(5, 10)
	for c := uint16(0); c < 10; c++ {
		p.Grid().SetCell(0, c, core.NewCell('X'))
	}

	p.Process([]byte("\x1b[2K"))
	for c := uint16(0); c < 10; c++ {
		assert.Equal(t, ' ', p.Grid().Cell(0, c).Character)
	}
}

// --- Insert/Delete lines and chars ---

func TestParserInsertLines(t *testing.T) {
	p := newParser(5, 5)
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	p.Process([]byte("\x1b[2;1H")) // (1,0)
	p.Process([]byte("\x1b[2L"))   // Insert 2 lines

	assert.Equal(t, 'A', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, ' ', p.Grid().Cell(1, 0).Character) // inserted
	assert.Equal(t, ' ', p.Grid().Cell(2, 0).Character) // inserted
	assert.Equal(t, 'B', p.Grid().Cell(3, 0).Character)
}

func TestParserDeleteLines(t *testing.T) {
	p := newParser(5, 5)
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	p.Process([]byte("\x1b[2;1H")) // (1,0)
	p.Process([]byte("\x1b[2M"))   // Delete 2 lines

	assert.Equal(t, 'A', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'D', p.Grid().Cell(1, 0).Character)
	assert.Equal(t, 'E', p.Grid().Cell(2, 0).Character)
	assert.Equal(t, ' ', p.Grid().Cell(3, 0).Character)
}

func TestParserInsertChars(t *testing.T) {
	p := newParser(1, 10)
	for c := uint16(0); c < 10; c++ {
		p.Grid().SetCell(0, c, core.NewCell(rune('0'+c)))
	}

	p.Process([]byte("\x1b[1;4H")) // (0,3)
	p.Process([]byte("\x1b[2@"))   // Insert 2 chars

	assert.Equal(t, '2', p.Grid().Cell(0, 2).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 3).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 4).Character)
	assert.Equal(t, '3', p.Grid().Cell(0, 5).Character)
}

func TestParserDeleteChars(t *testing.T) {
	p := newParser(1, 10)
	for c := uint16(0); c < 10; c++ {
		p.Grid().SetCell(0, c, core.NewCell(rune('0'+c)))
	}

	p.Process([]byte("\x1b[1;4H")) // (0,3)
	p.Process([]byte("\x1b[2P"))   // Delete 2 chars

	assert.Equal(t, '2', p.Grid().Cell(0, 2).Character)
	assert.Equal(t, '5', p.Grid().Cell(0, 3).Character)
	assert.Equal(t, ' ', p.Grid().Cell(0, 8).Character)
}

// --- SGR colors ---

func TestParserSGRForegroundColor(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[31mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, 'X', cell.Character)
	assert.Equal(t, core.ColorRed, cell.Fg.Type)
}

func TestParserSGRBackgroundColor(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[42mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorGreen, cell.Bg.Type)
}

func TestParserSGRBrightColors(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[91mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorBrightRed, cell.Fg.Type)
}

func TestParserSGRAttributes(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[1;4mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.True(t, cell.Attrs.Bold)
	assert.True(t, cell.Attrs.Underline)
}

func TestParserSGRReset(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[1;31mA\x1b[0mB"))

	cellA := p.Grid().Cell(0, 0)
	require.NotNil(t, cellA)
	assert.True(t, cellA.Attrs.Bold)
	assert.Equal(t, core.ColorRed, cellA.Fg.Type)

	cellB := p.Grid().Cell(0, 1)
	require.NotNil(t, cellB)
	assert.False(t, cellB.Attrs.Bold)
	assert.Equal(t, core.ColorDefault, cellB.Fg.Type)
}

func TestParserSGR256Color(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[38;5;42mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorIndexed, cell.Fg.Type)
	require.NotNil(t, cell.Fg.Index)
	assert.Equal(t, uint8(42), *cell.Fg.Index)
}

func TestParserSGRRGBColor(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[38;2;255;128;64mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorRGB, cell.Fg.Type)
	require.NotNil(t, cell.Fg.R)
	assert.Equal(t, uint8(255), *cell.Fg.R)
	assert.Equal(t, uint8(128), *cell.Fg.G)
	assert.Equal(t, uint8(64), *cell.Fg.B)
}

func TestParserSGR256Background(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[48;5;100mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorIndexed, cell.Bg.Type)
	require.NotNil(t, cell.Bg.Index)
	assert.Equal(t, uint8(100), *cell.Bg.Index)
}

func TestParserSGRRGBBackground(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("\x1b[48;2;10;20;30mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.Equal(t, core.ColorRGB, cell.Bg.Type)
	require.NotNil(t, cell.Bg.R)
	assert.Equal(t, uint8(10), *cell.Bg.R)
	assert.Equal(t, uint8(20), *cell.Bg.G)
	assert.Equal(t, uint8(30), *cell.Bg.B)
}

func TestParserSGRAllAttributes(t *testing.T) {
	p := newParser(24, 80)
	// bold, dim, italic, underline, blink, reverse, hidden, strikethrough
	p.Process([]byte("\x1b[1;2;3;4;5;7;8;9mX"))

	cell := p.Grid().Cell(0, 0)
	require.NotNil(t, cell)
	assert.True(t, cell.Attrs.Bold)
	assert.True(t, cell.Attrs.Dim)
	assert.True(t, cell.Attrs.Italic)
	assert.True(t, cell.Attrs.Underline)
	assert.True(t, cell.Attrs.Blink)
	assert.True(t, cell.Attrs.Reverse)
	assert.True(t, cell.Attrs.Hidden)
	assert.True(t, cell.Attrs.Strikethrough)

	// Reset each individually.
	p.Process([]byte("\x1b[22;23;24;25;27;28;29mY"))
	cell = p.Grid().Cell(0, 1)
	require.NotNil(t, cell)
	assert.False(t, cell.Attrs.Bold)
	assert.False(t, cell.Attrs.Dim)
	assert.False(t, cell.Attrs.Italic)
	assert.False(t, cell.Attrs.Underline)
	assert.False(t, cell.Attrs.Blink)
	assert.False(t, cell.Attrs.Reverse)
	assert.False(t, cell.Attrs.Hidden)
	assert.False(t, cell.Attrs.Strikethrough)
}

// --- Scroll sequences ---

func TestParserScrollUp(t *testing.T) {
	p := newParser(5, 5)
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	p.Process([]byte("\x1b[2S")) // Scroll up 2

	assert.Equal(t, 'C', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, ' ', p.Grid().Cell(3, 0).Character)
}

func TestParserScrollDown(t *testing.T) {
	p := newParser(5, 5)
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	p.Process([]byte("\x1b[2T")) // Scroll down 2

	assert.Equal(t, ' ', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'A', p.Grid().Cell(2, 0).Character)
}

func TestParserScrollRegion(t *testing.T) {
	p := newParser(5, 5)
	// Set scroll region rows 2-4 (1-based)
	p.Process([]byte("\x1b[2;4r"))

	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell(rune('A'+r)))
		}
	}

	p.Process([]byte("\x1b[1S")) // Scroll up 1

	// Row 0 untouched.
	assert.Equal(t, 'A', p.Grid().Cell(0, 0).Character)
	// Row 1 should have what was row 2 ('C').
	assert.Equal(t, 'C', p.Grid().Cell(1, 0).Character)
	// Row 2 should have what was row 3 ('D').
	assert.Equal(t, 'D', p.Grid().Cell(2, 0).Character)
	// Row 3 cleared.
	assert.Equal(t, ' ', p.Grid().Cell(3, 0).Character)
	// Row 4 untouched.
	assert.Equal(t, 'E', p.Grid().Cell(4, 0).Character)
}

// --- Mode sequences ---

func TestParserCursorVisibility(t *testing.T) {
	p := newParser(24, 80)
	assert.True(t, p.Grid().CursorVisible())

	p.Process([]byte("\x1b[?25l")) // hide
	assert.False(t, p.Grid().CursorVisible())

	p.Process([]byte("\x1b[?25h")) // show
	assert.True(t, p.Grid().CursorVisible())
}

func TestParserAlternateScreenBuffer(t *testing.T) {
	p := newParser(5, 5)
	for r := uint16(0); r < 5; r++ {
		for c := uint16(0); c < 5; c++ {
			p.Grid().SetCell(r, c, core.NewCell('X'))
		}
	}

	p.Process([]byte("\x1b[?1049h")) // Switch to alt screen — clears
	assert.Equal(t, ' ', p.Grid().Cell(0, 0).Character)
}

// --- OSC consumption ---

func TestParserOSCConsumed(t *testing.T) {
	p := newParser(24, 80)
	// OSC 0; title BEL then print A.
	p.Process([]byte("\x1b]0;My Title\x07A"))
	// 'A' should be printed after OSC is consumed.
	assert.Equal(t, 'A', p.Grid().Cell(0, 0).Character)
}

func TestParserOSCWithST(t *testing.T) {
	p := newParser(24, 80)
	// OSC terminated by ST (ESC \) then print B.
	p.Process([]byte("\x1b]0;Title\x1b\\B"))
	assert.Equal(t, 'B', p.Grid().Cell(0, 0).Character)
}

// --- Complex sequences ---

func TestParserMultiLineOutput(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("Line1\r\nLine2\r\nLine3"))

	row0 := extractRowText(p.Grid(), 0, 5)
	row1 := extractRowText(p.Grid(), 1, 5)
	row2 := extractRowText(p.Grid(), 2, 5)

	assert.Equal(t, "Line1", row0)
	assert.Equal(t, "Line2", row1)
	assert.Equal(t, "Line3", row2)
}

func TestParserOverwrite(t *testing.T) {
	p := newParser(24, 80)
	p.Process([]byte("AAAA\rBB"))

	assert.Equal(t, 'B', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'B', p.Grid().Cell(0, 1).Character)
	assert.Equal(t, 'A', p.Grid().Cell(0, 2).Character)
	assert.Equal(t, 'A', p.Grid().Cell(0, 3).Character)
}

func TestParserDefaultParam(t *testing.T) {
	// CSI H with no params should go to (0,0).
	p := newParser(24, 80)
	p.Process([]byte("\x1b[10;10H")) // (9,9)
	p.Process([]byte("\x1b[H"))      // should go to (0,0)
	assert.Equal(t, uint16(0), p.Grid().Cursor().Position.Row)
	assert.Equal(t, uint16(0), p.Grid().Cursor().Position.Col)
}

func TestParserWrapAround(t *testing.T) {
	p := newParser(3, 5)
	p.Process([]byte("ABCDE")) // fills row 0, cursor wraps
	p.Process([]byte("F"))     // should go to row 1

	assert.Equal(t, 'A', p.Grid().Cell(0, 0).Character)
	assert.Equal(t, 'E', p.Grid().Cell(0, 4).Character)
	assert.Equal(t, 'F', p.Grid().Cell(1, 0).Character)
}

// extractRowText extracts n characters from a grid row as a string.
func extractRowText(g *Grid, row uint16, n int) string {
	var s []byte
	for c := 0; c < n; c++ {
		cell := g.Cell(row, uint16(c))
		if cell != nil {
			s = append(s, byte(cell.Character))
		}
	}
	return string(s)
}
