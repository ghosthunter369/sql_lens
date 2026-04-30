package parser

import (
	"strings"
	"unicode"
)

type token struct {
	value string
}

type tokenizer struct {
	tokens []token
	pos    int
}

func newTokenizer(sql string) *tokenizer {
	t := &tokenizer{
		tokens: tokenize(sql),
		pos:    0,
	}
	return t
}

func tokenize(sql string) []token {
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
				j++
			}
			tokens = append(tokens, token{value: sql[i : j+1]})
			i = j + 1
			continue
		}

		// Double-quoted string
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
			tokens = append(tokens, token{value: sql[i : j+1]})
			i = j + 1
			continue
		}

		// Backtick-quoted identifier
		if ch == '`' {
			j := i + 1
			for j < len(sql) && sql[j] != '`' {
				j++
			}
			tokens = append(tokens, token{value: sql[i : j+1]})
			i = j + 1
			continue
		}

		// Parentheses and punctuation
		if strings.ContainsRune("(),;", ch) {
			tokens = append(tokens, token{value: string(ch)})
			i++
			continue
		}

		// Multi-char operators: !=, <>, >=, <=
		if ch == '!' && i+1 < len(sql) && sql[i+1] == '=' {
			tokens = append(tokens, token{value: "!="})
			i += 2
			continue
		}
		if ch == '<' {
			if i+1 < len(sql) && sql[i+1] == '>' {
				tokens = append(tokens, token{value: "<>"})
				i += 2
				continue
			}
			if i+1 < len(sql) && sql[i+1] == '=' {
				tokens = append(tokens, token{value: "<="})
				i += 2
				continue
			}
			tokens = append(tokens, token{value: "<"})
			i++
			continue
		}
		if ch == '>' && i+1 < len(sql) && sql[i+1] == '=' {
			tokens = append(tokens, token{value: ">="})
			i += 2
			continue
		}

		// Single char operators
		if strings.ContainsRune("=+-*/%<>", ch) {
			tokens = append(tokens, token{value: string(ch)})
			i++
			continue
		}

		// Dot (table.column or decimal)
		if ch == '.' {
			tokens = append(tokens, token{value: "."})
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
			tokens = append(tokens, token{value: current.String()})
			continue
		}

		// Identifiers and keywords
		if unicode.IsLetter(ch) || ch == '_' || ch == '*' {
			current.Reset()
			for i < len(sql) && (unicode.IsLetter(rune(sql[i])) || unicode.IsDigit(rune(sql[i])) || sql[i] == '_' || sql[i] == '*' || sql[i] == '?') {
				current.WriteByte(sql[i])
				i++
			}
			tokens = append(tokens, token{value: current.String()})
			continue
		}

		// Skip unknown characters
		i++
	}

	return tokens
}

func (t *tokenizer) skipToKeyword(keyword string) bool {
	upper := strings.ToUpper(keyword)
	for t.pos < len(t.tokens) {
		if strings.ToUpper(t.tokens[t.pos].value) == upper {
			t.pos++
			return true
		}
		t.pos++
	}
	return false
}

// joinTokens reconstructs source text from tokens, handling dots and parentheses spacing
func joinTokens(tokens []token, start, end int) string {
	var b strings.Builder
	for i := start; i < end; i++ {
		if i > start {
			prev := tokens[i-1].value
			curr := tokens[i].value
			// No space around dots
			if prev == "." || curr == "." {
				// no space
			} else if prev == "(" {
				// no space after opening paren
			} else if curr == ")" {
				// no space before closing paren
			} else if curr == "(" && isLikelyFunc(prev) {
				// no space between function name and (, e.g. IFNULL(
			} else {
				b.WriteString(" ")
			}
		}
		b.WriteString(tokens[i].value)
	}
	return b.String()
}

// isLikelyFunc returns true if token looks like a function name (not a keyword)
func isLikelyFunc(tok string) bool {
	upper := strings.ToUpper(tok)
	// It's a function name if it's not a SQL keyword
	switch upper {
	case "AND", "OR", "NOT", "IN", "LIKE", "BETWEEN", "IS", "NULL",
		"SELECT", "FROM", "WHERE", "GROUP", "ORDER", "LIMIT", "HAVING",
		"LEFT", "RIGHT", "INNER", "JOIN", "CROSS", "OUTER", "ON",
		"AS", "ASC", "DESC", "BY", "UNION", "ALL", "DISTINCT",
		"INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
		"SET", "INTO", "VALUES", "WHEN", "THEN", "ELSE", "END",
		"EXISTS", "ANY", "SOME", "OFFSET", "FETCH", "WITH", "RECURSIVE",
		"TRUE", "FALSE", "CASE", "INTERSECT", "EXCEPT":
		return false
	}
	// If it's a non-keyword identifier, likely a function name
	return true
}

// readUntilKeyword reads tokens until a keyword is found, respecting parenthesis depth
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

		upper := strings.ToUpper(val)
		isKw := false
		for _, kw := range keywords {
			if upper == strings.ToUpper(kw) {
				isKw = true
				break
			}
		}
		if isKw {
			break
		}
		t.pos++
	}

	return joinTokens(t.tokens, start, t.pos)
}

// readUntilCommaOrKeyword reads tokens until a top-level comma or clause keyword
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

		upper := strings.ToUpper(val)
		if upper == "FROM" || isClauseKeyword(upper) {
			break
		}
		t.pos++
	}

	return joinTokens(t.tokens, start, t.pos)
}

// currentToken returns the current token value without advancing
func (t *tokenizer) currentToken() string {
	if t.pos < len(t.tokens) {
		return t.tokens[t.pos].value
	}
	return ""
}

// advance moves past the current token
func (t *tokenizer) advance() {
	t.pos++
}
