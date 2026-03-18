package emulator

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/davidroman0O/mcp-terminator/core"
)

// PtyHandle manages a PTY process lifecycle.
type PtyHandle struct {
	mu   sync.Mutex
	file *os.File   // master PTY file
	cmd  *exec.Cmd  // child command
	done chan struct{}
}

// Spawn starts a new PTY process with the given command, dimensions, working directory,
// and environment variables. Returns a handle for interacting with the process.
func Spawn(command string, args []string, dims core.Dimensions, cwd string, env map[string]string) (*PtyHandle, error) {
	cmd := exec.Command(command, args...)

	if cwd != "" {
		cmd.Dir = cwd
	}

	// Build environment.
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	winSize := &pty.Winsize{
		Rows: dims.Rows,
		Cols: dims.Cols,
	}

	f, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return nil, fmt.Errorf("pty spawn: %w", err)
	}

	h := &PtyHandle{
		file: f,
		cmd:  cmd,
		done: make(chan struct{}),
	}

	// Background goroutine to detect process exit.
	go func() {
		_ = cmd.Wait()
		close(h.done)
	}()

	return h, nil
}

// Write sends data to the PTY stdin.
func (h *PtyHandle) Write(data []byte) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.file == nil {
		return 0, fmt.Errorf("pty closed")
	}
	return h.file.Write(data)
}

// Read reads available data from the PTY. It performs a non-blocking read by
// setting a short read deadline so the caller is not blocked indefinitely.
func (h *PtyHandle) Read(buf []byte) (int, error) {
	h.mu.Lock()
	f := h.file
	h.mu.Unlock()
	if f == nil {
		return 0, fmt.Errorf("pty closed")
	}
	// Set a short deadline so Read does not block forever.
	_ = f.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	n, err := f.Read(buf)
	if err != nil {
		if os.IsTimeout(err) {
			return 0, nil // no data available
		}
		return n, err
	}
	return n, nil
}

// Resize changes the PTY dimensions, sending SIGWINCH to the child.
func (h *PtyHandle) Resize(dims core.Dimensions) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.file == nil {
		return fmt.Errorf("pty closed")
	}
	return pty.Setsize(h.file, &pty.Winsize{
		Rows: dims.Rows,
		Cols: dims.Cols,
	})
}

// IsAlive reports whether the child process is still running.
func (h *PtyHandle) IsAlive() bool {
	select {
	case <-h.done:
		return false
	default:
		return true
	}
}

// Close terminates the child process. It sends SIGTERM first, then SIGKILL
// after a timeout, and waits for the process to exit. The PTY file is closed.
//
// The file is closed before waiting for the process to exit. This ensures
// that cmd.Wait() can complete even if there are outstanding reads on the
// PTY (which would otherwise block Wait on some platforms).
func (h *PtyHandle) Close() error {
	h.mu.Lock()

	if h.file == nil {
		h.mu.Unlock()
		return nil // already closed
	}

	proc := h.cmd.Process

	// Close the file first to unblock any outstanding reads and allow
	// cmd.Wait() to complete. This also causes the session readLoop to
	// exit when its next Read fails.
	err := h.file.Close()
	h.file = nil

	// Release the mutex before waiting for the process. The readLoop
	// goroutine may be blocked on mu.Lock() in Read(); after we release
	// the lock, it will see file==nil and return an error, allowing
	// the session to close its done channel.
	h.mu.Unlock()

	if proc != nil {
		// Try graceful termination.
		_ = proc.Signal(syscall.SIGTERM)

		// Wait up to 2 seconds for exit.
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-h.done:
			timer.Stop()
		case <-timer.C:
			// Force kill.
			_ = proc.Signal(syscall.SIGKILL)
			<-h.done
		}
	}

	return err
}
