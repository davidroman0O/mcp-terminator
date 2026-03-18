package detector

import (
	"sort"
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
)

// Pipeline runs detectors in priority order (highest first), claims
// regions between detector passes, and assembles a TerminalStateTree.
type Pipeline struct {
	detectors []Detector
}

// NewPipeline creates an empty pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{}
}

// AddDetector appends a detector and re-sorts by descending priority.
func (p *Pipeline) AddDetector(d Detector) {
	p.detectors = append(p.detectors, d)
	sort.Slice(p.detectors, func(i, j int) bool {
		return p.detectors[i].Priority() > p.detectors[j].Priority()
	})
}

// NewDefaultPipeline creates a pipeline with all 8 detectors in priority order.
func NewDefaultPipeline() *Pipeline {
	p := NewPipeline()
	p.AddDetector(NewBorderDetector())   // 100
	p.AddDetector(NewMenuDetector())     // 80
	p.AddDetector(NewTableDetector())    // 80
	p.AddDetector(NewInputDetector())    // 70
	p.AddDetector(NewButtonDetector())   // 60
	p.AddDetector(NewCheckboxDetector()) // 60
	p.AddDetector(NewProgressDetector()) // 60
	p.AddDetector(NewStatusBarDetector()) // 50
	return p
}

// DetectAll runs all detectors on the grid and returns flat detected elements.
func (p *Pipeline) DetectAll(grid GridReader) []DetectedElement {
	cursor := grid.CursorPosition()
	ctx := NewDetectionContext(cursor)
	ctx.CursorVisible = grid.CursorVisible()

	var allElements []DetectedElement

	for _, det := range p.detectors {
		elements := det.Detect(grid, ctx)
		for _, elem := range elements {
			ctx.ClaimRegion(elem.Bounds)
		}
		allElements = append(allElements, elements...)
	}

	return allElements
}

// Detect runs all detectors and assembles a TerminalStateTree.
// sessionID is attached to the resulting TST. rawText is the full
// text content of the terminal (caller can pass "" if not needed).
func (p *Pipeline) Detect(grid GridReader, sessionID, rawText string) *core.TerminalStateTree {
	detected := p.DetectAll(grid)

	elements := make([]core.Element, 0, len(detected))
	for _, d := range detected {
		elements = append(elements, d.Element)
	}

	// Assemble: nest elements inside borders based on containment.
	elements = assembleBorderHierarchy(elements)

	dims := grid.Dimensions()
	cursor := grid.CursorPosition()

	return &core.TerminalStateTree{
		SessionID:  sessionID,
		Dimensions: dims,
		Cursor:     cursor,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Elements:   elements,
		RawText:    rawText,
	}
}

// assembleBorderHierarchy nests non-border elements inside border elements
// when the non-border element is fully contained within a border's bounds.
// Returns only top-level elements; contained elements become children
// of their parent border by ref-id.
func assembleBorderHierarchy(elements []core.Element) []core.Element {
	// Separate borders from non-borders.
	var borders []core.Element
	var others []core.Element

	for _, e := range elements {
		if e.Type == core.ElementBorder {
			borders = append(borders, e)
		} else {
			others = append(others, e)
		}
	}

	if len(borders) == 0 {
		return elements
	}

	// Sort borders by area descending (largest first).
	sort.Slice(borders, func(i, j int) bool {
		areaI := int(borders[i].Bounds.Width) * int(borders[i].Bounds.Height)
		areaJ := int(borders[j].Bounds.Width) * int(borders[j].Bounds.Height)
		return areaI > areaJ
	})

	// Track which non-border elements are claimed by a border.
	claimed := make([]bool, len(others))

	for i := range borders {
		var children []string
		for j, o := range others {
			if claimed[j] {
				continue
			}
			if isFullyContained(o.Bounds, borders[i].Bounds) {
				children = append(children, o.RefID)
				claimed[j] = true
			}
		}
		if len(children) > 0 {
			borders[i].Children = children
		}
	}

	// Build result: borders first, then unclaimed others.
	result := make([]core.Element, 0, len(borders)+len(others))
	result = append(result, borders...)
	for j, o := range others {
		if !claimed[j] {
			result = append(result, o)
		}
	}

	return result
}

// isFullyContained reports whether inner is completely within outer.
func isFullyContained(inner, outer core.Bounds) bool {
	return inner.Row >= outer.Row &&
		inner.Col >= outer.Col &&
		(inner.Row+inner.Height) <= (outer.Row+outer.Height) &&
		(inner.Col+inner.Width) <= (outer.Col+outer.Width)
}
