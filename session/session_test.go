package session

import (
	"testing"
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/davidroman0O/mcp-terminator/emulator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shellConfig() core.SessionConfig {
	return core.SessionConfig{
		Dimensions: core.DefaultDimensions(),
		Shell:      "/bin/sh",
		Env:        map[string]string{},
	}
}

// --- Session tests ---

func TestSessionCreateAndSnapshot(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	// Run "echo hello" and wait for output.
	err = s.Type("echo hello\n", 0)
	require.NoError(t, err)

	// Give the command time to execute and the read loop to process.
	time.Sleep(300 * time.Millisecond)

	snap, err := Snapshot(s, DefaultSnapshotConfig())
	require.NoError(t, err)
	assert.NotNil(t, snap)
	assert.Contains(t, snap.RawText, "hello")
	assert.Equal(t, string(s.ID()), snap.SessionID)
}

func TestSessionTypeAndSnapshot(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	// Use cat so typed text is echoed back.
	err = s.Type("cat\n", 0)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	err = s.Type("typed_text_here", 0)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	snap, err := Snapshot(s, DefaultSnapshotConfig())
	require.NoError(t, err)
	assert.Contains(t, snap.RawText, "typed_text_here")
}

func TestSessionPressKey(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	// Press Enter — should not error.
	err = s.PressKey("Enter")
	assert.NoError(t, err)

	// Press Ctrl+C — should not error.
	err = s.PressKey("Ctrl+c")
	assert.NoError(t, err)

	// Press Up arrow.
	err = s.PressKey("Up")
	assert.NoError(t, err)

	// Invalid key should error.
	err = s.PressKey("InvalidKey")
	assert.Error(t, err)
}

func TestSessionResize(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	newDims := core.NewDimensions(30, 120)
	err = s.Resize(newDims)
	assert.NoError(t, err)

	s.WithGrid(func(grid *emulator.Grid) {
		assert.Equal(t, newDims, grid.Dimensions())
	})
}

func TestSessionClose(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)

	assert.Equal(t, core.SessionActive, s.Status())

	err = s.Close(true)
	assert.NoError(t, err)
	assert.Equal(t, core.SessionTerminated, s.Status())
}

func TestSessionTypeWithDelay(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	err = s.Type("cat\n", 0)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	err = s.Type("ab", 50)
	elapsed := time.Since(start)
	require.NoError(t, err)

	// 2 chars at 50ms each = at least 50ms (delay is between chars, after first).
	assert.True(t, elapsed >= 50*time.Millisecond, "expected >=50ms, got %v", elapsed)
}

// --- Wait tests ---

func TestWaitForText(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	err = s.Type("echo MAGIC_STRING_42\n", 0)
	require.NoError(t, err)

	pattern := "MAGIC_STRING_42"
	result, err := WaitFor(s, WaitCondition{
		Text:           &pattern,
		TimeoutMs:      5000,
		PollIntervalMs: 50,
	})
	require.NoError(t, err)
	assert.True(t, result.ConditionMet, "expected text condition to be met")
	assert.Contains(t, result.Snapshot.RawText, "MAGIC_STRING_42")
}

func TestWaitForTextTimeout(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	pattern := "nonexistent_text_xyz"
	result, err := WaitFor(s, WaitCondition{
		Text:           &pattern,
		TimeoutMs:      500,
		PollIntervalMs: 50,
	})
	require.NoError(t, err)
	assert.False(t, result.ConditionMet, "expected timeout, condition not met")
	assert.GreaterOrEqual(t, result.WaitedMs, 500)
}

func TestWaitForIdle(t *testing.T) {
	cfg := shellConfig()
	s, err := NewSession(cfg)
	require.NoError(t, err)
	defer s.Close(true)

	err = s.Type("echo done\n", 0)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	result, err := WaitFor(s, WaitCondition{
		Idle:           true,
		TimeoutMs:      3000,
		PollIntervalMs: 50,
	})
	require.NoError(t, err)
	assert.True(t, result.ConditionMet, "expected idle condition to be met")
}

// --- Manager tests ---

func TestManagerCreateAndList(t *testing.T) {
	m := NewManager(10)
	assert.Equal(t, 0, m.Count())

	cfg := shellConfig()
	s, err := m.Create(cfg)
	require.NoError(t, err)
	defer m.CloseAll()

	assert.Equal(t, 1, m.Count())

	infos := m.List()
	assert.Len(t, infos, 1)
	assert.Equal(t, s.ID(), infos[0].ID)
}

func TestManagerGet(t *testing.T) {
	m := NewManager(10)
	cfg := shellConfig()
	s, err := m.Create(cfg)
	require.NoError(t, err)
	defer m.CloseAll()

	got, err := m.Get(s.ID())
	require.NoError(t, err)
	assert.Equal(t, s.ID(), got.ID())

	_, err = m.Get(core.NewSessionID())
	assert.Error(t, err)
}

func TestManagerClose(t *testing.T) {
	m := NewManager(10)
	cfg := shellConfig()
	s, err := m.Create(cfg)
	require.NoError(t, err)

	assert.Equal(t, 1, m.Count())

	err = m.Close(s.ID(), true)
	assert.NoError(t, err)
	assert.Equal(t, 0, m.Count())
}

func TestManagerCloseAll(t *testing.T) {
	m := NewManager(10)
	cfg := shellConfig()
	_, err := m.Create(cfg)
	require.NoError(t, err)
	_, err = m.Create(cfg)
	require.NoError(t, err)

	assert.Equal(t, 2, m.Count())
	m.CloseAll()
	assert.Equal(t, 0, m.Count())
}

func TestManagerMaxSessions(t *testing.T) {
	m := NewManager(2)
	cfg := shellConfig()

	_, err := m.Create(cfg)
	require.NoError(t, err)
	_, err = m.Create(cfg)
	require.NoError(t, err)
	defer m.CloseAll()

	_, err = m.Create(cfg)
	assert.Error(t, err)

	var limitErr *core.SessionLimitReachedError
	assert.ErrorAs(t, err, &limitErr)
	assert.Equal(t, 2, limitErr.Max)
}

// --- GridAdapter tests ---

func TestGridAdapterSatisfiesGridReader(t *testing.T) {
	// Verify at compile time that GridAdapter satisfies detector.GridReader.
	// (This is really a compile-time check, but let's also exercise the adapter.)
	grid := emulator.NewGrid(core.DefaultDimensions())
	adapter := NewGridAdapter(grid)

	dims := adapter.Dimensions()
	assert.Equal(t, uint16(24), dims.Rows)
	assert.Equal(t, uint16(80), dims.Cols)

	pos := adapter.CursorPosition()
	assert.Equal(t, uint16(0), pos.Row)
	assert.Equal(t, uint16(0), pos.Col)

	assert.True(t, adapter.CursorVisible())

	// Out of bounds returns false.
	_, ok := adapter.Cell(-1, 0)
	assert.False(t, ok)
	_, ok = adapter.Cell(0, -1)
	assert.False(t, ok)
	_, ok = adapter.Cell(100, 0)
	assert.False(t, ok)

	// In bounds returns true.
	cell, ok := adapter.Cell(0, 0)
	assert.True(t, ok)
	assert.Equal(t, ' ', cell.Character)

	// ExtractText on empty grid returns spaces (or trimmed equivalent).
	text := adapter.ExtractText(core.NewBounds(0, 0, 5, 1))
	// An empty grid row is all spaces; RawText trims trailing whitespace.
	assert.NotNil(t, text) // just verify it runs without panic
}
