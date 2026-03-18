package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// MenuDetector detects vertical menus with selection indicators (prefix
// markers like >, *, arrow chars) or reverse-video / background-color
// highlighting. Also supports cursor-based selection as a fallback.
type MenuDetector struct {
	minItems       int
	prefixMarkers  []rune
}

// NewMenuDetector creates a new menu detector.
func NewMenuDetector() *MenuDetector {
	return &MenuDetector{
		minItems:      2,
		prefixMarkers: []rune{'>', '\u2192', '\u25b6', '\u2022', '*', '\u25ba'}, // > → ▶ • * ►
	}
}

func (d *MenuDetector) Name() string     { return "menu" }
func (d *MenuDetector) Priority() int    { return 80 }

func (d *MenuDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement

	regions := d.findMenuRegions(grid)
	for _, region := range regions {
		if ctx.IsRegionClaimed(region) {
			continue
		}
		if elem := d.detectMenuInRegion(grid, region, ctx.Cursor); elem != nil {
			results = append(results, *elem)
		}
	}
	return results
}

// detectMenuInRegion tries multiple strategies in confidence order.
func (d *MenuDetector) detectMenuInRegion(grid GridReader, region core.Bounds, cursor core.Position) *DetectedElement {
	type result struct {
		selectedIdx int
		items       []core.MenuItem
	}

	var res *result

	// Strategy 1: Reverse video.
	if r := d.detectByReverseVideo(grid, region); r != nil {
		res = &result{r.selectedIdx, r.items}
	}
	// Strategy 1b: Background color.
	if res == nil {
		if r := d.detectByBackgroundColor(grid, region); r != nil {
			res = &result{r.selectedIdx, r.items}
		}
	}
	// Strategy 2: Prefix marker.
	if res == nil {
		if r := d.detectByPrefixMarker(grid, region); r != nil {
			res = &result{r.selectedIdx, r.items}
		}
	}
	// Strategy 3: Cursor position.
	if res == nil {
		if r := d.detectByCursor(grid, region, cursor); r != nil {
			res = &result{r.selectedIdx, r.items}
		}
	}

	if res == nil {
		return nil
	}

	refID := fmt.Sprintf("menu_%d_%d", region.Row, region.Col)
	return &DetectedElement{
		Element:    core.NewMenuElement(refID, region, res.items, res.selectedIdx),
		Bounds:     region,
		Confidence: ConfidenceHigh,
	}
}

type menuResult struct {
	selectedIdx int
	items       []core.MenuItem
}

func (d *MenuDetector) detectByReverseVideo(grid GridReader, region core.Bounds) *menuResult {
	var items []core.MenuItem
	selectedIdx := -1

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		text := extractRowTextRegion(grid, row, int(region.Col), int(region.Width))
		if strings.TrimSpace(text) == "" {
			continue
		}

		hasReverse := d.rowHasAttribute(grid, row, region, func(a core.CellAttributes) bool { return a.Reverse })
		if hasReverse {
			selectedIdx = len(items)
		}
		items = append(items, core.MenuItem{
			Text:     strings.TrimSpace(text),
			Selected: hasReverse,
		})
	}

	if len(items) >= d.minItems && selectedIdx >= 0 {
		return &menuResult{selectedIdx, items}
	}
	return nil
}

func (d *MenuDetector) detectByBackgroundColor(grid GridReader, region core.Bounds) *menuResult {
	var items []core.MenuItem
	selectedIdx := -1

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		text := extractRowTextRegion(grid, row, int(region.Col), int(region.Width))
		if strings.TrimSpace(text) == "" {
			continue
		}

		hasBg := d.rowHasBackgroundColor(grid, row, region)
		if hasBg {
			selectedIdx = len(items)
		}
		items = append(items, core.MenuItem{
			Text:     strings.TrimSpace(text),
			Selected: hasBg,
		})
	}

	if len(items) >= d.minItems && selectedIdx >= 0 {
		return &menuResult{selectedIdx, items}
	}
	return nil
}

func (d *MenuDetector) detectByPrefixMarker(grid GridReader, region core.Bounds) *menuResult {
	var items []core.MenuItem
	selectedIdx := -1

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		text := extractRowTextRegion(grid, row, int(region.Col), int(region.Width))
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}

		runes := []rune(trimmed)
		isSelected := false
		if len(runes) > 0 {
			for _, m := range d.prefixMarkers {
				if runes[0] == m {
					isSelected = true
					break
				}
			}
		}

		itemText := trimmed
		if isSelected {
			itemText = strings.TrimSpace(string(runes[1:]))
		}

		if isSelected {
			selectedIdx = len(items)
		}
		items = append(items, core.MenuItem{
			Text:     itemText,
			Selected: isSelected,
		})
	}

	if len(items) >= d.minItems && selectedIdx >= 0 {
		return &menuResult{selectedIdx, items}
	}
	return nil
}

func (d *MenuDetector) detectByCursor(grid GridReader, region core.Bounds, cursor core.Position) *menuResult {
	if !region.Contains(cursor) {
		return nil
	}

	var items []core.MenuItem
	cursorRowOff := int(cursor.Row) - int(region.Row)

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		text := extractRowTextRegion(grid, row, int(region.Col), int(region.Width))
		if strings.TrimSpace(text) == "" {
			continue
		}
		items = append(items, core.MenuItem{
			Text:     strings.TrimSpace(text),
			Selected: rowOff == cursorRowOff,
		})
	}

	if len(items) >= d.minItems {
		selected := 0
		for i, it := range items {
			if it.Selected {
				selected = i
				break
			}
		}
		return &menuResult{selected, items}
	}
	return nil
}

// rowHasAttribute checks if >50% of non-space cells on a row satisfy the predicate.
func (d *MenuDetector) rowHasAttribute(grid GridReader, row int, region core.Bounds, pred func(core.CellAttributes) bool) bool {
	matchCount := 0
	nonSpaceCount := 0
	for colOff := 0; colOff < int(region.Width); colOff++ {
		col := int(region.Col) + colOff
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' {
				nonSpaceCount++
				if pred(cell.Attrs) {
					matchCount++
				}
			}
		}
	}
	return nonSpaceCount > 0 && matchCount > nonSpaceCount/2
}

// rowHasBackgroundColor checks if >50% of non-space cells have a non-default bg.
func (d *MenuDetector) rowHasBackgroundColor(grid GridReader, row int, region core.Bounds) bool {
	bgCount := 0
	nonSpaceCount := 0
	for colOff := 0; colOff < int(region.Width); colOff++ {
		col := int(region.Col) + colOff
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' {
				nonSpaceCount++
				if cell.Bg.Type != core.ColorDefault {
					bgCount++
				}
			}
		}
	}
	return nonSpaceCount > 0 && bgCount > nonSpaceCount/2
}

// findMenuRegions finds contiguous blocks of non-empty rows.
func (d *MenuDetector) findMenuRegions(grid GridReader) []core.Bounds {
	dims := grid.Dimensions()
	var regions []core.Bounds
	regionStart := -1

	for row := 0; row < int(dims.Rows); row++ {
		text := extractRowText(grid, row)
		hasContent := strings.TrimSpace(text) != ""

		if hasContent {
			if regionStart < 0 {
				regionStart = row
			}
		} else if regionStart >= 0 {
			height := row - regionStart
			if height >= d.minItems {
				regions = append(regions, core.NewBounds(uint16(regionStart), 0, dims.Cols, uint16(height)))
			}
			regionStart = -1
		}
	}

	if regionStart >= 0 {
		height := int(dims.Rows) - regionStart
		if height >= d.minItems {
			regions = append(regions, core.NewBounds(uint16(regionStart), 0, dims.Cols, uint16(height)))
		}
	}
	return regions
}
