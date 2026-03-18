package emulator

import (
	"testing"
	"time"

	"github.com/davidroman0O/mcp-terminator/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPtySpawnEcho(t *testing.T) {
	h, err := Spawn("/bin/echo", []string{"hello"}, core.NewDimensions(24, 80), "", nil)
	require.NoError(t, err)
	defer h.Close()

	// Give the process time to produce output.
	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 4096)
	n, err := h.Read(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0, "should have read some output")
	assert.Contains(t, string(buf[:n]), "hello")
}

func TestPtySpawnAndIsAlive(t *testing.T) {
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "", nil)
	require.NoError(t, err)

	assert.True(t, h.IsAlive())

	h.Close()
	// After close, process should be dead.
	time.Sleep(100 * time.Millisecond)
	assert.False(t, h.IsAlive())
}

func TestPtyWriteRead(t *testing.T) {
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "", nil)
	require.NoError(t, err)
	defer h.Close()

	// Write a command.
	_, err = h.Write([]byte("echo testoutput\n"))
	require.NoError(t, err)

	// Wait for output.
	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 4096)
	n, err := h.Read(buf)
	require.NoError(t, err)
	assert.Greater(t, n, 0)
	assert.Contains(t, string(buf[:n]), "testoutput")
}

func TestPtyResize(t *testing.T) {
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "", nil)
	require.NoError(t, err)
	defer h.Close()

	err = h.Resize(core.NewDimensions(40, 120))
	assert.NoError(t, err)
}

func TestPtyClose(t *testing.T) {
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "", nil)
	require.NoError(t, err)

	err = h.Close()
	assert.NoError(t, err)

	// Double close should be safe.
	err = h.Close()
	assert.NoError(t, err)
}

func TestPtyCwd(t *testing.T) {
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "/tmp", nil)
	require.NoError(t, err)
	defer h.Close()

	_, err = h.Write([]byte("pwd\n"))
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 4096)
	n, err := h.Read(buf)
	require.NoError(t, err)
	// The output should contain /tmp (or /private/tmp on macOS).
	output := string(buf[:n])
	assert.True(t, containsAny(output, "/tmp", "/private/tmp"), "expected /tmp in output, got: %s", output)
}

func TestPtyEnv(t *testing.T) {
	env := map[string]string{"MY_TEST_VAR": "hello_from_test"}
	h, err := Spawn("/bin/sh", nil, core.NewDimensions(24, 80), "", env)
	require.NoError(t, err)
	defer h.Close()

	_, err = h.Write([]byte("echo $MY_TEST_VAR\n"))
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	buf := make([]byte, 4096)
	n, err := h.Read(buf)
	require.NoError(t, err)
	assert.Contains(t, string(buf[:n]), "hello_from_test")
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
