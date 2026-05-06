package extractor

import (
	"errors"
	"regexp"
	"strings"
)

type ThinkPHPExtractor struct{}

var thinkphpSQLRe = regexp.MustCompile(`(?i)\[ SQL \]\s*(.+)`)

func (e *ThinkPHPExtractor) Match(raw string) bool {
	return strings.Contains(strings.ToUpper(raw), "[ SQL ]")
}

func (e *ThinkPHPExtractor) Extract(raw string) (*ExtractResult, error) {
	match := thinkphpSQLRe.FindStringSubmatch(raw)
	if match == nil {
		return nil, errors.New("could not extract SQL from ThinkPHP log")
	}

	sql := strings.TrimSpace(match[1])

	return &ExtractResult{
		SQL:      sql,
		Bindings: nil,
		LogType:  "thinkphp",
	}, nil
}
