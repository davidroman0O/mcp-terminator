package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidroman0O/mcp-terminator/core"
)

func TestBorderDetector_LightBox(t *testing.T) {
	grid := gridFromStringWithSize(
		"┌────────┐\n"+
			"│ Hello  │\n"+
			"└────────┘",
		5, 20,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, uint16(10), detected[0].Bounds.Width)
	assert.Equal(t, uint16(3), detected[0].Bounds.Height)
	assert.Equal(t, ConfidenceHigh, detected[0].Confidence)
}

func TestBorderDetector_WithTitle(t *testing.T) {
	grid := gridFromStringWithSize(
		"┌─ Title ─┐\n"+
			"│  Content│\n"+
			"└─────────┘",
		5, 20,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, core.ElementBorder, detected[0].Element.Type)
	require.NotNil(t, detected[0].Element.Title)
	assert.Equal(t, "Title", *detected[0].Element.Title)
}

func TestBorderDetector_ASCIIBox(t *testing.T) {
	grid := gridFromStringWithSize(
		"+------+\n"+
			"| Test |\n"+
			"+------+",
		5, 15,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, uint16(8), detected[0].Bounds.Width)
	assert.Equal(t, uint16(3), detected[0].Bounds.Height)
}

func TestBorderDetector_DoubleBox(t *testing.T) {
	grid := gridFromStringWithSize(
		"╔══════╗\n"+
			"║ Test ║\n"+
			"╚══════╝",
		5, 15,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, uint16(8), detected[0].Bounds.Width)
	assert.Equal(t, uint16(3), detected[0].Bounds.Height)
}

func TestBorderDetector_NestedBoxesFiltered(t *testing.T) {
	grid := gridFromStringWithSize(
		"┌──────────┐\n"+
			"│ ┌────┐  │\n"+
			"│ │    │  │\n"+
			"│ └────┘  │\n"+
			"└──────────┘",
		10, 20,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	// Filter removes the inner border (contained within outer).
	require.Len(t, detected, 1)
	assert.Equal(t, uint16(12), detected[0].Bounds.Width)
	assert.Equal(t, uint16(5), detected[0].Bounds.Height)
}

func TestBorderDetector_Priority(t *testing.T) {
	det := NewBorderDetector()
	assert.Equal(t, 100, det.Priority())
	assert.Equal(t, "border", det.Name())
}

func TestBorderDetector_NoBorder(t *testing.T) {
	grid := gridFromStringWithSize(
		"Hello World\n"+
			"No borders here",
		5, 20,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	assert.Len(t, detected, 0)
}

func TestBorderDetector_RoundedBox(t *testing.T) {
	grid := gridFromStringWithSize(
		"╭──────╮\n"+
			"│ Test │\n"+
			"╰──────╯",
		5, 15,
	)

	det := NewBorderDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	assert.Equal(t, uint16(8), detected[0].Bounds.Width)
	assert.Equal(t, uint16(3), detected[0].Bounds.Height)
}
