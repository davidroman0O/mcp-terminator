package session

import (
	"regexp"
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
)

// WaitCondition describes what to wait for in terminal output.
type WaitCondition struct {
	// Text is a regex pattern to match against the terminal raw text.
	Text *string

	// ElementType is a UI element type to look for (e.g. "menu", "button").
	ElementType *string

	// Gone reverses the condition — wait for the text/element to disappear.
	Gone bool

	// Idle waits for the terminal to become idle (no new output).
	Idle bool

	// TimeoutMs is the maximum time to wait in milliseconds.
	TimeoutMs int

	// PollIntervalMs is the interval between checks in milliseconds.
	PollIntervalMs int
}

// DefaultWaitCondition returns a WaitCondition with sensible defaults.
func DefaultWaitCondition() WaitCondition {
	return WaitCondition{
		TimeoutMs:      30000,
		PollIntervalMs: 100,
	}
}

// WaitResult is the outcome of a WaitFor call.
type WaitResult struct {
	ConditionMet bool
	WaitedMs     int
	Snapshot     *core.TerminalStateTree
}

// WaitFor polls the session at the configured interval, checking whether
// the condition is satisfied. Returns when the condition is met or the
// timeout expires.
func WaitFor(s *Session, cond WaitCondition) (*WaitResult, error) {
	if cond.TimeoutMs <= 0 {
		cond.TimeoutMs = 30000
	}
	if cond.PollIntervalMs <= 0 {
		cond.PollIntervalMs = 100
	}

	timeout := time.Duration(cond.TimeoutMs) * time.Millisecond
	interval := time.Duration(cond.PollIntervalMs) * time.Millisecond
	start := time.Now()

	// For idle condition, just monitor lastActivity.
	if cond.Idle {
		return waitForIdleCond(s, timeout, start)
	}

	// Compile regex if text pattern is provided.
	var re *regexp.Regexp
	if cond.Text != nil {
		var err error
		re, err = regexp.Compile(*cond.Text)
		if err != nil {
			return nil, &core.InvalidInputError{Message: "invalid regex: " + err.Error()}
		}
	}

	noIdleWait := SnapshotConfig{IncludeRawText: true}

	for {
		elapsed := time.Since(start)
		if elapsed >= timeout {
			snap, err := Snapshot(s, noIdleWait)
			if err != nil {
				return nil, err
			}
			return &WaitResult{
				ConditionMet: false,
				WaitedMs:     int(elapsed.Milliseconds()),
				Snapshot:     snap,
			}, nil
		}

		// Take snapshot (no idle wait — we poll ourselves).
		snap, err := Snapshot(s, noIdleWait)
		if err != nil {
			return nil, err
		}

		// Check text condition.
		if re != nil {
			found := re.MatchString(snap.RawText)
			if (found && !cond.Gone) || (!found && cond.Gone) {
				return &WaitResult{
					ConditionMet: true,
					WaitedMs:     int(time.Since(start).Milliseconds()),
					Snapshot:     snap,
				}, nil
			}
		}

		// Check element type condition.
		if cond.ElementType != nil {
			found := false
			for _, elem := range snap.Elements {
				if elem.TypeName() == *cond.ElementType {
					found = true
					break
				}
			}
			if (found && !cond.Gone) || (!found && cond.Gone) {
				return &WaitResult{
					ConditionMet: true,
					WaitedMs:     int(time.Since(start).Milliseconds()),
					Snapshot:     snap,
				}, nil
			}
		}

		time.Sleep(interval)
	}
}

// waitForIdleCond waits until the session's lastActivity timestamp indicates
// that no new output has been received for 100ms, or until timeout.
func waitForIdleCond(s *Session, timeout time.Duration, start time.Time) (*WaitResult, error) {
	idleThreshold := 100 * time.Millisecond

	for {
		elapsed := time.Since(start)
		if elapsed >= timeout {
			snap, err := Snapshot(s, SnapshotConfig{IncludeRawText: true})
			if err != nil {
				return nil, err
			}
			return &WaitResult{
				ConditionMet: false,
				WaitedMs:     int(elapsed.Milliseconds()),
				Snapshot:     snap,
			}, nil
		}

		since := time.Since(s.LastActivityTime())
		if since >= idleThreshold {
			snap, err := Snapshot(s, SnapshotConfig{IncludeRawText: true})
			if err != nil {
				return nil, err
			}
			return &WaitResult{
				ConditionMet: true,
				WaitedMs:     int(time.Since(start).Milliseconds()),
				Snapshot:     snap,
			}, nil
		}

		time.Sleep(10 * time.Millisecond)
	}
}
