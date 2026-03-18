package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/davidroman0O/mcp-terminator/emulator"
	"github.com/davidroman0O/mcp-terminator/session"
)

// registerTools wires all 10 MCP tools to the underlying MCPServer.
func (s *Server) registerTools() {
	s.mcp.AddTool(toolSessionCreate(), s.handleSessionCreate)
	s.mcp.AddTool(toolSessionList(), s.handleSessionList)
	s.mcp.AddTool(toolSessionClose(), s.handleSessionClose)
	s.mcp.AddTool(toolSessionResize(), s.handleSessionResize)
	s.mcp.AddTool(toolSnapshot(), s.handleSnapshot)
	s.mcp.AddTool(toolReadOutput(), s.handleReadOutput)
	s.mcp.AddTool(toolType(), s.handleType)
	s.mcp.AddTool(toolPressKey(), s.handlePressKey)
	s.mcp.AddTool(toolClick(), s.handleClick)
	s.mcp.AddTool(toolWaitFor(), s.handleWaitFor)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("json marshal: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func toolError(msg string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(msg, args...)), nil
}

func parseDimensions(args map[string]any, key string) *core.Dimensions {
	raw, ok := args[key]
	if !ok {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	rows, _ := toUint16(m["rows"])
	cols, _ := toUint16(m["cols"])
	if rows == 0 && cols == 0 {
		return nil
	}
	d := core.NewDimensions(rows, cols)
	return &d
}

func toUint16(v any) (uint16, bool) {
	switch n := v.(type) {
	case float64:
		return uint16(n), true
	case int:
		return uint16(n), true
	case int64:
		return uint16(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return uint16(i), true
	}
	return 0, false
}

func parseStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseStringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// intPtr returns a pointer to n, or nil if n is 0.
func intPtr(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

// ===========================================================================
// 1. terminal_session_create
// ===========================================================================

func toolSessionCreate() mcp.Tool {
	return mcp.NewTool("terminal_session_create",
		mcp.WithDescription("Create a new terminal session with the specified command"),
		mcp.WithString("command",
			mcp.Required(),
			mcp.Description("Command to execute (e.g. \"bash\", \"vim\", \"htop\")"),
		),
		mcp.WithArray("args",
			mcp.Description("Command arguments"),
			mcp.WithStringItems(),
		),
		mcp.WithObject("dimensions",
			mcp.Description("Terminal dimensions"),
			mcp.Properties(map[string]any{
				"rows": map[string]any{"type": "integer", "description": "Number of rows"},
				"cols": map[string]any{"type": "integer", "description": "Number of columns"},
			}),
		),
		mcp.WithString("cwd",
			mcp.Description("Working directory"),
		),
		mcp.WithObject("env",
			mcp.Description("Environment variables"),
			mcp.AdditionalProperties(map[string]any{"type": "string"}),
		),
	)
}

func (s *Server) handleSessionCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	command := mcp.ParseString(req, "command", "")
	if command == "" {
		return toolError("command is required")
	}

	args := req.GetArguments()

	// Build the shell command string including arguments.
	// core.SessionConfig.Shell is passed to emulator.Spawn as the command,
	// so we join command+args into a shell invocation if args are present.
	var cmdArgs []string
	if raw, ok := args["args"]; ok {
		cmdArgs = parseStringSlice(raw)
	}

	shell := command
	if len(cmdArgs) > 0 {
		// Wrap as a shell -c invocation so that args are passed correctly.
		shell = command + " " + strings.Join(cmdArgs, " ")
	}

	dims := core.DefaultDimensions()
	if d := parseDimensions(args, "dimensions"); d != nil {
		dims = *d
	}

	var cwd *string
	if v := mcp.ParseString(req, "cwd", ""); v != "" {
		cwd = &v
	}

	var env map[string]string
	if raw, ok := args["env"]; ok {
		env = parseStringMap(raw)
	}

	cfg := core.SessionConfig{
		Shell:            shell,
		Dimensions:       dims,
		WorkingDirectory: cwd,
		Env:              env,
	}

	sess, err := s.manager.Create(cfg)
	if err != nil {
		return toolError("failed to create session: %v", err)
	}

	return jsonResult(map[string]any{
		"session_id": sess.ID().String(),
		"dimensions": map[string]any{
			"rows": dims.Rows,
			"cols": dims.Cols,
		},
		"message": fmt.Sprintf("Session created for command '%s'", command),
	})
}

// ===========================================================================
// 2. terminal_session_list
// ===========================================================================

func toolSessionList() mcp.Tool {
	return mcp.NewTool("terminal_session_list",
		mcp.WithDescription("List all active terminal sessions"),
	)
}

func (s *Server) handleSessionList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	infos := s.manager.List()
	sessions := make([]map[string]any, 0, len(infos))
	for _, info := range infos {
		sessions = append(sessions, map[string]any{
			"session_id": info.ID.String(),
			"command":    info.Config.Shell,
			"status":     info.Status.String(),
		})
	}
	return jsonResult(map[string]any{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// ===========================================================================
// 3. terminal_session_close
// ===========================================================================

func toolSessionClose() mcp.Tool {
	return mcp.NewTool("terminal_session_close",
		mcp.WithDescription("Close and terminate a terminal session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to close"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force close (send SIGKILL instead of SIGTERM)"),
		),
	)
}

func (s *Server) handleSessionClose(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}
	force := mcp.ParseBoolean(req, "force", false)

	if err := s.manager.Close(core.SessionIDFromString(sessionID), force); err != nil {
		return toolError("failed to close session: %v", err)
	}

	return jsonResult(map[string]any{
		"session_id": sessionID,
		"message":    fmt.Sprintf("Session '%s' closed", sessionID),
	})
}

// ===========================================================================
// 4. terminal_session_resize
// ===========================================================================

func toolSessionResize() mcp.Tool {
	return mcp.NewTool("terminal_session_resize",
		mcp.WithDescription("Resize a terminal session to new dimensions"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to resize"),
		),
		mcp.WithObject("dimensions",
			mcp.Required(),
			mcp.Description("New dimensions"),
			mcp.Properties(map[string]any{
				"rows": map[string]any{"type": "integer", "description": "Number of rows"},
				"cols": map[string]any{"type": "integer", "description": "Number of columns"},
			}),
		),
	)
}

func (s *Server) handleSessionResize(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}

	args := req.GetArguments()
	dims := parseDimensions(args, "dimensions")
	if dims == nil {
		return toolError("dimensions is required")
	}

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	if err := sess.Resize(*dims); err != nil {
		return toolError("failed to resize session: %v", err)
	}

	return jsonResult(map[string]any{
		"session_id": sessionID,
		"dimensions": map[string]any{
			"rows": dims.Rows,
			"cols": dims.Cols,
		},
		"message": fmt.Sprintf("Session resized to %dx%d", dims.Rows, dims.Cols),
	})
}

// ===========================================================================
// 5. terminal_snapshot
// ===========================================================================

func toolSnapshot() mcp.Tool {
	return mcp.NewTool("terminal_snapshot",
		mcp.WithDescription("Capture the current terminal state as a structured Terminal State Tree with detected UI elements"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to snapshot"),
		),
		mcp.WithBoolean("include_raw_text",
			mcp.Description("Include raw text output (default: true)"),
		),
		mcp.WithNumber("idle_threshold_ms",
			mcp.Description("Idle threshold in milliseconds (wait for terminal to be idle)"),
		),
	)
}

func (s *Server) handleSnapshot(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}

	includeRaw := mcp.ParseBoolean(req, "include_raw_text", true)
	idleThreshold := mcp.ParseInt(req, "idle_threshold_ms", 0)

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	cfg := session.SnapshotConfig{
		IncludeRawText:  includeRaw,
		IdleThresholdMs: intPtr(idleThreshold),
	}

	tst, err := session.Snapshot(sess, cfg)
	if err != nil {
		return toolError("failed to capture snapshot: %v", err)
	}

	return jsonResult(tst)
}

// ===========================================================================
// 6. terminal_read_output
// ===========================================================================

func toolReadOutput() mcp.Tool {
	return mcp.NewTool("terminal_read_output",
		mcp.WithDescription("Read raw output from a terminal session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to read from"),
		),
		mcp.WithNumber("max_bytes",
			mcp.Description("Maximum bytes to read (default: read all available)"),
		),
	)
}

func (s *Server) handleReadOutput(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}
	maxBytes := mcp.ParseInt(req, "max_bytes", 0)

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	// Drain any pending PTY output into the grid.
	bytesRead, _ := sess.ReadOutput()

	// Extract the raw text from the grid.
	var output string
	sess.WithGrid(func(grid *emulator.Grid) {
		output = grid.RawText()
	})

	// Apply max_bytes truncation if requested.
	if maxBytes > 0 && len(output) > maxBytes {
		output = output[:maxBytes]
	}

	return jsonResult(map[string]any{
		"output":         output,
		"bytes_read":     bytesRead,
		"more_available": false,
	})
}

// ===========================================================================
// 7. terminal_type
// ===========================================================================

func toolType() mcp.Tool {
	return mcp.NewTool("terminal_type",
		mcp.WithDescription("Type text into a terminal session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to type into"),
		),
		mcp.WithString("text",
			mcp.Required(),
			mcp.Description("Text to type"),
		),
		mcp.WithNumber("delay_ms",
			mcp.Description("Delay between characters in milliseconds"),
		),
	)
}

func (s *Server) handleType(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}
	text := mcp.ParseString(req, "text", "")
	if text == "" {
		return toolError("text is required")
	}
	delayMs := mcp.ParseInt(req, "delay_ms", 0)

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	if err := sess.Type(text, delayMs); err != nil {
		return toolError("failed to type text: %v", err)
	}

	return jsonResult(map[string]any{
		"session_id":  sessionID,
		"chars_typed": len([]rune(text)),
		"message":     "Text typed successfully",
	})
}

// ===========================================================================
// 8. terminal_press_key
// ===========================================================================

func toolPressKey() mcp.Tool {
	return mcp.NewTool("terminal_press_key",
		mcp.WithDescription("Press a special key or key combination (arrows, F-keys, Ctrl+X, etc.)"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to send key to"),
		),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Key to press (e.g. \"Enter\", \"Up\", \"Ctrl+c\", \"F1\")"),
		),
	)
}

func (s *Server) handlePressKey(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}
	key := mcp.ParseString(req, "key", "")
	if key == "" {
		return toolError("key is required")
	}

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	if err := sess.PressKey(key); err != nil {
		return toolError("failed to press key '%s': %v", key, err)
	}

	// Get the escape sequence for the response.
	parsed, err := core.ParseKey(key)
	if err != nil {
		return toolError("invalid key format: %v", err)
	}
	seq := parsed.ToEscapeSequence()
	escStr := ""
	for _, b := range seq {
		escStr += fmt.Sprintf("\\x%02x", b)
	}

	return jsonResult(map[string]any{
		"session_id":      sessionID,
		"key":             key,
		"escape_sequence": escStr,
		"message":         fmt.Sprintf("Key '%s' pressed successfully", key),
	})
}

// ===========================================================================
// 9. terminal_click
// ===========================================================================

func toolClick() mcp.Tool {
	return mcp.NewTool("terminal_click",
		mcp.WithDescription("Click on a UI element by its ref_id (navigates and activates)"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to interact with"),
		),
		mcp.WithString("ref_id",
			mcp.Required(),
			mcp.Description("Element reference ID to click"),
		),
	)
}

func (s *Server) handleClick(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}
	refID := mcp.ParseString(req, "ref_id", "")
	if refID == "" {
		return toolError("ref_id is required")
	}

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	// Take a snapshot to find the element.
	snap, err := session.Snapshot(sess, session.DefaultSnapshotConfig())
	if err != nil {
		return toolError("failed to capture snapshot for click: %v", err)
	}

	elem := snap.FindElement(refID)
	if elem == nil {
		return toolError("element '%s' not found in current snapshot", refID)
	}

	// TODO: implement proper navigation to the element (tab ordering,
	// arrow keys, etc.). For now, return the element location so the
	// caller knows it was found.
	return jsonResult(map[string]any{
		"session_id": sessionID,
		"ref_id":     refID,
		"keys_sent":  []string{},
		"message":    fmt.Sprintf("Element '%s' found at row=%d col=%d (navigation not yet implemented)", refID, elem.Bounds.Row, elem.Bounds.Col),
	})
}

// ===========================================================================
// 10. terminal_wait_for
// ===========================================================================

func toolWaitFor() mcp.Tool {
	return mcp.NewTool("terminal_wait_for",
		mcp.WithDescription("Wait for text to appear, element to show, or terminal to be idle"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session to wait on"),
		),
		mcp.WithString("text",
			mcp.Description("Text to wait for (regex pattern)"),
		),
		mcp.WithString("element_type",
			mcp.Description("Element type to wait for"),
		),
		mcp.WithBoolean("gone",
			mcp.Description("Wait for text/element to disappear"),
		),
		mcp.WithBoolean("idle",
			mcp.Description("Wait for terminal to be idle"),
		),
		mcp.WithNumber("timeout_ms",
			mcp.Required(),
			mcp.Description("Timeout in milliseconds"),
		),
		mcp.WithNumber("poll_interval_ms",
			mcp.Description("Polling interval in milliseconds (default: 100)"),
		),
	)
}

func (s *Server) handleWaitFor(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := mcp.ParseString(req, "session_id", "")
	if sessionID == "" {
		return toolError("session_id is required")
	}

	timeoutMs := mcp.ParseInt(req, "timeout_ms", 5000)
	if timeoutMs <= 0 {
		return toolError("timeout_ms must be positive")
	}

	text := mcp.ParseString(req, "text", "")
	elementType := mcp.ParseString(req, "element_type", "")
	gone := mcp.ParseBoolean(req, "gone", false)
	idle := mcp.ParseBoolean(req, "idle", false)
	pollMs := mcp.ParseInt(req, "poll_interval_ms", 100)

	sess, err := s.manager.Get(core.SessionIDFromString(sessionID))
	if err != nil {
		return toolError("session not found: %v", err)
	}

	cond := session.WaitCondition{
		Text:           strPtr(text),
		ElementType:    strPtr(elementType),
		Gone:           gone,
		Idle:           idle,
		TimeoutMs:      timeoutMs,
		PollIntervalMs: pollMs,
	}

	result, err := session.WaitFor(sess, cond)
	if err != nil {
		return toolError("wait failed: %v", err)
	}

	msg := fmt.Sprintf("Condition met after %dms", result.WaitedMs)
	if !result.ConditionMet {
		msg = fmt.Sprintf("Timeout after %dms", result.WaitedMs)
	}

	resp := map[string]any{
		"session_id":    sessionID,
		"condition_met": result.ConditionMet,
		"waited_ms":     result.WaitedMs,
		"message":       msg,
	}
	if result.Snapshot != nil {
		resp["snapshot"] = result.Snapshot
	}

	return jsonResult(resp)
}
