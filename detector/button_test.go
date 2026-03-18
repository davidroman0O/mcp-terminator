package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/davidroman0O/mcp-terminator/core"
)

func TestButtonDetector_BracketPattern(t *testing.T) {
	grid := gridFromStringWithSize("[ OK ] [ Cancel ]", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 2)
	assert.Equal(t, core.ElementButton, detected[0].Element.Type)
	require.NotNil(t, detected[0].Element.Label)
	assert.Equal(t, "OK", *detected[0].Element.Label)
	require.NotNil(t, detected[1].Element.Label)
	assert.Equal(t, "Cancel", *detected[1].Element.Label)
}

func TestButtonDetector_AnglePattern(t *testing.T) {
	grid := gridFromStringWithSize("< Submit > < Reset >", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 2)
	require.NotNil(t, detected[0].Element.Label)
	assert.Equal(t, "Submit", *detected[0].Element.Label)
	require.NotNil(t, detected[1].Element.Label)
	assert.Equal(t, "Reset", *detected[1].Element.Label)
}

func TestButtonDetector_TightBracket(t *testing.T) {
	grid := gridFromStringWithSize("[OK] [Cancel]", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 2)
	require.NotNil(t, detected[0].Element.Label)
	assert.Equal(t, "OK", *detected[0].Element.Label)
	require.NotNil(t, detected[1].Element.Label)
	assert.Equal(t, "Cancel", *detected[1].Element.Label)
}

func TestButtonDetector_ShellPromptExcluded(t *testing.T) {
	grid := gridFromStringWithSize("user@host:/path$ [cmd]", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	assert.Len(t, detected, 0)
}

func TestButtonDetector_EmptyLabelRejected(t *testing.T) {
	grid := gridFromStringWithSize("[ ] [  ]", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	assert.Len(t, detected, 0)
}

func TestButtonDetector_CJKBrackets(t *testing.T) {
	grid := gridFromStringWithSize("\u300cHelp\u300d", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	detected := det.Detect(grid, ctx)

	require.Len(t, detected, 1)
	require.NotNil(t, detected[0].Element.Label)
	assert.Equal(t, "Help", *detected[0].Element.Label)
}

func TestButtonDetector_Priority(t *testing.T) {
	det := NewButtonDetector()
	assert.Equal(t, 60, det.Priority())
	assert.Equal(t, "button", det.Name())
}

func TestButtonDetector_RegionClaimed(t *testing.T) {
	grid := gridFromStringWithSize("[OK]", 5, 40)

	det := NewButtonDetector()
	ctx := NewDetectionContext(core.Origin())
	// Claim the entire first row.
	ctx.ClaimRegion(core.NewBounds(0, 0, 40, 1))
	detected := det.Detect(grid, ctx)

	assert.Len(t, detected, 0)
}
