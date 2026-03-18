// Package core provides fundamental types for the Terminal MCP Server.
//
// This is Layer 0 in the architecture - all other packages depend on this one,
// but this package has no dependencies on other mcp-terminator packages.
//
// It provides:
//   - Geometry types (Position, Bounds, Dimensions)
//   - Cell and color types for terminal grid
//   - Element types for Terminal State Tree (TST)
//   - Key types for input handling
//   - Session types
//   - Error types
package core

import (
	"encoding/json"
	"fmt"
)

// --- Geometry Types ---

// Position represents a location in the terminal grid (row, column).
type Position struct {
	Row uint16 `json:"row"`
	Col uint16 `json:"col"`
}

// NewPosition creates a new position.
func NewPosition(row, col uint16) Position {
	return Position{Row: row, Col: col}
}

// Origin returns the origin position (0, 0).
func Origin() Position {
	return Position{Row: 0, Col: 0}
}

// String implements fmt.Stringer.
func (p Position) String() string {
	return fmt.Sprintf("(%d, %d)", p.Row, p.Col)
}

// Dimensions represents the size of a terminal or region.
type Dimensions struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// NewDimensions creates new dimensions.
func NewDimensions(rows, cols uint16) Dimensions {
	return Dimensions{Rows: rows, Cols: cols}
}

// DefaultDimensions returns the standard 24x80 terminal dimensions.
func DefaultDimensions() Dimensions {
	return Dimensions{Rows: 24, Cols: 80}
}

// CellCount returns the total number of cells (rows * cols).
func (d Dimensions) CellCount() int {
	return int(d.Rows) * int(d.Cols)
}

// String implements fmt.Stringer.
func (d Dimensions) String() string {
	return fmt.Sprintf("%dx%d", d.Rows, d.Cols)
}

// Bounds represents a bounding box for a terminal region.
type Bounds struct {
	Row    uint16 `json:"row"`
	Col    uint16 `json:"col"`
	Width  uint16 `json:"width"`
	Height uint16 `json:"height"`
}

// NewBounds creates new bounds.
func NewBounds(row, col, width, height uint16) Bounds {
	return Bounds{Row: row, Col: col, Width: width, Height: height}
}

// Contains reports whether the position is within these bounds.
func (b Bounds) Contains(pos Position) bool {
	return pos.Row >= b.Row &&
		pos.Row < b.Row+b.Height &&
		pos.Col >= b.Col &&
		pos.Col < b.Col+b.Width
}

// Intersects reports whether these bounds overlap with other.
func (b Bounds) Intersects(other Bounds) bool {
	return !(b.Row+b.Height <= other.Row ||
		other.Row+other.Height <= b.Row ||
		b.Col+b.Width <= other.Col ||
		other.Col+other.Width <= b.Col)
}

// String implements fmt.Stringer.
func (b Bounds) String() string {
	return fmt.Sprintf("Bounds{row:%d, col:%d, %dx%d}", b.Row, b.Col, b.Width, b.Height)
}

// --- Color Types ---

// ColorType identifies which variant of Color is active.
type ColorType string

const (
	ColorDefault       ColorType = "default"
	ColorBlack         ColorType = "black"
	ColorRed           ColorType = "red"
	ColorGreen         ColorType = "green"
	ColorYellow        ColorType = "yellow"
	ColorBlue          ColorType = "blue"
	ColorMagenta       ColorType = "magenta"
	ColorCyan          ColorType = "cyan"
	ColorWhite         ColorType = "white"
	ColorBrightBlack   ColorType = "bright_black"
	ColorBrightRed     ColorType = "bright_red"
	ColorBrightGreen   ColorType = "bright_green"
	ColorBrightYellow  ColorType = "bright_yellow"
	ColorBrightBlue    ColorType = "bright_blue"
	ColorBrightMagenta ColorType = "bright_magenta"
	ColorBrightCyan    ColorType = "bright_cyan"
	ColorBrightWhite   ColorType = "bright_white"
	ColorIndexed       ColorType = "indexed"
	ColorRGB           ColorType = "rgb"
)

// Color represents a terminal color supporting ANSI, 256-color palette, and true RGB.
type Color struct {
	Type ColorType `json:"type"`

	// Index is used when Type == ColorIndexed (0-255).
	Index *uint8 `json:"index,omitempty"`

	// R, G, B are used when Type == ColorRGB.
	R *uint8 `json:"r,omitempty"`
	G *uint8 `json:"g,omitempty"`
	B *uint8 `json:"b,omitempty"`
}

// DefaultColor returns the default terminal color.
func DefaultColor() Color {
	return Color{Type: ColorDefault}
}

// ANSIColor returns a standard ANSI color.
func ANSIColor(t ColorType) Color {
	return Color{Type: t}
}

// IndexedColor returns a 256-color palette color.
func IndexedColor(index uint8) Color {
	return Color{Type: ColorIndexed, Index: &index}
}

// RGBColor returns a true-color RGB value.
func RGBColor(r, g, b uint8) Color {
	return Color{Type: ColorRGB, R: &r, G: &g, B: &b}
}

// String implements fmt.Stringer.
func (c Color) String() string {
	switch c.Type {
	case ColorIndexed:
		if c.Index != nil {
			return fmt.Sprintf("Indexed(%d)", *c.Index)
		}
		return "Indexed(?)"
	case ColorRGB:
		r, g, b := uint8(0), uint8(0), uint8(0)
		if c.R != nil {
			r = *c.R
		}
		if c.G != nil {
			g = *c.G
		}
		if c.B != nil {
			b = *c.B
		}
		return fmt.Sprintf("RGB(%d,%d,%d)", r, g, b)
	default:
		return string(c.Type)
	}
}

// --- Cell Types ---

// CellAttributes represents text formatting attributes for a terminal cell.
type CellAttributes struct {
	Bold          bool `json:"bold"`
	Dim           bool `json:"dim"`
	Italic        bool `json:"italic"`
	Underline     bool `json:"underline"`
	Blink         bool `json:"blink"`
	Reverse       bool `json:"reverse"`
	Hidden        bool `json:"hidden"`
	Strikethrough bool `json:"strikethrough"`
}

// IsDefault reports whether all attributes are at their zero values.
func (a CellAttributes) IsDefault() bool {
	return a == CellAttributes{}
}

// WithBold returns a copy with Bold enabled.
func (a CellAttributes) WithBold() CellAttributes {
	a.Bold = true
	return a
}

// WithReverse returns a copy with Reverse enabled.
func (a CellAttributes) WithReverse() CellAttributes {
	a.Reverse = true
	return a
}

// WithUnderline returns a copy with Underline enabled.
func (a CellAttributes) WithUnderline() CellAttributes {
	a.Underline = true
	return a
}

// WithItalic returns a copy with Italic enabled.
func (a CellAttributes) WithItalic() CellAttributes {
	a.Italic = true
	return a
}

// String implements fmt.Stringer.
func (a CellAttributes) String() string {
	attrs := ""
	if a.Bold {
		attrs += "B"
	}
	if a.Dim {
		attrs += "D"
	}
	if a.Italic {
		attrs += "I"
	}
	if a.Underline {
		attrs += "U"
	}
	if a.Blink {
		attrs += "K"
	}
	if a.Reverse {
		attrs += "R"
	}
	if a.Hidden {
		attrs += "H"
	}
	if a.Strikethrough {
		attrs += "S"
	}
	if attrs == "" {
		return "none"
	}
	return attrs
}

// Cell represents a single character cell in the terminal grid.
type Cell struct {
	Character rune           `json:"character"`
	Fg        Color          `json:"fg"`
	Bg        Color          `json:"bg"`
	Attrs     CellAttributes `json:"attrs"`
}

// NewCell creates a cell with the given character and default styling.
func NewCell(ch rune) Cell {
	return Cell{
		Character: ch,
		Fg:        DefaultColor(),
		Bg:        DefaultColor(),
	}
}

// DefaultCell returns an empty cell (space with default colors and attributes).
func DefaultCell() Cell {
	return NewCell(' ')
}

// CellWithFg creates a cell with the given character and foreground color.
func CellWithFg(ch rune, fg Color) Cell {
	return Cell{
		Character: ch,
		Fg:        fg,
		Bg:        DefaultColor(),
	}
}

// IsEmpty reports whether the cell is a space with default attributes.
func (c Cell) IsEmpty() bool {
	return c.Character == ' ' && c.Attrs.IsDefault()
}

// IsWhitespace reports whether the cell character is whitespace.
func (c Cell) IsWhitespace() bool {
	switch c.Character {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

// String implements fmt.Stringer.
func (c Cell) String() string {
	return fmt.Sprintf("Cell{%q fg:%s bg:%s attrs:%s}", c.Character, c.Fg, c.Bg, c.Attrs)
}

// MarshalJSON implements custom JSON marshalling for Cell to encode character as a string.
func (c Cell) MarshalJSON() ([]byte, error) {
	type cellJSON struct {
		Character string         `json:"character"`
		Fg        Color          `json:"fg"`
		Bg        Color          `json:"bg"`
		Attrs     CellAttributes `json:"attrs"`
	}
	return json.Marshal(cellJSON{
		Character: string(c.Character),
		Fg:        c.Fg,
		Bg:        c.Bg,
		Attrs:     c.Attrs,
	})
}

// UnmarshalJSON implements custom JSON unmarshalling for Cell.
func (c *Cell) UnmarshalJSON(data []byte) error {
	type cellJSON struct {
		Character string         `json:"character"`
		Fg        Color          `json:"fg"`
		Bg        Color          `json:"bg"`
		Attrs     CellAttributes `json:"attrs"`
	}
	var raw cellJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Character) > 0 {
		runes := []rune(raw.Character)
		c.Character = runes[0]
	} else {
		c.Character = ' '
	}
	c.Fg = raw.Fg
	c.Bg = raw.Bg
	c.Attrs = raw.Attrs
	return nil
}
