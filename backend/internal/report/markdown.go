package report

import (
	"fmt"
	"sql-lens/internal/model"
	"strings"
)

func BuildMarkdownReport(result *model.SQLAnalysisResult) string {
	var b strings.Builder

	b.WriteString("# SQL 解析报告\n\n")

	// SQL Type
	b.WriteString("## SQL 类型\n\n")
	b.WriteString(fmt.Sprintf("%s\n\n", result.StatementType))

	// Original SQL
	if result.RawSQL != "" {
		b.WriteString("## 原始 SQL\n\n")
		b.WriteString("```sql\n")
		b.WriteString(result.RawSQL)
		b.WriteString("\n```\n\n")
	}

	// Restored SQL (if different)
	if result.RestoredSQL != "" && result.RestoredSQL != result.RawSQL {
		b.WriteString("## 参数回填 SQL\n\n")
		b.WriteString("```sql\n")
		b.WriteString(result.RestoredSQL)
		b.WriteString("\n```\n\n")
	}

	// Summary
	b.WriteString("## 概览\n\n")
	b.WriteString("| 指标 | 值 |\n")
	b.WriteString("|---|---|\n")
	b.WriteString(fmt.Sprintf("| SQL 类型 | %s |\n", result.StatementType))
	b.WriteString(fmt.Sprintf("| 涉及表数 | %d |\n", result.Summary.TableCount))
	b.WriteString(fmt.Sprintf("| JOIN 数量 | %d |\n", result.Summary.JoinCount))
	b.WriteString(fmt.Sprintf("| 查询字段数 | %d |\n", result.Summary.FieldCount))
	b.WriteString(fmt.Sprintf("| WHERE 条件数 | %d |\n", result.Summary.WhereCount))
	b.WriteString(fmt.Sprintf("| 是否有 GROUP BY | %s |\n", boolToStr(result.Summary.HasGroupBy)))
	b.WriteString(fmt.Sprintf("| 是否有 ORDER BY | %s |\n", boolToStr(result.Summary.HasOrderBy)))
	b.WriteString(fmt.Sprintf("| 是否有 LIMIT | %s |\n", boolToStr(result.Summary.HasLimit)))
	b.WriteString(fmt.Sprintf("| 复杂度 | %s |\n\n", complexityLabel(result.Summary.Complexity)))

	// Tables
	b.WriteString("## 涉及表\n\n")
	b.WriteString("| 表名 | 别名 | 角色 |\n")
	b.WriteString("|---|---|---|\n")
	for _, table := range result.Tables {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", table.Name, orNA(table.Alias), roleLabel(table.Role)))
	}
	b.WriteString("\n")

	// JOIN relations
	if len(result.Joins) > 0 {
		b.WriteString("## JOIN 关系\n\n")
		for _, join := range result.Joins {
			b.WriteString(fmt.Sprintf("- %s %s\n", join.Type, join.RightTable))
			for _, cond := range join.Conditions {
				b.WriteString(fmt.Sprintf("  - ON %s %s %s\n", cond.Left, cond.Operator, cond.Right))
			}
		}
		b.WriteString("\n")
	}

	// Fields
	b.WriteString("## 查询字段\n\n")
	if len(result.Fields) == 0 {
		b.WriteString("无查询字段\n\n")
	} else {
		b.WriteString("| 输出字段 | 来源表 | 表达式 | 类型 |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, field := range result.Fields {
			sourceTable := orNA(field.SourceTable)
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				field.OutputName, sourceTable, field.Expression, fieldTypeLabel(field.FieldType)))
		}
		b.WriteString("\n")
	}

	// WHERE conditions
	if result.WhereTree != nil {
		b.WriteString("## WHERE 条件\n\n")
		writeWhereTree(&b, result.WhereTree, 0)
		b.WriteString("\n")
	} else {
		b.WriteString("## WHERE 条件\n\n")
		b.WriteString("无 WHERE 条件\n\n")
	}

	// GROUP BY
	if len(result.GroupBy) > 0 {
		b.WriteString("## GROUP BY\n\n")
		for _, gb := range result.GroupBy {
			b.WriteString(fmt.Sprintf("- %s", gb.Expression))
			if gb.SourceTable != "" {
				b.WriteString(fmt.Sprintf(" (来源表: %s)", gb.SourceTable))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// ORDER BY
	if len(result.OrderBy) > 0 {
		b.WriteString("## ORDER BY\n\n")
		for _, ob := range result.OrderBy {
			b.WriteString(fmt.Sprintf("- %s %s", ob.Expression, ob.Direction))
			if ob.SourceTable != "" {
				b.WriteString(fmt.Sprintf(" (来源表: %s)", ob.SourceTable))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// LIMIT
	if result.Limit != nil {
		b.WriteString("## LIMIT\n\n")
		b.WriteString(fmt.Sprintf("LIMIT %d", result.Limit.Limit))
		if result.Limit.Offset > 0 {
			b.WriteString(fmt.Sprintf(" OFFSET %d", result.Limit.Offset))
		}
		b.WriteString("\n\n")
	}

	// Risks
	if len(result.Risks) > 0 {
		b.WriteString("## 风险提示\n\n")
		for _, risk := range result.Risks {
			levelEmoji := "ℹ️"
			switch risk.Level {
			case "warning":
				levelEmoji = "⚠️"
			case "danger":
				levelEmoji = "🚫"
			}
			b.WriteString(fmt.Sprintf("- %s **%s**: %s\n", levelEmoji, risk.Level, risk.Message))
			b.WriteString(fmt.Sprintf("  - 建议: %s\n", risk.Suggestion))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## 风险提示\n\n")
		b.WriteString("- 无明显高危风险\n\n")
	}

	return b.String()
}

func writeWhereTree(b *strings.Builder, node *model.ConditionNode, depth int) {
	if node == nil {
		return
	}
	indent := strings.Repeat("  ", depth)
	if node.Type == "CONDITION" {
		b.WriteString(fmt.Sprintf("%s- %s\n", indent, node.Expr))
		if node.Table != "" {
			b.WriteString(fmt.Sprintf("%s  (来源表: %s)\n", indent, node.Table))
		}
	} else {
		b.WriteString(fmt.Sprintf("%s- **%s**\n", indent, node.Type))
		for _, child := range node.Children {
			writeWhereTree(b, child, depth+1)
		}
	}
}

func boolToStr(b bool) string {
	if b {
		return "是"
	}
	return "否"
}

func orNA(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func roleLabel(role string) string {
	switch role {
	case "main":
		return "主表"
	case "joined":
		return "JOIN 表"
	case "subquery":
		return "子查询表"
	case "cte":
		return "CTE 表"
	default:
		return role
	}
}

func complexityLabel(c string) string {
	switch c {
	case "LOW":
		return "低"
	case "MEDIUM":
		return "中等"
	case "HIGH":
		return "高"
	default:
		return c
	}
}

func fieldTypeLabel(ft string) string {
	switch ft {
	case "column":
		return "普通字段"
	case "function":
		return "函数字段"
	case "aggregate":
		return "聚合字段"
	case "case":
		return "CASE 字段"
	case "wildcard":
		return "通配字段"
	case "subquery":
		return "子查询字段"
	default:
		return ft
	}
}
