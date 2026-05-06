package parser

import (
	"fmt"
	"sql-lens/internal/model"
	"sql-lens/internal/utils"
	"strings"
)

// ExprNode is the interface for all expression AST nodes.
type ExprNode interface {
	String() string
}

// ColumnRef represents a column reference: table.column or just column.
type ColumnRef struct {
	Table  string // empty if unqualified
	Column string
}

func (c *ColumnRef) String() string {
	if c.Table != "" {
		return c.Table + "." + c.Column
	}
	return c.Column
}

// FunctionCall represents a function call: name(args) [OVER (...)].
type FunctionCall struct {
	Name   string
	Args   []ExprNode
	IsStar bool // COUNT(*)
	Over   *WindowSpec
}

func (f *FunctionCall) String() string {
	if f.IsStar {
		return f.Name + "(*)"
	}
	args := make([]string, len(f.Args))
	for i, a := range f.Args {
		args[i] = a.String()
	}
	s := f.Name + "(" + strings.Join(args, ", ") + ")"
	if f.Over != nil {
		s += " OVER (" + f.Over.String() + ")"
	}
	return s
}

// WindowSpec represents a window specification: OVER (PARTITION BY ... ORDER BY ...).
type WindowSpec struct {
	PartitionBy []ExprNode
	OrderBy     []OrderByExpr
	FrameClause string // raw text of ROWS/RANGE clause
}

func (w *WindowSpec) String() string {
	var parts []string
	if len(w.PartitionBy) > 0 {
		pb := make([]string, len(w.PartitionBy))
		for i, p := range w.PartitionBy {
			pb[i] = p.String()
		}
		parts = append(parts, "PARTITION BY "+strings.Join(pb, ", "))
	}
	if len(w.OrderBy) > 0 {
		ob := make([]string, len(w.OrderBy))
		for i, o := range w.OrderBy {
			ob[i] = o.String()
		}
		parts = append(parts, "ORDER BY "+strings.Join(ob, ", "))
	}
	if w.FrameClause != "" {
		parts = append(parts, w.FrameClause)
	}
	return strings.Join(parts, " ")
}

// AggregateCall represents an aggregate function call with optional DISTINCT.
type AggregateCall struct {
	Name     string
	Args     []ExprNode
	Distinct bool
	Over     *WindowSpec
}

func (a *AggregateCall) String() string {
	distinct := ""
	if a.Distinct {
		distinct = "DISTINCT "
	}
	args := make([]string, len(a.Args))
	for i, arg := range a.Args {
		args[i] = arg.String()
	}
	s := a.Name + "(" + distinct + strings.Join(args, ", ") + ")"
	if a.Over != nil {
		s += " OVER (" + a.Over.String() + ")"
	}
	return s
}

// CaseExpr represents CASE ... WHEN ... THEN ... ELSE ... END.
type CaseExpr struct {
	Operand ExprNode // simple CASE: CASE x WHEN ...; nil for searched CASE
	Whens   []WhenClause
	Else    ExprNode
}

func (c *CaseExpr) String() string {
	s := "CASE"
	if c.Operand != nil {
		s += " " + c.Operand.String()
	}
	for _, w := range c.Whens {
		s += " WHEN " + w.Condition.String() + " THEN " + w.Result.String()
	}
	if c.Else != nil {
		s += " ELSE " + c.Else.String()
	}
	s += " END"
	return s
}

// WhenClause represents a WHEN condition THEN result clause.
type WhenClause struct {
	Condition ExprNode
	Result    ExprNode
}

// CastExpr represents CAST(expr AS type).
type CastExpr struct {
	Expr ExprNode
	Type string
}

func (c *CastExpr) String() string {
	return "CAST(" + c.Expr.String() + " AS " + c.Type + ")"
}

// TypeCastExpr represents PostgreSQL ::type cast.
type TypeCastExpr struct {
	Expr ExprNode
	Type string
}

func (t *TypeCastExpr) String() string {
	return t.Expr.String() + "::" + t.Type
}

// BinaryExpr represents a binary operation: left op right.
type BinaryExpr struct {
	Left     ExprNode
	Operator string
	Right    ExprNode
}

func (b *BinaryExpr) String() string {
	return b.Left.String() + " " + b.Operator + " " + b.Right.String()
}

// UnaryExpr represents a unary operation: NOT expr, -expr.
type UnaryExpr struct {
	Operator string
	Expr     ExprNode
}

func (u *UnaryExpr) String() string {
	return u.Operator + " " + u.Expr.String()
}

// Literal represents a literal value: string, number, NULL, TRUE, FALSE.
type Literal struct {
	Value string
	Kind  string // "string", "number", "null", "bool"
}

func (l *Literal) String() string {
	return l.Value
}

// SubqueryExpr represents a (SELECT ...) subquery expression.
type SubqueryExpr struct {
	SQL string
}

func (s *SubqueryExpr) String() string {
	return "(" + s.SQL + ")"
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr ExprNode
}

func (p *ParenExpr) String() string {
	return "(" + p.Expr.String() + ")"
}

// BetweenExpr represents expr [NOT] BETWEEN low AND high.
type BetweenExpr struct {
	Expr ExprNode
	Low  ExprNode
	High ExprNode
	Not  bool
}

func (b *BetweenExpr) String() string {
	not := ""
	if b.Not {
		not = "NOT "
	}
	return b.Expr.String() + " " + not + "BETWEEN " + b.Low.String() + " AND " + b.High.String()
}

// InExpr represents expr [NOT] IN (values) or expr [NOT] IN (subquery).
type InExpr struct {
	Expr     ExprNode
	Values   []ExprNode
	Subquery *SubqueryExpr
	Not      bool
}

func (i *InExpr) String() string {
	not := ""
	if i.Not {
		not = "NOT "
	}
	if i.Subquery != nil {
		return i.Expr.String() + " " + not + "IN " + i.Subquery.String()
	}
	vals := make([]string, len(i.Values))
	for j, v := range i.Values {
		vals[j] = v.String()
	}
	return i.Expr.String() + " " + not + "IN (" + strings.Join(vals, ", ") + ")"
}

// IsNullExpr represents expr IS [NOT] NULL.
type IsNullExpr struct {
	Expr ExprNode
	Not  bool
}

func (i *IsNullExpr) String() string {
	not := ""
	if i.Not {
		not = "NOT "
	}
	return i.Expr.String() + " IS " + not + "NULL"
}

// LikeExpr represents expr [NOT] LIKE pattern.
type LikeExpr struct {
	Expr    ExprNode
	Pattern ExprNode
	Not     bool
}

func (l *LikeExpr) String() string {
	not := ""
	if l.Not {
		not = "NOT "
	}
	return l.Expr.String() + " " + not + "LIKE " + l.Pattern.String()
}

// OrderByExpr represents an expression with optional direction (for ORDER BY and window specs).
type OrderByExpr struct {
	Expr      ExprNode
	Direction string // "ASC" or "DESC"
}

func (o *OrderByExpr) String() string {
	return o.Expr.String() + " " + o.Direction
}

// ExprParser is a recursive-descent expression parser.
type ExprParser struct {
	tokens  []token
	pos     int
	dialect DialectConfig
}

// NewExprParser creates a new expression parser for the given tokens.
func NewExprParser(tokens []token, dialect DialectConfig) *ExprParser {
	return &ExprParser{tokens: tokens, pos: 0, dialect: dialect}
}

// ParseExpression parses a full expression (lowest precedence: OR).
func (ep *ExprParser) ParseExpression() (ExprNode, error) {
	return ep.parseOr()
}

// parseOr handles: expr OR expr OR ...
func (ep *ExprParser) parseOr() (ExprNode, error) {
	left, err := ep.parseAnd()
	if err != nil {
		return nil, err
	}

	for ep.pos < len(ep.tokens) && ep.currentUpper() == "OR" {
		ep.pos++ // skip OR
		right, err := ep.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Operator: "OR", Right: right}
	}
	return left, nil
}

// parseAnd handles: expr AND expr AND ...
func (ep *ExprParser) parseAnd() (ExprNode, error) {
	left, err := ep.parseNot()
	if err != nil {
		return nil, err
	}

	for ep.pos < len(ep.tokens) && ep.currentUpper() == "AND" {
		ep.pos++ // skip AND
		right, err := ep.parseNot()
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Left: left, Operator: "AND", Right: right}
	}
	return left, nil
}

// parseNot handles: NOT expr
func (ep *ExprParser) parseNot() (ExprNode, error) {
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "NOT" {
		ep.pos++ // skip NOT
		expr, err := ep.parseNot()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: "NOT", Expr: expr}, nil
	}
	return ep.parseComparison()
}

// parseComparison handles: expr op expr, expr LIKE, expr IN, expr BETWEEN, expr IS NULL, expr::type
func (ep *ExprParser) parseComparison() (ExprNode, error) {
	left, err := ep.parseAddSub()
	if err != nil {
		return nil, err
	}

	if ep.pos >= len(ep.tokens) {
		return left, nil
	}

	// PostgreSQL ::type cast (postfix)
	if ep.tokens[ep.pos].value == "::" {
		ep.pos++ // skip ::
		if ep.pos < len(ep.tokens) {
			typeName := ep.readTypeName()
			left = &TypeCastExpr{Expr: left, Type: typeName}
		}
		return left, nil
	}

	upper := ep.currentUpper()

	// IS [NOT] NULL
	if upper == "IS" {
		ep.pos++ // skip IS
		not := false
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "NOT" {
			not = true
			ep.pos++ // skip NOT
		}
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "NULL" {
			ep.pos++ // skip NULL
			return &IsNullExpr{Expr: left, Not: not}, nil
		}
		// IS TRUE/FALSE
		if ep.pos < len(ep.tokens) && (ep.currentUpper() == "TRUE" || ep.currentUpper() == "FALSE") {
			val := ep.tokens[ep.pos].value
			ep.pos++
			op := "IS"
			if not {
				op = "IS NOT"
			}
			return &BinaryExpr{Left: left, Operator: op, Right: &Literal{Value: val, Kind: "bool"}}, nil
		}
		return left, nil
	}

	// [NOT] LIKE
	if upper == "LIKE" || (upper == "NOT" && ep.pos+1 < len(ep.tokens) && strings.ToUpper(ep.tokens[ep.pos+1].value) == "LIKE") {
		not := false
		if upper == "NOT" {
			not = true
			ep.pos++ // skip NOT
		}
		ep.pos++ // skip LIKE
		pattern, err := ep.parseAddSub()
		if err != nil {
			return nil, err
		}
		return &LikeExpr{Expr: left, Pattern: pattern, Not: not}, nil
	}

	// [NOT] IN
	if upper == "IN" || (upper == "NOT" && ep.pos+1 < len(ep.tokens) && strings.ToUpper(ep.tokens[ep.pos+1].value) == "IN") {
		not := false
		if upper == "NOT" {
			not = true
			ep.pos++ // skip NOT
		}
		ep.pos++ // skip IN
		return ep.parseInValues(left, not)
	}

	// [NOT] BETWEEN
	if upper == "BETWEEN" || (upper == "NOT" && ep.pos+1 < len(ep.tokens) && strings.ToUpper(ep.tokens[ep.pos+1].value) == "BETWEEN") {
		not := false
		if upper == "NOT" {
			not = true
			ep.pos++ // skip NOT
		}
		ep.pos++ // skip BETWEEN
		low, err := ep.parseAddSub()
		if err != nil {
			return nil, err
		}
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "AND" {
			ep.pos++ // skip AND
		}
		high, err := ep.parseAddSub()
		if err != nil {
			return nil, err
		}
		return &BetweenExpr{Expr: left, Low: low, High: high, Not: not}, nil
	}

	// Comparison operators: =, !=, <>, >, <, >=, <=
	if ep.isComparisonOp() {
		op := ep.tokens[ep.pos].value
		ep.pos++
		right, err := ep.parseAddSub()
		if err != nil {
			return nil, err
		}
		return &BinaryExpr{Left: left, Operator: op, Right: right}, nil
	}

	return left, nil
}

// parseInValues handles the values part of IN (...)
func (ep *ExprParser) parseInValues(left ExprNode, not bool) (ExprNode, error) {
	if ep.pos >= len(ep.tokens) || ep.tokens[ep.pos].value != "(" {
		return left, nil
	}
	ep.pos++ // skip (

	// Check for subquery: ( SELECT ...
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "SELECT" {
		sql := ep.readUntilMatchingParen()
		return &InExpr{Expr: left, Subquery: &SubqueryExpr{SQL: sql}, Not: not}, nil
	}

	// Parse value list
	var values []ExprNode
	for ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value != ")" {
		val, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		values = append(values, val)
		if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "," {
			ep.pos++ // skip comma
		}
	}
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++ // skip )
	}
	return &InExpr{Expr: left, Values: values, Not: not}, nil
}

// parseAddSub handles: expr + expr, expr - expr, expr || expr
func (ep *ExprParser) parseAddSub() (ExprNode, error) {
	left, err := ep.parseMulDiv()
	if err != nil {
		return nil, err
	}

	for ep.pos < len(ep.tokens) {
		op := ep.tokens[ep.pos].value
		if op == "+" || op == "-" || op == "||" {
			ep.pos++
			right, err := ep.parseMulDiv()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

// parseMulDiv handles: expr * expr, expr / expr, expr % expr
func (ep *ExprParser) parseMulDiv() (ExprNode, error) {
	left, err := ep.parseUnary()
	if err != nil {
		return nil, err
	}

	for ep.pos < len(ep.tokens) {
		op := ep.tokens[ep.pos].value
		if op == "*" || op == "/" || op == "%" {
			ep.pos++
			right, err := ep.parseUnary()
			if err != nil {
				return nil, err
			}
			left = &BinaryExpr{Left: left, Operator: op, Right: right}
		} else {
			break
		}
	}
	return left, nil
}

// parseUnary handles: -expr, +expr
func (ep *ExprParser) parseUnary() (ExprNode, error) {
	if ep.pos < len(ep.tokens) {
		op := ep.tokens[ep.pos].value
		if op == "-" || op == "+" {
			ep.pos++
			expr, err := ep.parseUnary()
			if err != nil {
				return nil, err
			}
			return &UnaryExpr{Operator: op, Expr: expr}, nil
		}
	}
	return ep.parsePrimary()
}

// parsePrimary handles: literals, column refs, function calls, CASE, CAST, parenthesized exprs, subqueries
func (ep *ExprParser) parsePrimary() (ExprNode, error) {
	if ep.pos >= len(ep.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	tok := ep.tokens[ep.pos]
	upper := strings.ToUpper(tok.value)

	// NULL literal
	if upper == "NULL" && tok.kind == tokenKeyword {
		ep.pos++
		return &Literal{Value: "NULL", Kind: "null"}, nil
	}

	// TRUE/FALSE literals
	if (upper == "TRUE" || upper == "FALSE") && tok.kind == tokenKeyword {
		ep.pos++
		return &Literal{Value: tok.value, Kind: "bool"}, nil
	}

	// String literal
	if tok.kind == tokenString {
		ep.pos++
		return &Literal{Value: tok.value, Kind: "string"}, nil
	}

	// Number literal
	if tok.kind == tokenNumber {
		ep.pos++
		return &Literal{Value: tok.value, Kind: "number"}, nil
	}

	// Star (standalone *)
	if tok.kind == tokenStar {
		ep.pos++
		return &ColumnRef{Table: "", Column: "*"}, nil
	}

	// CASE expression
	if upper == "CASE" && tok.kind == tokenKeyword {
		return ep.parseCaseExpr()
	}

	// CAST expression
	if upper == "CAST" && tok.kind == tokenKeyword {
		return ep.parseCastExpr()
	}

	// [NOT] EXISTS (subquery)
	if upper == "EXISTS" && tok.kind == tokenKeyword {
		ep.pos++ // skip EXISTS
		if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "(" {
			ep.pos++ // skip (
			if ep.pos < len(ep.tokens) && ep.currentUpper() == "SELECT" {
				sql := ep.readUntilMatchingParen()
				return &FunctionCall{Name: "EXISTS", Args: []ExprNode{&SubqueryExpr{SQL: sql}}}, nil
			}
			// Not a subquery, try parsing as expression
			expr, err := ep.parseOr()
			if err == nil && ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
				ep.pos++
			}
			return &FunctionCall{Name: "EXISTS", Args: []ExprNode{expr}}, nil
		}
		return nil, fmt.Errorf("EXISTS must be followed by (")
	}

	// Parenthesized expression or subquery
	if tok.value == "(" {
		ep.pos++ // skip (
		// Check for subquery
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "SELECT" {
			sql := ep.readUntilMatchingParen()
			return &SubqueryExpr{SQL: sql}, nil
		}
		expr, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
			ep.pos++ // skip )
		}
		return &ParenExpr{Expr: expr}, nil
	}

	// Identifier: could be column ref, function call, or qualified name
	if tok.kind == tokenIdent || (tok.kind == tokenKeyword && !ep.isReservedKeyword(upper)) {
		return ep.parseIdentifierOrFunction()
	}

	return nil, fmt.Errorf("unexpected token: %s", tok.value)
}

// parseIdentifierOrFunction handles identifiers that could be columns, qualified columns, or function calls.
func (ep *ExprParser) parseIdentifierOrFunction() (ExprNode, error) {
	name := ep.tokens[ep.pos].value
	ep.pos++

	// Check for qualified name: table.column or schema.table.column
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].kind == tokenDot {
		ep.pos++ // skip dot
		if ep.pos < len(ep.tokens) {
			colName := ep.tokens[ep.pos].value
			ep.pos++

			// Check for function call on qualified name: schema.func(...)
			if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "(" {
				return ep.parseFunctionArgs(name + "." + colName)
			}

			return &ColumnRef{Table: name, Column: colName}, nil
		}
	}

	// Check for function call: name(...)
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "(" {
		return ep.parseFunctionArgs(name)
	}

	// Simple column reference
	return &ColumnRef{Table: "", Column: name}, nil
}

// parseFunctionArgs parses the argument list of a function call and optional OVER clause.
func (ep *ExprParser) parseFunctionArgs(name string) (ExprNode, error) {
	ep.pos++ // skip (

	upperName := strings.ToUpper(name)

	// Handle COUNT(*)
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].kind == tokenStar {
		ep.pos++ // skip *
		if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
			ep.pos++ // skip )
		}
		fn := &FunctionCall{Name: upperName, IsStar: true}
		ep.maybeParseOver(fn)
		if info, ok := LookupFunction(upperName, ep.dialect.ID); ok && info.IsAggregate {
			return &AggregateCall{Name: upperName, Args: nil, Over: fn.Over}, nil
		}
		return fn, nil
	}

	// Handle DISTINCT
	distinct := false
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "DISTINCT" {
		distinct = true
		ep.pos++ // skip DISTINCT
	}

	// Special handling for GROUP_CONCAT: it may contain ORDER BY and SEPARATOR inside args
	if upperName == "GROUP_CONCAT" {
		return ep.parseGroupConcatArgs(distinct)
	}

	// Parse arguments
	var args []ExprNode
	for ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value != ")" {
		arg, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "," {
			ep.pos++ // skip comma
		}
	}
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++ // skip )
	}

	// Determine if this is an aggregate or regular function
	if info, ok := LookupFunction(upperName, ep.dialect.ID); ok && info.IsAggregate {
		fn := &AggregateCall{Name: upperName, Args: args, Distinct: distinct}
		ep.maybeParseOver(fn)
		return fn, nil
	}

	fn := &FunctionCall{Name: upperName, Args: args}
	ep.maybeParseOver(fn)
	return fn, nil
}

// parseGroupConcatArgs handles GROUP_CONCAT's special syntax:
// GROUP_CONCAT([DISTINCT] expr [ORDER BY expr [ASC|DESC]] [SEPARATOR 'sep'])
func (ep *ExprParser) parseGroupConcatArgs(distinct bool) (ExprNode, error) {
	start := ep.pos
	depth := 0

	// Collect all tokens inside the parentheses, handling ORDER BY and SEPARATOR specially
	for ep.pos < len(ep.tokens) {
		val := ep.tokens[ep.pos].value
		if val == "(" {
			depth++
			ep.pos++
			continue
		}
		if val == ")" {
			if depth == 0 {
				break
			}
			depth--
			ep.pos++
			continue
		}
		ep.pos++
	}

	// Build the raw expression from collected tokens
	argStr := joinTokens(ep.tokens, start, ep.pos)

	// Skip closing )
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++
	}

	argNode := &Literal{Value: argStr, Kind: "string"}
	fn := &AggregateCall{
		Name:     "GROUP_CONCAT",
		Args:     []ExprNode{argNode},
		Distinct: distinct,
	}
	ep.maybeParseOver(fn)
	return fn, nil
}

// maybeParseOver checks for an OVER clause after a function/aggregate call.
func (ep *ExprParser) maybeParseOver(fn interface{}) {
	if ep.pos >= len(ep.tokens) || ep.currentUpper() != "OVER" {
		return
	}
	ep.pos++ // skip OVER

	if ep.pos >= len(ep.tokens) || ep.tokens[ep.pos].value != "(" {
		return
	}
	ep.pos++ // skip (

	ws := &WindowSpec{}

	// PARTITION BY
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "PARTITION" {
		ep.pos++ // skip PARTITION
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "BY" {
			ep.pos++ // skip BY
			for {
				expr, err := ep.parseOr()
				if err != nil {
					break
				}
				ws.PartitionBy = append(ws.PartitionBy, expr)
				if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "," {
					ep.pos++
					continue
				}
				break
			}
		}
	}

	// ORDER BY
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "ORDER" {
		ep.pos++ // skip ORDER
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "BY" {
			ep.pos++ // skip BY
			for {
				expr, err := ep.parseOr()
				if err != nil {
					break
				}
				dir := "ASC"
				if ep.pos < len(ep.tokens) {
					u := ep.currentUpper()
					if u == "ASC" || u == "DESC" {
						dir = u
						ep.pos++
					}
				}
				ws.OrderBy = append(ws.OrderBy, OrderByExpr{Expr: expr, Direction: dir})
				if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "," {
					ep.pos++
					continue
				}
				break
			}
		}
	}

	// Frame clause: ROWS/RANGE/GROUPS BETWEEN ...
	if ep.pos < len(ep.tokens) {
		u := ep.currentUpper()
		if u == "ROWS" || u == "RANGE" || u == "GROUPS" {
			frameStart := ep.pos
			depth := 0
			for ep.pos < len(ep.tokens) {
				if ep.tokens[ep.pos].value == "(" {
					depth++
				}
				if ep.tokens[ep.pos].value == ")" {
					if depth == 0 {
						break
					}
					depth--
				}
				ep.pos++
			}
			var parts []string
			for i := frameStart; i < ep.pos; i++ {
				parts = append(parts, ep.tokens[i].value)
			}
			ws.FrameClause = strings.Join(parts, " ")
		}
	}

	// Closing )
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++
	}

	// Assign window spec to the appropriate function type
	switch f := fn.(type) {
	case *FunctionCall:
		f.Over = ws
	case *AggregateCall:
		f.Over = ws
	}
}

// parseCaseExpr parses CASE ... WHEN ... THEN ... ELSE ... END
func (ep *ExprParser) parseCaseExpr() (ExprNode, error) {
	ep.pos++ // skip CASE

	ce := &CaseExpr{}

	// Check if this is a simple CASE (CASE expr WHEN ...) or searched CASE (CASE WHEN condition ...)
	// We peek ahead to determine: if the first significant token after CASE is WHEN, it's searched
	if ep.pos < len(ep.tokens) && ep.currentUpper() != "WHEN" {
		// Simple CASE: CASE operand WHEN ...
		operand, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		ce.Operand = operand
	}

	// Parse WHEN clauses
	for ep.pos < len(ep.tokens) && ep.currentUpper() == "WHEN" {
		ep.pos++ // skip WHEN
		condition, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		if ep.pos < len(ep.tokens) && ep.currentUpper() == "THEN" {
			ep.pos++ // skip THEN
		}
		result, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		ce.Whens = append(ce.Whens, WhenClause{Condition: condition, Result: result})
	}

	// Parse ELSE
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "ELSE" {
		ep.pos++ // skip ELSE
		elseExpr, err := ep.parseOr()
		if err != nil {
			return nil, err
		}
		ce.Else = elseExpr
	}

	// Skip END
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "END" {
		ep.pos++
	}

	return ce, nil
}

// parseCastExpr parses CAST(expr AS type)
func (ep *ExprParser) parseCastExpr() (ExprNode, error) {
	ep.pos++ // skip CAST

	if ep.pos >= len(ep.tokens) || ep.tokens[ep.pos].value != "(" {
		return nil, fmt.Errorf("expected ( after CAST")
	}
	ep.pos++ // skip (

	expr, err := ep.parseOr()
	if err != nil {
		return nil, err
	}

	// Skip AS
	if ep.pos < len(ep.tokens) && ep.currentUpper() == "AS" {
		ep.pos++
	}

	typeName := ep.readTypeName()

	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++ // skip )
	}

	return &CastExpr{Expr: expr, Type: typeName}, nil
}

// readTypeName reads a SQL type name like VARCHAR(255), INTEGER, TIMESTAMP, etc.
func (ep *ExprParser) readTypeName() string {
	if ep.pos >= len(ep.tokens) {
		return ""
	}
	var parts []string
	parts = append(parts, ep.tokens[ep.pos].value)
	ep.pos++

	// Check for precision: TYPE(N) or TYPE(N, M)
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == "(" {
		ep.pos++ // skip (
		precParts := []string{"("}
		depth := 1
		for ep.pos < len(ep.tokens) && depth > 0 {
			if ep.tokens[ep.pos].value == "(" {
				depth++
			}
			if ep.tokens[ep.pos].value == ")" {
				depth--
			}
			if depth > 0 {
				precParts = append(precParts, ep.tokens[ep.pos].value)
			}
			ep.pos++
		}
		if depth == 0 {
			precParts = append(precParts, ")")
		}
		parts = append(parts, strings.Join(precParts, ""))
	}

	// Check for WITH TIME ZONE / WITHOUT TIME ZONE
	if ep.pos+1 < len(ep.tokens) {
		u := ep.currentUpper()
		if u == "WITH" || u == "WITHOUT" {
			parts = append(parts, ep.tokens[ep.pos].value)
			ep.pos++
			if ep.pos < len(ep.tokens) && ep.currentUpper() == "TIME" {
				parts = append(parts, ep.tokens[ep.pos].value)
				ep.pos++
				if ep.pos < len(ep.tokens) && ep.currentUpper() == "ZONE" {
					parts = append(parts, ep.tokens[ep.pos].value)
					ep.pos++
				}
			}
		}
	}

	return strings.Join(parts, " ")
}

// readUntilMatchingParen reads tokens from the current position until the matching ) is found.
// Returns the reconstructed SQL string (without the outer parens).
func (ep *ExprParser) readUntilMatchingParen() string {
	start := ep.pos
	depth := 1
	for ep.pos < len(ep.tokens) && depth > 0 {
		if ep.tokens[ep.pos].value == "(" {
			depth++
		}
		if ep.tokens[ep.pos].value == ")" {
			depth--
		}
		if depth > 0 {
			ep.pos++
		}
	}
	if ep.pos < len(ep.tokens) && ep.tokens[ep.pos].value == ")" {
		ep.pos++ // skip closing )
	}
	return joinTokens(ep.tokens, start, ep.pos-1)
}

// currentUpper returns the uppercase value of the current token.
func (ep *ExprParser) currentUpper() string {
	if ep.pos < len(ep.tokens) {
		return strings.ToUpper(ep.tokens[ep.pos].value)
	}
	return ""
}

// isComparisonOp checks if the current token is a comparison operator.
func (ep *ExprParser) isComparisonOp() bool {
	if ep.pos >= len(ep.tokens) {
		return false
	}
	switch ep.tokens[ep.pos].value {
	case "=", "!=", "<>", ">", "<", ">=", "<=":
		return true
	}
	return false
}

// isReservedKeyword checks if a keyword cannot be used as an identifier.
func (ep *ExprParser) isReservedKeyword(upper string) bool {
	switch upper {
	case "SELECT", "FROM", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING",
		"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"JOIN", "INNER", "LEFT", "RIGHT", "CROSS", "OUTER", "FULL", "NATURAL",
		"UNION", "INTERSECT", "EXCEPT", "SET", "INTO", "VALUES",
		"ON", "AS", "BY", "ALL", "DISTINCT":
		return true
	}
	return false
}

// CollectColumnRefs walks an expression AST and returns all column references.
func CollectColumnRefs(node ExprNode) []ColumnRef {
	if node == nil {
		return nil
	}
	var refs []ColumnRef
	switch n := node.(type) {
	case *ColumnRef:
		if n.Column != "*" {
			refs = append(refs, *n)
		}
	case *FunctionCall:
		for _, arg := range n.Args {
			refs = append(refs, CollectColumnRefs(arg)...)
		}
	case *AggregateCall:
		for _, arg := range n.Args {
			refs = append(refs, CollectColumnRefs(arg)...)
		}
	case *BinaryExpr:
		refs = append(refs, CollectColumnRefs(n.Left)...)
		refs = append(refs, CollectColumnRefs(n.Right)...)
	case *UnaryExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
	case *CaseExpr:
		refs = append(refs, CollectColumnRefs(n.Operand)...)
		for _, w := range n.Whens {
			refs = append(refs, CollectColumnRefs(w.Condition)...)
			refs = append(refs, CollectColumnRefs(w.Result)...)
		}
		refs = append(refs, CollectColumnRefs(n.Else)...)
	case *CastExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
	case *TypeCastExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
	case *ParenExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
	case *BetweenExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
		refs = append(refs, CollectColumnRefs(n.Low)...)
		refs = append(refs, CollectColumnRefs(n.High)...)
	case *InExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
		for _, v := range n.Values {
			refs = append(refs, CollectColumnRefs(v)...)
		}
	case *IsNullExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
	case *LikeExpr:
		refs = append(refs, CollectColumnRefs(n.Expr)...)
		refs = append(refs, CollectColumnRefs(n.Pattern)...)
	case *Literal, *SubqueryExpr:
		// no column refs
	}
	return refs
}

// ExprToConditionTree converts an expression AST into the ConditionNode tree format
// used by the frontend for WHERE clause visualization.
func ExprToConditionTree(expr ExprNode, aliasMap map[string]string) *model.ConditionNode {
	if expr == nil {
		return nil
	}
	switch n := expr.(type) {
	case *BinaryExpr:
		upper := strings.ToUpper(n.Operator)
		if upper == "AND" || upper == "OR" {
			left := ExprToConditionTree(n.Left, aliasMap)
			right := ExprToConditionTree(n.Right, aliasMap)
			node := &model.ConditionNode{
				ID:       utils.NewID("cond"),
				Type:     upper,
				Children: make([]*model.ConditionNode, 0),
			}
			if left != nil {
				node.Children = append(node.Children, left)
			}
			if right != nil {
				node.Children = append(node.Children, right)
			}
			return node
		}
		// Comparison operator → leaf condition
		return makeLeafCondition(n.Left, n.Operator, n.Right, aliasMap)
	case *IsNullExpr:
		op := "IS NULL"
		if n.Not {
			op = "IS NOT NULL"
		}
		return makeLeafCondition(n.Expr, op, &Literal{Value: "NULL", Kind: "null"}, aliasMap)
	case *InExpr:
		op := "IN"
		if n.Not {
			op = "NOT IN"
		}
		valStr := ""
		if n.Subquery != nil {
			valStr = n.Subquery.SQL
		} else {
			vals := make([]string, len(n.Values))
			for i, v := range n.Values {
				vals[i] = v.String()
			}
			valStr = "(" + strings.Join(vals, ", ") + ")"
		}
		return &model.ConditionNode{
			ID:       utils.NewID("cond"),
			Type:     "CONDITION",
			Expr:     expr.String(),
			Field:    extractFieldFromExpr(n.Expr),
			Operator: op,
			Value:    valStr,
		}
	case *BetweenExpr:
		op := "BETWEEN"
		if n.Not {
			op = "NOT BETWEEN"
		}
		return &model.ConditionNode{
			ID:       utils.NewID("cond"),
			Type:     "CONDITION",
			Expr:     expr.String(),
			Field:    extractFieldFromExpr(n.Expr),
			Operator: op,
			Value:    n.Low.String() + " AND " + n.High.String(),
		}
	case *LikeExpr:
		op := "LIKE"
		if n.Not {
			op = "NOT LIKE"
		}
		return makeLeafCondition(n.Expr, op, n.Pattern, aliasMap)
	case *UnaryExpr:
		if strings.ToUpper(n.Operator) == "NOT" {
			child := ExprToConditionTree(n.Expr, aliasMap)
			if child != nil {
				return &model.ConditionNode{
					ID:       utils.NewID("cond"),
					Type:     "NOT",
					Children: []*model.ConditionNode{child},
				}
			}
		}
		return &model.ConditionNode{
			ID:   utils.NewID("cond"),
			Type: "CONDITION",
			Expr: expr.String(),
		}
	default:
		return &model.ConditionNode{
			ID:   utils.NewID("cond"),
			Type: "CONDITION",
			Expr: expr.String(),
		}
	}
}

// makeLeafCondition creates a leaf ConditionNode from a binary comparison.
func makeLeafCondition(left ExprNode, op string, right ExprNode, aliasMap map[string]string) *model.ConditionNode {
	node := &model.ConditionNode{
		ID:       utils.NewID("cond"),
		Type:     "CONDITION",
		Expr:     left.String() + " " + op + " " + right.String(),
		Field:    extractFieldFromExpr(left),
		Operator: op,
		Value:    right.String(),
	}
	// Try to resolve table from column ref
	if cr, ok := left.(*ColumnRef); ok && cr.Table != "" {
		if tableName, found := aliasMap[strings.ToLower(cr.Table)]; found {
			node.Table = tableName
			node.Field = cr.Column
		}
	}
	return node
}

// extractFieldFromExpr extracts a field name string from an expression node.
func extractFieldFromExpr(expr ExprNode) string {
	switch n := expr.(type) {
	case *ColumnRef:
		return n.String()
	case *FunctionCall:
		return n.Name + "(...)"
	default:
		return expr.String()
	}
}
