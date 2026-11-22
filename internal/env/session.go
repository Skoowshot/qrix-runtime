package env

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// SessionToken authenticates an Environment session.
type SessionToken string

// GenerateSessionToken returns a cryptographically secure random SessionToken.
func GenerateSessionToken() SessionToken {
	var b [32]byte
	_, _ = rand.Read(b[:])
	return SessionToken(base64.RawURLEncoding.EncodeToString(b[:]))
}

// Session represents an active Environment session.
type Session struct{}

// Close terminates the session and releases its resources.
func (s *Session) Close() error {
	return nil
}

type pendingSession struct {
	ttlTimer *time.Timer
	metadata Metadata
}

func (s *pendingSession) Close() error {
	s.ttlTimer.Stop()
	return nil
}

// SessionRegistry tracks pending and active Environment sessions with name and token indexes.
type SessionRegistry struct {
	nextID uint64

	pending    map[ID]*pendingSession
	pendingTTL time.Duration
	active     map[ID]*Session

	nameToID  map[Name]ID
	tokenToID map[SessionToken]ID
	idToName  map[ID]Name
	idToToken map[ID]SessionToken

	closed bool
	mu     sync.RWMutex
}

// NewSessionRegistry creates a SessionRegistry whose pending sessions expire after pendingTTL.
func NewSessionRegistry(pendingTTL time.Duration) *SessionRegistry {
	return &SessionRegistry{
		pending:    make(map[ID]*pendingSession),
		pendingTTL: pendingTTL,
		active:     make(map[ID]*Session),

		nameToID:  make(map[Name]ID),
		tokenToID: make(map[SessionToken]ID),
		idToName:  make(map[ID]Name),
		idToToken: make(map[ID]SessionToken),
	}
}

// PutPending registers a pending session with the given name, token, and metadata,
// starting an expiration timer for pendingTTL.
func (r *SessionRegistry) PutPending(name Name, token SessionToken, metadata Metadata) (ID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, ErrClosed
	}

	if _, exists := r.nameToID[name]; exists {
		return 0, fmt.Errorf("%w: %s", ErrNameAlreadyExists, name)
	}
	if _, exists := r.tokenToID[token]; exists {
		return 0, ErrTokenAlreadyExists
	}

	r.nextID++
	id := ID(r.nextID)
	ps := &pendingSession{
		ttlTimer: time.AfterFunc(r.pendingTTL, func() {
			r.RemovePending(id)
		}),
		metadata: metadata,
	}

	r.pending[id] = ps
	r.nameToID[name] = id
	r.tokenToID[token] = id
	r.idToName[id] = name
	r.idToToken[id] = token
	return id, nil
}

// RemovePending closes and removes a pending session and its indexes by ID.
func (r *SessionRegistry) RemovePending(id ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}

	p, ok := r.pending[id]
	if !ok {
		return
	}

	_ = p.Close()
	delete(r.pending, id)
	r.deleteIndexes(id)
}

// Close shuts down the registry, closing all pending and active sessions and joining any errors.
func (r *SessionRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}

	var errs []error

	if err := r.closePending(); err != nil {
		errs = append(errs, err)
	}
	if err := r.closeActive(); err != nil {
		errs = append(errs, err)
	}

	r.pending = nil
	r.active = nil
	r.nameToID = nil
	r.tokenToID = nil
	r.idToName = nil
	r.idToToken = nil
	r.closed = true
	return errors.Join(errs...)
}

func (r *SessionRegistry) closePending() error {
	var errs []error

	for _, s := range r.pending {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *SessionRegistry) closeActive() error {
	var errs []error

	for _, s := range r.active {
		if err := s.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (r *SessionRegistry) deleteIndexes(id ID) {
	if name, ok := r.idToName[id]; ok {
		delete(r.nameToID, name)
		delete(r.idToName, id)
	}
	if token, ok := r.idToToken[id]; ok {
		delete(r.tokenToID, token)
		delete(r.idToToken, id)
	}
}
