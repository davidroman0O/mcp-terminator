package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// boxCharSet defines a set of box-drawing characters for a single style.
type boxCharSet struct {
	topLeft     rune
	topRight    rune
	bottomLeft  rune
	bottomRight rune
	horizontal  rune
	vertical    rune
}

// BorderDetector finds box-drawing regions using single-line, double-line,
// heavy, rounded, and ASCII box characters.
type BorderDetector struct {
	boxSets []boxCharSet
}

// NewBorderDetector creates a border detector with all known box styles.
func NewBorderDetector() *BorderDetector {
	return &BorderDetector{
		boxSets: []boxCharSet{
			// Light box
			{topLeft: '\u250c', topRight: '\u2510', bottomLeft: '\u2514', bottomRight: '\u2518', horizontal: '\u2500', vertical: '\u2502'}, // ┌─┐│└┘
			// Heavy box
			{topLeft: '\u250f', topRight: '\u2513', bottomLeft: '\u2517', bottomRight: '\u251b', horizontal: '\u2501', vertical: '\u2503'}, // ┏━┓┃┗┛
			// Double box
			{topLeft: '\u2554', topRight: '\u2557', bottomLeft: '\u255a', bottomRight: '\u255d', horizontal: '\u2550', vertical: '\u2551'}, // ╔═╗║╚╝
			// Rounded
			{topLeft: '\u256d', topRight: '\u256e', bottomLeft: '\u2570', bottomRight: '\u256f', horizontal: '\u2500', vertical: '\u2502'}, // ╭─╮│╰╯
			// ASCII
			{topLeft: '+', topRight: '+', bottomLeft: '+', bottomRight: '+', horizontal: '-', vertical: '|'},
		},
	}
}

func (d *BorderDetector) Name() string     { return "border" }
func (d *BorderDetector) Priority() int    { return 100 }

func (d *BorderDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var borders []DetectedElement
	dims := grid.Dimensions()

	for row := 0; row < int(dims.Rows); row++ {
		for col := 0; col < int(dims.Cols); col++ {
			// Skip already-claimed single-cell regions.
			pointBounds := core.NewBounds(uint16(row), uint16(col), 1, 1)
			if ctx.IsRegionClaimed(pointBounds) {
				continue
			}

			cell, ok := grid.Cell(row, col)
			if !ok {
				continue
			}

			for i := range d.boxSets {
				if cell.Character == d.boxSets[i].topLeft {
					if elem := d.traceBorder(grid, row, col, &d.boxSets[i]); elem != nil {
						borders = append(borders, *elem)
					}
				}
			}
		}
	}

	return d.filterContainedBorders(borders)
}

// traceBorder traces a complete border starting from a top-left corner.
func (d *BorderDetector) traceBorder(grid GridReader, startRow, startCol int, bs *boxCharSet) *DetectedElement {
	dims := grid.Dimensions()

	// Find top-right corner (allow any chars in between for titles).
	width := 0
	for col := startCol + 1; col < int(dims.Cols); col++ {
		cell, ok := grid.Cell(startRow, col)
		if !ok {
			break
		}
		if cell.Character == bs.topRight {
			width = col - startCol + 1
			break
		}
	}
	if width == 0 {
		return nil
	}

	// Find bottom-left corner.
	height := 0
	for row := startRow + 1; row < int(dims.Rows); row++ {
		cell, ok := grid.Cell(row, startCol)
		if !ok {
			break
		}
		if cell.Character == bs.bottomLeft {
			height = row - startRow + 1
			break
		} else if cell.Character != bs.vertical && cell.Character != ' ' {
			return nil
		}
	}
	if height == 0 {
		return nil
	}

	// Verify bottom-right corner.
	bottomRow := startRow + height - 1
	rightCol := startCol + width - 1
	cell, ok := grid.Cell(bottomRow, rightCol)
	if !ok || cell.Character != bs.bottomRight {
		return nil
	}

	// Extract title from top border.
	title := d.extractTitle(grid, startRow, startCol, width, bs)

	bounds := core.NewBounds(uint16(startRow), uint16(startCol), uint16(width), uint16(height))
	refID := fmt.Sprintf("border_%d_%d", startRow, startCol)

	var titlePtr *string
	if title != "" {
		titlePtr = &title
	}

	return &DetectedElement{
		Element:    core.NewBorderElement(refID, bounds, titlePtr, nil),
		Bounds:     bounds,
		Confidence: ConfidenceHigh,
	}
}

// extractTitle extracts title text embedded in the top border row.
func (d *BorderDetector) extractTitle(grid GridReader, row, col, width int, bs *boxCharSet) string {
	var title strings.Builder
	inTitle := false

	for c := col + 1; c < col+width-1; c++ {
		cell, ok := grid.Cell(row, c)
		if !ok {
			break
		}
		ch := cell.Character
		if ch == bs.horizontal {
			if inTitle && strings.TrimSpace(title.String()) != "" {
				break
			}
		} else if ch != ' ' || inTitle {
			inTitle = true
			title.WriteRune(ch)
		}
	}

	return strings.TrimSpace(title.String())
}

// filterContainedBorders removes borders that are completely contained
// within larger borders, keeping only outermost ones.
func (d *BorderDetector) filterContainedBorders(borders []DetectedElement) []DetectedElement {
	// Sort by area descending.
	for i := 0; i < len(borders); i++ {
		for j := i + 1; j < len(borders); j++ {
			areaI := int(borders[i].Bounds.Width) * int(borders[i].Bounds.Height)
			areaJ := int(borders[j].Bounds.Width) * int(borders[j].Bounds.Height)
			if areaJ > areaI {
				borders[i], borders[j] = borders[j], borders[i]
			}
		}
	}

	var filtered []DetectedElement
	for _, border := range borders {
		contained := false
		for _, outer := range filtered {
			if border.Bounds.Row >= outer.Bounds.Row &&
				border.Bounds.Col >= outer.Bounds.Col &&
				(border.Bounds.Row+border.Bounds.Height) <= (outer.Bounds.Row+outer.Bounds.Height) &&
				(border.Bounds.Col+border.Bounds.Width) <= (outer.Bounds.Col+outer.Bounds.Width) {
				contained = true
				break
			}
		}
		if !contained {
			filtered = append(filtered, border)
		}
	}
	return filtered
}
