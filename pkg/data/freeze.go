package data

import (
	"sync/atomic"
	"time"
)

// FreezeToken is an opaque handle returned by Freeze. Pass it to Unfreeze
// to release the corresponding freeze. Tokens are unique and monotonically
// increasing.
type FreezeToken uint64

var nextToken uint64

func newFreezeToken() FreezeToken {
	return FreezeToken(atomic.AddUint64(&nextToken, 1))
}

// frozenState holds the frozen snapshot and any data that arrived while the
// series was frozen. Reference-counted: multiple callers can freeze the same
// series and data stays frozen until all of them unfreeze.
type frozenState struct {
	snapshot      *SeriesSnapshot
	pendingTimes  []time.Time
	pendingValues []float64
	count         int            // reference count
	tokens        map[FreezeToken]bool
}

// Freeze freezes the specified series (or all series if no names are given).
// While frozen, GetSeries returns the frozen snapshot and AddPoint buffers
// new data. Returns a FreezeToken that must be passed to Unfreeze.
func (s *Store) Freeze(names ...string) FreezeToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := newFreezeToken()

	// If no names specified, freeze all existing series.
	if len(names) == 0 {
		names = make([]string, 0, len(s.series))
		for name := range s.series {
			names = append(names, name)
		}
	}

	for _, name := range names {
		ser, ok := s.series[name]
		if !ok {
			continue
		}

		fs, exists := s.frozen[name]
		if !exists {
			fs = &frozenState{
				snapshot: snapshotFrom(ser),
				tokens:   make(map[FreezeToken]bool),
			}
			s.frozen[name] = fs
		}
		fs.count++
		fs.tokens[token] = true
	}

	return token
}

// Unfreeze releases a freeze identified by token. If the reference count for
// a series drops to zero, pending buffered data is merged back into the live
// series.
func (s *Store) Unfreeze(token FreezeToken) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, fs := range s.frozen {
		if !fs.tokens[token] {
			continue
		}
		delete(fs.tokens, token)
		fs.count--

		if fs.count <= 0 {
			// Merge pending data back into the live series.
			ser := s.getOrCreate(name)
			if len(fs.pendingTimes) > 0 {
				ser.Times = append(ser.Times, fs.pendingTimes...)
				ser.Values = append(ser.Values, fs.pendingValues...)
				s.enforceMaxPoints(ser)
			}
			delete(s.frozen, name)
		}
	}
}

// IsFrozen reports whether the named series is currently frozen.
func (s *Store) IsFrozen(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fs, ok := s.frozen[name]
	return ok && fs.count > 0
}
