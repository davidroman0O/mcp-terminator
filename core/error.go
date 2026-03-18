package core

import "fmt"

// SessionNotFoundError indicates the requested session does not exist.
type SessionNotFoundError struct {
	ID SessionID
}

func (e *SessionNotFoundError) Error() string {
	return fmt.Sprintf("session not found: %s", e.ID)
}

// PtyError indicates a PTY-related failure.
type PtyError struct {
	Message string
}

func (e *PtyError) Error() string {
	return fmt.Sprintf("pty error: %s", e.Message)
}

// InvalidInputError indicates invalid input or parameters.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string {
	return fmt.Sprintf("invalid input: %s", e.Message)
}

// SessionLimitReachedError indicates the maximum number of concurrent sessions is exceeded.
type SessionLimitReachedError struct {
	Max int
}

func (e *SessionLimitReachedError) Error() string {
	return fmt.Sprintf("session limit reached (max: %d)", e.Max)
}

// DetectionError indicates a UI element detection failure.
type DetectionError struct {
	Message string
}

func (e *DetectionError) Error() string {
	return fmt.Sprintf("detection error: %s", e.Message)
}

// TimeoutError indicates an operation timed out.
type TimeoutError struct {
	DurationMs uint64
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("timeout after %dms", e.DurationMs)
}

// ElementNotFoundError indicates the requested element was not found by ref ID.
type ElementNotFoundError struct {
	RefID string
}

func (e *ElementNotFoundError) Error() string {
	return fmt.Sprintf("element not found: %s", e.RefID)
}

// CommandNotAllowedError indicates a command is blocked by security policy.
type CommandNotAllowedError struct {
	Command string
}

func (e *CommandNotAllowedError) Error() string {
	return fmt.Sprintf("command not allowed: %s", e.Command)
}

// InvalidDimensionsError indicates invalid terminal dimensions.
type InvalidDimensionsError struct {
	Rows uint16
	Cols uint16
}

func (e *InvalidDimensionsError) Error() string {
	return fmt.Sprintf("invalid dimensions: %dx%d", e.Rows, e.Cols)
}

// SessionTerminatedError indicates the session has already terminated.
type SessionTerminatedError struct{}

func (e *SessionTerminatedError) Error() string {
	return "session already terminated"
}

// ConfigError indicates a configuration problem.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error: %s", e.Message)
}
