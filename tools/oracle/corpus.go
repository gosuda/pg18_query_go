//go:build ignore

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pg "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/proto"
)

type corpusFixture struct {
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

var executeCommand = regexp.MustCompile(`\\g(?:exec|set)?(?:\s+[^\r\n]*)?$`)

func main() {
	file, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	zip := gzip.NewWriter(file)
	encoder := json.NewEncoder(zip)
	files, queries, accepted, rejected, plCount, copyLines, commands := 0, 0, 0, 0, 0, 0, 0
	err = filepath.WalkDir(os.Args[1], func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(os.Args[1], path)
		if err != nil {
			return err
		}
		files++
		var buffer []byte
		startLine, copying := 1, false
		emit := func(force bool) error {
			if len(bytes.TrimSpace(buffer)) == 0 {
				buffer = nil
				return nil
			}
			sql := string(buffer)
			scan, scanErr := pg.Scan(sql)
			if !force {
				if scanErr != nil {
					if strings.Contains(scanErr.Error(), "unterminated") {
						return nil
					}
				} else {
					last := pg.Token(0)
					for _, token := range scan.Tokens {
						if token.Token != pg.Token_SQL_COMMENT && token.Token != pg.Token_C_COMMENT {
							last = token.Token
						}
					}
					if last != pg.Token_ASCII_59 {
						return nil
					}
				}
			}
			tree, parseErr := pg.Parse(sql)
			if !force && parseErr != nil && strings.Contains(parseErr.Error(), "at end of input") {
				return nil
			}
			f := corpusFixture{File: filepath.ToSlash(relative), Line: startLine, SQL: buffer}
			if parseErr != nil {
				f.ParseError = parseErr.Error()
				rejected++
			} else {
				if len(tree.Stmts) == 0 {
					buffer = nil
					return nil
				}
				f.Parse, err = proto.Marshal(tree)
				if err != nil {
					return err
				}
				f.Deparse, err = pg.Deparse(tree)
				if err != nil {
					f.DeparseError = err.Error()
				}
				for _, statement := range tree.Stmts {
					if statement.Stmt.GetDoStmt() != nil {
						f.PL = true
					}
					if fn := statement.Stmt.GetCreateFunctionStmt(); fn != nil {
						for _, option := range fn.Options {
							if d := option.GetDefElem(); d != nil && d.Defname == "language" && d.Arg.GetString_().GetSval() == "plpgsql" {
								f.PL = true
							}
						}
					}
				}
				if f.PL {
					f.PLJSON, err = pg.ParsePlPgSqlToJSON(sql)
					if err != nil {
						f.PLError = err.Error()
					}
					plCount++
				}
				last := tree.Stmts[len(tree.Stmts)-1].Stmt.GetCopyStmt()
				copying = last != nil && last.IsFrom && last.Filename == "" && !last.IsProgram
				accepted++
			}
			if err := encoder.Encode(f); err != nil {
				return err
			}
			queries++
			buffer = nil
			return nil
		}
		for index, line := range bytes.SplitAfter(data, []byte("\n")) {
			trimmed := strings.TrimSpace(string(line))
			if copying {
				if trimmed == `\.` {
					copying = false
				} else {
					copyLines++
				}
				continue
			}
			if strings.HasPrefix(trimmed, `\`) {
				// psql metacommands are outside SQL only when no quoted token is open.
				_, scanErr := pg.Scan(string(buffer))
				if scanErr == nil {
					commands++
					if executeCommand.MatchString(trimmed) {
						if err := emit(true); err != nil {
							return err
						}
					}
					continue
				}
			}
			if len(buffer) == 0 {
				startLine = index + 1
			}
			if match := executeCommand.FindIndex(line); match != nil {
				prefix := append(append([]byte(nil), buffer...), line[:match[0]]...)
				scan, scanErr := pg.Scan(string(prefix))
				outside := scanErr == nil
				if outside && len(scan.Tokens) > 0 {
					last := scan.Tokens[len(scan.Tokens)-1]
					outside = last.Token != pg.Token_SQL_COMMENT && last.Token != pg.Token_C_COMMENT
				}
				if outside {
					buffer = prefix
					commands++
					if err := emit(true); err != nil {
						return err
					}
					continue
				}
			}
			buffer = append(buffer, line...)
			if bytes.ContainsRune(line, ';') {
				if err := emit(false); err != nil {
					return err
				}
			}
		}
		if err := emit(true); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	if err := zip.Close(); err != nil {
		panic(err)
	}
	if err := file.Close(); err != nil {
		panic(err)
	}
	fmt.Printf("files=%d queries=%d accepted=%d rejected=%d plpgsql=%d copy_data_lines=%d psql_commands=%d\n", files, queries, accepted, rejected, plCount, copyLines, commands)
}
