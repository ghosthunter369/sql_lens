package extractor

import (
	"errors"
	"regexp"
	"strings"
)

type ExtractResult struct {
	SQL      string        `json:"sql"`
	Bindings []interface{} `json:"bindings"`
	LogType  string        `json:"logType"`
}

type LogExtractor interface {
	Match(raw string) bool
	Extract(raw string) (*ExtractResult, error)
}

func AutoExtract(raw string) (*ExtractResult, error) {
	extractors := []LogExtractor{
		&MyBatisExtractor{},
		&LaravelExtractor{},
		&ThinkPHPExtractor{},
		&PlainSQLExtractor{},
	}

	for _, e := range extractors {
		if e.Match(raw) {
			return e.Extract(raw)
		}
	}

	return nil, errors.New("unsupported log format")
}

// PlainSQLExtractor handles raw SQL text
type PlainSQLExtractor struct{}

func (e *PlainSQLExtractor) Match(raw string) bool {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	return strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "INSERT") ||
		strings.HasPrefix(upper, "UPDATE") ||
		strings.HasPrefix(upper, "DELETE") ||
		strings.HasPrefix(upper, "WITH") ||
		strings.HasPrefix(upper, "CREATE") ||
		strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP")
}

func (e *PlainSQLExtractor) Extract(raw string) (*ExtractResult, error) {
	return &ExtractResult{
		SQL:      strings.TrimSpace(raw),
		Bindings: nil,
		LogType:  "plain",
	}, nil
}

// MyBatisExtractor handles MyBatis log format
type MyBatisExtractor struct{}

var mybatisPreparingRe = regexp.MustCompile(`(?i)Preparing:\s*(.+)`)
var mybatisParamsRe = regexp.MustCompile(`(?i)Parameters:\s*(.+)`)

func (e *MyBatisExtractor) Match(raw string) bool {
	return strings.Contains(raw, "Preparing:") || strings.Contains(raw, "==>")
}

func (e *MyBatisExtractor) Extract(raw string) (*ExtractResult, error) {
	sqlMatch := mybatisPreparingRe.FindStringSubmatch(raw)
	paramsMatch := mybatisParamsRe.FindStringSubmatch(raw)

	sql := ""
	if sqlMatch != nil {
		sql = strings.TrimSpace(sqlMatch[1])
	}

	var bindings []interface{}
	if paramsMatch != nil {
		paramsStr := paramsMatch[1]
		parts := strings.Split(paramsStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			// Remove type hints like (Long), (Integer), (String)
			p = regexp.MustCompile(`\(.*\)$`).ReplaceAllString(p, "")
			p = strings.TrimSpace(p)
			// Try to parse as number
			bindings = append(bindings, parseValue(p))
		}
	}

	if sql == "" {
		// Try to extract SQL from the raw text
		lines := strings.Split(raw, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "SELECT") ||
				strings.HasPrefix(upper, "INSERT") ||
				strings.HasPrefix(upper, "UPDATE") ||
				strings.HasPrefix(upper, "DELETE") {
				sql = line
				break
			}
		}
	}

	if sql == "" {
		return nil, errors.New("could not extract SQL from MyBatis log")
	}

	return &ExtractResult{
		SQL:      sql,
		Bindings: bindings,
		LogType:  "mybatis",
	}, nil
}

func parseValue(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "null" || s == "NULL" || s == "" {
		return nil
	}
	// Remove surrounding quotes
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}
	// Try to parse as number - just return as string for simplicity
	// The binding resolver will handle formatting
	return s
}
