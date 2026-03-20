package main

import (
	"log"
	"os"
	"strconv"

	"github.com/davidroman0O/mcp-terminator/server"
)

func main() {
	maxSessions := 16
	if raw := os.Getenv("MCP_TERMINATOR_MAX_SESSIONS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			maxSessions = n
		}
	}

	if err := server.New(maxSessions).Run(); err != nil {
		log.Fatal(err)
	}
}
