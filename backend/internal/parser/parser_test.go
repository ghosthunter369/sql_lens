package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestBasicSelect tests a simple SELECT statement
func TestBasicSelect(t *testing.T) {
	sql := "SELECT id, name FROM users WHERE status = 1"
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if result.StatementType != "SELECT" {
		t.Errorf("Expected SELECT, got %s", result.StatementType)
	}
	if len(result.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(result.Tables))
	}
	if len(result.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(result.Fields))
	}
}

// TestJoinWithSubquery tests JOIN with subquery-based tables
func TestJoinWithSubquery(t *testing.T) {
	sql := `SELECT a.id, b.total
FROM users a
LEFT JOIN (
    SELECT user_id, SUM(amount) as total
    FROM payments
    GROUP BY user_id
) b ON a.id = b.user_id
WHERE a.status = 1`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(result.Joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(result.Joins))
	}
	join := result.Joins[0]
	if join.LeftTable == "" {
		t.Error("Join LeftTable is empty")
	}
	if join.RightTable == "" {
		t.Error("Join RightTable is empty")
	}
	if join.Type != "LEFT JOIN" {
		t.Errorf("Expected LEFT JOIN, got %s", join.Type)
	}
	t.Logf("Join: %s -> %s (%s), conditions=%d, rawExpr=%q",
		join.LeftTable, join.RightTable, join.Type, len(join.Conditions), join.RawExpr)
}

// TestGroupConcatOrderBy tests GROUP_CONCAT with ORDER BY inside
func TestGroupConcatOrderBy(t *testing.T) {
	sql := `SELECT
    id,
    GROUP_CONCAT(DISTINCT tag_name ORDER BY tag_name ASC SEPARATOR ',') as tags
FROM items
LEFT JOIN item_tags ON items.id = item_tags.item_id
GROUP BY id`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(result.Fields) < 2 {
		t.Fatalf("Expected at least 2 fields, got %d", len(result.Fields))
	}
	// Check that the GROUP_CONCAT field is parsed
	foundGroupConcat := false
	for _, f := range result.Fields {
		if strings.Contains(f.Expression, "GROUP_CONCAT") {
			foundGroupConcat = true
			t.Logf("GROUP_CONCAT field: outputName=%s, fieldType=%s, funcCategory=%s",
				f.OutputName, f.FieldType, f.FuncCategory)
			if f.FieldType == "column" {
				t.Error("GROUP_CONCAT should not be parsed as 'column' type")
			}
		}
	}
	if !foundGroupConcat {
		t.Error("GROUP_CONCAT field not found in result")
	}
}

// TestExistsSubquery tests WHERE with EXISTS and NOT EXISTS
func TestExistsSubquery(t *testing.T) {
	sql := `SELECT id FROM users u
WHERE u.status = 1
AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)
AND NOT EXISTS (SELECT 1 FROM blacklist b WHERE b.user_id = u.id)`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if result.WhereTree == nil {
		t.Fatal("WhereTree should not be nil")
	}
	t.Logf("WhereTree: %+v", result.WhereTree)
}

// TestComplexSQL tests the full complex SQL that causes white screen
func TestComplexSQL(t *testing.T) {
	sql := `SELECT
    clue.id,
    clue.clue_name,
    clue.contact_name,
    clue.contact_phone,
    clue.channel_id,
    ch.channel_name,
    clue.created_at,
    clue.is_deal,
    IFNULL(pay_summary.pay_amount, 0) - IFNULL(refund_summary.refund_amount, 0) AS real_income_amount,
    COUNT(DISTINCT order_info.id) AS order_count,
    GROUP_CONCAT(DISTINCT tag.tag_name ORDER BY tag.tag_name ASC SEPARATOR ',') AS tag_names,
    CASE
        WHEN clue.is_deal = 1 THEN '已成交'
        WHEN clue.is_deal = 2 THEN '跟进中'
        ELSE '待跟进'
    END AS deal_status,
    JSON_UNQUOTE(JSON_EXTRACT(clue.extra_info, '$.source')) AS source_info
FROM clue_info clue
LEFT JOIN channel ch ON clue.channel_id = ch.id
LEFT JOIN order_info ON clue.id = order_info.clue_id
LEFT JOIN clue_tag ct ON clue.id = ct.clue_id
LEFT JOIN tag ON ct.tag_id = tag.id
LEFT JOIN (
    SELECT clue_id, SUM(amount) AS pay_amount
    FROM payment_records
    WHERE status = 1
    GROUP BY clue_id
) pay_summary ON clue.id = pay_summary.clue_id
LEFT JOIN (
    SELECT clue_id, SUM(amount) AS refund_amount
    FROM refund_records
    WHERE status = 1
    GROUP BY clue_id
) refund_summary ON clue.id = refund_summary.clue_id
LEFT JOIN follow_record fr ON clue.id = fr.clue_id
LEFT JOIN user_info u ON clue.owner_id = u.id
LEFT JOIN department dept ON u.dept_id = dept.id
LEFT JOIN region r ON clue.region_code = r.code
LEFT JOIN source_config sc ON clue.source_id = sc.id
WHERE clue.created_at >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
AND clue.created_at < CURDATE()
AND clue.is_deal IN (0, 1, 2)
AND EXISTS (SELECT 1 FROM follow_record fr2 WHERE fr2.clue_id = clue.id)
AND NOT EXISTS (SELECT 1 FROM blacklist bl WHERE bl.phone = clue.contact_phone)
GROUP BY clue.id, clue.clue_name, clue.contact_name, clue.contact_phone,
    clue.channel_id, ch.channel_name, clue.created_at, clue.is_deal,
    pay_summary.pay_amount, refund_summary.refund_amount
HAVING real_income_amount > 0 AND clue_count >= 1
ORDER BY real_income_amount DESC, clue.created_at DESC
LIMIT 100`

	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Basic checks
	if result.StatementType != "SELECT" {
		t.Errorf("Expected SELECT, got %s", result.StatementType)
	}

	// Check tables
	t.Logf("Tables: %d", len(result.Tables))
	for _, tbl := range result.Tables {
		t.Logf("  Table: name=%s, alias=%s, role=%s", tbl.Name, tbl.Alias, tbl.Role)
	}

	// Check joins
	t.Logf("Joins: %d", len(result.Joins))
	for _, j := range result.Joins {
		t.Logf("  Join: %s -> %s (%s), conditions=%d, rawExpr=%q",
			j.LeftTable, j.RightTable, j.Type, len(j.Conditions), j.RawExpr)
		for ci, cond := range j.Conditions {
			t.Logf("    Condition[%d]: left=%q, op=%q, right=%q", ci, cond.Left, cond.Operator, cond.Right)
		}
	}

	// Check fields
	t.Logf("Fields: %d", len(result.Fields))
	for _, f := range result.Fields {
		t.Logf("  Field: outputName=%s, fieldType=%s, funcCategory=%s, expr=%s",
			f.OutputName, f.FieldType, f.FuncCategory, f.Expression)
	}

	// Check WHERE tree
	if result.WhereTree != nil {
		t.Logf("WhereTree type=%s, children=%d", result.WhereTree.Type, len(result.WhereTree.Children))
	} else {
		t.Log("WhereTree is nil")
	}

	// Check summary
	t.Logf("Summary: tables=%d, joins=%d, fields=%d, whereCount=%d, complexity=%s",
		result.Summary.TableCount, result.Summary.JoinCount, result.Summary.FieldCount,
		result.Summary.WhereCount, result.Summary.Complexity)

	// Critical: verify JSON serialization doesn't panic
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}
	t.Logf("JSON output size: %d bytes", len(jsonBytes))

	// Verify the JSON can be deserialized back
	var check map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &check); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// Check that joins array is present and valid
	joinsRaw, ok := check["joins"]
	if !ok {
		t.Fatal("JSON missing 'joins' field")
	}
	joinsArr, ok := joinsRaw.([]interface{})
	if !ok {
		t.Fatalf("Expected joins to be array, got %T", joinsRaw)
	}
	t.Logf("JSON joins count: %d", len(joinsArr))

	// Check each join has required fields
	for i, j := range joinsArr {
		joinMap, ok := j.(map[string]interface{})
		if !ok {
			t.Errorf("Join[%d] is not a map", i)
			continue
		}
		for _, field := range []string{"id", "type", "leftTable", "rightTable", "conditions", "rawExpr"} {
			if _, exists := joinMap[field]; !exists {
				t.Errorf("Join[%d] missing field %q", i, field)
			}
		}
		// Check conditions
		conds, ok := joinMap["conditions"].([]interface{})
		if !ok {
			t.Errorf("Join[%d] conditions is not array: %T", i, joinMap["conditions"])
			continue
		}
		for ci, c := range conds {
			condMap, ok := c.(map[string]interface{})
			if !ok {
				t.Errorf("Join[%d].conditions[%d] is not a map", i, ci)
				continue
			}
			for _, field := range []string{"left", "operator", "right"} {
				if _, exists := condMap[field]; !exists {
					t.Errorf("Join[%d].conditions[%d] missing field %q", i, ci, field)
				}
			}
		}
	}
}

// TestMultipleJoins tests parsing with many JOINs
func TestMultipleJoins(t *testing.T) {
	sql := `SELECT a.id
FROM t1 a
LEFT JOIN t2 b ON a.id = b.t1_id
LEFT JOIN t3 c ON a.id = c.t1_id
LEFT JOIN t4 d ON a.id = d.t1_id
LEFT JOIN t5 e ON a.id = e.t1_id
LEFT JOIN t6 f ON a.id = f.t1_id
LEFT JOIN t7 g ON a.id = g.t1_id
LEFT JOIN t8 h ON a.id = h.t1_id
LEFT JOIN t9 i ON a.id = i.t1_id
LEFT JOIN t10 j ON a.id = j.t1_id
LEFT JOIN t11 k ON a.id = k.t1_id
LEFT JOIN t12 l ON a.id = l.t1_id`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(result.Joins) != 12 {
		t.Errorf("Expected 12 joins, got %d", len(result.Joins))
	}
	for _, j := range result.Joins {
		if j.LeftTable == "" || j.RightTable == "" {
			t.Errorf("Empty table in join: left=%q, right=%q", j.LeftTable, j.RightTable)
		}
	}
}

// TestHavingClause tests HAVING with column aliases
func TestHavingClause(t *testing.T) {
	sql := `SELECT user_id, COUNT(*) AS order_count
FROM orders
GROUP BY user_id
HAVING order_count > 5`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("GroupBy: %d items", len(result.GroupBy))
	t.Logf("Summary: hasGroupBy=%v", result.Summary.HasGroupBy)
}

// TestCaseExpression tests CASE WHEN expressions in SELECT
func TestCaseExpression(t *testing.T) {
	sql := `SELECT
    id,
    CASE
        WHEN status = 1 THEN 'active'
        WHEN status = 2 THEN 'inactive'
        ELSE 'unknown'
    END AS status_label
FROM users`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	foundCase := false
	for _, f := range result.Fields {
		if f.FieldType == "case" {
			foundCase = true
			t.Logf("CASE field: outputName=%s", f.OutputName)
		}
	}
	if !foundCase {
		t.Error("CASE expression not detected")
	}
}

// TestNestedFunctions tests nested function calls like IFNULL(x, 0) - IFNULL(y, 0)
func TestNestedFunctions(t *testing.T) {
	sql := `SELECT IFNULL(a, 0) - IFNULL(b, 0) AS diff FROM t1`
	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(result.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(result.Fields))
	}
	f := result.Fields[0]
	t.Logf("Field: outputName=%s, fieldType=%s, expr=%s", f.OutputName, f.FieldType, f.Expression)
}

// TestTokenizerGroupConcat tests that GROUP_CONCAT is tokenized correctly
func TestTokenizerGroupConcat(t *testing.T) {
	sql := "GROUP_CONCAT(DISTINCT tag_name ORDER BY tag_name ASC SEPARATOR ',')"
	tokens := tokenize(sql, GetDialectConfig(DialectMySQL))
	for i, tok := range tokens {
		t.Logf("Token[%d]: value=%q, kind=%d", i, tok.value, tok.kind)
	}
}

// TestTokenizerExists tests that EXISTS is tokenized correctly
func TestTokenizerExists(t *testing.T) {
	sql := "EXISTS (SELECT 1 FROM t1)"
	tokens := tokenize(sql, GetDialectConfig(DialectMySQL))
	for i, tok := range tokens {
		t.Logf("Token[%d]: value=%q, kind=%d", i, tok.value, tok.kind)
	}
	// EXISTS should be a keyword
	if tokens[0].kind != tokenKeyword {
		t.Errorf("EXISTS should be keyword, got kind=%d", tokens[0].kind)
	}
}

// TestExprParserExists tests that the expression parser handles EXISTS
func TestExprParserExists(t *testing.T) {
	tokens := tokenize("EXISTS (SELECT 1 FROM t1)", GetDialectConfig(DialectMySQL))
	ep := NewExprParser(tokens, GetDialectConfig(DialectMySQL))
	expr, err := ep.ParseExpression()
	if err != nil {
		t.Fatalf("ExprParser failed on EXISTS: %v", err)
	}
	t.Logf("EXISTS parsed as: %T - %s", expr, expr.String())
}

// TestExprParserGroupConcat tests that the expression parser handles GROUP_CONCAT with ORDER BY
func TestExprParserGroupConcat(t *testing.T) {
	tokens := tokenize("GROUP_CONCAT(DISTINCT tag_name ORDER BY tag_name ASC SEPARATOR ',')",
		GetDialectConfig(DialectMySQL))
	ep := NewExprParser(tokens, GetDialectConfig(DialectMySQL))
	expr, err := ep.ParseExpression()
	if err != nil {
		t.Fatalf("ExprParser failed on GROUP_CONCAT: %v", err)
	}
	t.Logf("GROUP_CONCAT parsed as: %T - %s", expr, expr.String())
}

// TestJSONRoundTrip verifies the full parse result can be serialized and deserialized
func TestJSONRoundTrip(t *testing.T) {
	sql := `SELECT
    a.id,
    b.name,
    CASE WHEN a.status = 1 THEN 'yes' ELSE 'no' END AS active,
    GROUP_CONCAT(DISTINCT c.tag SEPARATOR ',') AS tags,
    IFNULL(d.total, 0) AS total
FROM main_table a
LEFT JOIN ref_table b ON a.id = b.ref_id
LEFT JOIN tag_table c ON a.id = c.owner_id
LEFT JOIN (
    SELECT owner_id, SUM(amount) AS total FROM payments GROUP BY owner_id
) d ON a.id = d.owner_id
WHERE a.status IN (1, 2, 3)
AND EXISTS (SELECT 1 FROM log_table l WHERE l.ref_id = a.id)
GROUP BY a.id, b.name
HAVING total > 0
ORDER BY a.id DESC
LIMIT 50`

	p := NewCustomParser()
	result, err := p.Parse(sql)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("JSON marshal error: %v", err)
	}

	var check map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &check); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	// Print for visual inspection
	fmt.Printf("JSON output:\n%s\n", string(jsonBytes))
}
