package pg_query_test

import (
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	pg_query "github.com/gosuda/pg18_query_go"
)

// Consumer-observable port of SuperSQL internal/neoparse
// TestParseRetainsSemanticValuesAcrossGC. It stresses the public API while a
// background goroutine forces GC, proving deeply nested string/int constants
// survive goyacc semantic-value handling without unsafe lifetime bugs.
func TestPublicParseRetainsValuesAcrossGC(t *testing.T) {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				runtime.GC()
			}
		}
	}()
	defer func() {
		close(stop)
		wg.Wait()
	}()

	for i := range 32 {
		value := "retained-" + strconv.Itoa(i) + strings.Repeat("x", 256)
		query := "SELECT " + strings.Repeat("(", 128) + "'" + value + "'" + strings.Repeat(")", 128) + ", " + strconv.Itoa(i)
		tree, err := pg_query.Parse(query)
		if err != nil {
			t.Fatal(err)
		}
		runtime.GC()
		if len(tree.Stmts) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(tree.Stmts))
		}
		targets := tree.Stmts[0].Stmt.GetSelectStmt().GetTargetList()
		if len(targets) != 2 {
			t.Fatalf("expected 2 targets, got %d", len(targets))
		}
		gotStr := targets[0].GetResTarget().GetVal().GetAConst().GetSval().GetSval()
		if gotStr != value {
			t.Fatalf("string value after GC has length %d, want length %d", len(gotStr), len(value))
		}
		gotInt := targets[1].GetResTarget().GetVal().GetAConst().GetIval().GetIval()
		if gotInt != int32(i) {
			t.Fatalf("integer value after GC = %d, want %d", gotInt, i)
		}
	}
}
