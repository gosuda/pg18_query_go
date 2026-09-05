package pg_query_test

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	pg "github.com/gosuda/pg18_query_go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

type postgresFixture struct {
	File         string `json:"file"`
	Line         int    `json:"line"`
	SQL          []byte `json:"sql"`
	Parse        []byte `json:"parse,omitempty"`
	ParseError   string `json:"parse_error,omitempty"`
	Deparse      string `json:"deparse,omitempty"`
	DeparseError string `json:"deparse_error,omitempty"`
	PL           bool   `json:"pl,omitempty"`
	PLJSON       string `json:"pl_json,omitempty"`
	PLError      string `json:"pl_error,omitempty"`
}

func postgresFixtures(t *testing.T, run func(*testing.T, postgresFixture), include func(postgresFixture) bool) {
	t.Helper()
	file, err := os.Open("testdata/postgresql-18.4.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	zip, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer zip.Close()
	decoder := json.NewDecoder(zip)
	for {
		var f postgresFixture
		err := decoder.Decode(&f)
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		if !include(f) {
			continue
		}
		t.Run(fmt.Sprintf("%s/%d", f.File, f.Line), func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("SQL: %s\npanic: %v", f.SQL, err)
				}
			}()
			run(t, f)
		})
	}
}

func TestPostgreSQLCorpusAST(t *testing.T) {
	postgresFixtures(t, func(t *testing.T, f postgresFixture) {
		got, err := pg.Parse(string(f.SQL))
		if f.ParseError != "" {
			if err == nil {
				t.Fatalf("accepted upstream-invalid SQL: %s\nupstream error: %s", f.SQL, f.ParseError)
			}
			return
		}
		if err != nil {
			t.Fatalf("SQL: %s\n%v", f.SQL, err)
		}
		want := new(pg.ParseResult)
		if err := proto.Unmarshal(f.Parse, want); err != nil {
			t.Fatal(err)
		}
		if !proto.Equal(want, got) {
			t.Errorf("SQL: %s\n%s", f.SQL, cmp.Diff(want, got, protocmp.Transform()))
		}
	}, func(postgresFixture) bool { return true })
}

func TestPostgreSQLCorpusDeparse(t *testing.T) {
	postgresFixtures(t, func(t *testing.T, f postgresFixture) {
		tree := new(pg.ParseResult)
		if err := proto.Unmarshal(f.Parse, tree); err != nil {
			t.Fatal(err)
		}
		got, err := pg.Deparse(tree)
		if f.DeparseError != "" {
			if err == nil {
				t.Fatalf("expected upstream deparse failure for %s: %s", f.SQL, f.DeparseError)
			}
			return
		}
		if err != nil {
			t.Fatalf("SQL: %s\n%v", f.SQL, err)
		}
		if got != f.Deparse {
			t.Errorf("SQL: %s\nwant: %s\ngot: %s", f.SQL, f.Deparse, got)
		}
	}, func(f postgresFixture) bool { return f.ParseError == "" })
}

func TestPostgreSQLCorpusPLpgSQL(t *testing.T) {
	postgresFixtures(t, func(t *testing.T, f postgresFixture) {
		got, err := pg.ParsePlPgSqlToJSON(string(f.SQL))
		if f.PLError != "" {
			if err == nil {
				t.Fatalf("accepted upstream-invalid PL/pgSQL: %s\nupstream error: %s", f.SQL, f.PLError)
			}
			return
		}
		if err != nil {
			t.Fatalf("SQL: %s\n%v", f.SQL, err)
		}
		var expected, actual any
		if err := json.Unmarshal([]byte(f.PLJSON), &expected); err != nil {
			t.Fatalf("invalid reference JSON: %v", err)
		}
		if err := json.Unmarshal([]byte(got), &actual); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("SQL: %s\n%s", f.SQL, diff)
		}
	}, func(f postgresFixture) bool { return f.PL })
}
