# MCP Terminator

**Playwright for terminals.** An MCP server that lets AI agents interact with terminal applications through structured UI element detection.

Instead of raw terminal text, your AI agent gets a **Terminal State Tree** — buttons, inputs, tables, menus, borders, checkboxes, progress bars, and status bars detected and labeled with positions and ref IDs.

```json
{
  "elements": [
    { "type": "border", "ref_id": "border_1", "bounds": {"row": 0, "col": 0, "width": 80, "height": 24}, "title": "My App" },
    { "type": "button", "ref_id": "button_1", "bounds": {"row": 5, "col": 10, "width": 8, "height": 1}, "label": "Submit" },
    { "type": "input", "ref_id": "input_1", "bounds": {"row": 3, "col": 10, "width": 30, "height": 1}, "value": "hello" }
  ]
}
```

## Features

- **10 MCP tools** for terminal interaction (create sessions, type, press keys, snapshot, wait for conditions)
- **8 UI element detectors** — borders, menus, tables, inputs, buttons, checkboxes, progress bars, status bars
- **Terminal State Tree (TST)** — structured snapshot with detected elements, cursor position, and raw text
- **Multi-session** — manage multiple terminal sessions concurrently
- **PTY-based** — real terminal emulation with full ANSI escape sequence support
- **Pure Go** — single binary, no external dependencies, no CGo

## Install

```bash
go install github.com/davidroman0O/mcp-terminator/cmd/mcp-terminator@latest
```

Or build from source:

```bash
git clone https://github.com/davidroman0O/mcp-terminator.git
cd mcp-terminator
go build -o mcp-terminator ./cmd/mcp-terminator/
```

## Configure

### Claude Code

Add to `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "terminator": {
      "command": "mcp-terminator"
    }
  }
}
```

Or in your project's `.mcp.json`:

```json
{
  "mcpServers": {
    "terminator": {
      "command": "/path/to/mcp-terminator"
    }
  }
}
```

### Other MCP Clients

MCP Terminator uses stdio transport. Point your MCP client at the `mcp-terminator` binary.

## Tools

### Session Management

| Tool | Description |
|------|-------------|
| `terminal_session_create` | Spawn a new terminal session with a command |
| `terminal_session_list` | List all active sessions |
| `terminal_session_close` | Close a session (SIGTERM or SIGKILL) |
| `terminal_session_resize` | Resize terminal dimensions |

### Observation

| Tool | Description |
|------|-------------|
| `terminal_snapshot` | Capture terminal state with UI element detection |
| `terminal_read_output` | Read raw output from a session |

### Interaction

| Tool | Description |
|------|-------------|
| `terminal_type` | Type text into a session (with optional per-char delay) |
| `terminal_press_key` | Send a key press (Enter, Ctrl+C, arrow keys, F1-F12, etc.) |
| `terminal_click` | Click on a detected UI element by ref_id |

### Synchronization

| Tool | Description |
|------|-------------|
| `terminal_wait_for` | Wait for text, element type, or idle state with timeout |

## Terminal State Tree

The `terminal_snapshot` tool returns a structured representation of the terminal:

```json
{
  "session_id": "abc-123",
  "dimensions": { "rows": 24, "cols": 80 },
  "cursor": { "row": 10, "col": 5, "visible": true },
  "timestamp": "2026-03-17T20:00:00Z",
  "elements": [...],
  "raw_text": "full screen content as string"
}
```

### Detected Element Types

| Type | Detection | Key Fields |
|------|-----------|------------|
| **Border** | Box-drawing chars (`+─│`, `┌─┐│└┘`, `╔═╗║╚╝`) | `title`, `children` |
| **Menu** | Selection indicators, reverse-video highlighting | `items[]`, `selected` |
| **Table** | Bold/colored headers, column alignment | `headers[]`, `rows[][]` |
| **Input** | Underlined regions, cursor presence | `value`, `cursor_position` |
| **Button** | Bracket patterns `[ OK ]`, `< Cancel >` | `label` |
| **Checkbox** | `[x]`/`[ ]` patterns | `label`, `checked` |
| **Progress** | Block chars `█░`, percentage text | `percent` |
| **Status Bar** | Top/bottom edge lines | `content` |

### Detection Pipeline

Detectors run in priority order with region claiming to prevent overlaps:

1. **Borders** (priority 100) — box-drawing regions
2. **Menus** (priority 80) — selection lists
3. **Tables** (priority 80) — header + data grids
4. **Inputs** (priority 70) — editable fields
5. **Buttons** (priority 60) — bracketed labels
6. **Checkboxes** (priority 60) — toggle indicators
7. **Progress bars** (priority 60) — fill indicators
8. **Status bars** (priority 50) — edge info lines

## Key Support

The `terminal_press_key` tool accepts human-readable key names:

| Key | Sequence |
|-----|----------|
| `Enter` | `\r` |
| `Tab`, `Shift+Tab` | `\t`, `\x1b[Z` |
| `Escape` | `\x1b` |
| `Backspace` | `\x7f` |
| `Up`, `Down`, `Left`, `Right` | Arrow escape sequences |
| `Home`, `End`, `PageUp`, `PageDown` | Navigation sequences |
| `F1`–`F12` | Function key sequences |
| `Ctrl+c`, `Ctrl+d`, `Ctrl+z` | Control characters |
| `Alt+f`, `Alt+b` | Alt key combinations |
| `Shift+Up`, `Shift+Down` | Modified arrow keys |

## Examples

### Interact with a TUI application

```
1. terminal_session_create(command: "htop")
2. terminal_snapshot(session_id: "...", idle_threshold_ms: 1000)
   → See the full htop UI with detected elements
3. terminal_press_key(session_id: "...", key: "q")
   → Quit htop
```

### Test a bubbletea app

```
1. terminal_session_create(command: "./my-tui-app")
2. terminal_wait_for(session_id: "...", text: "Welcome", timeout_ms: 5000)
   → Wait for the app to render
3. terminal_snapshot(session_id: "...")
   → Get detected buttons, inputs, menus
4. terminal_type(session_id: "...", text: "hello")
   → Type into the focused input
5. terminal_press_key(session_id: "...", key: "Enter")
   → Submit
6. terminal_snapshot(session_id: "...")
   → Verify the UI updated
```

### Automate a CLI workflow

```
1. terminal_session_create(command: "bash", cwd: "/my/project")
2. terminal_type(session_id: "...", text: "npm test\n")
3. terminal_wait_for(session_id: "...", text: "passing|failing", timeout_ms: 30000)
   → Wait for test results
4. terminal_snapshot(session_id: "...")
   → Read the test output
```

## Architecture

```
mcp-terminator/
├── cmd/mcp-terminator/    Entry point (stdio MCP server)
├── core/                  Foundation types (Cell, Element, Key, Session, Error)
├── emulator/              Terminal emulation (Grid, ANSI parser, PTY)
├── detector/              UI element detection (8 detectors + pipeline)
├── session/               Session lifecycle (manager, snapshots, wait)
└── server/                MCP protocol binding (10 tools)
```

| Package | What It Does |
|---------|-------------|
| **core** | Types shared across all packages — cells, elements, keys, geometry, errors |
| **emulator** | Grid data structure, custom ANSI/VTE parser (~30 escape sequences), PTY management via `creack/pty` |
| **detector** | 8 pattern-matching detectors that scan the grid for UI elements, plus a pipeline that orchestrates them with priority ordering and region claiming |
| **session** | Individual terminal sessions with background PTY reading, snapshot capture, and wait-for-condition polling |
| **server** | MCP tool definitions and handlers using `mark3labs/mcp-go` |

## Testing

```bash
# Run all tests
go test ./... -timeout 120s

# Run with verbose output
go test ./... -v -timeout 120s

# Run specific package
go test ./detector/ -v
go test ./emulator/ -v
go test ./session/ -v -timeout 60s
```

**154 tests** across all packages — unit tests for parsing, grid operations, and element detection, plus end-to-end tests that spawn real processes and verify snapshots.

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) | MCP SDK (stdio server, tool registration) |
| [creack/pty](https://github.com/creack/pty) | PTY management |
| [google/uuid](https://github.com/google/uuid) | Session IDs |
| [stretchr/testify](https://github.com/stretchr/testify) | Testing |

No CGo. No external VTE library. The ANSI parser is custom-built (~400 lines).

## License

MIT
