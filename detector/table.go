package detector

import (
	"fmt"
	"strings"

	"github.com/davidroman0O/mcp-terminator/core"
)

// TableDetector detects data tables by finding column alignment patterns
// (consistent whitespace gaps), header rows (bold, colored, or followed
// by separator lines), and extracting structured row data.
type TableDetector struct {
	minColumns int
	minRows    int
}

// NewTableDetector creates a new table detector.
func NewTableDetector() *TableDetector {
	return &TableDetector{
		minColumns: 2,
		minRows:    2,
	}
}

func (d *TableDetector) Name() string     { return "table" }
func (d *TableDetector) Priority() int    { return 80 }

func (d *TableDetector) Detect(grid GridReader, ctx *DetectionContext) []DetectedElement {
	var results []DetectedElement

	regions := d.findTableRegions(grid)
	for _, region := range regions {
		if ctx.IsRegionClaimed(region) {
			continue
		}
		if elem := d.detectTableInRegion(grid, region); elem != nil {
			results = append(results, *elem)
		}
	}
	return results
}

func (d *TableDetector) detectTableInRegion(grid GridReader, region core.Bounds) *DetectedElement {
	columns := d.detectColumns(grid, region)
	if len(columns) < d.minColumns {
		return nil
	}

	headerRowIdx := d.detectHeaderRow(grid, region)

	headers, rows := d.parseTableRows(grid, region, columns, headerRowIdx)

	totalRows := len(rows)
	if len(headers) > 0 {
		totalRows++
	}
	if totalRows < d.minRows {
		return nil
	}

	refID := fmt.Sprintf("table_%d_%d", region.Row, region.Col)
	return &DetectedElement{
		Element:    core.NewTableElement(refID, region, headers, rows),
		Bounds:     region,
		Confidence: ConfidenceMedium,
	}
}

// detectColumns builds a column occupancy map and finds separator gaps.
func (d *TableDetector) detectColumns(grid GridReader, region core.Bounds) []int {
	colOccupancy := make([]int, region.Width)

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		for colOff := 0; colOff < int(region.Width); colOff++ {
			col := int(region.Col) + colOff
			if cell, ok := grid.Cell(row, col); ok {
				if cell.Character != ' ' {
					colOccupancy[colOff]++
				}
			}
		}
	}

	// Find consistently empty columns.
	threshold := int(region.Height) / 2
	var separators []int
	for idx, count := range colOccupancy {
		if count < threshold {
			separators = append(separators, idx)
		}
	}

	return d.separatorsToColumns(separators, int(region.Width))
}

// separatorsToColumns groups consecutive separators and derives column start positions.
func (d *TableDetector) separatorsToColumns(separators []int, width int) []int {
	if len(separators) == 0 {
		return nil
	}

	columns := []int{0}

	i := 0
	for i < len(separators) {
		start := separators[i]
		end := start

		for i+1 < len(separators) && separators[i+1] == end+1 {
			i++
			end = separators[i]
		}

		// Skip trailing whitespace (>80% of width).
		if end >= width*8/10 {
			break
		}

		columns = append(columns, (start+end)/2+1)
		i++
	}

	if len(columns) >= d.minColumns {
		return columns
	}
	return nil
}

// detectHeaderRow uses heuristics: bold first row, separator after first row,
// or distinct background on first row.
func (d *TableDetector) detectHeaderRow(grid GridReader, region core.Bounds) int {
	// Strategy 1: Bold first row.
	if d.rowIsBold(grid, int(region.Row), region) {
		return 0
	}
	// Strategy 2: Separator line after first row.
	if region.Height > 1 {
		secondRow := int(region.Row) + 1
		if d.isSeparatorLine(grid, secondRow, region) {
			return 0
		}
	}
	// Strategy 3: First row has different background.
	if d.rowHasDifferentBg(grid, int(region.Row), region) {
		return 0
	}
	return -1 // no header detected
}

func (d *TableDetector) rowIsBold(grid GridReader, row int, region core.Bounds) bool {
	boldCount := 0
	nonSpaceCount := 0
	for colOff := 0; colOff < int(region.Width); colOff++ {
		col := int(region.Col) + colOff
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' {
				nonSpaceCount++
				if cell.Attrs.Bold {
					boldCount++
				}
			}
		}
	}
	return nonSpaceCount > 0 && boldCount > nonSpaceCount/2
}

func (d *TableDetector) rowHasDifferentBg(grid GridReader, row int, region core.Bounds) bool {
	for colOff := 0; colOff < int(region.Width); colOff++ {
		col := int(region.Col) + colOff
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' && cell.Bg.Type != core.ColorDefault {
				return true
			}
		}
	}
	return false
}

func (d *TableDetector) isSeparatorLine(grid GridReader, row int, region core.Bounds) bool {
	separatorChars := map[rune]bool{
		'\u2500': true, '\u2501': true, '-': true, '=': true,
		'\u2550': true, '|': true, '\u2502': true,
	}
	sepCount := 0
	totalNonSpace := 0

	for col := int(region.Col); col < int(region.Col)+int(region.Width); col++ {
		if cell, ok := grid.Cell(row, col); ok {
			if cell.Character != ' ' {
				totalNonSpace++
				if separatorChars[cell.Character] {
					sepCount++
				}
			}
		}
	}
	return totalNonSpace > 0 && float64(sepCount)/float64(totalNonSpace) > 0.8
}

// parseTableRows extracts headers and data rows using column boundaries.
func (d *TableDetector) parseTableRows(grid GridReader, region core.Bounds, columns []int, headerRowIdx int) ([]string, [][]string) {
	type indexedRow struct {
		rowOff int
		cells  []string
	}
	var allRows []indexedRow

	for rowOff := 0; rowOff < int(region.Height); rowOff++ {
		row := int(region.Row) + rowOff
		if d.isSeparatorLine(grid, row, region) {
			continue
		}

		var cells []string
		for i := 0; i < len(columns); i++ {
			colStart := columns[i]
			colEnd := int(region.Width)
			if i+1 < len(columns) {
				colEnd = columns[i+1]
			}

			content := d.extractCell(grid, row, region, colStart, colEnd)

			// Skip empty trailing column.
			if i == len(columns)-1 && content == "" && len(cells) > 0 {
				hasContent := false
				for _, c := range cells {
					if c != "" {
						hasContent = true
						break
					}
				}
				if hasContent {
					continue
				}
			}
			cells = append(cells, content)
		}
		allRows = append(allRows, indexedRow{rowOff, cells})
	}

	var headers []string
	var dataRows [][]string

	headerIdx := headerRowIdx
	if headerIdx < 0 {
		headerIdx = 0
	}

	for _, ir := range allRows {
		if ir.rowOff == headerIdx {
			headers = ir.cells
		} else {
			dataRows = append(dataRows, ir.cells)
		}
	}

	return headers, dataRows
}

func (d *TableDetector) extractCell(grid GridReader, row int, region core.Bounds, colStart, colEnd int) string {
	start := int(region.Col) + colStart
	width := colEnd - colStart
	return strings.TrimSpace(extractRowTextRegion(grid, row, start, width))
}

func (d *TableDetector) findTableRegions(grid GridReader) []core.Bounds {
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
			if height >= d.minRows {
				regions = append(regions, core.NewBounds(uint16(regionStart), 0, dims.Cols, uint16(height)))
			}
			regionStart = -1
		}
	}

	if regionStart >= 0 {
		height := int(dims.Rows) - regionStart
		if height >= d.minRows {
			regions = append(regions, core.NewBounds(uint16(regionStart), 0, dims.Cols, uint16(height)))
		}
	}
	return regions
}
