package utils

import (
	"fmt"
	"sync"
)

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
