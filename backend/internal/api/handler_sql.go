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
		c.JSON(http.StatusOK, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "LOG_EXTRACT_ERROR",
				Message: "无法从输入中提取 SQL",
				Detail:  err.Error(),
			},
		})
		return
	}

	sql := extractResult.SQL

	// 2. Restore bindings
	restoredSQL := sql
	if req.Options.RestoreBindings && len(extractResult.Bindings) > 0 {
		restoredSQL, err = binding.RestoreBindings(sql, extractResult.Bindings)
		if err != nil {
			c.JSON(http.StatusOK, model.APIResponse{
				Success: false,
				Error: &model.APIError{
					Code:    "BINDING_RESTORE_ERROR",
					Message: "SQL 参数回填失败",
					Detail:  err.Error(),
				},
			})
			return
		}
	}

	// 3. SQL Parse
	sqlParser := parser.NewCustomParser()
	analysisResult, err := sqlParser.Parse(restoredSQL)
	if err != nil {
		c.JSON(http.StatusOK, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "SQL_PARSE_ERROR",
				Message: "SQL 解析失败",
				Detail:  err.Error(),
			},
		})
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
		c.JSON(http.StatusOK, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "AST_TRANSFORM_ERROR",
				Message: "AST 转换失败",
				Detail:  err.Error(),
			},
		})
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

	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data:    analysisResult,
	})
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
		c.JSON(http.StatusOK, model.APIResponse{
			Success: false,
			Error: &model.APIError{
				Code:    "LOG_EXTRACT_ERROR",
				Message: "无法从输入中提取 SQL",
				Detail:  err.Error(),
			},
		})
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

	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data: model.ExtractResultResponse{
			SQL:         extractResult.SQL,
			Bindings:    extractResult.Bindings,
			RestoredSQL: restoredSQL,
			LogType:     extractResult.LogType,
		},
	})
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
	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data: model.FormatResultResponse{
			FormattedSQL: formatted,
		},
	})
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

	c.JSON(http.StatusOK, model.APIResponse{
		Success: true,
		Data: model.MarkdownReportResponse{
			Markdown: markdown,
		},
	})
}

// Simple SQL formatting (basic version, for a more advanced formatter use sql-formatter on frontend)
func formatSQLSimple(sql string) string {
	if sql == "" {
		return ""
	}

	// This is a basic formatter. The formatted SQL from the parser's transformer
	// provides the primary formatting. This is a fallback.
	return sql
}
