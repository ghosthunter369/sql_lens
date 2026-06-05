package api

import (
	"net/http"

	"sql-lens/internal/analyzer"
	"sql-lens/internal/binding"
	"sql-lens/internal/extractor"
	"sql-lens/internal/model"
	"sql-lens/internal/parser"
	"sql-lens/internal/report"

	"github.com/gin-gonic/gin"
)

func ParseSQLHandler(c *gin.Context) {
	var req model.ParseSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "INVALID_REQUEST",
				Message: "请求参数错误",
				Detail:  err.Error(),
			},
		})
		return
	}

	// Set defaults
	if req.Dialect == "" {
		req.Dialect = "mysql"
	}
	if req.LogType == "" {
		req.LogType = "auto"
	}

	// 1. Log extraction
	extractResult, err := extractor.AutoExtract(req.RawText)
	if err != nil {
		code, resp := model.ErrorResponse("LOG_EXTRACT_ERROR", "无法从输入中提取 SQL", err.Error())
		c.JSON(code, resp)
		return
	}

	sql := extractResult.SQL

	// 2. Restore bindings
	restoredSQL := sql
	if req.Options.RestoreBindings && len(extractResult.Bindings) > 0 {
		restoredSQL, err = binding.RestoreBindings(sql, extractResult.Bindings)
		if err != nil {
			code, resp := model.ErrorResponse("BINDING_RESTORE_ERROR", "SQL 参数回填失败", err.Error())
			c.JSON(code, resp)
			return
		}
	}

	// 3. SQL Parse
	dialectID := parser.DialectID(req.Dialect)
	if !parser.IsValidDialectID(req.Dialect) {
		dialectID = parser.DialectMySQL
	}
	sqlParser := parser.NewCustomParserWithDialect(dialectID)
	analysisResult, err := sqlParser.Parse(restoredSQL)
	if err != nil {
		code, resp := model.ErrorResponse("SQL_PARSE_ERROR", "SQL 解析失败", err.Error())
		c.JSON(code, resp)
		return
	}

	// 4. Enrich with raw/restored SQL info
	analysisResult.RawSQL = sql
	analysisResult.RestoredSQL = restoredSQL
	analysisResult.Dialect = req.Dialect

	// 5. AST Transform (enrich table info, format SQL)
	transformer := analyzer.NewASTTransformer()
	analysisResult, err = transformer.Transform(analysisResult)
	if err != nil {
		code, resp := model.ErrorResponse("AST_TRANSFORM_ERROR", "AST 转换失败", err.Error())
		c.JSON(code, resp)
		return
	}

	// 6. Risk analysis
	if req.Options.EnableRiskCheck {
		analysisResult.Risks = analyzer.AnalyzeRisks(analysisResult)
	}

	// 7. Format SQL
	if req.Options.FormatSQL {
		if analysisResult.RestoredSQL != "" && analysisResult.RestoredSQL != sql {
			analysisResult.FormattedSQL = formatSQLSimple(analysisResult.RestoredSQL)
		} else {
			analysisResult.FormattedSQL = formatSQLSimple(analysisResult.RawSQL)
		}
	}

	code, resp := model.SuccessResponse(analysisResult)
	c.JSON(code, resp)
}

func ExtractSQLHandler(c *gin.Context) {
	var req model.ExtractSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "INVALID_REQUEST",
				Message: "请求参数错误",
				Detail:  err.Error(),
			},
		})
		return
	}

	extractResult, err := extractor.AutoExtract(req.RawLog)
	if err != nil {
		code, resp := model.ErrorResponse("LOG_EXTRACT_ERROR", "无法从输入中提取 SQL", err.Error())
		c.JSON(code, resp)
		return
	}

	restoredSQL := extractResult.SQL
	if len(extractResult.Bindings) > 0 {
		var err error
		restoredSQL, err = binding.RestoreBindings(extractResult.SQL, extractResult.Bindings)
		if err != nil {
			restoredSQL = extractResult.SQL
		}
	}

	code, resp := model.SuccessResponse(model.ExtractResultResponse{
		SQL:         extractResult.SQL,
		Bindings:    extractResult.Bindings,
		RestoredSQL: restoredSQL,
		LogType:     extractResult.LogType,
	})
	c.JSON(code, resp)
}

func FormatSQLHandler(c *gin.Context) {
	var req model.FormatSQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "INVALID_REQUEST",
				Message: "请求参数错误",
				Detail:  err.Error(),
			},
		})
		return
	}

	formatted := formatSQLSimple(req.SQL)
	code, resp := model.SuccessResponse(model.FormatResultResponse{
		FormattedSQL: formatted,
	})
	c.JSON(code, resp)
}

func BuildMarkdownReportHandler(c *gin.Context) {
	var req model.BuildMarkdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "INVALID_REQUEST",
				Message: "请求参数错误",
				Detail:  err.Error(),
			},
		})
		return
	}

	markdown := report.BuildMarkdownReport(req.AnalysisResult)

	code, resp := model.SuccessResponse(model.MarkdownReportResponse{
		Markdown: markdown,
	})
	c.JSON(code, resp)
}

// formatSQLSimple formats SQL by capitalizing keywords and adding newlines.
func formatSQLSimple(sql string) string {
	return analyzer.FormatSQL(sql)
}
