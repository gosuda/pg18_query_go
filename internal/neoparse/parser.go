package neoparse

import (
	"errors"
	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

// Parse parses a SQL string and returns a slice of parsed AST Statements.
func Parse(sql string) ([]Statement, error) {
	lex := newLexer(sql)
	yyParse(lex)

	if lex.err != nil {
		return nil, lex.err
	}
	return lex.statements, nil
}

// ParseOne parses a SQL string containing a single statement.
func ParseOne(sql string) (Statement, error) {
	stmts, err := Parse(sql)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, errors.New("empty SQL statement")
	}
	if len(stmts) > 1 {
		return nil, errors.New("expected single SQL statement, found multiple")
	}
	return stmts[0], nil
}

// ParseMode uses PostgreSQL RawParseMode values without modifying the input.
func ParseMode(sql string, mode int) (*pg_query.ParseResult, error) {
	lex := newLexer(sql)
	switch mode {
	case 0:
	case 1:
		lex.initialToken = MODE_TYPE_NAME
	case 2:
		lex.initialToken = MODE_PLPGSQL_EXPR
	case 3:
		lex.initialToken = MODE_PLPGSQL_ASSIGN1
	case 4:
		lex.initialToken = MODE_PLPGSQL_ASSIGN2
	case 5:
		lex.initialToken = MODE_PLPGSQL_ASSIGN3
	default:
		return nil, &Error{Message: "invalid raw parse mode", Location: -1}
	}
	yyParse(lex)
	if lex.err != nil {
		return nil, lex.err
	}
	result := &pg_query.ParseResult{Version: 180004}
	for i, stmt := range lex.statements {
		location := lex.statementLocations[i]
		result.Stmts = append(result.Stmts, &pg_query.RawStmt{
			Stmt:         stmt,
			StmtLocation: location.start,
			StmtLen:      location.length,
		})
	}
	return result, nil
}
