package main

import (
	"fmt"
	"sync"
	"time"
)

// Session holds the state for a single ACP session.
// agy-acp-bridge supports one active session at a time.
// Continuity is provided via `agy --continue` which resumes the last conversation.
type Session struct {
	ID        string
	Cwd       string
	CreatedAt time.Time
	// hasHistory tracks whether this session has sent at least one prompt,
	// enabling --continue on subsequent prompts.
	HasHistory bool
}

type SessionStore struct {
	mu      sync.Mutex
	current *Session
}

var store = &SessionStore{}

// NewSession creates a new session, replacing any existing one.
func (s *SessionStore) NewSession(cwd string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess := &Session{
		ID:        generateSessionID(),
		Cwd:       cwd,
		CreatedAt: time.Now(),
	}
	s.current = sess
	return sess
}

// Get returns the current session, or nil if none exists.
func (s *SessionStore) Get(id string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && s.current.ID == id {
		return s.current
	}
	return nil
}

// MarkHistory marks the current session as having history (used --continue next time).
func (s *SessionStore) MarkHistory(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && s.current.ID == id {
		s.current.HasHistory = true
	}
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
