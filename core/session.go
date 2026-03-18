package core

import (
	"fmt"

	"github.com/google/uuid"
)

// SessionID is a unique identifier for a terminal session (UUID string).
type SessionID string

// NewSessionID generates a new random session ID.
func NewSessionID() SessionID {
	return SessionID(uuid.New().String())
}

// SessionIDFromString creates a SessionID from an existing UUID string.
func SessionIDFromString(s string) SessionID {
	return SessionID(s)
}

// String implements fmt.Stringer.
func (id SessionID) String() string {
	return string(id)
}

// SessionStatus represents the lifecycle state of a terminal session.
type SessionStatus string

const (
	SessionInitializing SessionStatus = "initializing"
	SessionActive       SessionStatus = "active"
	SessionPaused       SessionStatus = "paused"
	SessionTerminated   SessionStatus = "terminated"
)

// String implements fmt.Stringer.
func (s SessionStatus) String() string {
	return string(s)
}

// SessionConfig holds configuration for creating a new terminal session.
type SessionConfig struct {
	Dimensions       Dimensions        `json:"dimensions"`
	Shell            string            `json:"shell"`
	WorkingDirectory *string           `json:"working_directory,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
}

// DefaultSessionConfig returns a SessionConfig with sensible defaults.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		Dimensions: DefaultDimensions(),
		Shell:      defaultShell(),
		Env:        make(map[string]string),
	}
}

// defaultShell returns the default shell for the platform.
func defaultShell() string {
	// On macOS and Linux, use /bin/bash; on Windows, use powershell.exe.
	// Since Go doesn't have cfg!(windows) like Rust, we use runtime detection
	// but keep it simple for the core package.
	return "/bin/bash"
}

// String implements fmt.Stringer.
func (c SessionConfig) String() string {
	return fmt.Sprintf("SessionConfig{shell:%s, dims:%s}", c.Shell, c.Dimensions)
}

// SessionInfo holds information about an active session.
type SessionInfo struct {
	ID     SessionID     `json:"id"`
	Status SessionStatus `json:"status"`
	Config SessionConfig `json:"config"`
}

// NewSessionInfo creates a new SessionInfo.
func NewSessionInfo(id SessionID, status SessionStatus, config SessionConfig) SessionInfo {
	return SessionInfo{ID: id, Status: status, Config: config}
}

// String implements fmt.Stringer.
func (i SessionInfo) String() string {
	return fmt.Sprintf("Session{id:%s, status:%s, shell:%s}", i.ID, i.Status, i.Config.Shell)
}
