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

// Inputs: libpg_query bbfab3948719782cc4ce48dbf2485f8ed0519089/test/deparse_tests.c.
// Expected protobufs and SQL: pg_query_go e6a9b9881a8b, PostgreSQL 18.4.
type officialDeparseFixture struct {
	SQL          string `json:"sql"`
	Parse        []byte `json:"parse,omitempty"`
	ParseError   string `json:"parse_error,omitempty"`
	Deparse      string `json:"deparse,omitempty"`
	DeparseError string `json:"deparse_error,omitempty"`
}

func officialDeparseFixtures(t *testing.T) []officialDeparseFixture {
	t.Helper()
	file, err := os.Open("testdata/libpg_query_deparse.json.gz")
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
	var fixtures []officialDeparseFixture
	for {
		var f officialDeparseFixture
		err := decoder.Decode(&f)
		if err == io.EOF {
			return fixtures
		}
		if err != nil {
			t.Fatal(err)
		}
		fixtures = append(fixtures, f)
	}
}

func TestOfficialDeparseAST(t *testing.T) {
	for i, f := range officialDeparseFixtures(t) {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("SQL: %s\npanic: %v", f.SQL, err)
				}
			}()
			got, err := pg.Parse(f.SQL)
			if f.ParseError != "" {
				if err == nil {
					t.Fatalf("accepted upstream-invalid SQL: %s", f.SQL)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s: %v", f.SQL, err)
			}
			want := new(pg.ParseResult)
			if err := proto.Unmarshal(f.Parse, want); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(want, got, protocmp.Transform()); diff != "" {
				t.Errorf("SQL: %s\n%s", f.SQL, diff)
			}
		})
	}
}

func TestOfficialDeparse(t *testing.T) {
	for i, f := range officialDeparseFixtures(t) {
		if f.ParseError != "" {
			continue
		}
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					t.Fatalf("SQL: %s\npanic: %v", f.SQL, err)
				}
			}()
			tree := new(pg.ParseResult)
			if err := proto.Unmarshal(f.Parse, tree); err != nil {
				t.Fatal(err)
			}
			got, err := pg.Deparse(tree)
			if f.DeparseError != "" {
				if err == nil {
					t.Fatalf("expected deparse error for %s", f.SQL)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s: %v", f.SQL, err)
			}
			if got != f.Deparse {
				t.Errorf("SQL: %s\nwant: %s\ngot: %s", f.SQL, f.Deparse, got)
			}
		})
	}
}
