package neoparse

import pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"

func ParseToPgQuery(sql string) (*pg_query.ParseResult, error) {
	return ParseMode(sql, 0)
}
