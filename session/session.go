// Package session provides terminal session lifecycle management.
//
// This is Layer 2 in the architecture - it depends on core, emulator, and
// detector to manage terminal session lifecycles. It provides:
//   - Session creation and initialization
//   - Session state tracking
//   - Session cleanup and termination
//   - Session registry management (Manager)
//   - Terminal snapshot capture
//   - Wait-for-condition polling
package session

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/davidroman0O/mcp-terminator/emulator"
)

// Session is an individual terminal session backed by a PTY, parser, and grid.
type Session struct {
	id        core.SessionID
	pty       *emulator.PtyHandle
	parser    *emulator.Parser
	grid      *emulator.Grid
	config    core.SessionConfig
	status    core.SessionStatus
	createdAt time.Time

	// lastActivity is updated (atomically, as unix-nano) by the readLoop
	// whenever new bytes arrive from the PTY. Snapshot and wait functions
	// use this to detect idleness without competing for PTY reads.
	lastActivity atomic.Int64

	mu   sync.Mutex
	done chan struct{} // closed when the background reader goroutine exits
}

// NewSession spawns a new PTY process, initialises the parser and grid, and
// starts a background goroutine that continuously reads PTY output and feeds
// it through the parser into the grid.
func NewSession(cfg core.SessionConfig) (*Session, error) {
	cwd := ""
	if cfg.WorkingDirectory != nil {
		cwd = *cfg.WorkingDirectory
	}

	// When Shell contains spaces (e.g. "bash -c 'echo hello'"), it is a
	// full command string rather than a bare executable path. Wrap it in
	// /bin/sh -c so the OS shell handles argument splitting correctly.
	// Without this, exec.Command would try to find an executable whose
	// filename is the entire string, which fails.
	command := cfg.Shell
	var args []string
	if strings.ContainsAny(command, " \t") {
		args = []string{"-c", command}
		command = "/bin/sh"
	}

	ptyHandle, err := emulator.Spawn(command, args, cfg.Dimensions, cwd, cfg.Env)
	if err != nil {
		return nil, fmt.Errorf("session: spawn pty: %w", err)
	}

	grid := emulator.NewGrid(cfg.Dimensions)
	parser := emulator.NewParser(grid)

	s := &Session{
		id:        core.NewSessionID(),
		pty:       ptyHandle,
		parser:    parser,
		grid:      grid,
		config:    cfg,
		status:    core.SessionActive,
		createdAt: time.Now(),
		done:      make(chan struct{}),
	}
	s.lastActivity.Store(time.Now().UnixNano())

	go s.readLoop()
	return s, nil
}

// readLoop continuously reads from the PTY and feeds bytes through the parser.
// It exits when the PTY is closed or returns an error.
func (s *Session) readLoop() {
	defer close(s.done)
	buf := make([]byte, 4096)
	for {
		n, err := s.pty.Read(buf)
		if err != nil {
			// PTY closed or errored — mark exited.
			s.mu.Lock()
			if s.status == core.SessionActive {
				s.status = core.SessionTerminated
			}
			s.mu.Unlock()
			return
		}
		if n > 0 {
			s.mu.Lock()
			s.parser.Process(buf[:n])
			s.mu.Unlock()
			s.lastActivity.Store(time.Now().UnixNano())
		}
	}
}

// ID returns the session identifier.
func (s *Session) ID() core.SessionID {
	return s.id
}

// Config returns the session configuration.
func (s *Session) Config() core.SessionConfig {
	return s.config
}

// Status returns the current session status.
func (s *Session) Status() core.SessionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// CreatedAt returns the session creation time.
func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

// LastActivityTime returns the time of the last PTY read activity.
func (s *Session) LastActivityTime() time.Time {
	return time.Unix(0, s.lastActivity.Load())
}

// ReadOutput is a no-op kept for API compatibility. The background goroutine
// handles all reading. Returns (0, nil).
func (s *Session) ReadOutput() (int, error) {
	return 0, nil
}

// Type writes text to the PTY, optionally with a per-character delay.
// If delayMs is 0 or negative the text is written in one shot.
func (s *Session) Type(text string, delayMs int) error {
	if delayMs <= 0 {
		_, err := s.pty.Write([]byte(text))
		return err
	}
	delay := time.Duration(delayMs) * time.Millisecond
	for _, ch := range text {
		_, err := s.pty.Write([]byte(string(ch)))
		if err != nil {
			return err
		}
		time.Sleep(delay)
	}
	return nil
}

// PressKey parses the human-readable key name (e.g. "Enter", "Ctrl+c"),
// converts it to an escape sequence, and writes it to the PTY.
func (s *Session) PressKey(key string) error {
	k, err := core.ParseKey(key)
	if err != nil {
		return err
	}
	seq := k.ToEscapeSequence()
	if seq == nil {
		return &core.InvalidInputError{Message: fmt.Sprintf("key %q produces no escape sequence", key)}
	}
	_, err = s.pty.Write(seq)
	return err
}

// Resize changes the terminal dimensions for both the PTY and the grid.
func (s *Session) Resize(dims core.Dimensions) error {
	if err := s.pty.Resize(dims); err != nil {
		return err
	}
	s.mu.Lock()
	s.grid.Resize(dims)
	s.mu.Unlock()
	return nil
}

// Close terminates the session. If force is true the PTY is killed
// immediately; otherwise a graceful close is attempted.
func (s *Session) Close(force bool) error {
	s.mu.Lock()
	s.status = core.SessionTerminated
	s.mu.Unlock()

	err := s.pty.Close()

	// Wait for the read loop to finish.
	<-s.done
	return err
}

// Grid returns the underlying grid (caller must hold lock or use WithGrid).
func (s *Session) Grid() *emulator.Grid {
	return s.grid
}

// WithGrid runs fn while holding the session mutex, giving safe access to the
// grid and parser.
func (s *Session) WithGrid(fn func(grid *emulator.Grid)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.grid)
}
