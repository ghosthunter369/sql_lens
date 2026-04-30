package parser

import (
	"sql-lens/internal/model"
)

type SQLParser interface {
	Parse(sql string) (*model.SQLAnalysisResult, error)
}
