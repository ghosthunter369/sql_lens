package parser

import "strings"

// FuncCategory classifies a SQL function by its purpose.
type FuncCategory string

const (
	FuncCatAggregate   FuncCategory = "aggregate"
	FuncCatWindow      FuncCategory = "window"
	FuncCatScalar      FuncCategory = "scalar"
	FuncCatDateTime    FuncCategory = "datetime"
	FuncCatString      FuncCategory = "string"
	FuncCatMath        FuncCategory = "math"
	FuncCatConditional FuncCategory = "conditional"
	FuncCatCast        FuncCategory = "cast"
	FuncCatJSON        FuncCategory = "json"
)

// FuncInfo describes a known SQL function.
type FuncInfo struct {
	Name        string
	Category    FuncCategory
	Dialects    []DialectID // empty means all dialects
	IsWindow    bool        // can be used with OVER()
	IsAggregate bool        // is an aggregate function
}

// funcRegistry contains ~30 aggregate and window functions.
// All other ident(...) calls are auto-detected as scalar functions.
var funcRegistry = map[string]FuncInfo{
	// --- Aggregate functions (also usable as window functions) ---
	"COUNT":          {Name: "COUNT", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"SUM":            {Name: "SUM", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"AVG":            {Name: "AVG", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"MAX":            {Name: "MAX", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"MIN":            {Name: "MIN", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"GROUP_CONCAT":   {Name: "GROUP_CONCAT", Category: FuncCatAggregate, IsAggregate: true, IsWindow: false, Dialects: []DialectID{DialectMySQL, DialectSQLite}},
	"STRING_AGG":     {Name: "STRING_AGG", Category: FuncCatAggregate, IsAggregate: true, IsWindow: false, Dialects: []DialectID{DialectPostgreSQL}},
	"LISTAGG":        {Name: "LISTAGG", Category: FuncCatAggregate, IsAggregate: true, IsWindow: false, Dialects: []DialectID{DialectOracle}},
	"STDDEV":         {Name: "STDDEV", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"STDDEV_POP":     {Name: "STDDEV_POP", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"STDDEV_SAMP":    {Name: "STDDEV_SAMP", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"VARIANCE":       {Name: "VARIANCE", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"VAR_POP":        {Name: "VAR_POP", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"VAR_SAMP":       {Name: "VAR_SAMP", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true},
	"BIT_AND":        {Name: "BIT_AND", Category: FuncCatAggregate, IsAggregate: true, Dialects: []DialectID{DialectMySQL}},
	"BIT_OR":         {Name: "BIT_OR", Category: FuncCatAggregate, IsAggregate: true, Dialects: []DialectID{DialectMySQL}},
	"BIT_XOR":        {Name: "BIT_XOR", Category: FuncCatAggregate, IsAggregate: true, Dialects: []DialectID{DialectMySQL}},
	"JSON_ARRAYAGG":  {Name: "JSON_ARRAYAGG", Category: FuncCatJSON, IsAggregate: true, Dialects: []DialectID{DialectMySQL, DialectPostgreSQL}},
	"JSON_OBJECTAGG": {Name: "JSON_OBJECTAGG", Category: FuncCatJSON, IsAggregate: true, Dialects: []DialectID{DialectMySQL, DialectPostgreSQL}},
	"EVERY":          {Name: "EVERY", Category: FuncCatAggregate, IsAggregate: true, IsWindow: true, Dialects: []DialectID{DialectPostgreSQL}},
	"SOME":           {Name: "SOME", Category: FuncCatAggregate, IsAggregate: true, Dialects: []DialectID{DialectPostgreSQL}},
	"ANY":            {Name: "ANY", Category: FuncCatAggregate, IsAggregate: true, Dialects: []DialectID{DialectPostgreSQL}},

	// --- Window-only functions ---
	"ROW_NUMBER":   {Name: "ROW_NUMBER", Category: FuncCatWindow, IsWindow: true},
	"RANK":         {Name: "RANK", Category: FuncCatWindow, IsWindow: true},
	"DENSE_RANK":   {Name: "DENSE_RANK", Category: FuncCatWindow, IsWindow: true},
	"NTILE":        {Name: "NTILE", Category: FuncCatWindow, IsWindow: true},
	"LAG":          {Name: "LAG", Category: FuncCatWindow, IsWindow: true},
	"LEAD":         {Name: "LEAD", Category: FuncCatWindow, IsWindow: true},
	"FIRST_VALUE":  {Name: "FIRST_VALUE", Category: FuncCatWindow, IsWindow: true},
	"LAST_VALUE":   {Name: "LAST_VALUE", Category: FuncCatWindow, IsWindow: true},
	"NTH_VALUE":    {Name: "NTH_VALUE", Category: FuncCatWindow, IsWindow: true},
	"PERCENT_RANK": {Name: "PERCENT_RANK", Category: FuncCatWindow, IsWindow: true},
	"CUME_DIST":    {Name: "CUME_DIST", Category: FuncCatWindow, IsWindow: true},
}

// LookupFunction checks if a function is in the registry and returns its info.
// The dialect parameter is used to filter dialect-specific functions.
func LookupFunction(name string, dialect DialectID) (FuncInfo, bool) {
	upper := strings.ToUpper(name)
	info, ok := funcRegistry[upper]
	if !ok {
		return FuncInfo{}, false
	}
	// Check dialect compatibility
	if len(info.Dialects) > 0 {
		found := false
		for _, d := range info.Dialects {
			if d == dialect {
				found = true
				break
			}
		}
		if !found {
			return FuncInfo{}, false
		}
	}
	return info, true
}

// IsKnownFunction checks if a function name is in the registry.
func IsKnownFunction(name string) bool {
	_, ok := funcRegistry[strings.ToUpper(name)]
	return ok
}

// GetFuncCategory returns the category for a function, or "scalar" for unknown functions.
func GetFuncCategory(name string, dialect DialectID) FuncCategory {
	if info, ok := LookupFunction(name, dialect); ok {
		return info.Category
	}
	return FuncCatScalar
}
