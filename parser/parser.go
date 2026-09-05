// Package parser exposes the low-level protobuf API using the native Go
// parser. It mirrors the upstream pg_query_go parser subpackage surface
// without cgo: callers marshal Parse/Scan/Summary results with protobuf and
// unmarshal them back before calling Deparse.
package parser

import (
	pg_query "github.com/gosuda/pg18_query_go"
	"google.golang.org/protobuf/proto"
)

type Error = pg_query.Error
type FingerprintOption = pg_query.FingerprintOption

const (
	FingerprintDefault               = pg_query.FingerprintDefault
	FingerprintRangeVarIgnoreAliases = pg_query.FingerprintRangeVarIgnoreAliases
	FingerprintRangeVarIncludeSchema = pg_query.FingerprintRangeVarIncludeSchema
	FingerprintRangeVarPG17Compat    = pg_query.FingerprintRangeVarPG17Compat
	FingerprintRelnameFull           = pg_query.FingerprintRelnameFull
)

func ParseToJSON(input string) (string, error)        { return pg_query.ParseToJSON(input) }
func ParsePlPgSqlToJSON(input string) (string, error) { return pg_query.ParsePlPgSqlToJSON(input) }
func Normalize(input string) (string, error)          { return pg_query.Normalize(input) }
func NormalizeUtility(input string) (string, error)   { return pg_query.NormalizeUtility(input) }
func SplitWithScanner(input string, trimSpace bool) ([]string, error) {
	return pg_query.SplitWithScanner(input, trimSpace)
}
func SplitWithParser(input string, trimSpace bool) ([]string, error) {
	return pg_query.SplitWithParser(input, trimSpace)
}
func FingerprintToUInt64(input string) (uint64, error) { return pg_query.FingerprintToUInt64(input) }
func FingerprintToUInt64WithOpts(input string, opts FingerprintOption) (uint64, error) {
	return pg_query.FingerprintToUInt64WithOpts(input, opts)
}
func FingerprintToHexStr(input string) (string, error) { return pg_query.Fingerprint(input) }
func FingerprintToHexStrWithOpts(input string, opts FingerprintOption) (string, error) {
	return pg_query.FingerprintWithOpts(input, opts)
}
func HashXXH3_64(input []byte, seed uint64) uint64 { return pg_query.HashXXH3_64(input, seed) }
func IsUtilityStmt(input string) ([]bool, error)   { return pg_query.IsUtilityStmt(input) }

func ParseToProtobuf(input string) ([]byte, error) {
	tree, err := pg_query.Parse(input)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(tree)
}

func ScanToProtobuf(input string) ([]byte, error) {
	tree, err := pg_query.Scan(input)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(tree)
}

func SummaryToProtobuf(input string, truncateLimit int) ([]byte, error) {
	summary, err := pg_query.Summary(input, truncateLimit)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(summary)
}

func DeparseFromProtobuf(input []byte) (string, error) {
	tree := new(pg_query.ParseResult)
	if err := proto.Unmarshal(input, tree); err != nil {
		return "", err
	}
	return pg_query.Deparse(tree)
}
