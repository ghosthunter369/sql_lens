package model

type ParseSQLRequest struct {
	RawText string       `json:"rawText" binding:"required"`
	Dialect string       `json:"dialect"`
	LogType string       `json:"logType"`
	Options ParseOptions `json:"options"`
}

type ParseOptions struct {
	RestoreBindings bool `json:"restoreBindings"`
	FormatSQL       bool `json:"formatSql"`
	EnableRiskCheck bool `json:"enableRiskCheck"`
}

type ExtractSQLRequest struct {
	RawLog  string `json:"rawLog" binding:"required"`
	LogType string `json:"logType"`
}

type FormatSQLRequest struct {
	SQL     string `json:"sql" binding:"required"`
	Dialect string `json:"dialect"`
}

type BuildMarkdownRequest struct {
	AnalysisResult *SQLAnalysisResult `json:"analysisResult" binding:"required"`
}
