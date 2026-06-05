package utils

import (
	"fmt"
	"sync"
)

// IDGenerator is a request-scoped ID counter. Each parse request gets its own
// instance so concurrent requests never race on shared counters.
type IDGenerator struct {
	mu       sync.Mutex
	counters map[string]int
}

// NewIDGenerator creates a fresh ID generator for a single request.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{counters: make(map[string]int)}
}

// NewID returns a unique ID with the given prefix (e.g. "table-1").
func (g *IDGenerator) NewID(prefix string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counters[prefix]++
	return fmt.Sprintf("%s-%d", prefix, g.counters[prefix])
}

// --- Legacy global ID generator (kept for backward compatibility) ---

var (
	idCounters = make(map[string]int)
	idMu       sync.Mutex
)

func NewID(prefix string) string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounters[prefix]++
	return fmt.Sprintf("%s-%d", prefix, idCounters[prefix])
}

func ResetIDs() {
	idMu.Lock()
	defer idMu.Unlock()
	idCounters = make(map[string]int)
}
