//go:build ignore

package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
	"os"
)

type fixture struct {
	SQL          string `json:"sql"`
	Parse        []byte `json:"parse,omitempty"`
	ParseError   string `json:"parse_error,omitempty"`
	Deparse      string `json:"deparse,omitempty"`
	DeparseError string `json:"deparse_error,omitempty"`
}

func main() {
	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var queries []string
	if err := json.Unmarshal(input, &queries); err != nil {
		panic(err)
	}
	file, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	zip := gzip.NewWriter(file)
	encoder := json.NewEncoder(zip)
	for _, sql := range queries {
		f := fixture{SQL: sql}
		tree, err := pg.Parse(sql)
		if err != nil {
			f.ParseError = err.Error()
		} else {
			f.Parse, err = proto.Marshal(tree)
			if err != nil {
				panic(err)
			}
			f.Deparse, err = pg.Deparse(tree)
			if err != nil {
				f.DeparseError = err.Error()
			}
		}
		if err := encoder.Encode(f); err != nil {
			panic(err)
		}
	}
	if err := zip.Close(); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("Generated %d upstream fixtures\n", len(queries))
}
