package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKey_Char(t *testing.T) {
	tests := []struct {
		input string
		want  rune
	}{
		{"a", 'a'},
		{"Z", 'Z'},
		{"5", '5'},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err, "ParseKey(%q)", tt.input)
		assert.Equal(t, KeyChar, k.Type)
		assert.Equal(t, tt.want, k.Char)
	}
}

func TestParseKey_Named(t *testing.T) {
	tests := []struct {
		input string
		want  KeyType
	}{
		{"Enter", KeyEnter},
		{"Return", KeyEnter},
		{"Tab", KeyTab},
		{"Escape", KeyEscape},
		{"Esc", KeyEscape},
		{"Backspace", KeyBackspace},
		{"Delete", KeyDelete},
		{"Del", KeyDelete},
		{"Space", KeySpace},
		{"Insert", KeyInsert},
		{"Ins", KeyInsert},
		{"Up", KeyUp},
		{"Down", KeyDown},
		{"Left", KeyLeft},
		{"Right", KeyRight},
		{"Home", KeyHome},
		{"End", KeyEnd},
		{"PageUp", KeyPageUp},
		{"PgUp", KeyPageUp},
		{"PageDown", KeyPageDown},
		{"PgDn", KeyPageDown},
		{"F1", KeyF1},
		{"F2", KeyF2},
		{"F3", KeyF3},
		{"F4", KeyF4},
		{"F5", KeyF5},
		{"F6", KeyF6},
		{"F7", KeyF7},
		{"F8", KeyF8},
		{"F9", KeyF9},
		{"F10", KeyF10},
		{"F11", KeyF11},
		{"F12", KeyF12},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err, "ParseKey(%q)", tt.input)
		assert.Equal(t, tt.want, k.Type, "ParseKey(%q).Type", tt.input)
	}
}

func TestParseKey_Ctrl(t *testing.T) {
	k, err := ParseKey("Ctrl+c")
	require.NoError(t, err)
	assert.Equal(t, KeyCtrl, k.Type)
	assert.Equal(t, 'c', k.Char)

	// Ctrl+C (uppercase) normalizes to lowercase
	k2, err := ParseKey("Ctrl+C")
	require.NoError(t, err)
	assert.Equal(t, KeyCtrl, k2.Type)
	assert.Equal(t, 'c', k2.Char)

	k3, err := ParseKey("Ctrl+a")
	require.NoError(t, err)
	assert.Equal(t, KeyCtrl, k3.Type)
	assert.Equal(t, 'a', k3.Char)
}

func TestParseKey_Alt(t *testing.T) {
	k, err := ParseKey("Alt+f")
	require.NoError(t, err)
	assert.Equal(t, KeyAlt, k.Type)
	assert.Equal(t, 'f', k.Char)

	k2, err := ParseKey("Alt+x")
	require.NoError(t, err)
	assert.Equal(t, KeyAlt, k2.Type)
	assert.Equal(t, 'x', k2.Char)
}

func TestParseKey_Shift(t *testing.T) {
	k, err := ParseKey("Shift+Tab")
	require.NoError(t, err)
	assert.Equal(t, KeyShift, k.Type)
	require.NotNil(t, k.Inner)
	assert.Equal(t, KeyTab, k.Inner.Type)

	k2, err := ParseKey("Shift+Up")
	require.NoError(t, err)
	assert.Equal(t, KeyShift, k2.Type)
	require.NotNil(t, k2.Inner)
	assert.Equal(t, KeyUp, k2.Inner.Type)
}

func TestParseKey_CtrlAlt(t *testing.T) {
	k, err := ParseKey("Ctrl+Alt+c")
	require.NoError(t, err)
	assert.Equal(t, KeyCtrlAlt, k.Type)
	assert.Equal(t, 'c', k.Char)
}

func TestParseKey_Invalid(t *testing.T) {
	_, err := ParseKey("InvalidKey")
	assert.Error(t, err)

	_, err = ParseKey("Ctrl+")
	assert.Error(t, err)

	_, err = ParseKey("Alt+")
	assert.Error(t, err)
}

// --- Escape Sequence Tests ---

func TestEscapeSequence_Ctrl(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"Ctrl+c", []byte{0x03}},
		{"Ctrl+a", []byte{0x01}},
		{"Ctrl+z", []byte{0x1A}},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err, "ParseKey(%q)", tt.input)
		assert.Equal(t, tt.want, k.ToEscapeSequence(), "escape for %q", tt.input)
	}
}

func TestEscapeSequence_Enter(t *testing.T) {
	k, _ := ParseKey("Enter")
	assert.Equal(t, []byte{0x0D}, k.ToEscapeSequence())
}

func TestEscapeSequence_Arrows(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"Up", []byte{0x1B, '[', 'A'}},
		{"Down", []byte{0x1B, '[', 'B'}},
		{"Right", []byte{0x1B, '[', 'C'}},
		{"Left", []byte{0x1B, '[', 'D'}},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.want, k.ToEscapeSequence(), "escape for %q", tt.input)
	}
}

func TestEscapeSequence_FunctionKeys(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"F1", []byte{0x1B, 'O', 'P'}},
		{"F2", []byte{0x1B, 'O', 'Q'}},
		{"F3", []byte{0x1B, 'O', 'R'}},
		{"F4", []byte{0x1B, 'O', 'S'}},
		{"F5", []byte{0x1B, '[', '1', '5', '~'}},
		{"F6", []byte{0x1B, '[', '1', '7', '~'}},
		{"F7", []byte{0x1B, '[', '1', '8', '~'}},
		{"F8", []byte{0x1B, '[', '1', '9', '~'}},
		{"F9", []byte{0x1B, '[', '2', '0', '~'}},
		{"F10", []byte{0x1B, '[', '2', '1', '~'}},
		{"F11", []byte{0x1B, '[', '2', '3', '~'}},
		{"F12", []byte{0x1B, '[', '2', '4', '~'}},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.want, k.ToEscapeSequence(), "escape for %q", tt.input)
	}
}

func TestEscapeSequence_Alt(t *testing.T) {
	k, _ := ParseKey("Alt+f")
	assert.Equal(t, []byte{0x1B, 'f'}, k.ToEscapeSequence())

	k2, _ := ParseKey("Alt+x")
	assert.Equal(t, []byte{0x1B, 'x'}, k2.ToEscapeSequence())
}

func TestEscapeSequence_ShiftTab(t *testing.T) {
	k, _ := ParseKey("Shift+Tab")
	assert.Equal(t, []byte{0x1B, '[', 'Z'}, k.ToEscapeSequence())
}

func TestEscapeSequence_ShiftArrows(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"Shift+Up", []byte{0x1B, '[', '1', ';', '2', 'A'}},
		{"Shift+Down", []byte{0x1B, '[', '1', ';', '2', 'B'}},
		{"Shift+Right", []byte{0x1B, '[', '1', ';', '2', 'C'}},
		{"Shift+Left", []byte{0x1B, '[', '1', ';', '2', 'D'}},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.want, k.ToEscapeSequence(), "escape for %q", tt.input)
	}
}

func TestEscapeSequence_Special(t *testing.T) {
	tests := []struct {
		input string
		want  []byte
	}{
		{"Tab", []byte{0x09}},
		{"Escape", []byte{0x1B}},
		{"Backspace", []byte{0x7F}},
		{"Space", []byte{0x20}},
		{"Delete", []byte{0x1B, '[', '3', '~'}},
		{"Insert", []byte{0x1B, '[', '2', '~'}},
		{"Home", []byte{0x1B, '[', 'H'}},
		{"End", []byte{0x1B, '[', 'F'}},
		{"PageUp", []byte{0x1B, '[', '5', '~'}},
		{"PageDown", []byte{0x1B, '[', '6', '~'}},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.want, k.ToEscapeSequence(), "escape for %q", tt.input)
	}
}

func TestEscapeSequence_Char(t *testing.T) {
	k, _ := ParseKey("a")
	assert.Equal(t, []byte("a"), k.ToEscapeSequence())

	k2, _ := ParseKey("Z")
	assert.Equal(t, []byte("Z"), k2.ToEscapeSequence())
}

func TestEscapeSequence_CtrlAlt(t *testing.T) {
	k, _ := ParseKey("Ctrl+Alt+c")
	// ESC + Ctrl+C (0x03)
	assert.Equal(t, []byte{0x1B, 0x03}, k.ToEscapeSequence())
}

func TestKey_String(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"a", "a"},
		{"Enter", "Enter"},
		{"Ctrl+c", "Ctrl+c"},
		{"Alt+f", "Alt+f"},
		{"Shift+Tab", "Shift+Tab"},
		{"F1", "F1"},
		{"Up", "Up"},
	}
	for _, tt := range tests {
		k, err := ParseKey(tt.input)
		require.NoError(t, err)
		assert.Equal(t, tt.want, k.String(), "String() for %q", tt.input)
	}
}

func TestParseKey_RoundTrip(t *testing.T) {
	cases := []string{
		"a", "Enter", "Tab", "Up", "F1", "Ctrl+c", "Alt+f", "Shift+Tab",
	}
	for _, input := range cases {
		k, err := ParseKey(input)
		require.NoError(t, err, "ParseKey(%q)", input)
		// Verify ToEscapeSequence doesn't panic
		seq := k.ToEscapeSequence()
		assert.NotNil(t, seq, "ToEscapeSequence() for %q", input)
	}
}
