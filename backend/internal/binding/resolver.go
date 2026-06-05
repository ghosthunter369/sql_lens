package binding

import (
	"errors"
	"fmt"
	"strconv"
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
		// Try to detect numeric strings and booleans
		if v == "" {
			return "''"
		}
		// Check if it's a numeric value (int or float)
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			// Check if it's an integer
			if f == float64(int64(f)) {
				return fmt.Sprintf("%d", int64(f))
			}
			return fmt.Sprintf("%v", f)
		}
		// Check if it's a boolean
		upper := strings.ToUpper(v)
		if upper == "TRUE" {
			return "1"
		}
		if upper == "FALSE" {
			return "0"
		}
		// It's a string value
		escaped := strings.ReplaceAll(v, "'", "''")
		return "'" + escaped + "'"
	case bool:
		if v {
			return "1"
		}
		return "0"
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case float32:
		if float64(v) == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
