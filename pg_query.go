// Package pg_query is a CGO-free, pure Go PostgreSQL 18.4 parser.
//
// It mirrors the public surface of pg_query_go v6 (Parse, Scan, ParseToJSON,
// Deparse, Normalize, Fingerprint, Split, Summary, and related helpers)
// on top of the native Ragel/goyacc parser in internal/neoparse. Message
// and enum types are protobuf-compatible aliases defined in types.go and
// implemented in internal/pgproto; see NOTICE for provenance.
package pg_query

//go:generate go run ./internal/cmd/genapi
//go:generate go run ./internal/cmd/genast

import (
	"errors"
	"strings"

	"github.com/gosuda/pg18_query_go/internal/neoparse"
)

// Error describes a PostgreSQL parser diagnostic. Cursorpos is one-based.
// It mirrors the upstream parser.Error shape (Message, Funcname, Filename,
// Lineno, Cursorpos, Context) so callers can distinguish scan.l grammar
// failures from internal errors.
type Error struct {
	Message   string
	Funcname  string
	Filename  string
	Lineno    int
	Cursorpos int
	Context   string
}

func (e *Error) Error() string { return e.Message }

func Parse(input string) (*ParseResult, error) {
	result, err := neoparse.ParseToPgQuery(input)
	return result, publicParseError(err)
}

func Scan(input string) (*ScanResult, error) {
	result, err := neoparse.Scan(input)
	return result, publicParseError(err)
}

func publicParseError(err error) error {
	if err == nil {
		return nil
	}
	var native *neoparse.Error
	if errors.As(err, &native) {
		converted := &Error{Message: native.Message, Funcname: native.Funcname, Filename: native.Filename, Lineno: native.Lineno, Cursorpos: native.Cursorpos, Context: native.Context}
		// The native scanner reports syntax errors without C-style provenance.
		// Upstream pg_query_go reports them as scan.l/scanner_yyerror; preserve
		// that contract for syntax errors only, leaving internal errors unlabeled.
		if strings.HasPrefix(converted.Message, "syntax error") {
			if converted.Funcname == "" {
				converted.Funcname = "scanner_yyerror"
			}
			if converted.Filename == "" {
				converted.Filename = "scan.l"
			}
		}
		return converted
	}
	return err
}
