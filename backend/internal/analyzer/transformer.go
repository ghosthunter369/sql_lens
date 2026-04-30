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
	// Table field usage is now populated by parser.populateTableFields

	if result.FormattedSQL == "" && result.RestoredSQL != "" {
		result.FormattedSQL = formatSQL(result.RestoredSQL)
	}

	return result, nil
}

func (tr *ASTTransformer) enrichTableFieldUsage(result *model.SQLAnalysisResult) {
	tableMap := make(map[string]*model.TableMeta)
	for i := range result.Tables {
		tableMap[result.Tables[i].Name] = &result.Tables[i]
		if result.Tables[i].Alias != "" {
			tableMap[result.Tables[i].Alias] = &result.Tables[i]
		}
	}

	// Track selected fields per table
	for _, field := range result.Fields {
		if field.SourceTable != "" {
			if t, ok := tableMap[field.SourceTable]; ok {
				t.SelectedFields = append(t.SelectedFields, field.OutputName)
			}
		}
		if field.SourceAlias != "" {
			if t, ok := tableMap[field.SourceAlias]; ok {
				if !containsStr(t.SelectedFields, field.OutputName) {
					t.SelectedFields = append(t.SelectedFields, field.OutputName)
				}
			}
		}
	}

	// Track filter fields per table (from WHERE)
	if result.WhereTree != nil {
		collectWhereTables(result.WhereTree, tableMap)
	}

	// Track join fields per table
	for _, join := range result.Joins {
		for _, cond := range join.Conditions {
			if t, ok := tableMap[join.LeftTable]; ok {
				t.JoinFields = append(t.JoinFields, cond.Left, cond.Right)
			}
			if t, ok := tableMap[join.RightTable]; ok {
				t.JoinFields = append(t.JoinFields, cond.Left, cond.Right)
			}
		}
	}
}

func collectWhereTables(node *model.ConditionNode, tableMap map[string]*model.TableMeta) {
	if node == nil {
		return
	}
	if node.Type == "CONDITION" && node.Table != "" {
		if t, ok := tableMap[node.Table]; ok {
			t.FilterFields = append(t.FilterFields, node.Field)
		}
	}
	for _, child := range node.Children {
		collectWhereTables(child, tableMap)
	}
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// formatSQL does simple SQL keyword capitalization and indentation
func formatSQL(sql string) string {
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
