package pg_query

import "strings"

func SplitWithParser(input string, trimSpace bool) ([]string, error) {
	tree, err := Parse(input)
	if err != nil {
		return nil, err
	}
	if len(tree.Stmts) == 0 {
		return nil, nil
	}
	result := make([]string, len(tree.Stmts))
	for i, stmt := range tree.Stmts {
		start, end := int(stmt.StmtLocation), len(input)
		if stmt.StmtLen != 0 {
			end = start + int(stmt.StmtLen)
		}
		result[i] = input[start:end]
		if trimSpace {
			result[i] = strings.TrimSpace(result[i])
		}
	}
	return result, nil
}

func SplitWithScanner(input string, trimSpace bool) ([]string, error) {
	scan, err := Scan(input)
	if err != nil {
		return nil, err
	}
	var result []string
	start, parens, keyword := 0, 0, false
	lastEnd := 0
	appendStmt := func(end int) {
		stmt := input[start:end]
		if trimSpace {
			stmt = strings.TrimSpace(stmt)
		}
		result = append(result, stmt)
		start, keyword = end+1, false
	}
	for _, token := range scan.Tokens {
		if token.KeywordKind != KeywordKind_NO_KEYWORD {
			keyword = true
		} else {
			switch token.Token {
			case Token_ASCII_40:
				parens++
			case Token_ASCII_41:
				parens--
			case Token_ASCII_59:
				if parens == 0 {
					if keyword {
						appendStmt(int(token.Start))
					} else {
						start = int(token.Start) + 1
					}
				}
			}
		}
		lastEnd = int(token.End)
	}
	if keyword && parens == 0 {
		appendStmt(lastEnd)
	}
	return result, nil
}
