package parser

import (
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokenIdent    tokenKind = iota
	tokenKeyword            // SQL keywords (SELECT, FROM, etc.)
	tokenNumber             // numeric literals
	tokenString             // string literals ('...', "...", $$...$$)
	tokenOperator           // operators (=, !=, <>, +, -, *, /, etc.)
	tokenPunct              // punctuation (, ;)
	tokenDot                // .
	tokenLParen             // (
	tokenRParen             // )
	tokenStar               // *
)

type token struct {
	value string
	kind  tokenKind
}

type tokenizer struct {
	tokens  []token
	pos     int
	dialect DialectConfig
}

// newTokenizer creates a tokenizer with MySQL dialect (backward-compatible).
func newTokenizer(sql string) *tokenizer {
	return newDialectTokenizer(sql, GetDialectConfig(DialectMySQL))
}

// newDialectTokenizer creates a tokenizer with the specified dialect config.
func newDialectTokenizer(sql string, dialect DialectConfig) *tokenizer {
	return &tokenizer{
		tokens:  tokenize(sql, dialect),
		pos:     0,
		dialect: dialect,
	}
}

func tokenize(sql string, dialect DialectConfig) []token {
	var tokens []token
	var current strings.Builder
	i := 0

	for i < len(sql) {
		ch := rune(sql[i])

		// Skip whitespace
		if unicode.IsSpace(ch) {
			i++
			continue
		}

		// Single-quoted string
		if ch == '\'' {
			j := i + 1
			for j < len(sql) {
				if sql[j] == '\'' {
					if j+1 < len(sql) && sql[j+1] == '\'' {
						j += 2
						continue
					}
					break
				}
				// MySQL backslash escapes
				if dialect.StringEscapes && sql[j] == '\\' && j+1 < len(sql) {
					j += 2
					continue
				}
				j++
			}
			tokens = append(tokens, token{value: sql[i : j+1], kind: tokenString})
			i = j + 1
			continue
		}

		// Double-quoted string (or identifier in PG/Oracle)
		if ch == '"' {
			j := i + 1
			for j < len(sql) {
				if sql[j] == '"' {
					if j+1 < len(sql) && sql[j+1] == '"' {
						j += 2
						continue
					}
					break
				}
				j++
			}
			quoted := sql[i : j+1]
			// In PG/Oracle, double quotes are identifiers; in MySQL/SQL Server, they're strings
			if dialect.IdentifierQuote == '"' {
				tokens = append(tokens, token{value: quoted, kind: tokenIdent})
			} else {
				tokens = append(tokens, token{value: quoted, kind: tokenString})
			}
			i = j + 1
			continue
		}

		// Backtick-quoted identifier (MySQL, SQLite)
		if ch == '`' {
			j := i + 1
			for j < len(sql) && sql[j] != '`' {
				j++
			}
			tokens = append(tokens, token{value: sql[i : j+1], kind: tokenIdent})
			i = j + 1
			continue
		}

		// Bracket-quoted identifier [identifier] (SQL Server)
		if dialect.SupportBracketIdent && ch == '[' {
			j := i + 1
			for j < len(sql) && sql[j] != ']' {
				j++
			}
			if j < len(sql) {
				tokens = append(tokens, token{value: sql[i : j+1], kind: tokenIdent})
				i = j + 1
			} else {
				tokens = append(tokens, token{value: string(ch), kind: tokenPunct})
				i++
			}
			continue
		}

		// Dollar-quoted string $$...$$ (PostgreSQL)
		if dialect.SupportDoubleDollar && ch == '$' && i+1 < len(sql) && sql[i+1] == '$' {
			j := i + 2
			for j+1 < len(sql) {
				if sql[j] == '$' && sql[j+1] == '$' {
					j += 2
					break
				}
				j++
			}
			tokens = append(tokens, token{value: sql[i:j], kind: tokenString})
			i = j
			continue
		}

		// PostgreSQL :: type cast operator
		if dialect.SupportTypeCast && ch == ':' && i+1 < len(sql) && sql[i+1] == ':' {
			tokens = append(tokens, token{value: "::", kind: tokenOperator})
			i += 2
			continue
		}

		// PostgreSQL/Oracle || concatenation operator
		if dialect.SupportConcatOp && ch == '|' && i+1 < len(sql) && sql[i+1] == '|' {
			tokens = append(tokens, token{value: "||", kind: tokenOperator})
			i += 2
			continue
		}

		// Parentheses
		if ch == '(' {
			tokens = append(tokens, token{value: "(", kind: tokenLParen})
			i++
			continue
		}
		if ch == ')' {
			tokens = append(tokens, token{value: ")", kind: tokenRParen})
			i++
			continue
		}

		// Punctuation: comma, semicolon
		if ch == ',' || ch == ';' {
			kind := tokenPunct
			if ch == ',' {
				kind = tokenPunct
			}
			tokens = append(tokens, token{value: string(ch), kind: kind})
			i++
			continue
		}

		// Multi-char operators: !=, <>, >=, <=
		if ch == '!' && i+1 < len(sql) && sql[i+1] == '=' {
			tokens = append(tokens, token{value: "!=", kind: tokenOperator})
			i += 2
			continue
		}
		if ch == '<' {
			if i+1 < len(sql) && sql[i+1] == '>' {
				tokens = append(tokens, token{value: "<>", kind: tokenOperator})
				i += 2
				continue
			}
			if i+1 < len(sql) && sql[i+1] == '=' {
				tokens = append(tokens, token{value: "<=", kind: tokenOperator})
				i += 2
				continue
			}
			tokens = append(tokens, token{value: "<", kind: tokenOperator})
			i++
			continue
		}
		if ch == '>' && i+1 < len(sql) && sql[i+1] == '=' {
			tokens = append(tokens, token{value: ">=", kind: tokenOperator})
			i += 2
			continue
		}

		// Single-char operators
		if ch == '=' || ch == '>' {
			tokens = append(tokens, token{value: string(ch), kind: tokenOperator})
			i++
			continue
		}

		// Arithmetic operators: +, -, /, %
		if ch == '+' || ch == '-' || ch == '/' || ch == '%' {
			tokens = append(tokens, token{value: string(ch), kind: tokenOperator})
			i++
			continue
		}

		// Star/wildcard
		if ch == '*' {
			tokens = append(tokens, token{value: "*", kind: tokenStar})
			i++
			continue
		}

		// Dot
		if ch == '.' {
			tokens = append(tokens, token{value: ".", kind: tokenDot})
			i++
			continue
		}

		// Numbers
		if unicode.IsDigit(ch) {
			current.Reset()
			for i < len(sql) && (unicode.IsDigit(rune(sql[i])) || sql[i] == '.') {
				current.WriteByte(sql[i])
				i++
			}
			tokens = append(tokens, token{value: current.String(), kind: tokenNumber})
			continue
		}

		// Identifiers and keywords
		if unicode.IsLetter(ch) || ch == '_' {
			current.Reset()
			for i < len(sql) && (unicode.IsLetter(rune(sql[i])) || unicode.IsDigit(rune(sql[i])) || sql[i] == '_') {
				current.WriteByte(sql[i])
				i++
			}
			val := current.String()
			kind := tokenIdent
			if isSQLKeyword(strings.ToUpper(val)) {
				kind = tokenKeyword
			}
			tokens = append(tokens, token{value: val, kind: kind})
			continue
		}

		// Skip unknown characters (e.g. ?)
		i++
	}

	return tokens
}

// isSQLKeyword checks if a string is a SQL keyword.
func isSQLKeyword(s string) bool {
	switch s {
	case "SELECT", "FROM", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING",
		"LEFT", "RIGHT", "INNER", "JOIN", "CROSS", "OUTER", "ON",
		"AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL",
		"AS", "ASC", "DESC", "BY", "UNION", "ALL", "DISTINCT",
		"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"SET", "INTO", "VALUES", "CASE", "WHEN", "THEN", "ELSE", "END",
		"EXISTS", "ANY", "SOME", "OFFSET", "FETCH", "WITH", "RECURSIVE",
		"FOR", "NOWAIT", "WAIT", "INTERSECT", "EXCEPT", "TRUE", "FALSE",
		"CAST", "OVER", "PARTITION", "ROWS", "RANGE", "GROUPS",
		"TOP", "PERCENT", "TIES", "NOLOCK", "ROWNUM", "ROWID",
		"LATERAL", "NATURAL", "FULL", "USING",
		"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CONSTRAINT",
		"INDEX", "TABLE", "VIEW", "DATABASE", "SCHEMA",
		"IF", "REPLACE", "CONFLICT", "RETURNING", "DO", "NOTHING",
		"CONNECT", "PRIOR", "START", "LEVEL":
		return true
	}
	return false
}

func (t *tokenizer) skipToKeyword(keyword string) bool {
	upper := strings.ToUpper(keyword)
	for t.pos < len(t.tokens) {
		if t.tokens[t.pos].kind == tokenKeyword && strings.ToUpper(t.tokens[t.pos].value) == upper {
			t.pos++
			return true
		}
		t.pos++
	}
	return false
}

// joinTokens reconstructs source text from tokens, handling dots and parentheses spacing.
func joinTokens(tokens []token, start, end int) string {
	var b strings.Builder
	for i := start; i < end; i++ {
		if i > start {
			prev := tokens[i-1]
			curr := tokens[i]
			// No space around dots
			if prev.kind == tokenDot || curr.kind == tokenDot {
				// no space
			} else if prev.value == "(" {
				// no space after opening paren
			} else if curr.value == ")" {
				// no space before closing paren
			} else if curr.value == "(" && isLikelyFunc(prev.value) {
				// no space between function name and (
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString(tokens[i].value)
	}
	return b.String()
}

// isLikelyFunc returns true if token looks like a function name (not a keyword).
func isLikelyFunc(tok string) bool {
	return !isSQLKeyword(strings.ToUpper(tok))
}

// readUntilKeyword reads tokens until a keyword is found, respecting parenthesis depth.
func (t *tokenizer) readUntilKeyword(keywords ...string) string {
	start := t.pos
	depth := 0

	for t.pos < len(t.tokens) {
		val := t.tokens[t.pos].value

		if val == "(" {
			depth++
			t.pos++
			continue
		}
		if val == ")" {
			if depth > 0 {
				depth--
			}
			t.pos++
			continue
		}

		if depth > 0 {
			t.pos++
			continue
		}

		if t.tokens[t.pos].kind == tokenKeyword {
			upper := strings.ToUpper(val)
			for _, kw := range keywords {
				if upper == strings.ToUpper(kw) {
					return joinTokens(t.tokens, start, t.pos)
				}
			}
		}
		t.pos++
	}

	return joinTokens(t.tokens, start, t.pos)
}

// collectUntilKeywords returns the tokens from current position until a keyword is found.
func (t *tokenizer) collectUntilKeywords(keywords ...string) []token {
	start := t.pos
	depth := 0

	for t.pos < len(t.tokens) {
		val := t.tokens[t.pos].value

		if val == "(" {
			depth++
			t.pos++
			continue
		}
		if val == ")" {
			if depth > 0 {
				depth--
			}
			t.pos++
			continue
		}

		if depth == 0 && t.tokens[t.pos].kind == tokenKeyword {
			upper := strings.ToUpper(val)
			for _, kw := range keywords {
				if upper == strings.ToUpper(kw) {
					result := make([]token, t.pos-start)
					copy(result, t.tokens[start:t.pos])
					return result
				}
			}
		}
		t.pos++
	}

	result := make([]token, t.pos-start)
	copy(result, t.tokens[start:t.pos])
	return result
}

// readUntilCommaOrKeyword reads tokens until a top-level comma or clause keyword.
func (t *tokenizer) readUntilCommaOrKeyword() string {
	start := t.pos
	depth := 0

	for t.pos < len(t.tokens) {
		val := t.tokens[t.pos].value

		if val == "(" {
			depth++
			t.pos++
			continue
		}
		if val == ")" {
			if depth > 0 {
				depth--
			}
			t.pos++
			continue
		}

		if depth > 0 {
			t.pos++
			continue
		}

		// Only break on comma at depth 0
		if val == "," {
			break
		}

		if t.tokens[t.pos].kind == tokenKeyword {
			upper := strings.ToUpper(val)
			if upper == "FROM" || isClauseKeyword(upper) {
				break
			}
		}
		t.pos++
	}

	return joinTokens(t.tokens, start, t.pos)
}

// currentToken returns the current token value without advancing.
func (t *tokenizer) currentToken() string {
	if t.pos < len(t.tokens) {
		return t.tokens[t.pos].value
	}
	return ""
}

// advance moves past the current token.
func (t *tokenizer) advance() {
	t.pos++
}
