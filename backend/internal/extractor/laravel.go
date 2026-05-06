package extractor

import (
	"errors"
	"regexp"
	"strings"
)

type LaravelExtractor struct{}

var laravelSQLRe = regexp.MustCompile(`(?i)SQL:\s*(.+)`)
var laravelBindingsRe = regexp.MustCompile(`(?i)(?:Bindings|bindings):\s*\[(.*?)\]`)

func (e *LaravelExtractor) Match(raw string) bool {
	return strings.Contains(raw, "SQL:") &&
		(strings.Contains(raw, "Bindings:") || strings.Contains(raw, "bindings:"))
}

func (e *LaravelExtractor) Extract(raw string) (*ExtractResult, error) {
	sqlMatch := laravelSQLRe.FindStringSubmatch(raw)
	bindingsMatch := laravelBindingsRe.FindStringSubmatch(raw)

	sql := ""
	if sqlMatch != nil {
		sql = strings.TrimSpace(sqlMatch[1])
	}

	var bindings []interface{}
	if bindingsMatch != nil {
		bindingsStr := bindingsMatch[1]
		parts := strings.Split(bindingsStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			bindings = append(bindings, parseValue(p))
		}
	}

	if sql == "" {
		return nil, errors.New("could not extract SQL from Laravel log")
	}

	// Remove time info from end like {"time":"0.23"}
	if idx := strings.LastIndex(sql, "{"); idx > 0 {
		sql = strings.TrimSpace(sql[:idx])
	}

	return &ExtractResult{
		SQL:      sql,
		Bindings: bindings,
		LogType:  "laravel",
	}, nil
}
