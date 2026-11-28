package env

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Skoowshot/qrix-runtime/internal/rpc"
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

type PendingSession struct {
	ttlTimer *time.Timer
	metadata Metadata

	callIn    <-chan rpc.CallFrame
	callOut   chan rpc.CallFrame
	streamIn  <-chan rpc.StreamFrame
	streamOut chan rpc.StreamFrame
}

func (s *PendingSession) AttachCall(in <-chan rpc.CallFrame, out chan rpc.CallFrame) {
	s.callIn = in
	s.callOut = out
}

func (s *PendingSession) AttachStream(in <-chan rpc.StreamFrame, out chan rpc.StreamFrame) {
	s.streamIn = in
	s.streamOut = out
}

func (s *PendingSession) Close() error {
	s.ttlTimer.Stop() //todo: add cancellation signal
	return nil
}

func (s *PendingSession) tryCompleteConnection() {
	if s.callIn == nil || s.callOut == nil {
		return
	}
	if s.streamIn == nil || s.streamOut == nil {
		return
	}
	s.ttlTimer.Stop()
	//todo: put a session and remove this pending session in registry
}

// SessionRegistry tracks pending and active Environment sessions with name and token indexes.
type SessionRegistry struct {
	nextID uint64

	pending    map[ID]*PendingSession
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
		pending:    make(map[ID]*PendingSession),
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
	ps := &PendingSession{
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

// GetPending returns a pending session by the given ID.
func (r *SessionRegistry) GetPending(id ID) (*PendingSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.pending[id]
	return p, ok
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

// GetByToken returns an ID by the given SessionToken.
func (r *SessionRegistry) GetByToken(token SessionToken) (ID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.tokenToID[token]
	return id, ok
}

// AttachCall retrieves pending session by the given token, and attaches the call io channels.
func (r *SessionRegistry) AttachCall(token SessionToken, in <-chan rpc.CallFrame, out chan rpc.CallFrame) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.tokenToID[token]
	if !ok {
		return ErrSessionInvalid
	}

	sess, ok := r.pending[id]
	if !ok {
		return ErrSessionNotPending
	}

	sess.AttachCall(in, out)
	return nil
}

// AttachStream retrieves pending session by the given token, and attaches the stream io channels.
func (r *SessionRegistry) AttachStream(token SessionToken, in <-chan rpc.StreamFrame, out chan rpc.StreamFrame) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.tokenToID[token]
	if !ok {
		return ErrSessionInvalid
	}
	sess, ok := r.pending[id]
	if !ok {
		return ErrSessionNotPending
	}

	sess.AttachStream(in, out)
	return nil
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
