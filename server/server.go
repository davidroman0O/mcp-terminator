// Package server provides the MCP protocol layer for mcp-terminator.
//
// It registers 10 tools over stdio transport using the mcp-go SDK,
// bridging MCP tool calls to the session management layer.
package server

import (
	"log"
	"os"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/davidroman0O/mcp-terminator/session"
)

// Server is the MCP server for terminal session management.
type Server struct {
	manager *session.Manager
	mcp     *mcpserver.MCPServer
}

// New creates a new MCP server backed by a session manager with the given
// maximum number of concurrent sessions.
func New(maxSessions int) *Server {
	s := &Server{
		manager: session.NewManager(maxSessions),
		mcp: mcpserver.NewMCPServer(
			"mcp-terminator",
			"0.1.0",
			mcpserver.WithToolCapabilities(true),
		),
	}
	s.registerTools()
	return s
}

// Run starts the MCP server on stdio and blocks until the client disconnects
// or a signal is received.
func (s *Server) Run() error {
	return mcpserver.ServeStdio(s.mcp,
		mcpserver.WithErrorLogger(log.New(os.Stderr, "[mcp-terminator] ", log.LstdFlags)),
	)
}
