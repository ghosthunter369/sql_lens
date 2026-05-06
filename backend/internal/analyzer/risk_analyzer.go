package analyzer

import (
	"fmt"
	"sql-lens/internal/model"
	"sql-lens/internal/utils"
	"strings"
)

type RiskRule interface {
	Check(result *model.SQLAnalysisResult) []model.RiskMeta
}

var rules = []RiskRule{
	&SelectStarRule{},
	&NoWhereUpdateDeleteRule{},
	&LikePrefixWildcardRule{},
	&WhereFunctionRule{},
	&TooManyJoinRule{},
	&OrderByRandRule{},
	&TooManyORRule{},
	&SubqueryDepthRule{},
	&WindowFuncWithoutOrderByRule{},
	&CTENoTerminationRule{},
}

func AnalyzeRisks(result *model.SQLAnalysisResult) []model.RiskMeta {
	risks := make([]model.RiskMeta, 0)
	for _, rule := range rules {
		risks = append(risks, rule.Check(result)...)
	}
	return risks
}

// SELECT * rule
type SelectStarRule struct{}

func (r *SelectStarRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	var risks []model.RiskMeta
	for _, field := range result.Fields {
		if field.FieldType == "wildcard" {
			risks = append(risks, model.RiskMeta{
				ID:          utils.NewID("risk"),
				Level:       "warning",
				Type:        "SELECT_STAR",
				Message:     "当前 SQL 使用了 SELECT *",
				Suggestion:  "建议明确指定查询字段，减少不必要的数据传输。",
				RelatedExpr: field.Expression,
			})
		}
	}
	return risks
}

// No WHERE on UPDATE/DELETE
type NoWhereUpdateDeleteRule struct{}

func (r *NoWhereUpdateDeleteRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	stmtType := strings.ToUpper(result.StatementType)
	if stmtType != "UPDATE" && stmtType != "DELETE" {
		return nil
	}
	if result.WhereTree == nil {
		return []model.RiskMeta{
			{
				ID:         utils.NewID("risk"),
				Level:      "danger",
				Type:       "NO_WHERE_" + stmtType,
				Message:    stmtType + " 语句没有 WHERE 条件",
				Suggestion: "禁止执行没有 WHERE 条件的 " + stmtType + " 语句，可能导致全表数据被修改/删除。",
			},
		}
	}
	return nil
}

// LIKE prefix wildcard
type LikePrefixWildcardRule struct{}

func (r *LikePrefixWildcardRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	var risks []model.RiskMeta
	if result.WhereTree == nil {
		return risks
	}
	checkLikeNode(result.WhereTree, &risks)
	return risks
}

func checkLikeNode(node *model.ConditionNode, risks *[]model.RiskMeta) {
	if node == nil {
		return
	}
	if node.Type == "CONDITION" && (node.Operator == "LIKE" || node.Operator == "NOT LIKE") {
		val := strings.Trim(node.Value, "'\"")
		if strings.HasPrefix(val, "%") || strings.HasPrefix(val, "_") {
			*risks = append(*risks, model.RiskMeta{
				ID:          utils.NewID("risk"),
				Level:       "warning",
				Type:        "LIKE_PREFIX_WILDCARD",
				Message:     "LIKE 使用了前缀通配符: " + node.Expr,
				Suggestion:  "前缀通配符会导致索引失效，建议检查是否可以使用后缀通配符替代。",
				RelatedExpr: node.Expr,
			})
		}
	}
	for _, child := range node.Children {
		checkLikeNode(child, risks)
	}
}

// WHERE field uses function
type WhereFunctionRule struct{}

func (r *WhereFunctionRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	var risks []model.RiskMeta
	if result.WhereTree == nil {
		return risks
	}
	checkWhereFuncNode(result.WhereTree, &risks)
	return risks
}

func checkWhereFuncNode(node *model.ConditionNode, risks *[]model.RiskMeta) {
	if node == nil {
		return
	}
	if node.Type == "CONDITION" && strings.Contains(node.Field, "(") {
		*risks = append(*risks, model.RiskMeta{
			ID:          utils.NewID("risk"),
			Level:       "warning",
			Type:        "WHERE_FIELD_FUNCTION",
			Message:     "WHERE 条件中对字段使用了函数: " + node.Expr,
			Suggestion:  "对字段使用函数可能影响索引使用，建议尽量改为范围查询或在应用层处理。",
			RelatedExpr: node.Expr,
		})
	}
	for _, child := range node.Children {
		checkWhereFuncNode(child, risks)
	}
}

// Too many JOINs
type TooManyJoinRule struct{}

func (r *TooManyJoinRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	if result.Summary.JoinCount >= 4 {
		return []model.RiskMeta{
			{
				ID:         utils.NewID("risk"),
				Level:      "warning",
				Type:       "TOO_MANY_JOINS",
				Message:    fmt.Sprintf("当前 SQL 包含 %d 个 JOIN", result.Summary.JoinCount),
				Suggestion: "JOIN 数量过多会影响查询可读性和性能，建议检查是否可以拆分为多个简单查询。",
			},
		}
	}
	return nil
}

// ORDER BY RAND()
type OrderByRandRule struct{}

func (r *OrderByRandRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	for _, ob := range result.OrderBy {
		upper := strings.ToUpper(ob.Expression)
		if strings.Contains(upper, "RAND()") || strings.Contains(upper, "RAND ()") {
			return []model.RiskMeta{
				{
					ID:          utils.NewID("risk"),
					Level:       "danger",
					Type:        "ORDER_BY_RAND",
					Message:     "使用了 ORDER BY RAND()",
					Suggestion:  "ORDER BY RAND() 在大表上性能极差，建议在应用层实现随机排序。",
					RelatedExpr: ob.Expression,
				},
			}
		}
	}
	return nil
}

// Too many OR conditions
type TooManyORRule struct{}

func (r *TooManyORRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	if result.WhereTree == nil {
		return nil
	}
	orCount := countORNodes(result.WhereTree)
	if orCount >= 3 {
		return []model.RiskMeta{
			{
				ID:         utils.NewID("risk"),
				Level:      "info",
				Type:       "TOO_MANY_OR",
				Message:    "WHERE 条件中包含多个 OR 条件",
				Suggestion: "大量 OR 条件可能影响索引使用效率，检查是否可以改写为 UNION 或 IN。",
			},
		}
	}
	return nil
}

func countORNodes(node *model.ConditionNode) int {
	if node == nil {
		return 0
	}
	count := 0
	if node.Type == "OR" {
		count++
	}
	for _, child := range node.Children {
		count += countORNodes(child)
	}
	return count
}

// Subquery depth
type SubqueryDepthRule struct{}

func (r *SubqueryDepthRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	depth := countSubqueryDepth(result.RawSQL)
	if depth >= 2 {
		return []model.RiskMeta{
			{
				ID:         utils.NewID("risk"),
				Level:      "warning",
				Type:       "SUBQUERY_DEPTH",
				Message:    "子查询嵌套层级较深",
				Suggestion: "深层子查询会降低 SQL 可读性和性能，建议检查是否可以改写为 JOIN 或 CTE。",
			},
		}
	}
	return nil
}

func countSubqueryDepth(sql string) int {
	maxDepth := 0
	currentDepth := 0
	upper := strings.ToUpper(sql)
	for i := 0; i < len(upper)-7; i++ {
		if upper[i:i+8] == "( SELECT" || (i < len(upper)-6 && upper[i:i+7] == "(SELECT") {
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		}
	}
	return maxDepth
}

// Window function without ORDER BY in OVER clause
type WindowFuncWithoutOrderByRule struct{}

func (r *WindowFuncWithoutOrderByRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	var risks []model.RiskMeta
	for _, field := range result.Fields {
		if field.WindowSpec != nil {
			// Window functions that need ORDER BY for deterministic results
			upperExpr := strings.ToUpper(field.Expression)
			needsOrder := strings.Contains(upperExpr, "ROW_NUMBER") ||
				strings.Contains(upperExpr, "RANK") ||
				strings.Contains(upperExpr, "DENSE_RANK") ||
				strings.Contains(upperExpr, "NTILE") ||
				strings.Contains(upperExpr, "LAG") ||
				strings.Contains(upperExpr, "LEAD")
			if needsOrder && len(field.WindowSpec.OrderBy) == 0 {
				risks = append(risks, model.RiskMeta{
					ID:          utils.NewID("risk"),
					Level:       "warning",
					Type:        "WINDOW_NO_ORDER",
					Message:     "窗口函数缺少 ORDER BY: " + field.Expression,
					Suggestion:  "ROW_NUMBER/RANK/LAG/LEAD 等窗口函数需要 ORDER BY 才能保证结果确定性。",
					RelatedExpr: field.Expression,
				})
			}
		}
	}
	return risks
}

// Recursive CTE without termination condition
type CTENoTerminationRule struct{}

func (r *CTENoTerminationRule) Check(result *model.SQLAnalysisResult) []model.RiskMeta {
	if len(result.CTEs) == 0 {
		return nil
	}
	// Simple heuristic: check if any CTE's raw SQL contains the CTE's own name (self-reference)
	// and lacks a clear termination (WHERE or LIMIT)
	var risks []model.RiskMeta
	for _, cte := range result.CTEs {
		upper := strings.ToUpper(cte.RawSQL)
		cteNameUpper := strings.ToUpper(cte.Name)
		if strings.Contains(upper, cteNameUpper) {
			// Self-referencing CTE — check for WHERE or LIMIT as termination
			if !strings.Contains(upper, "WHERE") && !strings.Contains(upper, "LIMIT") {
				risks = append(risks, model.RiskMeta{
					ID:         utils.NewID("risk"),
					Level:      "warning",
					Type:       "CTE_NO_TERMINATION",
					Message:    "递归 CTE 可能缺少终止条件: " + cte.Name,
					Suggestion: "递归 CTE 应包含 WHERE 或 LIMIT 来防止无限递归。",
				})
			}
		}
	}
	return risks
}
