package session

import (
	"context"
	"errors"
	"sync"
)

type Persister interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	ListSessionsByIdentity(ctx context.Context, identityID string) ([]*Session, error)
}

type MemoryPersister struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewMemoryPersister() *MemoryPersister {
	return &MemoryPersister{
		sessions: make(map[string]*Session),
	}
}

func (m *MemoryPersister) CreateSession(ctx context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := *s
	m.sessions[s.ID] = &copied
	return nil
}

func (m *MemoryPersister) GetSession(ctx context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	copied := *s
	return &copied, nil
}

func (m *MemoryPersister) DeleteSession(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	return nil
}

func (m *MemoryPersister) ListSessionsByIdentity(ctx context.Context, identityID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var res []*Session
	for _, s := range m.sessions {
		if s.IdentityID == identityID {
			copied := *s
			res = append(res, &copied)
		}
	}
	return res, nil
}
