# pg18_query_go

Pure Go PostgreSQL 18.4 SQL and PL/pgSQL parser. No CGO or external parser runtime required. Go 1.25+.

Provides parsing, deparsing, normalization, fingerprints, statement splitting, and query summaries through a `pg_query_go`-style API.

## Install

```sh
go get github.com/gosuda/pg18_query_go
```

```go
import pg "github.com/gosuda/pg18_query_go"

tree, err := pg.Parse("SELECT id FROM accounts WHERE balance > 100")
```

## Compatibility

Verified against 49,125 PostgreSQL corpus inputs, 854 PL/pgSQL cases, and upstream API tests. This does not guarantee identical behavior for every input or live-database validation.

Trigger output intentionally fixes malformed JSON in the upstream parser. Other catalog-free PL/pgSQL limitations remain.

## Development

```sh
CGO_ENABLED=0 go test ./...
go generate ./...                         # requires Ragel
python3 tools/regenerate_fixtures.py --check  # requires Python 3.10+ and a C compiler
```

Tests pass on Go 1.25 and 1.27, including race and pointer-safety checks. AST lint findings remain.

## License

BSD-3-Clause. See [NOTICE](NOTICE) and `licenses/` for third-party terms and attribution, including gFire Projects Groups.
