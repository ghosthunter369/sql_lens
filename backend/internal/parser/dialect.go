package parser

// DialectID identifies a SQL dialect.
type DialectID string

const (
	DialectMySQL      DialectID = "mysql"
	DialectPostgreSQL DialectID = "postgresql"
	DialectOracle     DialectID = "oracle"
	DialectSQLServer  DialectID = "sqlserver"
	DialectSQLite     DialectID = "sqlite"
)

// DialectConfig holds syntax flags that control tokenization and parsing behavior.
type DialectConfig struct {
	ID                  DialectID
	IdentifierQuote     byte   // '`' for MySQL, '"' for PG/Oracle, '[' for SQL Server
	AltIdentifierQuote  byte   // '"' also valid in MySQL
	SupportDoubleDollar bool   // PostgreSQL $$ string literals
	SupportTypeCast     bool   // PostgreSQL ::type casts
	SupportConcatOp     bool   // || concatenation operator (PG, Oracle)
	SupportBracketIdent bool   // [identifier] (SQL Server)
	SupportTopN         bool   // SELECT TOP N (SQL Server)
	StringEscapes       bool   // backslash escapes in strings (MySQL)
	LimitSyntax         string // "LIMIT_OFFSET" (MySQL/PG/SQLite), "OFFSET_FETCH" (SQL Server/Oracle), "TOP" (SQL Server)
}

var dialectConfigs = map[DialectID]DialectConfig{
	DialectMySQL: {
		ID:                  DialectMySQL,
		IdentifierQuote:     '`',
		AltIdentifierQuote:  '"',
		SupportDoubleDollar: false,
		SupportTypeCast:     false,
		SupportConcatOp:     false,
		SupportBracketIdent: false,
		SupportTopN:         false,
		StringEscapes:       true,
		LimitSyntax:         "LIMIT_OFFSET",
	},
	DialectPostgreSQL: {
		ID:                  DialectPostgreSQL,
		IdentifierQuote:     '"',
		AltIdentifierQuote:  '"',
		SupportDoubleDollar: true,
		SupportTypeCast:     true,
		SupportConcatOp:     true,
		SupportBracketIdent: false,
		SupportTopN:         false,
		StringEscapes:       false,
		LimitSyntax:         "LIMIT_OFFSET",
	},
	DialectOracle: {
		ID:                  DialectOracle,
		IdentifierQuote:     '"',
		AltIdentifierQuote:  '"',
		SupportDoubleDollar: false,
		SupportTypeCast:     false,
		SupportConcatOp:     true,
		SupportBracketIdent: false,
		SupportTopN:         false,
		StringEscapes:       false,
		LimitSyntax:         "OFFSET_FETCH",
	},
	DialectSQLServer: {
		ID:                  DialectSQLServer,
		IdentifierQuote:     '[',
		AltIdentifierQuote:  '"',
		SupportDoubleDollar: false,
		SupportTypeCast:     false,
		SupportConcatOp:     false,
		SupportBracketIdent: true,
		SupportTopN:         true,
		StringEscapes:       false,
		LimitSyntax:         "TOP",
	},
	DialectSQLite: {
		ID:                  DialectSQLite,
		IdentifierQuote:     '`',
		AltIdentifierQuote:  '"',
		SupportDoubleDollar: false,
		SupportTypeCast:     false,
		SupportConcatOp:     false,
		SupportBracketIdent: false,
		SupportTopN:         false,
		StringEscapes:       true,
		LimitSyntax:         "LIMIT_OFFSET",
	},
}

// GetDialectConfig returns the configuration for the given dialect ID.
// Falls back to MySQL if the ID is unknown.
func GetDialectConfig(id DialectID) DialectConfig {
	if cfg, ok := dialectConfigs[id]; ok {
		return cfg
	}
	return dialectConfigs[DialectMySQL]
}

// AllDialectIDs returns all supported dialect IDs.
func AllDialectIDs() []DialectID {
	return []DialectID{DialectMySQL, DialectPostgreSQL, DialectOracle, DialectSQLServer, DialectSQLite}
}

// IsValidDialectID checks if the given string is a valid dialect ID.
func IsValidDialectID(id string) bool {
	_, ok := dialectConfigs[DialectID(id)]
	return ok
}
