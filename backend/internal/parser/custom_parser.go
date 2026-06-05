package parser

import (
	"fmt"
	"sql-lens/internal/model"
	"sql-lens/internal/utils"
	"strings"
	"unicode"
)

type CustomParser struct {
	dialect DialectConfig
}

func NewCustomParser() *CustomParser {
	return &CustomParser{dialect: GetDialectConfig(DialectMySQL)}
}

func NewCustomParserWithDialect(id DialectID) *CustomParser {
	return &CustomParser{dialect: GetDialectConfig(id)}
}

func (p *CustomParser) Parse(sql string) (*model.SQLAnalysisResult, error) {
	utils.ResetIDs()

	stmtType := ""
	upper := strings.ToUpper(strings.TrimSpace(sql))

	switch {
	case strings.HasPrefix(upper, "SELECT"):
		stmtType = "SELECT"
	case strings.HasPrefix(upper, "WITH"):
		stmtType = "SELECT" // CTE starts with WITH
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

	// Handle DML statements
	if stmtType == "INSERT" {
		return p.parseInsert(sql)
	}
	if stmtType == "UPDATE" {
		return p.parseUpdate(sql)
	}
	if stmtType == "DELETE" {
		return p.parseDelete(sql)
	}
	if stmtType != "SELECT" {
		return nil, fmt.Errorf("currently only SELECT/INSERT/UPDATE/DELETE statements are supported")
	}

	result := &model.SQLAnalysisResult{
		StatementType: stmtType,
		Dialect:       string(p.dialect.ID),
	}

	// Tokenize once, reuse the token array for all parse passes
	tokens := tokenize(sql, p.dialect)

	// Parse CTEs if present
	var ctes []model.CTEDefinition
	if len(tokens) > 0 && strings.ToUpper(tokens[0].value) == "WITH" {
		ctes, tokens = p.parseCTEs(tokens)
		if len(ctes) > 0 {
			result.CTEs = ctes
			// Register CTE names as virtual tables in alias map
		}
	}

	// Build table alias map and parse tables
	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string) // tableName -> alias

	// Register CTE names as virtual tables
	for _, cte := range ctes {
		aliasMap[strings.ToLower(cte.Name)] = cte.Name
	}

	t := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	tables, joins, err := p.parseFromAndJoins(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, fmt.Errorf("parse FROM/JOIN error: %w", err)
	}

	// Reuse token array for SELECT fields
	t2 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	fields := p.parseSelectFields(t2, aliasMap)

	// Reuse token array for WHERE
	t3 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	whereTree, whereCount := p.parseWhereClause(t3, aliasMap)

	// Reuse token array for GROUP BY
	t4 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	groupBy := p.parseGroupBy(t4, aliasMap)

	// Reuse token array for HAVING
	t4b := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	havingTree, havingCount := p.parseHaving(t4b, aliasMap)

	// Reuse token array for ORDER BY
	t5 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	orderBy := p.parseOrderBy(t5, aliasMap)

	// Reuse token array for LIMIT
	t6 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	limit := p.parseLimit(t6)

	// Check for UNION/INTERSECT/EXCEPT after the main SELECT
	setOps := p.parseSetOperations(tokens, result)

	// Post-process: resolve join conditions to use table names instead of aliases
	p.resolveJoinConditionAliases(joins, tableAliasMap)

	result.Tables = tables
	result.Joins = joins
	result.Fields = fields
	result.WhereTree = whereTree
	result.HavingTree = havingTree
	result.GroupBy = groupBy
	result.OrderBy = orderBy
	result.Limit = limit
	result.Risks = make([]model.RiskMeta, 0)
	result.SetOperations = setOps

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

	// Check for window functions in fields
	hasWindowFunc := false
	for _, f := range fields {
		if f.WindowSpec != nil {
			hasWindowFunc = true
			break
		}
	}

	// Build summary
	result.Summary = model.SQLSummary{
		TableCount:    len(tables),
		JoinCount:     len(joins),
		FieldCount:    len(fields),
		WhereCount:    whereCount,
		HavingCount:   havingCount,
		HasGroupBy:    len(groupBy) > 0,
		HasOrderBy:    len(orderBy) > 0,
		HasLimit:      limit != nil,
		HasHaving:     havingCount > 0,
		Complexity:    p.calculateComplexity(len(joins), whereCount, len(groupBy), len(orderBy), sql),
		HasWindowFunc: hasWindowFunc,
		HasCTE:        len(ctes) > 0,
		HasUnion:      len(setOps) > 0,
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

	// Parse main FROM table (could be subquery)
	mainTable, err := p.parseTableRefOrSubquery(t, aliasMap, tableAliasMap)
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

		joinTable, err := p.parseTableRefOrSubquery(t, aliasMap, tableAliasMap)
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

// parseTableRefOrSubquery handles both regular table refs and (SELECT ...) subquery aliases
func (p *CustomParser) parseTableRefOrSubquery(t *tokenizer, aliasMap map[string]string, tableAliasMap map[string]string) (*model.TableMeta, error) {
	if t.pos >= len(t.tokens) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Check if this is a subquery: ( SELECT ...
	if t.tokens[t.pos].value == "(" {
		// Skip the entire subquery to find the alias
		t.pos++ // skip (
		depth := 1
		for t.pos < len(t.tokens) && depth > 0 {
			if t.tokens[t.pos].value == "(" {
				depth++
			} else if t.tokens[t.pos].value == ")" {
				depth--
			}
			if depth > 0 {
				t.pos++
			}
		}
		if t.pos < len(t.tokens) {
			t.pos++ // skip closing )
		}

		// Now read the alias (AS alias or just alias)
		alias := ""
		if t.pos < len(t.tokens) {
			nextVal := t.tokens[t.pos].value
			nextUpper := strings.ToUpper(nextVal)
			if nextUpper == "AS" {
				t.pos++ // skip AS
				if t.pos < len(t.tokens) {
					alias = stripBackticks(t.tokens[t.pos].value)
					t.pos++
				}
			} else if !isKeyword(nextUpper) && !isOperator(nextVal) && nextVal != "(" && nextVal != ")" && nextVal != "," {
				alias = stripBackticks(nextVal)
				t.pos++
			}
		}

		// Generate a name for the subquery
		name := alias
		if name == "" {
			name = fmt.Sprintf("subquery_%d", utils.NewID("sub"))
		}

		// Register in alias maps
		aliasMap[strings.ToLower(name)] = name
		if alias != "" {
			tableAliasMap[name] = alias
		}

		return &model.TableMeta{
			ID:    utils.NewID("table"),
			Name:  name,
			Alias: alias,
			Role:  "subquery",
		}, nil
	}

	// Regular table reference
	return p.parseTableRef(t, aliasMap, tableAliasMap)
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

	// Handle SQL Server TOP N
	if p.dialect.SupportTopN && t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "TOP" {
		t.pos++ // skip TOP
		// Skip optional PERCENT
		if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "PERCENT" {
			t.pos++
		}
		// Skip optional (expr) or N
		if t.pos < len(t.tokens) {
			if t.tokens[t.pos].value == "(" {
				t.pos++ // skip (
				// skip until )
				depth := 1
				for t.pos < len(t.tokens) && depth > 0 {
					if t.tokens[t.pos].value == "(" {
						depth++
					}
					if t.tokens[t.pos].value == ")" {
						depth--
					}
					t.pos++
				}
			} else if t.tokens[t.pos].kind == tokenNumber {
				t.pos++ // skip N
			}
		}
		// Skip optional WITH TIES
		if t.pos+1 < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "WITH" && strings.ToUpper(t.tokens[t.pos+1].value) == "TIES" {
			t.pos += 2
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
		field.OutputName = strings.TrimSpace(expr[asIdx+3:])
		baseExpr = strings.TrimSpace(expr[:asIdx-1])
	} else {
		baseExpr = expr
	}

	// --- Parse expression with ExprParser ---
	tokens := tokenize(baseExpr, p.dialect)
	ep := NewExprParser(tokens, p.dialect)
	ast, err := ep.ParseExpression()
	if err != nil {
		// Fallback: treat as raw expression
		field.FieldType = "column"
		field.OutputName = stripBackticksAll(baseExpr)
		return field
	}

	// --- Determine field type from AST ---
	switch n := ast.(type) {
	case *ColumnRef:
		field.FieldType = "column"
		if n.Table != "" {
			if tableName, ok := aliasMap[strings.ToLower(n.Table)]; ok {
				field.SourceTable = tableName
				field.SourceAlias = n.Table
			}
			field.SourceColumn = n.Column
		} else {
			field.SourceColumn = stripBackticksAll(n.Column)
		}
	case *FunctionCall:
		if n.Over != nil {
			field.FieldType = "window"
		} else {
			field.FieldType = "function"
		}
	case *AggregateCall:
		if n.Over != nil {
			field.FieldType = "window"
		} else {
			field.FieldType = "aggregate"
		}
	case *CaseExpr:
		field.FieldType = "case"
	case *SubqueryExpr:
		field.FieldType = "subquery"
	case *CastExpr, *TypeCastExpr:
		field.FieldType = "function"
	default:
		field.FieldType = "column"
	}

	// --- Set func category from registry ---
	if field.FieldType == "function" || field.FieldType == "aggregate" || field.FieldType == "window" {
		funcName := ""
		switch n := ast.(type) {
		case *FunctionCall:
			funcName = n.Name
		case *AggregateCall:
			funcName = n.Name
		}
		if funcName != "" {
			if info, ok := LookupFunction(funcName, p.dialect.ID); ok {
				field.FuncCategory = string(info.Category)
			} else {
				field.FuncCategory = "scalar"
			}
		}
	}

	// --- Set window spec ---
	switch n := ast.(type) {
	case *FunctionCall:
		if n.Over != nil {
			field.WindowSpec = buildWindowSpecMeta(n.Over)
		}
	case *AggregateCall:
		if n.Over != nil {
			field.WindowSpec = buildWindowSpecMeta(n.Over)
		}
	}

	// --- Set OutputName if not set by AS ---
	if field.OutputName == "" {
		if isSimpleColumn(baseExpr) {
			if dotIdx := strings.LastIndex(baseExpr, "."); dotIdx > 0 {
				field.OutputName = stripBackticks(baseExpr[dotIdx+1:])
			} else {
				field.OutputName = stripBackticks(baseExpr)
			}
		} else {
			// For functions, use funcName(firstCol) pattern
			funcName := ""
			switch n := ast.(type) {
			case *FunctionCall:
				funcName = n.Name
			case *AggregateCall:
				funcName = n.Name
			}
			if funcName != "" {
				refs := CollectColumnRefs(ast)
				if len(refs) > 0 {
					col := refs[0].Column
					if col != "" && col != "*" {
						field.OutputName = strings.ToLower(funcName) + "_" + col
					} else {
						field.OutputName = funcName + "(...)"
					}
				} else {
					field.OutputName = funcName + "(...)"
				}
			} else {
				field.OutputName = stripBackticksAll(baseExpr)
			}
		}
	}

	// --- Extract deep sources from all column references ---
	refs := CollectColumnRefs(ast)
	deepSources := make(map[string]model.DeepSourceRef)
	for _, ref := range refs {
		if ref.Table != "" {
			alias := ref.Table
			colName := ref.Column
			if tableName, ok := aliasMap[strings.ToLower(alias)]; ok {
				key := tableName + "." + colName
				if _, exists := deepSources[key]; !exists {
					deepSources[key] = model.DeepSourceRef{
						Table:  tableName,
						Alias:  alias,
						Column: colName,
					}
				}
			}
		}
	}
	if len(deepSources) > 0 {
		field.DeepSources = make([]model.DeepSourceRef, 0, len(deepSources))
		for _, ds := range deepSources {
			field.DeepSources = append(field.DeepSources, ds)
		}
	}

	// --- Set shallow source for non-column fields from first column ref ---
	if field.FieldType != "column" && field.SourceTable == "" {
		for _, ref := range refs {
			if ref.Table != "" {
				if tableName, ok := aliasMap[strings.ToLower(ref.Table)]; ok {
					field.SourceTable = tableName
					field.SourceAlias = ref.Table
					field.SourceColumn = ref.Column
				}
				break
			}
		}
	}

	return field
}

// buildWindowSpecMeta converts a WindowSpec AST to the model's WindowSpecMeta.
func buildWindowSpecMeta(ws *WindowSpec) *model.WindowSpecMeta {
	if ws == nil {
		return nil
	}
	meta := &model.WindowSpecMeta{}
	for _, p := range ws.PartitionBy {
		meta.PartitionBy = append(meta.PartitionBy, p.String())
	}
	for _, o := range ws.OrderBy {
		meta.OrderBy = append(meta.OrderBy, model.OrderByMeta{
			Expression: o.Expr.String(),
			Direction:  o.Direction,
		})
	}
	meta.FrameClause = ws.FrameClause
	return meta
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

	whereTokens := t.collectUntilKeywords("GROUP", "ORDER", "LIMIT", "HAVING")
	if len(whereTokens) == 0 {
		return nil, 0
	}

	// Parse the WHERE expression using the expression parser
	ep := NewExprParser(whereTokens, p.dialect)
	expr, err := ep.ParseExpression()
	if err != nil {
		// Fallback: try string-based parsing
		whereStr := joinTokens(whereTokens, 0, len(whereTokens))
		whereStr = strings.TrimSpace(whereStr)
		if whereStr == "" {
			return nil, 0
		}
		node, count := p.buildConditionTreeFallback(whereStr, aliasMap)
		return node, count
	}

	// Convert AST to ConditionNode tree
	node := ExprToConditionTree(expr, aliasMap)
	count := countConditions(node)
	return node, count
}

// countConditions counts the number of leaf CONDITION nodes in a tree.
func countConditions(node *model.ConditionNode) int {
	if node == nil {
		return 0
	}
	if node.Type == "CONDITION" {
		return 1
	}
	count := 0
	for _, child := range node.Children {
		count += countConditions(child)
	}
	return count
}

// buildConditionTreeFallback is the old string-based condition parser, used as fallback.
func (p *CustomParser) buildConditionTreeFallback(whereStr string, aliasMap map[string]string) (*model.ConditionNode, int) {
	whereStr = strings.TrimSpace(whereStr)
	if whereStr == "" {
		return nil, 0
	}
	whereStr = unwrapOuterParens(whereStr)
	parts, connectors := splitByLogicOp(whereStr)
	if len(parts) == 1 {
		cond := p.parseSingleCondition(parts[0], aliasMap)
		return cond, 1
	}
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
		child, c := p.buildConditionTreeFallback(part, aliasMap)
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

	// Operators ordered by length (longer first) to avoid partial matches
	operators := []string{"!=", "<>", ">=", "<=", "=", ">", "<"}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		for _, op := range operators {
			idx := findOpOutsideQuotes(part, op)
			if idx > 0 {
				conditions = append(conditions, model.JoinCondition{
					Left:     strings.TrimSpace(part[:idx]),
					Operator: op,
					Right:    strings.TrimSpace(part[idx+len(op):]),
				})
				break
			}
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

// parseHaving parses the HAVING clause into a condition tree.
func (p *CustomParser) parseHaving(t *tokenizer, aliasMap map[string]string) (*model.ConditionNode, int) {
	if !t.skipToKeyword("HAVING") {
		return nil, 0
	}

	havingTokens := t.collectUntilKeywords("ORDER", "LIMIT")
	if len(havingTokens) == 0 {
		return nil, 0
	}

	// Parse the HAVING expression using the expression parser
	ep := NewExprParser(havingTokens, p.dialect)
	expr, err := ep.ParseExpression()
	if err != nil {
		// Fallback: try string-based parsing
		havingStr := joinTokens(havingTokens, 0, len(havingTokens))
		havingStr = strings.TrimSpace(havingStr)
		if havingStr == "" {
			return nil, 0
		}
		node, count := p.buildConditionTreeFallback(havingStr, aliasMap)
		return node, count
	}

	node := ExprToConditionTree(expr, aliasMap)
	count := countConditions(node)
	return node, count
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
	// Handle SQL Server / PostgreSQL OFFSET...FETCH syntax
	if t.pos < len(t.tokens) {
		// Try to find OFFSET keyword for OFFSET...FETCH
		savedPos := t.pos
		if t.skipToKeyword("OFFSET") {
			if t.pos < len(t.tokens) {
				offsetVal := 0
				fmt.Sscanf(t.tokens[t.pos].value, "%d", &offsetVal)
				t.pos++
				// Skip ROWS/ROW
				if t.pos < len(t.tokens) {
					upper := strings.ToUpper(t.tokens[t.pos].value)
					if upper == "ROWS" || upper == "ROW" {
						t.pos++
					}
				}
				// Check for FETCH NEXT/LIMIT
				if t.pos < len(t.tokens) {
					upper := strings.ToUpper(t.tokens[t.pos].value)
					if upper == "FETCH" {
						t.pos++
						if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "NEXT" {
							t.pos++
						}
						limitVal := 0
						if t.pos < len(t.tokens) {
							fmt.Sscanf(t.tokens[t.pos].value, "%d", &limitVal)
							t.pos++
						}
						// Skip ROWS/ROW ONLY
						if t.pos < len(t.tokens) {
							upper := strings.ToUpper(t.tokens[t.pos].value)
							if upper == "ROWS" || upper == "ROW" {
								t.pos++
							}
						}
						if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "ONLY" {
							t.pos++
						}
						return &model.LimitMeta{Limit: limitVal, Offset: offsetVal}
					}
				}
				// OFFSET without FETCH
				return &model.LimitMeta{Limit: 0, Offset: offsetVal}
			}
		}
		t.pos = savedPos
	}

	// Standard LIMIT syntax (MySQL, PostgreSQL, SQLite)
	if !t.skipToKeyword("LIMIT") {
		return nil
	}

	if t.pos >= len(t.tokens) {
		return nil
	}

	// Handle LIMIT ALL (PostgreSQL)
	if strings.ToUpper(t.tokens[t.pos].value) == "ALL" {
		t.pos++
		return nil // no limit
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

// parseCTEs parses WITH [RECURSIVE] cte_name AS (SELECT ...) [, ...] and returns
// the CTE definitions plus the remaining tokens after the CTE clause.
func (p *CustomParser) parseCTEs(tokens []token) ([]model.CTEDefinition, []token) {
	if len(tokens) == 0 || strings.ToUpper(tokens[0].value) != "WITH" {
		return nil, tokens
	}

	pos := 1
	// Check for RECURSIVE
	if pos < len(tokens) && strings.ToUpper(tokens[pos].value) == "RECURSIVE" {
		pos++
	}

	var ctes []model.CTEDefinition
	for {
		if pos >= len(tokens) {
			break
		}

		// CTE name
		if tokens[pos].kind != tokenIdent && !isSQLKeyword(strings.ToUpper(tokens[pos].value)) {
			break
		}
		cteName := stripBackticks(tokens[pos].value)
		pos++

		// Optional column list: cte(col1, col2)
		var columns []string
		if pos < len(tokens) && tokens[pos].value == "(" {
			pos++ // skip (
			for pos < len(tokens) && tokens[pos].value != ")" {
				if tokens[pos].value != "," {
					columns = append(columns, stripBackticks(tokens[pos].value))
				}
				pos++
			}
			if pos < len(tokens) {
				pos++ // skip )
			}
		}

		// AS keyword
		if pos < len(tokens) && strings.ToUpper(tokens[pos].value) == "AS" {
			pos++
		}

		// ( SELECT ... ) — find matching paren
		if pos < len(tokens) && tokens[pos].value == "(" {
			pos++ // skip (
			depth := 1
			bodyStart := pos
			for pos < len(tokens) && depth > 0 {
				if tokens[pos].value == "(" {
					depth++
				}
				if tokens[pos].value == ")" {
					depth--
				}
				if depth > 0 {
					pos++
				}
			}
			// tokens[bodyStart:pos] is the CTE body (without outer parens)
			innerSQL := joinTokens(tokens, bodyStart, pos)
			if pos < len(tokens) {
				pos++ // skip closing )
			}

			ctes = append(ctes, model.CTEDefinition{
				ID:      utils.NewID("cte"),
				Name:    cteName,
				Columns: columns,
				RawSQL:  innerSQL,
			})
		}

		// Check for comma (next CTE) or end
		if pos < len(tokens) && tokens[pos].value == "," {
			pos++
			continue
		}
		break
	}

	return ctes, tokens[pos:]
}

// parseSetOperations checks for UNION/INTERSECT/EXCEPT after the main SELECT.
func (p *CustomParser) parseSetOperations(tokens []token, firstResult *model.SQLAnalysisResult) []model.SetOperation {
	// Find the position after the main SELECT statement
	// We look for UNION/INTERSECT/EXCEPT at the top level (depth 0)
	var ops []model.SetOperation
	depth := 0
	i := 0
	for i < len(tokens) {
		if tokens[i].value == "(" {
			depth++
		} else if tokens[i].value == ")" {
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && tokens[i].kind == tokenKeyword {
			upper := strings.ToUpper(tokens[i].value)
			opType := ""
			switch upper {
			case "UNION":
				opType = "UNION"
				if i+1 < len(tokens) && strings.ToUpper(tokens[i+1].value) == "ALL" {
					opType = "UNION ALL"
				}
			case "INTERSECT":
				opType = "INTERSECT"
			case "EXCEPT":
				opType = "EXCEPT"
			}
			if opType != "" {
				// Parse the right side using a separate parser instance to avoid
				// resetting the shared ID generator mid-parse.
				skipCount := 1
				if opType == "UNION ALL" {
					skipCount = 2
				}
				remaining := make([]token, len(tokens)-i-skipCount)
				copy(remaining, tokens[i+skipCount:])
				rightParser := NewCustomParserWithDialect(p.dialect.ID)
				rightResult, err := rightParser.parseSelectOnly(remaining)
				if err == nil {
					ops = append(ops, model.SetOperation{
						Type:  opType,
						Left:  firstResult,
						Right: rightResult,
					})
				}
				break
			}
		}
		i++
	}
	return ops
}

// parseSelectOnly parses a SELECT statement from tokens without resetting IDs.
// This is used internally by set operation parsing to avoid ID counter corruption.
func (p *CustomParser) parseSelectOnly(tokens []token) (*model.SQLAnalysisResult, error) {
	result := &model.SQLAnalysisResult{
		StatementType: "SELECT",
		Dialect:       string(p.dialect.ID),
	}

	// Parse CTEs if present
	var ctes []model.CTEDefinition
	if len(tokens) > 0 && strings.ToUpper(tokens[0].value) == "WITH" {
		ctes, tokens = p.parseCTEs(tokens)
		if len(ctes) > 0 {
			result.CTEs = ctes
		}
	}

	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string)

	for _, cte := range ctes {
		aliasMap[strings.ToLower(cte.Name)] = cte.Name
	}

	t := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	tables, joins, err := p.parseFromAndJoins(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, err
	}

	t2 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	fields := p.parseSelectFields(t2, aliasMap)

	t3 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	whereTree, whereCount := p.parseWhereClause(t3, aliasMap)

	t4 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	groupBy := p.parseGroupBy(t4, aliasMap)

	t4b := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	havingTree, havingCount := p.parseHaving(t4b, aliasMap)

	t5 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	orderBy := p.parseOrderBy(t5, aliasMap)

	t6 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	limit := p.parseLimit(t6)

	p.resolveJoinConditionAliases(joins, tableAliasMap)

	result.Tables = tables
	result.Joins = joins
	result.Fields = fields
	result.WhereTree = whereTree
	result.HavingTree = havingTree
	result.GroupBy = groupBy
	result.OrderBy = orderBy
	result.Limit = limit
	result.Risks = make([]model.RiskMeta, 0)

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

	p.populateTableFields(tables, fields, whereTree, joins)

	hasWindowFunc := false
	for _, f := range fields {
		if f.WindowSpec != nil {
			hasWindowFunc = true
			break
		}
	}

	result.Summary = model.SQLSummary{
		TableCount:    len(tables),
		JoinCount:     len(joins),
		FieldCount:    len(fields),
		WhereCount:    whereCount,
		HavingCount:   havingCount,
		HasGroupBy:    len(groupBy) > 0,
		HasOrderBy:    len(orderBy) > 0,
		HasLimit:      limit != nil,
		HasHaving:     havingCount > 0,
		HasWindowFunc: hasWindowFunc,
		HasCTE:        len(ctes) > 0,
	}

	result.Graph = p.buildGraph(tables, joins)

	return result, nil
}

// parseInsert handles INSERT INTO table (columns) VALUES/subquery.
func (p *CustomParser) parseInsert(sql string) (*model.SQLAnalysisResult, error) {
	utils.ResetIDs()
	result := &model.SQLAnalysisResult{
		StatementType: "INSERT",
		Dialect:       string(p.dialect.ID),
		Risks:         make([]model.RiskMeta, 0),
	}
	tokens := tokenize(sql, p.dialect)
	t := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}

	// Skip INSERT [INTO]
	t.skipToKeyword("INSERT")
	if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "INTO" {
		t.pos++
	}

	// Parse table name
	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string)
	table, err := p.parseTableRef(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, fmt.Errorf("parse INSERT table error: %w", err)
	}
	table.Role = "main"
	result.Tables = []model.TableMeta{*table}

	// Parse column list if present
	if t.pos < len(t.tokens) && t.tokens[t.pos].value == "(" {
		t.pos++ // skip (
		for t.pos < len(t.tokens) && t.tokens[t.pos].value != ")" {
			if t.tokens[t.pos].kind == tokenIdent {
				col := stripBackticks(t.tokens[t.pos].value)
				result.Fields = append(result.Fields, model.FieldMeta{
					ID:           utils.NewID("field"),
					OutputName:   col,
					SourceColumn: col,
					SourceTable:  table.Name,
					FieldType:    "column",
				})
			}
			t.pos++
			if t.pos < len(t.tokens) && t.tokens[t.pos].value == "," {
				t.pos++
			}
		}
		if t.pos < len(t.tokens) {
			t.pos++ // skip )
		}
	}

	result.Summary = model.SQLSummary{TableCount: 1}
	result.Graph = p.buildGraph(result.Tables, result.Joins)
	return result, nil
}

// parseUpdate handles UPDATE table SET assignments WHERE conditions.
func (p *CustomParser) parseUpdate(sql string) (*model.SQLAnalysisResult, error) {
	utils.ResetIDs()
	result := &model.SQLAnalysisResult{
		StatementType: "UPDATE",
		Dialect:       string(p.dialect.ID),
		Risks:         make([]model.RiskMeta, 0),
	}
	tokens := tokenize(sql, p.dialect)
	t := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}

	// Skip UPDATE
	t.skipToKeyword("UPDATE")

	// Parse table name
	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string)
	table, err := p.parseTableRef(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, fmt.Errorf("parse UPDATE table error: %w", err)
	}
	table.Role = "main"
	result.Tables = []model.TableMeta{*table}

	// Parse SET assignments
	t2 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	setFields := p.parseSetAssignments(t2, table.Name)
	result.Fields = setFields

	// Parse WHERE clause
	t3 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	whereTree, whereCount := p.parseWhereClause(t3, aliasMap)
	result.WhereTree = whereTree

	result.Summary = model.SQLSummary{
		TableCount: 1,
		WhereCount: whereCount,
		FieldCount: len(setFields),
	}
	result.Graph = p.buildGraph(result.Tables, result.Joins)
	return result, nil
}

// parseSetAssignments parses SET col1 = val1, col2 = val2 in UPDATE statements.
func (p *CustomParser) parseSetAssignments(t *tokenizer, tableName string) []model.FieldMeta {
	if !t.skipToKeyword("SET") {
		return nil
	}

	var fields []model.FieldMeta
	for t.pos < len(t.tokens) {
		// Stop at WHERE
		if strings.ToUpper(t.tokens[t.pos].value) == "WHERE" {
			break
		}

		// Read column name
		col := stripBackticks(t.tokens[t.pos].value)
		t.pos++

		// Skip = sign
		if t.pos < len(t.tokens) && t.tokens[t.pos].value == "=" {
			t.pos++
		}

		// Read value until comma or WHERE
		value := t.readUntilCommaOrKeyword()
		value = strings.TrimSpace(value)

		fields = append(fields, model.FieldMeta{
			ID:           utils.NewID("field"),
			OutputName:   col,
			SourceColumn: col,
			SourceTable:  tableName,
			Expression:   col + " = " + value,
			FieldType:    "column",
		})

		// Skip comma
		if t.pos < len(t.tokens) && t.tokens[t.pos].value == "," {
			t.pos++
		}
	}
	return fields
}

// parseDelete handles DELETE FROM table WHERE conditions.
func (p *CustomParser) parseDelete(sql string) (*model.SQLAnalysisResult, error) {
	utils.ResetIDs()
	result := &model.SQLAnalysisResult{
		StatementType: "DELETE",
		Dialect:       string(p.dialect.ID),
		Risks:         make([]model.RiskMeta, 0),
	}
	tokens := tokenize(sql, p.dialect)
	t := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}

	// Skip DELETE FROM
	t.skipToKeyword("DELETE")
	if t.pos < len(t.tokens) && strings.ToUpper(t.tokens[t.pos].value) == "FROM" {
		t.pos++
	}

	// Parse table name
	aliasMap := make(map[string]string)
	tableAliasMap := make(map[string]string)
	table, err := p.parseTableRef(t, aliasMap, tableAliasMap)
	if err != nil {
		return nil, fmt.Errorf("parse DELETE table error: %w", err)
	}
	table.Role = "main"
	result.Tables = []model.TableMeta{*table}

	// Parse WHERE clause
	t3 := &tokenizer{tokens: tokens, pos: 0, dialect: p.dialect}
	whereTree, whereCount := p.parseWhereClause(t3, aliasMap)
	result.WhereTree = whereTree

	result.Summary = model.SQLSummary{
		TableCount: 1,
		WhereCount: whereCount,
	}
	result.Graph = p.buildGraph(result.Tables, result.Joins)
	return result, nil
}
