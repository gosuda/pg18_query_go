package neoparse

import (
	"strings"

	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

// ScanResult and ScanToken are aliased to pg_query types for 100% surface compatibility.
type ScanResult = pg_query.ScanResult
type ScanToken = pg_query.ScanToken

// Single-character ASCII tokens matching PostgreSQL scan.l and pg_query.Token
const (
	ASCII_36 = 36 // $
	ASCII_37 = 37 // %
	ASCII_40 = 40 // (
	ASCII_41 = 41 // )
	ASCII_42 = 42 // *
	ASCII_43 = 43 // +
	ASCII_44 = 44 // ,
	ASCII_45 = 45 // -
	ASCII_46 = 46 // .
	ASCII_47 = 47 // /
	ASCII_58 = 58 // :
	ASCII_59 = 59 // ;
	ASCII_60 = 60 // <
	ASCII_61 = 61 // =
	ASCII_62 = 62 // >
	ASCII_63 = 63 // ?
	ASCII_91 = 91 // [
	ASCII_92 = 92 // \
	ASCII_93 = 93 // ]
	ASCII_94 = 94 // ^
)

type KeywordKind int32

const (
	KeywordKind_NO_KEYWORD KeywordKind = iota
	KeywordKind_UNRESERVED_KEYWORD
	KeywordKind_COL_NAME_KEYWORD
	KeywordKind_TYPE_FUNC_NAME_KEYWORD
	KeywordKind_RESERVED_KEYWORD
)

type KeywordInfo struct {
	Token       int
	KeywordKind KeywordKind
}

func LookupKeyword(word string) (KeywordInfo, bool) {
	keyword, ok := ScanKeywords[strings.ToLower(word)]
	return keyword, ok
}
