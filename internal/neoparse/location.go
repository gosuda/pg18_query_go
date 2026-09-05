package neoparse

// yyLocDefault follows PostgreSQL's first nonempty RHS location rule.
func yyLocDefault(cur *int, rhs []yySymType, n int) {
	*cur = -1
	for i := 1; i <= n; i++ {
		if rhs[i].yyloc >= 0 {
			*cur = rhs[i].yyloc
			return
		}
	}
}

type statementLocation struct {
	start  int32
	length int32
}

func (l *Lexer) startStatement(stmt Statement, location int) {
	if stmt != nil {
		l.statementLocations = append(l.statementLocations, statementLocation{start: int32(location)})
	}
}

func (l *Lexer) finishStatement(end int, stmt Statement, location int) {
	if len(l.statementLocations) != 0 {
		last := &l.statementLocations[len(l.statementLocations)-1]
		if last.length == 0 {
			last.length = int32(end) - last.start
		}
	}
	l.startStatement(stmt, location)
}

// Error exposes PostgreSQL-compatible diagnostics. Location is a zero-based byte
// offset; Cursorpos is a one-based Unicode character position.
// Funcname/Filename report the logical PostgreSQL diagnostic provenance
// (scan.l/scanner_yyerror, gram.y/parser_yyerror) for the error path.
type Error struct {
	Message   string
	Funcname  string
	Filename  string
	Lineno    int
	Cursorpos int
	Context   string
	Location  int
}

func (e *Error) Error() string { return e.Message }
