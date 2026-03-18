package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidroman0O/mcp-terminator/core"
)

func TestPipeline_DefaultDetectors(t *testing.T) {
	p := NewDefaultPipeline()
	assert.Len(t, p.detectors, 8)

	// Verify priority ordering (descending).
	for i := 0; i < len(p.detectors)-1; i++ {
		assert.GreaterOrEqual(t, p.detectors[i].Priority(), p.detectors[i+1].Priority(),
			"detectors should be sorted by descending priority")
	}
}

func TestPipeline_BorderWithButton(t *testing.T) {
	grid := gridFromStringWithSize(
		"┌──────────────┐\n"+
			"│   [OK]       │\n"+
			"└──────────────┘",
		5, 20,
	)

	p := NewDefaultPipeline()
	tst := p.Detect(grid, "test-session", "")

	assert.Equal(t, "test-session", tst.SessionID)
	require.NotEmpty(t, tst.Elements)

	// Should have a border element.
	var hasBorder bool
	for _, e := range tst.Elements {
		if e.Type == core.ElementBorder {
			hasBorder = true
		}
	}
	assert.True(t, hasBorder, "should detect a border element")
}

func TestPipeline_CheckboxAndProgress(t *testing.T) {
	// Use standalone detectors to test checkbox and progress individually,
	// since the full pipeline's menu/table detectors may claim regions
	// via cursor-based heuristics when content rows are contiguous.
	grid := gridFromStringWithSize(
		"[x] Enable logging\n"+
			"[ ] Verbose mode",
		5, 25,
	)
	grid.cursorPos = core.NewPosition(4, 0) // cursor outside content

	cbDet := NewCheckboxDetector()
	ctx := NewDetectionContext(grid.cursorPos)
	cbs := cbDet.Detect(grid, ctx)
	assert.Equal(t, 2, len(cbs), "should detect 2 checkboxes")

	grid2 := gridFromStringWithSize("████░░░░░░", 5, 25)
	grid2.cursorPos = core.NewPosition(4, 0)
	ctx2 := NewDetectionContext(grid2.cursorPos)
	pDet := NewProgressDetector()
	progs := pDet.Detect(grid2, ctx2)
	assert.Equal(t, 1, len(progs), "should detect 1 progress bar")
}

func TestPipeline_RegionClaimingPreventsOverlap(t *testing.T) {
	// A border should claim its region, preventing a button
	// from being detected on the same spot by the border chars.
	grid := gridFromStringWithSize(
		"┌────┐\n"+
			"│ AB │\n"+
			"└────┘",
		5, 10,
	)

	p := NewDefaultPipeline()
	elems := p.DetectAll(grid)

	// Count how many border elements we have.
	borderCount := 0
	for _, e := range elems {
		if e.Element.Type == core.ElementBorder {
			borderCount++
		}
	}
	assert.Equal(t, 1, borderCount)
}

func TestPipeline_EmptyGrid(t *testing.T) {
	grid := gridFromStringWithSize("", 5, 10)

	p := NewDefaultPipeline()
	elems := p.DetectAll(grid)

	assert.Empty(t, elems)
}

func TestPipeline_StatusBarDetection(t *testing.T) {
	// Test the status bar detector directly to avoid menu/table claiming.
	grid := gridFromStringWithSize(
		"Content line 1\n"+
			"Content line 2\n"+
			"Press q to quit | F1 Help",
		3, 40,
	)

	det := NewStatusBarDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, core.ElementStatusBar, detected[0].Element.Type)
	assert.Equal(t, ConfidenceMedium, detected[0].Confidence)
	require.NotNil(t, detected[0].Element.Content)
	assert.Contains(t, *detected[0].Element.Content, "Press q to quit")
}

func TestPipeline_ConfidenceOrdering(t *testing.T) {
	assert.True(t, ConfidenceHigh > ConfidenceMedium)
	assert.True(t, ConfidenceMedium > ConfidenceLow)
}

func TestPipeline_DetectReturnsTST(t *testing.T) {
	grid := gridFromStringWithSize("[OK]", 3, 10)

	p := NewDefaultPipeline()
	tst := p.Detect(grid, "sess-1", "raw text here")

	assert.Equal(t, "sess-1", tst.SessionID)
	assert.Equal(t, "raw text here", tst.RawText)
	assert.NotEmpty(t, tst.Timestamp)
	assert.Equal(t, uint16(3), tst.Dimensions.Rows)
	assert.Equal(t, uint16(10), tst.Dimensions.Cols)
}

func TestPipeline_AssemblyNestsChildrenInBorder(t *testing.T) {
	// When a border is detected, it claims its full region. Lower-priority
	// detectors skip claimed regions, so buttons inside the border are not
	// independently detected. To test assembly nesting, we feed the
	// assembler directly with pre-built elements.
	borderBounds := core.NewBounds(0, 0, 18, 3)
	buttonBounds := core.NewBounds(1, 3, 4, 1)

	title := "Test"
	elements := []core.Element{
		core.NewBorderElement("border_0_0", borderBounds, &title, nil),
		core.NewButtonElement("button_1_3", buttonBounds, "OK"),
	}

	assembled := assembleBorderHierarchy(elements)

	// The border should now list the button as a child.
	require.Len(t, assembled, 1, "button should be nested inside border")
	assert.Equal(t, core.ElementBorder, assembled[0].Type)
	require.Len(t, assembled[0].Children, 1)
	assert.Equal(t, "button_1_3", assembled[0].Children[0])
}

func TestPipeline_BorderClaimsRegion(t *testing.T) {
	// Verify that when a border is detected, elements inside its
	// claimed region are not duplicated as independent detections.
	grid := gridFromStringWithSize(
		"┌────────────────┐\n"+
			"│  [OK] [Cancel] │\n"+
			"└────────────────┘",
		5, 25,
	)

	p := NewDefaultPipeline()
	elems := p.DetectAll(grid)

	borderCount := 0
	buttonCount := 0
	for _, e := range elems {
		switch e.Element.Type {
		case core.ElementBorder:
			borderCount++
		case core.ElementButton:
			buttonCount++
		}
	}
	assert.Equal(t, 1, borderCount, "should detect one border")
	// Buttons are inside the border's claimed region, so they are blocked.
	assert.Equal(t, 0, buttonCount, "buttons inside border region are claimed")
}
