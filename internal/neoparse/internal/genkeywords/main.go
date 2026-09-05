package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"regexp"
)

func main() {
	data, err := os.ReadFile("pg18/kwlist.h")
	if err != nil {
		log.Fatal(err)
	}
	pattern := regexp.MustCompile(`(?m)^PG_KEYWORD\("([^"]+)", ([A-Z_0-9]+), ([A-Z_]+), [A-Z_]+\)`)
	keywords := pattern.FindAllSubmatch(data, -1)
	if len(keywords) == 0 {
		log.Fatal("no PG_KEYWORD entries in pg18/kwlist.h")
	}
	var output bytes.Buffer
	output.WriteString("// Code generated from PostgreSQL 18 kwlist.h; DO NOT EDIT.\n\npackage neoparse\n\nvar ScanKeywords = map[string]KeywordInfo{\n")
	for _, keyword := range keywords {
		fmt.Fprintf(&output, "%q: {Token: %s, KeywordKind: KeywordKind_%s},\n", keyword[1], keyword[2], keyword[3])
	}
	output.WriteString("}\n")
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("keywords.go", formatted, 0644); err != nil {
		log.Fatal(err)
	}
}
