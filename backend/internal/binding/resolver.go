package binding

import (
	"errors"
	"fmt"
	"strings"
)

func RestoreBindings(sql string, bindings []interface{}) (string, error) {
	var builder strings.Builder
	bindIndex := 0

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch == '?' {
			if bindIndex >= len(bindings) {
				return "", errors.New("binding count mismatch: more placeholders than bindings")
			}
			builder.WriteString(formatBinding(bindings[bindIndex]))
			bindIndex++
			continue
		}
		builder.WriteByte(ch)
	}

	if bindIndex != len(bindings) {
		return "", fmt.Errorf("binding count mismatch: placeholder count: %d, binding count: %d", bindIndex, len(bindings))
	}

	return builder.String(), nil
}

func formatBinding(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		escaped := strings.ReplaceAll(v, "'", "''")
		return "'" + escaped + "'"
	case bool:
		if v {
			return "1"
		}
		return "0"
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
