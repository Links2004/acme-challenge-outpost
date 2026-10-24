package store

import "sync"

// Store keeps acme token -> keyauthorization pairs in memory.
type Store struct {
	mu     sync.RWMutex
	tokens map[string]string
}

func New() *Store {
	return &Store{tokens: map[string]string{}}
}

func (s *Store) Set(token, keyAuth string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = keyAuth
}

func (s *Store) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.tokens[token]
	return v, ok
}

func (s *Store) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}
