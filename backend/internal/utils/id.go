package utils

import "fmt"

var idCounters = make(map[string]int)

func NewID(prefix string) string {
	idCounters[prefix]++
	return fmt.Sprintf("%s-%d", prefix, idCounters[prefix])
}

func ResetIDs() {
	idCounters = make(map[string]int)
}
