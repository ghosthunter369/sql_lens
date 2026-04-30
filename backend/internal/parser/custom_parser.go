package parser

import (
	"fmt"
	"sql-lens/internal/model"
	"sql-lens/internal/utils"
	"strings"
	"unicode"
)

type CustomParser struct{}

func NewCustomParser() *CustomParser {
	return &CustomParser{}
}

func (p *CustomParser) Parse(sql string) (*model.SQLAnalysisResult, error) {
	utils.ResetIDs()

	t := newTokenizer(sql)
	stmtType := ""
	upper := strings.ToUpper(strings.TrimSpace(sql))

	switch {
	case strings.HasPrefix(upper, "SELECT"):
		stmtType = "SELECT"
	case strings.HasPrefix(upper, "INSERT"):
		stmtType = "INSERT"
	case strings.HasPrefix(upper, "UPDATE"):
		stmtType = "UPDATE"
	case strings.HasPrefix(upper, "DELETE"):
		stmtType = "DELETE"
	case strings.HasPrefix(upper, "CREATE"):
		stmtType = "CREATE"
	default:
		return nil, fmt.Errorf("unsupported statement type")
	}

	if stmtType != "SELECT" {
		return nil, fmt.Errorf("currently only SELECT statements are supported")
	}

	result := &model.SQLAnalysisResult{
		StatementType: stmtType,
		Dialect:       "mysql",
	}

	// Build table alias map and parse tables
	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string) // tableName -> alias

	tables, joins, err := p.parseFromAndJoins(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, fmt.Errorf("parse FROM/JOIN error: %w", err)
	}

	// Reset tokenizer to parse SELECT fields
	t2 := newTokenizer(sql)
	fields := p.parseSelectFields(t2, aliasMap)

	// Reset to parse WHERE
	t3 := newTokenizer(sql)
	whereTree, whereCount := p.parseWhereClause(t3, aliasMap)

	// Reset to parse GROUP BY
	t4 := newTokenizer(sql)
	groupBy := p.parseGroupBy(t4, aliasMap)

	// Reset to parse ORDER BY
	t5 := newTokenizer(sql)
	orderBy := p.parseOrderBy(t5, aliasMap)

	// Reset to parse LIMIT
	t6 := newTokenizer(sql)
	limit := p.parseLimit(t6)

	// Post-process: resolve join conditions to use table names instead of aliases
	p.resolveJoinConditionAliases(joins, tableAliasMap)

	result.Tables = tables
	result.Joins = joins
	result.Fields = fields
	result.WhereTree = whereTree
	result.GroupBy = groupBy
	result.OrderBy = orderBy
	result.Limit = limit
	result.Risks = make([]model.RiskMeta, 0)

	// Ensure no nil slices
	if result.Tables == nil {
		result.Tables = make([]model.TableMeta, 0)
	}
	if result.Joins == nil {
		result.Joins = make([]model.JoinMeta, 0)
	}
	if result.Fields == nil {
		result.Fields = make([]model.FieldMeta, 0)
	}
	if result.GroupBy == nil {
		result.GroupBy = make([]model.GroupByMeta, 0)
	}
	if result.OrderBy == nil {
		result.OrderBy = make([]model.OrderByMeta, 0)
	}

	// For single-table queries, attribute unqualified columns to that table
	if len(tables) == 1 {
		tableName := tables[0].Name
		for i := range fields {
			if fields[i].SourceTable == "" && fields[i].FieldType == "column" {
				fields[i].SourceTable = tableName
			}
		}
		if whereTree != nil {
			p.resolveUnqualifiedWhere(whereTree, tableName)
		}
	}

	// Populate table field lists (selected/join/filter)
	p.populateTableFields(tables, fields, whereTree, joins)

	// Build summary
	result.Summary = model.SQLSummary{
		TableCount: len(tables),
		JoinCount:  len(joins),
		FieldCount: len(fields),
		WhereCount: whereCount,
		HasGroupBy: len(groupBy) > 0,
		HasOrderBy: len(orderBy) > 0,
		HasLimit:   limit != nil,
		Complexity: p.calculateComplexity(len(joins), whereCount, len(groupBy), len(orderBy), sql),
	}

	// Build graph
	result.Graph = p.buildGraph(tables, joins)

	return result, nil
}

func (p *CustomParser) parseFromAndJoins(t *tokenizer, aliasMap map[string]string, tableAliasMap map[string]string) ([]model.TableMeta, []model.JoinMeta, error) {
	var tables []model.TableMeta
	var joins []model.JoinMeta

	if !t.skipToKeyword("FROM") {
		return nil, nil, fmt.Errorf("FROM clause not found")
	}

	// Parse main FROM table
	mainTable, err := p.parseTableRef(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, nil, fmt.Errorf("parse FROM table error: %w", err)
	}
	mainTable.Role = "main"
	tables = append(tables, *mainTable)

	// Parse JOINs
	for {
		pos := t.pos
		if pos >= len(t.tokens) {
			break
		}

		tok := t.tokens[pos]
		upperVal := strings.ToUpper(tok.value)

		joinType := ""
		switch upperVal {
		case "LEFT":
			if pos+1 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "JOIN" {
				joinType = "LEFT JOIN"
				t.pos = pos + 2
			} else if pos+2 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "OUTER" && strings.ToUpper(t.tokens[pos+2].value) == "JOIN" {
				joinType = "LEFT OUTER JOIN"
				t.pos = pos + 3
			} else {
				break
			}
		case "RIGHT":
			if pos+1 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "JOIN" {
				joinType = "RIGHT JOIN"
				t.pos = pos + 2
			} else if pos+2 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "OUTER" && strings.ToUpper(t.tokens[pos+2].value) == "JOIN" {
				joinType = "RIGHT OUTER JOIN"
				t.pos = pos + 3
			} else {
				break
			}
		case "INNER":
			if pos+1 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "JOIN" {
				joinType = "INNER JOIN"
				t.pos = pos + 2
			} else {
				break
			}
		case "JOIN":
			joinType = "JOIN"
			t.pos = pos + 1
		case "CROSS":
			if pos+1 < len(t.tokens) && strings.ToUpper(t.tokens[pos+1].value) == "JOIN" {
				joinType = "CROSS JOIN"
				t.pos = pos + 2
			} else {
				break
			}
		default:
			// Check if we've hit WHERE, GROUP, ORDER, LIMIT, HAVING, etc.
			if isClauseKeyword(upperVal) {
				return tables, joins, nil
			}
			break
		}

		if joinType == "" {
			break
		}

		joinTable, err := p.parseTableRef(t, aliasMap, tableAliasMap)
		if err != nil {
			return nil, nil, fmt.Errorf("parse JOIN table error: %w", err)
		}
		joinTable.Role = "joined"
		tables = append(tables, *joinTable)

		// Parse ON condition
		joinMeta := model.JoinMeta{
			ID:         utils.NewID("join"),
			Type:       joinType,
			LeftTable:  tables[len(tables)-2].Name,
			RightTable: joinTable.Name,
			RawExpr:    "",
		}

		if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "ON" {
			t.pos++
			condStr := t.readUntilKeyword("WHERE", "GROUP", "ORDER", "LIMIT", "HAVING", "LEFT", "RIGHT", "INNER", "JOIN", "CROSS")
			joinMeta.RawExpr = strings.TrimSpace(condStr)
			joinMeta.Conditions = parseJoinConditions(condStr)
		}

		joins = append(joins, joinMeta)
	}

	return tables, joins, nil
}

func (p *CustomParser) parseTableRef(t *tokenizer, aliasMap map[string]string, tableAliasMap map[string]string) (*model.TableMeta, error) {
	if t.pos >= len(t.tokens) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Strip backticks from table name
	name := stripBackticks(t.tokens[t.pos].value)
	t.pos++

	alias := ""
	// Check for alias (no AS keyword)
	if t.pos < len(t.tokens) {
		nextVal := t.tokens[t.pos].value
		nextUpper := strings.ToUpper(nextVal)
		if !isKeyword(nextUpper) && !isOperator(nextVal) && nextVal != "(" && nextVal != ")" {
			alias = stripBackticks(nextVal)
			t.pos++
		} else if nextUpper == "AS" {
			t.pos++ // skip AS
			if t.pos < len(t.tokens) {
				alias = stripBackticks(t.tokens[t.pos].value)
				t.pos++
			}
		}
	}

	// Register alias -> table name (and table name -> itself for direct references)
	aliasMap[strings.ToLower(name)] = name
	if alias != "" {
		aliasMap[strings.ToLower(alias)] = name
		tableAliasMap[name] = alias
	}

	return &model.TableMeta{
		ID:    utils.NewID("table"),
		Name:  name,
		Alias: alias,
	}, nil
}

func isClauseKeyword(s string) bool {
	switch s {
	case "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING", "UNION", "INTERSECT", "EXCEPT", "FOR":
		return true
	}
	return false
}

func isKeyword(s string) bool {
	switch s {
	case "SELECT", "FROM", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING",
		"LEFT", "RIGHT", "INNER", "JOIN", "CROSS", "OUTER", "ON",
		"AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL",
		"AS", "ASC", "DESC", "BY", "UNION", "ALL", "DISTINCT",
		"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"SET", "INTO", "VALUES", "CASE", "WHEN", "THEN", "ELSE", "END",
		"EXISTS", "ANY", "SOME", "OFFSET", "FETCH", "WITH", "RECURSIVE",
		"FOR", "NOWAIT", "WAIT", "INTERSECT", "EXCEPT", "TRUE", "FALSE":
		return true
	}
	return false
}

func isOperator(s string) bool {
	switch s {
	case "=", "!=", "<>", ">", "<", ">=", "<=", "+", "-", "*", "/", "%", "||":
		return true
	}
	return false
}

func (p *CustomParser) parseSelectFields(t *tokenizer, aliasMap map[string]string) []model.FieldMeta {
	var fields []model.FieldMeta

	if !t.skipToKeyword("SELECT") {
		return fields
	}

	// Skip DISTINCT, ALL if present
	if t.pos < len(t.tokens) {
		upper := strings.ToUpper(t.tokens[t.pos].value)
		if upper == "DISTINCT" || upper == "ALL" {
			t.pos++
		}
	}

	for t.pos < len(t.tokens) {
		tok := t.tokens[t.pos]
		upper := strings.ToUpper(tok.value)
		if upper == "FROM" || isClauseKeyword(upper) && upper != "LIMIT" {
			break
		}

		expr := t.readUntilCommaOrKeyword()
		expr = strings.TrimSpace(expr)
		if expr == "" {
			if t.pos < len(t.tokens) && t.tokens[t.pos].value == "," {
				t.pos++
			}
			continue
		}

		field := p.analyzeField(expr, aliasMap)
		fields = append(fields, field)

		// Skip comma
		if t.pos < len(t.tokens) && t.tokens[t.pos].value == "," {
			t.pos++
		}
	}

	return fields
}

func (p *CustomParser) analyzeField(expr string, aliasMap map[string]string) model.FieldMeta {
	expr = strings.TrimSpace(expr)
	field := model.FieldMeta{
		ID:         utils.NewID("field"),
		Expression: expr,
	}

	// --- Wildcard ---
	if expr == "*" || strings.HasSuffix(expr, ".*") {
		field.FieldType = "wildcard"
		field.OutputName = expr
		if dotIdx := strings.Index(expr, "."); dotIdx > 0 {
			alias := expr[:dotIdx]
			if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
				field.SourceTable = tableName
				field.SourceAlias = alias
			}
		}
		return field
	}

	// --- Find AS alias (outside parens and quotes) ---
	asIdx := findASKeyword(expr)
	var baseExpr string
	if asIdx >= 0 {
		// asIdx points to 'A' of "AS", so skip "AS " = 3 chars
		field.OutputName = strings.TrimSpace(expr[asIdx+3:])
		// baseExpr is everything before " AS "
		baseExpr = strings.TrimSpace(expr[:asIdx-1])
	} else {
		// No AS — for simple columns use the column name as output
		if isSimpleColumn(expr) {
			if dotIdx := strings.LastIndex(expr, "."); dotIdx > 0 {
				field.OutputName = stripBackticks(expr[dotIdx+1:])
			} else {
				field.OutputName = stripBackticks(expr)
			}
		} else {
			// For functions without AS, use a simplified name
			funcName, _ := extractFunctionName(expr)
			if funcName != "" {
				firstCol := extractFirstColumn(expr)
				if firstCol != "" {
					if dotIdx := strings.LastIndex(firstCol, "."); dotIdx > 0 {
						field.OutputName = stripBackticks(firstCol[dotIdx+1:])
					} else if isSimpleColumn(firstCol) {
						field.OutputName = stripBackticks(firstCol)
					} else {
						field.OutputName = funcName + "(...)"
					}
				} else {
					field.OutputName = funcName + "(...)"
				}
			} else {
				field.OutputName = stripBackticksAll(expr)
			}
		}
		baseExpr = expr
	}

	// --- Detect function name and type ---
	funcName, isFunc := extractFunctionName(baseExpr)

	// --- Determine field type ---
	switch {
	case isFunc && isAggregateFunc(funcName):
		field.FieldType = "aggregate"
	case isFunc && funcName == "CASE":
		field.FieldType = "case"
	case isFunc:
		field.FieldType = "function"
	default:
		field.FieldType = "column"
	}

	// --- Resolve source table/column ---
	if field.FieldType == "column" {
		p.resolveSourceTable(baseExpr, aliasMap, &field)
	} else {
		// For function/aggregate: extract the first column reference from inside
		firstCol := extractFirstColumn(baseExpr)
		if firstCol != "" {
			p.resolveSourceTable(firstCol, aliasMap, &field)
		}
	}

	return field
}

// findASKeyword finds " AS " outside parentheses and quotes, returning the index of the space before AS.
// Only returns the LAST valid AS (the effective alias).
func findASKeyword(expr string) int {
	upper := strings.ToUpper(expr)
	depth := 0
	inQuote := false
	var quoteChar byte
	lastAS := -1

	for i := 0; i <= len(expr)-4; i++ {
		ch := expr[i]
		if ch == '\'' || ch == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if inQuote {
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && upper[i:i+4] == " AS " {
			lastAS = i + 1 // index of "AS" keyword
		}
	}
	return lastAS
}

// stripBackticks removes surrounding backtick quotes from an identifier.
func stripBackticks(s string) string {
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}

// stripBackticksAll strips backticks from all dot-separated parts of a qualified identifier.
func stripBackticksAll(expr string) string {
	parts := strings.Split(expr, ".")
	for i, p := range parts {
		parts[i] = stripBackticks(p)
	}
	return strings.Join(parts, ".")
}

// isSimpleColumn checks if the expression is just a column reference (possibly with table alias)
func isSimpleColumn(expr string) bool {
	// Strip backticks first so backtick-quoted identifiers are recognized as simple
	expr = stripBackticksAll(expr)
	for _, ch := range expr {
		if ch == '(' || ch == ')' || ch == '\'' || ch == '"' {
			return false
		}
	}
	for _, ch := range expr {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '.' {
			continue
		}
		return false
	}
	return true
}

// extractFunctionName returns the function name and whether this is a function call
func extractFunctionName(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	// Find first opening paren
	parenIdx := strings.Index(expr, "(")
	if parenIdx <= 0 {
		return "", false
	}
	// Walk back from paren to get function name
	name := strings.TrimSpace(expr[:parenIdx])
	name = strings.ToUpper(name)

	// If name contains spaces, it's likely not a simple function call
	// (could be "CASE WHEN" or other constructs)
	if strings.Contains(name, " ") {
		// Check for CASE WHEN pattern
		if strings.HasPrefix(name, "CASE") {
			return "CASE", true
		}
		return "", false
	}

	return name, true
}

// extractFirstColumn extracts the first column reference from a function's arguments
func extractFirstColumn(expr string) string {
	// Find the content inside the outermost parentheses
	start := strings.Index(expr, "(")
	if start < 0 {
		return ""
	}
	start++
	depth := 1
	end := start
	for end < len(expr) {
		ch := expr[end]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				break
			}
		}
		end++
	}

	if end >= len(expr) {
		return ""
	}

	inner := strings.TrimSpace(expr[start:end])

	// For CASE WHEN, extract differently
	upper := strings.ToUpper(expr)
	if strings.Contains(upper, "CASE") {
		return extractFromCaseWhen(inner)
	}

	// Split arguments by top-level commas, take first
	firstArg := splitFirstArg(inner)

	// If first arg contains a dot, return it
	firstArg = strings.TrimSpace(firstArg)
	if dotIdx := strings.Index(firstArg, "."); dotIdx > 0 {
		return firstArg
	}

	// If it looks like a column name (no spaces, no parens), return it
	if isSimpleColumn(firstArg) {
		return firstArg
	}

	return ""
}

// splitFirstArg splits on the first top-level comma
func splitFirstArg(s string) string {
	depth := 0
	inQuote := false
	var qc byte
	for i, ch := range s {
		if ch == '\'' || ch == '"' {
			if !inQuote {
				inQuote = true
				qc = byte(ch)
			} else if byte(ch) == qc {
				inQuote = false
			}
			continue
		}
		if inQuote {
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth > 0 {
				depth--
			}
		}
		if ch == ',' && depth == 0 {
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

func extractFromCaseWhen(inner string) string {
	upper := strings.ToUpper(inner)
	// CASE column WHEN ... or CASE WHEN column = ...
	whenIdx := strings.Index(upper, "WHEN")
	if whenIdx < 0 {
		return ""
	}
	afterWhen := strings.TrimSpace(inner[whenIdx+4:])
	// If "WHEN column", return the first word
	if dotIdx := strings.Index(afterWhen, "."); dotIdx > 0 {
		// Has table.column
		parts := strings.Fields(afterWhen)
		if len(parts) > 0 && strings.Contains(parts[0], ".") {
			return parts[0]
		}
	}
	return ""
}

// isAggregateFunc checks if a function name is an aggregate function
func isAggregateFunc(name string) bool {
	switch name {
	case "COUNT", "SUM", "AVG", "MAX", "MIN", "GROUP_CONCAT",
		"STDDEV", "STDDEV_POP", "STDDEV_SAMP",
		"VARIANCE", "VAR_POP", "VAR_SAMP",
		"BIT_AND", "BIT_OR", "BIT_XOR",
		"JSON_ARRAYAGG", "JSON_OBJECTAGG":
		return true
	}
	return false
}

func (p *CustomParser) resolveSourceTable(expr string, aliasMap map[string]string, field *model.FieldMeta) {
	expr = strings.TrimSpace(expr)
	// Strip backticks from the whole expression for alias lookup
	cleanExpr := stripBackticksAll(expr)
	if dotIdx := strings.Index(cleanExpr, "."); dotIdx > 0 {
		alias := cleanExpr[:dotIdx]
		col := cleanExpr[dotIdx+1:]
		// Strip any trailing parens or commas that might have leaked
		col = strings.TrimRight(col, "), ")
		if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
			field.SourceTable = tableName
			field.SourceAlias = alias
			field.SourceColumn = col
		}
	}
	if field.SourceTable == "" && field.FieldType == "column" {
		field.SourceColumn = cleanExpr
	}
}

func (p *CustomParser) parseWhereClause(t *tokenizer, aliasMap map[string]string) (*model.ConditionNode, int) {
	if !t.skipToKeyword("WHERE") {
		return nil, 0
	}

	whereStr := t.readUntilKeyword("GROUP", "ORDER", "LIMIT", "HAVING")
	whereStr = strings.TrimSpace(whereStr)
	if whereStr == "" {
		return nil, 0
	}

	node, count := p.buildConditionTree(whereStr, aliasMap)
	return node, count
}

func (p *CustomParser) buildConditionTree(whereStr string, aliasMap map[string]string) (*model.ConditionNode, int) {
	whereStr = strings.TrimSpace(whereStr)
	if whereStr == "" {
		return nil, 0
	}

	// Remove outer parentheses if they wrap the entire expression
	whereStr = unwrapOuterParens(whereStr)

	// Split by AND/OR at the top level
	parts, connectors := splitByLogicOp(whereStr)

	if len(parts) == 1 {
		// Single condition
		cond := p.parseSingleCondition(parts[0], aliasMap)
		return cond, 1
	}

	// Determine if this is AND or OR (use first connector)
	firstConnector := "AND"
	if len(connectors) > 0 && strings.ToUpper(connectors[0]) == "OR" {
		firstConnector = "OR"
	}

	node := &model.ConditionNode{
		ID:       utils.NewID("cond"),
		Type:     firstConnector,
		Children: make([]*model.ConditionNode, 0),
	}

	count := 0
	for _, part := range parts {
		child, c := p.buildConditionTree(part, aliasMap)
		if child != nil {
			node.Children = append(node.Children, child)
			count += c
		}
	}

	return node, count
}

func (p *CustomParser) parseSingleCondition(cond string, aliasMap map[string]string) *model.ConditionNode {
	cond = strings.TrimSpace(cond)

	node := &model.ConditionNode{
		ID:   utils.NewID("cond"),
		Type: "CONDITION",
		Expr: cond,
	}

	// Try to parse table.field operator value pattern
	cond = strings.TrimSpace(cond)
	upper := strings.ToUpper(cond)

	// Handle IS NULL / IS NOT NULL
	if strings.HasSuffix(upper, " IS NULL") {
		fieldPart := strings.TrimSpace(cond[:len(cond)-8])
		node.Field = fieldPart
		node.Operator = "IS NULL"
		node.Value = "NULL"
		p.resolveConditionTable(fieldPart, aliasMap, node)
		return node
	}
	if strings.HasSuffix(upper, " IS NOT NULL") {
		fieldPart := strings.TrimSpace(cond[:len(cond)-12])
		node.Field = fieldPart
		node.Operator = "IS NOT NULL"
		node.Value = "NULL"
		p.resolveConditionTable(fieldPart, aliasMap, node)
		return node
	}

	// Handle IN / NOT IN
	inIdx := strings.Index(upper, " IN (")
	if inIdx > 0 {
		fieldPart := strings.TrimSpace(cond[:inIdx])
		notPrefix := strings.HasSuffix(strings.TrimSpace(upper[:inIdx]), " NOT")
		if notPrefix {
			fieldPart = strings.TrimSpace(cond[:inIdx-4])
			node.Operator = "NOT IN"
		} else {
			node.Operator = "IN"
		}
		node.Field = fieldPart
		node.Value = cond[inIdx+1:] // includes the parenthesized list
		node.Expr = cond
		p.resolveConditionTable(fieldPart, aliasMap, node)
		return node
	}

	// Handle BETWEEN
	betweenIdx := strings.Index(upper, " BETWEEN ")
	if betweenIdx > 0 {
		fieldPart := strings.TrimSpace(cond[:betweenIdx])
		node.Field = fieldPart
		node.Operator = "BETWEEN"
		node.Value = strings.TrimSpace(cond[betweenIdx+9:])
		node.Expr = cond
		p.resolveConditionTable(fieldPart, aliasMap, node)
		return node
	}

	// Handle LIKE
	likeIdx := strings.Index(upper, " LIKE ")
	if likeIdx > 0 {
		fieldPart := strings.TrimSpace(cond[:likeIdx])
		notPrefix := strings.HasSuffix(strings.TrimSpace(upper[:likeIdx]), " NOT")
		if notPrefix {
			fieldPart = strings.TrimSpace(cond[:likeIdx-4])
			node.Operator = "NOT LIKE"
		} else {
			node.Operator = "LIKE"
		}
		node.Field = fieldPart
		node.Value = strings.TrimSpace(cond[likeIdx+6:])
		node.Expr = cond
		p.resolveConditionTable(fieldPart, aliasMap, node)
		return node
	}

	// Handle basic operators: =, !=, <>, >=, <=, >, <
	operators := []string{"!=", "<>", ">=", "<=", "=", ">", "<"}
	for _, op := range operators {
		idx := findOpOutsideQuotes(cond, op)
		if idx > 0 {
			node.Field = strings.TrimSpace(cond[:idx])
			node.Operator = op
			node.Value = strings.TrimSpace(cond[idx+len(op):])
			node.Expr = cond
			p.resolveConditionTable(node.Field, aliasMap, node)
			return node
		}
	}

	return node
}

// resolveUnqualifiedWhere sets the table for WHERE conditions that have no table set,
// for single-table queries where the table context is unambiguous.
func (p *CustomParser) resolveUnqualifiedWhere(node *model.ConditionNode, tableName string) {
	if node == nil {
		return
	}
	if node.Type == "CONDITION" && node.Table == "" {
		node.Table = tableName
	}
	for _, child := range node.Children {
		p.resolveUnqualifiedWhere(child, tableName)
	}
}

func (p *CustomParser) resolveConditionTable(fieldExpr string, aliasMap map[string]string, node *model.ConditionNode) {
	fieldExpr = strings.TrimSpace(fieldExpr)
	cleanExpr := stripBackticksAll(fieldExpr)
	if dotIdx := strings.Index(cleanExpr, "."); dotIdx > 0 {
		alias := cleanExpr[:dotIdx]
		col := cleanExpr[dotIdx+1:]
		if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
			node.Table = tableName
			if node.Field == fieldExpr {
				node.Field = col
			}
		}
	}
}

func findOpOutsideQuotes(s, op string) int {
	inQuote := false
	var quoteChar byte
	for i := 0; i <= len(s)-len(op); i++ {
		ch := s[i]
		if ch == '\'' || ch == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if !inQuote && s[i:i+len(op)] == op {
			return i
		}
	}
	return -1
}

func unwrapOuterParens(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '(' {
		return s
	}
	depth := 0
	for i, ch := range s {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 && i == len(s)-1 {
				return unwrapOuterParens(s[1 : len(s)-1])
			}
			if depth == 0 && i < len(s)-1 {
				return s
			}
		}
	}
	return s
}

func splitByLogicOp(s string) ([]string, []string) {
	var parts []string
	var connectors []string
	last := 0
	depth := 0
	inQuote := false
	var quoteChar byte

	upper := strings.ToUpper(s)

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\'' || ch == '"' {
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
			}
			continue
		}
		if inQuote {
			continue
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		}

		if depth == 0 {
			// Check for AND
			if i+5 <= len(s) && upper[i:i+5] == " AND " {
				parts = append(parts, strings.TrimSpace(s[last:i]))
				connectors = append(connectors, "AND")
				last = i + 5
				i += 4
				continue
			}
			// Check for OR (but not part of ORDER, FOR, etc.)
			if i+4 <= len(s) && upper[i:i+4] == " OR " {
				// Make sure it's not part of "ORDER" or "FOR"
				if i >= 1 && s[i-1] != ' ' {
					// Check context - if preceded by word char, might be part of a word
					wordStart := i - 1
					for wordStart > last && s[wordStart] != ' ' {
						wordStart--
					}
					if wordStart > last {
						wordStart++
					}
					if strings.ToUpper(s[wordStart:i]) == "ORDER" || strings.ToUpper(s[wordStart:i]) == "FOR" {
						continue
					}
				}
				parts = append(parts, strings.TrimSpace(s[last:i]))
				connectors = append(connectors, "OR")
				last = i + 4
				i += 3
				continue
			}
		}
	}

	if last < len(s) {
		parts = append(parts, strings.TrimSpace(s[last:]))
	}

	return parts, connectors
}

func parseJoinConditions(condStr string) []model.JoinCondition {
	var conditions []model.JoinCondition

	condStr = strings.TrimSpace(condStr)
	// Split by AND
	parts, _ := splitByLogicOp(condStr)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Look for = operator
		eqIdx := findOpOutsideQuotes(part, "=")
		if eqIdx > 0 {
			conditions = append(conditions, model.JoinCondition{
				Left:     strings.TrimSpace(part[:eqIdx]),
				Operator: "=",
				Right:    strings.TrimSpace(part[eqIdx+1:]),
			})
		}
	}

	return conditions
}

// resolveJoinConditionAliases replaces aliases in join conditions with actual table names
func (p *CustomParser) resolveJoinConditionAliases(joins []model.JoinMeta, tableAliasMap map[string]string) {
	// Build reverse map: alias -> tableName
	aliasToName := make(map[string]string)
	for tableName, alias := range tableAliasMap {
		aliasToName[alias] = tableName
	}

	for i := range joins {
		for j := range joins[i].Conditions {
			cond := &joins[i].Conditions[j]
			// Strip backticks from Left/Right first
			cond.Left = stripBackticksAll(cond.Left)
			cond.Right = stripBackticksAll(cond.Right)
			// Replace alias in Left
			if dotIdx := strings.Index(cond.Left, "."); dotIdx > 0 {
				alias := cond.Left[:dotIdx]
				if tblName, ok := aliasToName[alias]; ok {
					cond.Left = tblName + cond.Left[dotIdx:]
				}
			}
			// Replace alias in Right
			if dotIdx := strings.Index(cond.Right, "."); dotIdx > 0 {
				alias := cond.Right[:dotIdx]
				if tblName, ok := aliasToName[alias]; ok {
					cond.Right = tblName + cond.Right[dotIdx:]
				}
			}
		}
		// Also resolve rawExpr — strip backticks and replace aliases
		joins[i].RawExpr = stripBackticksAll(joins[i].RawExpr)
		for alias, tblName := range aliasToName {
			joins[i].RawExpr = strings.ReplaceAll(joins[i].RawExpr, alias+".", tblName+".")
		}
	}
}

func (p *CustomParser) parseGroupBy(t *tokenizer, aliasMap map[string]string) []model.GroupByMeta {
	var items []model.GroupByMeta

	if !t.skipToKeyword("GROUP") {
		return items
	}
	if t.pos >= len(t.tokens) || strings.ToUpper(t.tokens[t.pos].value) != "BY" {
		return items
	}
	t.pos++

	groupStr := t.readUntilKeyword("HAVING", "ORDER", "LIMIT")
	parts := strings.Split(groupStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		item := model.GroupByMeta{Expression: stripBackticksAll(part)}
		cleanPart := stripBackticksAll(part)
		if dotIdx := strings.Index(cleanPart, "."); dotIdx > 0 {
			alias := cleanPart[:dotIdx]
			if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
				item.SourceTable = tableName
			}
		}
		items = append(items, item)
	}

	return items
}

func (p *CustomParser) parseOrderBy(t *tokenizer, aliasMap map[string]string) []model.OrderByMeta {
	var items []model.OrderByMeta

	if !t.skipToKeyword("ORDER") {
		return items
	}
	if t.pos >= len(t.tokens) || strings.ToUpper(t.tokens[t.pos].value) != "BY" {
		return items
	}
	t.pos++

	orderStr := t.readUntilKeyword("LIMIT")
	parts := strings.Split(orderStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		direction := "ASC"
		expr := part
		upper := strings.ToUpper(part)
		if strings.HasSuffix(upper, " DESC") {
			direction = "DESC"
			expr = strings.TrimSpace(part[:len(part)-5])
		} else if strings.HasSuffix(upper, " ASC") {
			expr = strings.TrimSpace(part[:len(part)-4])
		}

		item := model.OrderByMeta{
			Expression: stripBackticksAll(expr),
			Direction:  direction,
		}
		cleanExpr := stripBackticksAll(expr)
		if dotIdx := strings.Index(cleanExpr, "."); dotIdx > 0 {
			alias := cleanExpr[:dotIdx]
			if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
				item.SourceTable = tableName
			}
		}
		items = append(items, item)
	}

	return items
}

func (p *CustomParser) parseLimit(t *tokenizer) *model.LimitMeta {
	if !t.skipToKeyword("LIMIT") {
		return nil
	}

	if t.pos >= len(t.tokens) {
		return nil
	}

	limitVal := 0
	fmt.Sscanf(t.tokens[t.pos].value, "%d", &limitVal)
	t.pos++

	offsetVal := 0
	if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "OFFSET" {
		t.pos++
		if t.pos < len(t.tokens) {
			fmt.Sscanf(t.tokens[t.pos].value, "%d", &offsetVal)
			t.pos++
		}
	} else if t.pos < len(t.tokens) && t.tokens[t.pos].value == "," {
		// MySQL LIMIT offset, limit syntax
		t.pos++
		if t.pos < len(t.tokens) {
			offsetVal = limitVal
			fmt.Sscanf(t.tokens[t.pos].value, "%d", &limitVal)
			t.pos++
		}
	}

	return &model.LimitMeta{
		Limit:  limitVal,
		Offset: offsetVal,
	}
}

func (p *CustomParser) calculateComplexity(joinCount, whereCount, groupByCount, orderByCount int, sql string) string {
	score := joinCount*2 + whereCount + groupByCount*2 + orderByCount

	// Count subqueries
	upper := strings.ToUpper(sql)
	subCount := strings.Count(upper, "(SELECT")
	score += subCount * 3

	switch {
	case score <= 5:
		return "LOW"
	case score <= 12:
		return "MEDIUM"
	default:
		return "HIGH"
	}
}

func (p *CustomParser) buildGraph(tables []model.TableMeta, joins []model.JoinMeta) model.GraphMeta {
	graph := model.GraphMeta{
		Nodes: make([]model.GraphNode, 0, len(tables)),
		Edges: make([]model.GraphEdge, 0, len(joins)),
	}

	tableNameToID := make(map[string]string)

	// Build nodes
	for i, table := range tables {
		tableNameToID[table.Name] = table.ID
		node := model.GraphNode{
			ID:   table.ID,
			Type: "tableNode",
			Position: model.GraphPosition{
				X: float64(i) * 360,
				Y: float64(i%2) * 200,
			},
			Data: map[string]interface{}{
				"tableName":      table.Name,
				"alias":          table.Alias,
				"role":           table.Role,
				"selectedFields": table.SelectedFields,
				"filterFields":   table.FilterFields,
				"joinFields":     table.JoinFields,
			},
		}
		graph.Nodes = append(graph.Nodes, node)
	}

	// Build edges
	for _, join := range joins {
		sourceID := tableNameToID[join.LeftTable]
		targetID := tableNameToID[join.RightTable]
		if sourceID == "" {
			sourceID = join.LeftTable
		}
		if targetID == "" {
			targetID = join.RightTable
		}

		edge := model.GraphEdge{
			ID:     join.ID,
			Source: sourceID,
			Target: targetID,
			Label:  join.Type,
			Type:   "joinEdge",
			Data: map[string]interface{}{
				"conditions": join.Conditions,
				"rawExpr":    join.RawExpr,
			},
		}
		graph.Edges = append(graph.Edges, edge)
	}

	return graph
}

// populateTableFields fills SelectedFields, FilterFields, JoinFields on each table
func (p *CustomParser) populateTableFields(tables []model.TableMeta, fields []model.FieldMeta, whereTree *model.ConditionNode, joins []model.JoinMeta) {
	tableByName := make(map[string]*model.TableMeta)
	for i := range tables {
		// Reset field lists to avoid duplicates
		tables[i].SelectedFields = make([]string, 0)
		tables[i].FilterFields = make([]string, 0)
		tables[i].JoinFields = make([]string, 0)
		tableByName[tables[i].Name] = &tables[i]
	}

	// SelectedFields: from parsed fields
	for _, f := range fields {
		if f.SourceTable == "" {
			continue
		}
		if tbl, ok := tableByName[f.SourceTable]; ok {
			fieldName := f.OutputName
			if fieldName == "" {
				fieldName = f.SourceColumn
			}
			if !containsStr(tbl.SelectedFields, fieldName) {
				tbl.SelectedFields = append(tbl.SelectedFields, fieldName)
			}
		}
	}

	// FilterFields: from WHERE condition tree
	var collectConditions func(node *model.ConditionNode)
	collectConditions = func(node *model.ConditionNode) {
		if node == nil {
			return
		}
		if node.Type == "CONDITION" && node.Table != "" && node.Field != "" {
			col := node.Field
			if dotIdx := strings.Index(col, "."); dotIdx >= 0 {
				col = col[dotIdx+1:]
			}
			if tbl, ok := tableByName[node.Table]; ok {
				if !containsStr(tbl.FilterFields, col) {
					tbl.FilterFields = append(tbl.FilterFields, col)
				}
			}
		}
		for _, child := range node.Children {
			collectConditions(child)
		}
	}
	collectConditions(whereTree)

	// JoinFields: from JOIN ON conditions (aliases already resolved to table names)
	for _, join := range joins {
		for _, cond := range join.Conditions {
			resolveJoinField(cond.Left, tableByName)
			resolveJoinField(cond.Right, tableByName)
		}
	}
}

func resolveJoinField(colRef string, tableByName map[string]*model.TableMeta) {
	colRef = stripBackticksAll(colRef)
	if dotIdx := strings.Index(colRef, "."); dotIdx >= 0 {
		tblName := colRef[:dotIdx]
		col := colRef[dotIdx+1:]
		if tbl, ok := tableByName[tblName]; ok {
			if !containsStr(tbl.JoinFields, col) {
				tbl.JoinFields = append(tbl.JoinFields, col)
			}
		}
	}
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
