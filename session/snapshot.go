package session

import (
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/davidroman0O/mcp-terminator/detector"
)

// SnapshotConfig controls how a snapshot is captured.
type SnapshotConfig struct {
	// IncludeRawText controls whether the TST includes the full raw text.
	IncludeRawText bool

	// IdleThresholdMs, when non-nil, causes the snapshot to wait until no
	// new output has been received for this many milliseconds before
	// capturing. A nil value skips the idle wait.
	IdleThresholdMs *int
}

// DefaultSnapshotConfig returns a SnapshotConfig with sensible defaults.
func DefaultSnapshotConfig() SnapshotConfig {
	threshold := 100
	return SnapshotConfig{
		IncludeRawText:  true,
		IdleThresholdMs: &threshold,
	}
}

// Snapshot captures the current terminal state. It optionally waits for
// idle (by watching when the background reader last received data), then
// runs the detection pipeline and assembles a TerminalStateTree.
func Snapshot(s *Session, cfg SnapshotConfig) (*core.TerminalStateTree, error) {
	// Optionally wait for idle by monitoring lastActivity.
	if cfg.IdleThresholdMs != nil && *cfg.IdleThresholdMs > 0 {
		waitForIdle(s, *cfg.IdleThresholdMs)
	}

	// Build snapshot under the session lock.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract raw text.
	rawText := ""
	if cfg.IncludeRawText {
		rawText = s.grid.RawText()
	}

	// Create adapter so the grid satisfies detector.GridReader.
	adapter := NewGridAdapter(s.grid)

	// Run the default detection pipeline.
	pipeline := detector.NewDefaultPipeline()
	tst := pipeline.Detect(adapter, string(s.id), rawText)

	return tst, nil
}

// waitForIdle polls the session's lastActivity timestamp until no new
// output has been received for at least thresholdMs, or until a hard
// timeout of 5 seconds is reached. This does NOT read from the PTY
// itself; the background readLoop handles that.
func waitForIdle(s *Session, thresholdMs int) {
	threshold := time.Duration(thresholdMs) * time.Millisecond
	timeout := 5 * time.Second
	start := time.Now()

	for {
		if time.Since(start) > timeout {
			return
		}
		since := time.Since(s.LastActivityTime())
		if since >= threshold {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
