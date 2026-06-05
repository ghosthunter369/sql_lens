package analyzer

import (
	"regexp"
	"strings"
	"sql-lens/internal/model"
)

// ASTTransformer transforms parse results to the unified analysis model.
type ASTTransformer struct{}

func NewASTTransformer() *ASTTransformer {
	return &ASTTransformer{}
}

func (tr *ASTTransformer) Transform(result *model.SQLAnalysisResult) (*model.SQLAnalysisResult, error) {
	if result.FormattedSQL == "" && result.RestoredSQL != "" {
		result.FormattedSQL = FormatSQL(result.RestoredSQL)
	}
	return result, nil
}

// FormatSQL does simple SQL keyword capitalization and indentation.
func FormatSQL(sql string) string {
	if sql == "" {
		return ""
	}

	sql = strings.TrimSpace(sql)

	// Capitalize major keywords
	keywordPatterns := []string{
		"SELECT", "FROM", "WHERE", "AND", "OR",
		"LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "JOIN", "CROSS JOIN",
		"ON", "GROUP BY", "ORDER BY", "LIMIT", "OFFSET", "HAVING",
		"AS", "IN", "LIKE", "BETWEEN", "IS NULL", "IS NOT NULL",
		"NOT IN", "DISTINCT", "CASE WHEN", "INSERT INTO", "VALUES",
		"UPDATE", "DELETE FROM", "SET", "CREATE TABLE", "ALTER TABLE",
		"DROP TABLE", "UNION ALL", "UNION",
	}

	result := sql
	for _, kw := range keywordPatterns {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kw) + `\b`)
		result = re.ReplaceAllString(result, kw)
	}

	// Add newlines before major clauses
	newlineBeforeList := []string{
		"LEFT JOIN", "RIGHT JOIN", "INNER JOIN", "JOIN", "CROSS JOIN",
		"WHERE", "GROUP BY", "ORDER BY", "LIMIT", "HAVING", "UNION", "UNION ALL",
	}
	for _, kw := range newlineBeforeList {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(kw) + `\b`)
		result = re.ReplaceAllString(result, "\n"+kw)
	}

	// Indent AND/OR within WHERE
	result = regexp.MustCompile(`(?i)\b(AND)\b`).ReplaceAllString(result, "\n  AND")
	result = regexp.MustCompile(`(?i)\b(OR)\b`).ReplaceAllString(result, "\n   OR")

	// Clean up extra whitespace
	result = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(result, "\n")
	result = strings.TrimSpace(result)

	return result
}
