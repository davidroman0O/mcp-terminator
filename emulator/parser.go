package emulator

import (
	"github.com/davidroman0O/mcp-terminator/core"
)

// parserState is the state of the ANSI escape sequence parser.
type parserState int

const (
	stateNormal parserState = iota
	stateEscape             // Received ESC (\x1b)
	stateCSI                // Received ESC [
	stateCSIParam           // Collecting CSI parameters
	stateOSC                // Receiving OSC payload (until BEL or ST)
	stateOSCEsc             // Received ESC inside OSC (looking for \)
)

// Parser is a custom ANSI/VTE escape sequence state machine that writes into a Grid.
type Parser struct {
	grid  *Grid
	state parserState

	// CSI parameter accumulation.
	csiParams        []uint16 // parsed numeric parameters
	csiCurrentParam  uint16   // parameter being built
	csiHasParam      bool     // whether we are building a param digit
	csiPrivateMarker byte     // '?' for private mode sequences, 0 otherwise
}

// NewParser creates a parser wrapping the given grid.
func NewParser(grid *Grid) *Parser {
	return &Parser{
		grid:  grid,
		state: stateNormal,
	}
}

// Grid returns the underlying grid.
func (p *Parser) Grid() *Grid {
	return p.grid
}

// Process feeds a slice of bytes through the parser state machine, updating the grid.
func (p *Parser) Process(data []byte) {
	for _, b := range data {
		p.feed(b)
	}
}

// feed processes a single byte.
func (p *Parser) feed(b byte) {
	switch p.state {
	case stateNormal:
		p.handleNormal(b)
	case stateEscape:
		p.handleEscape(b)
	case stateCSI, stateCSIParam:
		p.handleCSI(b)
	case stateOSC:
		p.handleOSC(b)
	case stateOSCEsc:
		p.handleOSCEsc(b)
	}
}

// --- Normal state ---

func (p *Parser) handleNormal(b byte) {
	switch {
	case b == 0x1b: // ESC
		p.state = stateEscape
	case b == 0x07: // BEL — ignore
	case b == 0x08: // BS
		p.grid.CursorLeft(1)
	case b == 0x09: // HT (tab)
		cur := p.grid.cursor.Position.Col
		nextTab := ((cur / 8) + 1) * 8
		maxCol := p.grid.dimensions.Cols
		if maxCol > 0 {
			maxCol--
		}
		if nextTab > maxCol {
			nextTab = maxCol
		}
		p.grid.cursor.Position.Col = nextTab
	case b == 0x0A: // LF
		p.linefeed()
	case b == 0x0D: // CR
		p.grid.cursor.Position.Col = 0
	case b >= 0x20: // printable ASCII and start of UTF-8
		p.printByte(b)
	}
	// Other C0 controls are ignored.
}

// printByte handles a printable byte. For simplicity this treats each byte >= 0x20
// as a single-byte character. True multi-byte UTF-8 is not decoded here; each
// byte in a multi-byte sequence will produce its own cell. This is acceptable for
// terminal emulation where the vast majority of output is ASCII.
func (p *Parser) printByte(b byte) {
	ch := rune(b)
	pos := p.grid.cursor.Position
	dims := p.grid.dimensions

	cell := p.grid.Cell(pos.Row, pos.Col)
	if cell != nil {
		cell.Character = ch
		cell.Attrs = p.grid.currentAttrs
		cell.Fg = p.grid.currentFg
		cell.Bg = p.grid.currentBg
	}

	// Advance cursor.
	p.grid.cursor.Position.Col++
	if p.grid.cursor.Position.Col >= dims.Cols {
		p.grid.cursor.Position.Col = 0
		nextRow := p.grid.cursor.Position.Row + 1
		maxRow := dims.Rows
		if maxRow > 0 {
			maxRow--
		}
		if nextRow > maxRow {
			// At bottom of scroll region — scroll up.
			p.grid.ScrollUp(1)
			nextRow = maxRow
		}
		p.grid.cursor.Position.Row = nextRow
		p.grid.SetLineWrapped(nextRow, true)
	}
}

// linefeed moves the cursor down one row. If at the bottom of the scroll region, scrolls up.
func (p *Parser) linefeed() {
	bottom := p.grid.scrollBottom()
	if p.grid.cursor.Position.Row >= bottom {
		p.grid.ScrollUp(1)
	} else {
		p.grid.cursor.Position.Row++
	}
	newRow := p.grid.cursor.Position.Row
	p.grid.SetLineWrapped(newRow, false)
}

// --- Escape state ---

func (p *Parser) handleEscape(b byte) {
	switch b {
	case '[': // CSI introducer
		p.csiReset()
		p.state = stateCSI
	case ']': // OSC introducer
		p.state = stateOSC
	case '7': // DECSC — save cursor
		p.grid.SaveCursor()
		p.state = stateNormal
	case '8': // DECRC — restore cursor
		p.grid.RestoreCursor()
		p.state = stateNormal
	case 'D': // IND — index (move cursor down, scroll if needed)
		p.linefeed()
		p.state = stateNormal
	case 'M': // RI — reverse index (move cursor up, scroll if needed)
		top := p.grid.scrollTop()
		if p.grid.cursor.Position.Row <= top {
			p.grid.ScrollDown(1)
		} else {
			p.grid.cursor.Position.Row--
		}
		p.state = stateNormal
	case 'c': // RIS — full reset
		p.grid.Clear()
		p.grid.cursor = DefaultCursor()
		p.grid.currentAttrs = core.CellAttributes{}
		p.grid.currentFg = core.DefaultColor()
		p.grid.currentBg = core.DefaultColor()
		p.state = stateNormal
	default:
		// Unknown ESC sequence — return to normal.
		p.state = stateNormal
	}
}

// --- CSI state ---

func (p *Parser) csiReset() {
	p.csiParams = p.csiParams[:0]
	p.csiCurrentParam = 0
	p.csiHasParam = false
	p.csiPrivateMarker = 0
}

func (p *Parser) handleCSI(b byte) {
	switch {
	case b == '?':
		p.csiPrivateMarker = '?'
		p.state = stateCSIParam
	case b >= '0' && b <= '9':
		p.csiHasParam = true
		p.csiCurrentParam = p.csiCurrentParam*10 + uint16(b-'0')
		p.state = stateCSIParam
	case b == ';':
		p.csiParams = append(p.csiParams, p.csiCurrentParam)
		p.csiCurrentParam = 0
		p.csiHasParam = false
		p.state = stateCSIParam
	case b >= 0x40 && b <= 0x7E: // final byte
		// Push last param if we were collecting digits.
		if p.csiHasParam || len(p.csiParams) > 0 {
			p.csiParams = append(p.csiParams, p.csiCurrentParam)
		}
		p.dispatchCSI(b)
		p.state = stateNormal
	default:
		// Intermediate bytes (0x20-0x2F) — currently ignored; stay in CSI.
		p.state = stateCSIParam
	}
}

// csiParam returns param at index, or defaultVal if not present.
func (p *Parser) csiParam(index int, defaultVal uint16) uint16 {
	if index < len(p.csiParams) {
		v := p.csiParams[index]
		if v == 0 && defaultVal > 0 {
			return defaultVal
		}
		return v
	}
	return defaultVal
}

func (p *Parser) dispatchCSI(final byte) {
	isPrivate := p.csiPrivateMarker == '?'

	switch final {
	case 'A': // CUU — cursor up
		p.grid.CursorUp(p.csiParam(0, 1))

	case 'B': // CUD — cursor down
		n := p.csiParam(0, 1)
		newRow := p.grid.cursor.Position.Row + n
		maxRow := p.grid.dimensions.Rows
		if maxRow > 0 {
			maxRow--
		}
		if newRow > maxRow {
			newRow = maxRow
		}
		p.grid.cursor.Position.Row = newRow

	case 'C': // CUF — cursor forward (right)
		p.grid.CursorRight(p.csiParam(0, 1))

	case 'D': // CUB — cursor backward (left)
		p.grid.CursorLeft(p.csiParam(0, 1))

	case 'H', 'f': // CUP / HVP — absolute cursor position (1-based)
		row := p.csiParam(0, 1)
		col := p.csiParam(1, 1)
		if row > 0 {
			row--
		}
		if col > 0 {
			col--
		}
		p.grid.MoveCursor(row, col)

	case 'J': // ED — erase in display
		p.grid.ClearScreen(ClearScreenMode(p.csiParam(0, 0)))

	case 'K': // EL — erase in line
		p.grid.ClearLine(p.grid.cursor.Position.Row, ClearLineMode(p.csiParam(0, 0)))

	case 'L': // IL — insert lines
		p.grid.InsertLines(p.csiParam(0, 1))

	case 'M': // DL — delete lines
		p.grid.DeleteLines(p.csiParam(0, 1))

	case '@': // ICH — insert characters
		p.grid.InsertChars(p.csiParam(0, 1))

	case 'P': // DCH — delete characters
		p.grid.DeleteChars(p.csiParam(0, 1))

	case 'S': // SU — scroll up
		p.grid.ScrollUp(p.csiParam(0, 1))

	case 'T': // SD — scroll down
		p.grid.ScrollDown(p.csiParam(0, 1))

	case 'r': // DECSTBM — set scroll region (1-based params)
		if len(p.csiParams) >= 2 {
			top := p.csiParams[0]
			bottom := p.csiParams[1]
			if top > 0 {
				top--
			}
			if bottom > 0 {
				bottom--
			}
			maxRow := p.grid.dimensions.Rows
			if maxRow > 0 {
				maxRow--
			}
			if bottom > maxRow {
				bottom = maxRow
			}
			if top < bottom {
				p.grid.SetScrollRegion(top, bottom)
			}
		} else {
			p.grid.ClearScrollRegion()
		}

	case 'm': // SGR — select graphic rendition
		p.processSGR()

	case 's': // SCP — save cursor position
		p.grid.SaveCursor()

	case 'u': // RCP — restore cursor position
		p.grid.RestoreCursor()

	case 'h': // SM — set mode
		if isPrivate {
			p.setPrivateMode(true)
		}

	case 'l': // RM — reset mode
		if isPrivate {
			p.setPrivateMode(false)
		}

	case 'c': // DA — device attributes — ignore

	case 'd': // VPA — vertical position absolute (1-based)
		row := p.csiParam(0, 1)
		if row > 0 {
			row--
		}
		maxRow := p.grid.dimensions.Rows
		if maxRow > 0 {
			maxRow--
		}
		if row > maxRow {
			row = maxRow
		}
		p.grid.cursor.Position.Row = row

	case 'G': // CHA — cursor character absolute (1-based column)
		col := p.csiParam(0, 1)
		if col > 0 {
			col--
		}
		maxCol := p.grid.dimensions.Cols
		if maxCol > 0 {
			maxCol--
		}
		if col > maxCol {
			col = maxCol
		}
		p.grid.cursor.Position.Col = col

	case 'X': // ECH — erase characters
		n := p.csiParam(0, 1)
		row := p.grid.cursor.Position.Row
		col := p.grid.cursor.Position.Col
		for i := uint16(0); i < n && col+i < p.grid.dimensions.Cols; i++ {
			if c := p.grid.Cell(row, col+i); c != nil {
				*c = core.DefaultCell()
			}
		}
	}
}

// setPrivateMode handles DEC private mode set/reset (CSI ? n h/l).
func (p *Parser) setPrivateMode(enable bool) {
	mode := p.csiParam(0, 0)
	switch mode {
	case 25: // DECTCEM — cursor visibility
		p.grid.cursor.Visible = enable
	case 1049, 47, 1047: // Alternate screen buffer
		if enable {
			p.grid.Clear()
		}
	}
	// Other private modes are ignored.
}

// --- SGR processing ---

func (p *Parser) processSGR() {
	// If no params, treat as reset (SGR 0).
	if len(p.csiParams) == 0 {
		p.grid.SetCurrentAttrs(core.CellAttributes{})
		p.grid.SetCurrentFg(core.DefaultColor())
		p.grid.SetCurrentBg(core.DefaultColor())
		return
	}

	i := 0
	for i < len(p.csiParams) {
		code := p.csiParams[i]
		i++

		switch {
		case code == 0:
			p.grid.SetCurrentAttrs(core.CellAttributes{})
			p.grid.SetCurrentFg(core.DefaultColor())
			p.grid.SetCurrentBg(core.DefaultColor())

		// Bold, dim, italic, underline, blink, reverse, hidden, strikethrough
		case code == 1:
			a := p.grid.CurrentAttrs()
			a.Bold = true
			p.grid.SetCurrentAttrs(a)
		case code == 2:
			a := p.grid.CurrentAttrs()
			a.Dim = true
			p.grid.SetCurrentAttrs(a)
		case code == 3:
			a := p.grid.CurrentAttrs()
			a.Italic = true
			p.grid.SetCurrentAttrs(a)
		case code == 4:
			a := p.grid.CurrentAttrs()
			a.Underline = true
			p.grid.SetCurrentAttrs(a)
		case code == 5:
			a := p.grid.CurrentAttrs()
			a.Blink = true
			p.grid.SetCurrentAttrs(a)
		case code == 7:
			a := p.grid.CurrentAttrs()
			a.Reverse = true
			p.grid.SetCurrentAttrs(a)
		case code == 8:
			a := p.grid.CurrentAttrs()
			a.Hidden = true
			p.grid.SetCurrentAttrs(a)
		case code == 9:
			a := p.grid.CurrentAttrs()
			a.Strikethrough = true
			p.grid.SetCurrentAttrs(a)

		// Reset attributes
		case code == 22:
			a := p.grid.CurrentAttrs()
			a.Bold = false
			a.Dim = false
			p.grid.SetCurrentAttrs(a)
		case code == 23:
			a := p.grid.CurrentAttrs()
			a.Italic = false
			p.grid.SetCurrentAttrs(a)
		case code == 24:
			a := p.grid.CurrentAttrs()
			a.Underline = false
			p.grid.SetCurrentAttrs(a)
		case code == 25:
			a := p.grid.CurrentAttrs()
			a.Blink = false
			p.grid.SetCurrentAttrs(a)
		case code == 27:
			a := p.grid.CurrentAttrs()
			a.Reverse = false
			p.grid.SetCurrentAttrs(a)
		case code == 28:
			a := p.grid.CurrentAttrs()
			a.Hidden = false
			p.grid.SetCurrentAttrs(a)
		case code == 29:
			a := p.grid.CurrentAttrs()
			a.Strikethrough = false
			p.grid.SetCurrentAttrs(a)

		// Foreground ANSI colors 30-37
		case code >= 30 && code <= 37:
			p.grid.SetCurrentFg(ansiColor(code - 30))
		case code == 39:
			p.grid.SetCurrentFg(core.DefaultColor())

		// Background ANSI colors 40-47
		case code >= 40 && code <= 47:
			p.grid.SetCurrentBg(ansiColor(code - 40))
		case code == 49:
			p.grid.SetCurrentBg(core.DefaultColor())

		// Bright foreground 90-97
		case code >= 90 && code <= 97:
			p.grid.SetCurrentFg(brightColor(code - 90))

		// Bright background 100-107
		case code >= 100 && code <= 107:
			p.grid.SetCurrentBg(brightColor(code - 100))

		// 256-color / RGB foreground (38;5;n or 38;2;r;g;b)
		case code == 38:
			if i < len(p.csiParams) {
				sub := p.csiParams[i]
				i++
				if sub == 5 && i < len(p.csiParams) {
					idx := uint8(p.csiParams[i])
					i++
					p.grid.SetCurrentFg(core.IndexedColor(idx))
				} else if sub == 2 && i+2 < len(p.csiParams) {
					r := uint8(p.csiParams[i])
					g := uint8(p.csiParams[i+1])
					b := uint8(p.csiParams[i+2])
					i += 3
					p.grid.SetCurrentFg(core.RGBColor(r, g, b))
				}
			}

		// 256-color / RGB background (48;5;n or 48;2;r;g;b)
		case code == 48:
			if i < len(p.csiParams) {
				sub := p.csiParams[i]
				i++
				if sub == 5 && i < len(p.csiParams) {
					idx := uint8(p.csiParams[i])
					i++
					p.grid.SetCurrentBg(core.IndexedColor(idx))
				} else if sub == 2 && i+2 < len(p.csiParams) {
					r := uint8(p.csiParams[i])
					g := uint8(p.csiParams[i+1])
					b := uint8(p.csiParams[i+2])
					i += 3
					p.grid.SetCurrentBg(core.RGBColor(r, g, b))
				}
			}
		}
	}
}

// ansiColor maps offset 0-7 to an ANSI Color.
func ansiColor(offset uint16) core.Color {
	switch offset {
	case 0:
		return core.ANSIColor(core.ColorBlack)
	case 1:
		return core.ANSIColor(core.ColorRed)
	case 2:
		return core.ANSIColor(core.ColorGreen)
	case 3:
		return core.ANSIColor(core.ColorYellow)
	case 4:
		return core.ANSIColor(core.ColorBlue)
	case 5:
		return core.ANSIColor(core.ColorMagenta)
	case 6:
		return core.ANSIColor(core.ColorCyan)
	case 7:
		return core.ANSIColor(core.ColorWhite)
	default:
		return core.DefaultColor()
	}
}

// brightColor maps offset 0-7 to a bright ANSI Color.
func brightColor(offset uint16) core.Color {
	switch offset {
	case 0:
		return core.ANSIColor(core.ColorBrightBlack)
	case 1:
		return core.ANSIColor(core.ColorBrightRed)
	case 2:
		return core.ANSIColor(core.ColorBrightGreen)
	case 3:
		return core.ANSIColor(core.ColorBrightYellow)
	case 4:
		return core.ANSIColor(core.ColorBrightBlue)
	case 5:
		return core.ANSIColor(core.ColorBrightMagenta)
	case 6:
		return core.ANSIColor(core.ColorBrightCyan)
	case 7:
		return core.ANSIColor(core.ColorBrightWhite)
	default:
		return core.DefaultColor()
	}
}

// --- OSC state ---

func (p *Parser) handleOSC(b byte) {
	switch b {
	case 0x07: // BEL — terminates OSC
		p.state = stateNormal
	case 0x1b: // ESC — could be start of ST (ESC \)
		p.state = stateOSCEsc
	}
	// All other bytes within OSC are consumed and ignored.
}

func (p *Parser) handleOSCEsc(b byte) {
	if b == '\\' {
		// ST (String Terminator) — end OSC.
		p.state = stateNormal
	} else {
		// Not a valid ST. Return to normal.
		p.state = stateNormal
	}
}
