package model

type SQLAnalysisResult struct {
	StatementType string          `json:"statementType"`
	Dialect       string          `json:"dialect"`
	RawSQL        string          `json:"rawSql"`
	RestoredSQL   string          `json:"restoredSql"`
	FormattedSQL  string          `json:"formattedSql"`
	Summary       SQLSummary      `json:"summary"`
	Tables        []TableMeta     `json:"tables"`
	Joins         []JoinMeta      `json:"joins"`
	Fields        []FieldMeta     `json:"fields"`
	WhereTree     *ConditionNode  `json:"whereTree,omitempty"`
	GroupBy       []GroupByMeta   `json:"groupBy"`
	OrderBy       []OrderByMeta   `json:"orderBy"`
	Limit         *LimitMeta      `json:"limit,omitempty"`
	Graph         GraphMeta       `json:"graph"`
	Risks         []RiskMeta      `json:"risks"`
	CTEs          []CTEDefinition `json:"ctes,omitempty"`
	SetOperations []SetOperation  `json:"setOperations,omitempty"`
}

type SQLSummary struct {
	TableCount    int    `json:"tableCount"`
	JoinCount     int    `json:"joinCount"`
	FieldCount    int    `json:"fieldCount"`
	WhereCount    int    `json:"whereCount"`
	HasGroupBy    bool   `json:"hasGroupBy"`
	HasOrderBy    bool   `json:"hasOrderBy"`
	HasLimit      bool   `json:"hasLimit"`
	Complexity    string `json:"complexity"`
	HasWindowFunc bool   `json:"hasWindowFunc"`
	HasCTE        bool   `json:"hasCTE"`
	HasUnion      bool   `json:"hasUnion"`
}

type TableMeta struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Alias          string   `json:"alias"`
	Role           string   `json:"role"`
	SelectedFields []string `json:"selectedFields"`
	FilterFields   []string `json:"filterFields"`
	JoinFields     []string `json:"joinFields"`
}

type JoinMeta struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	LeftTable  string          `json:"leftTable"`
	RightTable string          `json:"rightTable"`
	Conditions []JoinCondition `json:"conditions"`
	RawExpr    string          `json:"rawExpr"`
}

type JoinCondition struct {
	Left     string `json:"left"`
	Operator string `json:"operator"`
	Right    string `json:"right"`
}

type FieldMeta struct {
	ID           string          `json:"id"`
	OutputName   string          `json:"outputName"`
	SourceTable  string          `json:"sourceTable"`
	SourceAlias  string          `json:"sourceAlias"`
	SourceColumn string          `json:"sourceColumn"`
	Expression   string          `json:"expression"`
	FieldType    string          `json:"fieldType"`
	DeepSources  []DeepSourceRef `json:"deepSources,omitempty"`
	FuncCategory string          `json:"funcCategory,omitempty"`
	WindowSpec   *WindowSpecMeta `json:"windowSpec,omitempty"`
}

// WindowSpecMeta describes a window function's OVER clause.
type WindowSpecMeta struct {
	PartitionBy []string      `json:"partitionBy,omitempty"`
	OrderBy     []OrderByMeta `json:"orderBy,omitempty"`
	FrameClause string        `json:"frameClause,omitempty"`
}

type DeepSourceRef struct {
	Table  string `json:"table"`
	Alias  string `json:"alias,omitempty"`
	Column string `json:"column"`
}

type ConditionNode struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Expr     string           `json:"expr,omitempty"`
	Table    string           `json:"table,omitempty"`
	Field    string           `json:"field,omitempty"`
	Operator string           `json:"operator,omitempty"`
	Value    string           `json:"value,omitempty"`
	Children []*ConditionNode `json:"children,omitempty"`
}

type GroupByMeta struct {
	Expression  string `json:"expression"`
	SourceTable string `json:"sourceTable,omitempty"`
}

type OrderByMeta struct {
	Expression  string `json:"expression"`
	SourceTable string `json:"sourceTable,omitempty"`
	Direction   string `json:"direction"`
}

type LimitMeta struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset,omitempty"`
}

type GraphMeta struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type GraphNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position GraphPosition          `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

type GraphEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Label  string                 `json:"label"`
	Type   string                 `json:"type"`
	Data   map[string]interface{} `json:"data"`
}

type GraphPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type RiskMeta struct {
	ID          string `json:"id"`
	Level       string `json:"level"`
	Type        string `json:"type"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	RelatedExpr string `json:"relatedExpr,omitempty"`
}

// CTEDefinition represents a Common Table Expression (WITH clause).
type CTEDefinition struct {
	ID      string             `json:"id"`
	Name    string             `json:"name"`
	Columns []string           `json:"columns,omitempty"`
	Query   *SQLAnalysisResult `json:"query,omitempty"`
	RawSQL  string             `json:"rawSql"`
}

// SetOperation represents UNION/INTERSECT/EXCEPT between SELECT statements.
type SetOperation struct {
	Type  string             `json:"type"`
	Left  *SQLAnalysisResult `json:"left"`
	Right *SQLAnalysisResult `json:"right"`
}
