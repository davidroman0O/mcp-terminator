package session

import (
	"sync"

	"github.com/davidroman0O/mcp-terminator/core"
)

// Manager is a session registry that enforces a maximum session limit.
type Manager struct {
	sessions    map[core.SessionID]*Session
	mu          sync.RWMutex
	maxSessions int
}

// NewManager creates a Manager with the given maximum session count.
func NewManager(maxSessions int) *Manager {
	if maxSessions <= 0 {
		maxSessions = 10
	}
	return &Manager{
		sessions:    make(map[core.SessionID]*Session),
		maxSessions: maxSessions,
	}
}

// Create creates a new session with the given configuration and registers it.
// Returns SessionLimitReachedError if the maximum number of sessions is exceeded.
func (m *Manager) Create(cfg core.SessionConfig) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSessions {
		return nil, &core.SessionLimitReachedError{Max: m.maxSessions}
	}

	s, err := NewSession(cfg)
	if err != nil {
		return nil, err
	}

	m.sessions[s.ID()] = s
	return s, nil
}

// Get retrieves a session by ID. Returns SessionNotFoundError if not found.
func (m *Manager) Get(id core.SessionID) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	s, ok := m.sessions[id]
	if !ok {
		return nil, &core.SessionNotFoundError{ID: id}
	}
	return s, nil
}

// List returns info about all registered sessions.
func (m *Manager) List() []core.SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]core.SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		infos = append(infos, core.SessionInfo{
			ID:     s.ID(),
			Status: s.Status(),
			Config: s.Config(),
		})
	}
	return infos
}

// Close terminates a session by ID and removes it from the registry.
func (m *Manager) Close(id core.SessionID, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return &core.SessionNotFoundError{ID: id}
	}
	err := s.Close(force)
	delete(m.sessions, id)
	return err
}

// CloseAll terminates all sessions and clears the registry.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, s := range m.sessions {
		_ = s.Close(true)
		delete(m.sessions, id)
	}
}

// Count returns the number of active sessions.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}
