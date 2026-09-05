package neoparse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

%%{
    machine pg_scanner;
    alphtype byte;

    newline     = '\r'? '\n';
    whitespace  = [ \t\r\n\f\v]+;
    linecomment = '--' [^\r\n]*;

    ident_start = [a-zA-Z_] | ( 0x80..0xFF );
    ident_cont  = [a-zA-Z_0-9] | 0x24 | ( 0x80..0xFF );
    ident       = ident_start ident_cont*;

    qident      = '"' ( [^"] | '""' )* '"';
    uqident     = [uU] '&"' ( [^"] | '""' )* '"';

    param       = 0x24 [0-9]+;

    decdigit    = [0-9];
    hexdigit    = [0-9a-fA-F];
    octdigit    = [0-7];
    bindigit    = [0-1];

    decinteger  = decdigit ('_'? decdigit)*;
    hexinteger  = '0' [xX] ('_'? hexdigit)+;
    octinteger  = '0' [oO] ('_'? octdigit)+;
    bininteger  = '0' [bB] ('_'? bindigit)+;

    numeric     = (decinteger '.' decinteger?) | ('.' decinteger);
    numericfail = decinteger '..';
    real        = (decinteger | numeric) [eE] [+\-]? decinteger;
    realfail    = (decinteger | numeric) [eE] [+\-];
    numericjunk = (decinteger | numeric | real) ident;

    bconst      = [bB] '\'';
    xconst      = [xX] '\'';
    usconst     = [uU] '&' '\'';

    op_chars    = [\~\!\@\#\^\&\|\`\?\+\-\*\/\%\<\>\=];
    operator    = op_chars+;

    main := |*
        whitespace => {
            // whitespace ignored
        };

        linecomment => {
            tok = SQL_COMMENT
            tokStart = int32(ts)
            tokEnd = int32(te)
            keywordKind = KeywordKind_NO_KEYWORD
            fbreak;
        };

        '/*' => {
            // Handle potentially nested C-style block comments
            depth := 1
            i := ts + 2
            for i < len(data) && depth > 0 {
                if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
                    depth++
                    i += 2
                } else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
                    depth--
                    i += 2
                } else {
                    i++
                }
            }
            if depth != 0 {
                l.errorAt("unterminated /* comment", ts)
                fbreak;
            }
            tok = C_COMMENT
            tokStart = int32(ts)
            tokEnd = int32(i)
            p = i - 1
            keywordKind = KeywordKind_NO_KEYWORD
            fbreak;
        };

        '::' => {
            tok = TYPECAST
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        '..' => {
            tok = DOT_DOT
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        ':=' => {
            tok = COLON_EQUALS
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        '=>' => {
            tok = EQUALS_GREATER
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        '<=' => {
            tok = LESS_EQUALS
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        '>=' => {
            tok = GREATER_EQUALS
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        '!=' | '<>' => {
            tok = NOT_EQUALS
            tokStart = int32(ts)
            tokEnd = int32(te)
            fbreak;
        };

        param => {
            tok = PARAM
            tokStart = int32(ts)
            tokEnd = int32(te)
            numStr := data[ts+1 : te]
            n, err := strconv.ParseInt(numStr, 10, 32)
            if err != nil {
                l.errorAt("parameter number too large", ts)
                fbreak;
            }
            if lval != nil {
                lval.setint(n)
            }
            fbreak;
        };

        numericfail => {
            // Integer followed by '..'; emit integer and rewind 2 bytes
            tokStart = int32(ts)
            tokEnd = int32(te - 2)
            p = te - 3
            raw := strings.ReplaceAll(data[ts:te-2], "_", "")
            if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
                tok = ICONST
                if lval != nil {
                    lval.setint(n)
                }
            } else {
                tok = FCONST
                if lval != nil {
                    lval.setstr(data[ts:te-2])
                }
            }
            fbreak;
        };

        real => {
            tok = FCONST
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := data[ts:te]
            if lval != nil {
                lval.setstr(raw)
            }
            fbreak;
        };

        numeric => {
            tok = FCONST
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := data[ts:te]
            if lval != nil {
                lval.setstr(raw)
            }
            fbreak;
        };

        hexinteger => {
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := strings.ReplaceAll(data[ts+2:te], "_", "")
            if n, err := strconv.ParseInt(raw, 16, 32); err == nil {
                tok = ICONST
                if lval != nil {
                    lval.setint(n)
                }
            } else {
                tok = FCONST
                if lval != nil {
                    lval.setstr(data[ts:te])
                }
            }
            fbreak;
        };

        octinteger => {
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := strings.ReplaceAll(data[ts+2:te], "_", "")
            if n, err := strconv.ParseInt(raw, 8, 32); err == nil {
                tok = ICONST
                if lval != nil {
                    lval.setint(n)
                }
            } else {
                tok = FCONST
                if lval != nil {
                    lval.setstr(data[ts:te])
                }
            }
            fbreak;
        };

        bininteger => {
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := strings.ReplaceAll(data[ts+2:te], "_", "")
            if n, err := strconv.ParseInt(raw, 2, 32); err == nil {
                tok = ICONST
                if lval != nil {
                    lval.setint(n)
                }
            } else {
                tok = FCONST
                if lval != nil {
                    lval.setstr(data[ts:te])
                }
            }
            fbreak;
        };

        decinteger => {
            tokStart = int32(ts)
            tokEnd = int32(te)
            raw := strings.ReplaceAll(data[ts:te], "_", "")
            if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
                tok = ICONST
                if lval != nil {
                    lval.setint(n)
                }
            } else {
                tok = FCONST
                if lval != nil {
                    lval.setstr(data[ts:te])
                }
            }
            fbreak;
        };

        '0' [xX] '_'? | '0' [oO] '_'? | '0' [bB] '_'? => {
            l.errorAt("invalid non-decimal integer", ts)
            fbreak;
        };

        numericjunk | realfail => {
            l.errorAt("trailing junk after numeric literal", ts)
            fbreak;
        };


        [nN] '\'' => {
            tok = NCHAR
            tokStart = int32(ts)
            tokEnd = int32(ts + 1)
            keywordKind = KeywordKind_COL_NAME_KEYWORD
            p = ts
            if lval != nil {
                lval.setstr("nchar")
            }
            fbreak;
        };

        bconst | xconst => {
            tok = BCONST
            prefix := "b"
            if data[ts] == 'x' || data[ts] == 'X' {
                tok = XCONST
                prefix = "x"
            }
            value, end, err := scanQuotedLiteral(data, te-1, literalBits)
            if err != nil {
                l.errorAt(err.Error(), ts)
                fbreak;
            }
            tokStart = int32(ts)
            tokEnd = int32(end)
            p = end - 1
            if lval != nil {
                lval.setstr(prefix + value)
            }
            fbreak;
        };

        usconst => {
            value, end, err := scanQuotedLiteral(data, te-1, literalStandard)
            if err != nil {
                l.errorAt(err.Error(), ts)
                fbreak;
            }
            tok = USCONST
            tokStart = int32(ts)
            tokEnd = int32(end)
            p = end - 1
            if lval != nil {
                lval.setstr(value)
            }
            fbreak;
        };

        # Dollar-quoted strings: $tag$...$tag$ or $$...$$
        0x24 ( [a-zA-Z\x80-\xFF_] [a-zA-Z\x80-\xFF_0-9]* )? 0x24 => {
            tag := data[ts:te]
            // search for closing tag
            endIdx := strings.Index(data[te:], tag)
            if endIdx >= 0 {
                fullEnd := te + endIdx + len(tag)
                if !utf8.ValidString(data[te:te+endIdx]) {
                    l.errorAt("invalid UTF-8 string", ts)
                    fbreak;
                }
                tok = SCONST
                tokStart = int32(ts)
                tokEnd = int32(fullEnd)
                p = fullEnd - 1
                if lval != nil {
                    lval.setstr(data[te : te+endIdx])
                }
                fbreak;
            }
            l.errorAt("unterminated dollar-quoted string", ts)
            fbreak;
        };

        [eE]? '\'' => {
            mode := literalStandard
            if data[ts] != '\'' {
                mode = literalEscape
            }
            value, end, err := scanQuotedLiteral(data, te-1, mode)
            if err != nil {
                l.errorAt(err.Error(), ts)
                fbreak;
            }
            tok = SCONST
            tokStart = int32(ts)
            tokEnd = int32(end)
            p = end - 1
            if lval != nil {
                lval.setstr(value)
            }
            fbreak;
        };

        uqident => {
            tok = UIDENT
            tokStart = int32(ts)
            tokEnd = int32(te)
            if lval != nil {
                lval.setstr(unquoteIdent(data[ts:te]))
            }
            fbreak;
        };

        qident => {
            value := unquoteIdent(data[ts:te])
            if value == "" {
                l.errorAt("zero-length delimited identifier", ts)
                fbreak;
            }
            if !utf8.ValidString(value) {
                l.errorAt("invalid UTF-8 identifier", ts)
                fbreak;
            }
            tok = IDENT
            tokStart = int32(ts)
            tokEnd = int32(te)
            if lval != nil {
                lval.setstr(truncateIdentifier(value))
            }
            fbreak;
        };

        ident => {
            word := data[ts:te]
            if !utf8.ValidString(word) {
                l.errorAt("invalid UTF-8 identifier", ts)
                fbreak;
            }
            tokStart = int32(ts)
            tokEnd = int32(te)
            if kw, ok := LookupKeyword(word); ok {
                tok = kw.Token
                keywordKind = kw.KeywordKind
                if lval != nil {
                    lval.setstr(strings.ToLower(word))
                }
            } else {
                tok = IDENT
                keywordKind = KeywordKind_NO_KEYWORD
                if lval != nil {
                    lval.setstr(truncateIdentifier(asciiDowncase(word)))
                }
            }
            fbreak;
        };

        operator => {
            // Check for embedded comment starts: /* or --
            nchars := te - ts
            rawOp := data[ts:te]

            if strings.HasPrefix(rawOp, "/*") {
                depth := 1
                i := ts + 2
                for i < len(data) && depth > 0 {
                    if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
                        depth++
                        i += 2
                    } else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
                        depth--
                        i += 2
                    } else {
                        i++
                    }
                }
                if depth != 0 {
                    l.errorAt("unterminated /* comment", ts)
                    fbreak;
                }
                tok = C_COMMENT
                tokStart = int32(ts)
                tokEnd = int32(i)
                p = i - 1
                keywordKind = KeywordKind_NO_KEYWORD
                fbreak;
            }
            if strings.HasPrefix(rawOp, "--") {
                i := ts + 2
                for i < len(data) && data[i] != '\n' && data[i] != '\r' {
                    i++
                }
                tok = SQL_COMMENT
                tokStart = int32(ts)
                tokEnd = int32(i)
                p = i - 1
                keywordKind = KeywordKind_NO_KEYWORD
                fbreak;
            }

            slashstar := strings.Index(rawOp, "/*")
            dashdash := strings.Index(rawOp, "--")
            cutIdx := -1
            if slashstar >= 0 && dashdash >= 0 {
                if slashstar < dashdash {
                    cutIdx = slashstar
                } else {
                    cutIdx = dashdash
                }
            } else if slashstar >= 0 {
                cutIdx = slashstar
            } else if dashdash >= 0 {
                cutIdx = dashdash
            }
            if cutIdx > 0 {
                nchars = cutIdx
                rawOp = rawOp[:nchars]
            }

            // If nchars > 1 and ends with + or - preceded only by SQL ops, strip trailing [+-]
            if nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
                hasSpecial := false
                for ic := nchars - 2; ic >= 0; ic-- {
                    c := rawOp[ic]
                    if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
                        c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
                        hasSpecial = true
                        break
                    }
                }
                if !hasSpecial {
                    for nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
                        nchars--
                    }
                    rawOp = rawOp[:nchars]
                }
            }

            if nchars >= 64 {
                l.Error("operator too long")
                fbreak;
            }
            tokStart = int32(ts)
            tokEnd = int32(ts + nchars)
            p = ts + nchars - 1

            if nchars == 1 && strings.ContainsRune(",()[].;:+-*/%^<>=", rune(rawOp[0])) {
                tok = int(rawOp[0])
            } else if nchars == 2 {
                switch rawOp {
                case "=>":
                    tok = EQUALS_GREATER
                case ">=":
                    tok = GREATER_EQUALS
                case "<=":
                    tok = LESS_EQUALS
                case "<>", "!=":
                    tok = NOT_EQUALS
                default:
                    tok = Op
                    if lval != nil {
                        lval.setstr(rawOp)
                    }
                }
            } else {
                tok = Op
                if lval != nil {
                    lval.setstr(rawOp)
                }
            }
            fbreak;
        };

        any => {
            if data[ts] == '"' {
                l.errorAt("unterminated quoted identifier", ts)
                fbreak;
            }
            tok = int(data[ts])
            tokStart = int32(ts)
            tokEnd = int32(ts + 1)
            fbreak;
        };
    *|;

    write data;
}%%

type Lexer struct {
	data        string
	p           int
	pe          int
	eof         int
	cs          int
	ts          int
	te          int
	act         int
	statements  []Statement
	statementLocations []statementLocation
	tokenStart int
	tokenEnd int
	initialToken int
	err         error
	scanMode    bool

	haveLookahead bool
	lookaheadTok  int
	lookaheadVal  yySymType
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		data: input,
		pe:   len(input),
		eof:  len(input),
	}
	var cs, ts, te, act int
	%% write init;
	l.cs = cs
	l.ts = ts
	l.te = te
	l.act = act
	return l
}

var newLexer = NewLexer
type lexer = Lexer

func (l *Lexer) rawLex(lval *yySymType) int {
	if l.err != nil {
		return 0
	}
	data := l.data
	p := l.p
	pe := l.pe
	eof := l.eof
	cs := l.cs
	ts := l.ts
	te := l.te
	act := l.act

	for p < pe {
		tok := 0
		var tokStart, tokEnd int32
		var keywordKind KeywordKind
		_ = tokStart
		_ = tokEnd
		_ = keywordKind

		oldP := p
		%% write exec;

		l.p = p
		l.cs = cs
		l.ts = ts
		l.te = te
		l.act = act

		if l.err != nil {
			return 0
		}
		if tok != 0 {
			lval.yyloc = int(tokStart)
			// In parser mode, ignore SQL comments and C comments
			if !l.scanMode && (tok == SQL_COMMENT || tok == C_COMMENT) {
				continue
			}
			return tok
		}

		if p <= oldP {
			p = oldP + 1
			l.p = p
		}
	}

	l.p = p
	lval.yyloc = len(data)
	l.cs = cs
	return 0
}

func (l *Lexer) Lex(lval *yySymType) int {
	if l.initialToken != 0 {
		token := l.initialToken
		l.initialToken = 0
		lval.yyloc = 0
		return token
	}
	var curTok int
	if l.haveLookahead {
		curTok = l.lookaheadTok
		if lval != nil {
			*lval = l.lookaheadVal
		}
		l.haveLookahead = false
	} else {
		curTok = l.rawLex(lval)
	}
	l.tokenStart = lval.yyloc
	l.tokenEnd = l.te

	if curTok == 0 {
		return 0
	}

	if l.scanMode {
		return curTok
	}

	switch curTok {
	case NOT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case BETWEEN, IN_P, LIKE, ILIKE, SIMILAR:
			return NOT_LA
		}

	case NULLS_P:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case FIRST_P, LAST_P:
			return NULLS_LA
		}

	case WITH:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case TIME, ORDINALITY:
			return WITH_LA
		}

	case WITHOUT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case TIME:
			return WITHOUT_LA
		}

	case FORMAT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case JSON:
			return FORMAT_LA
		}

	case USCONST, UIDENT:
		escape := byte('\\')
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		if nextTok == UESCAPE {
			var escVal yySymType
			if l.rawLex(&escVal) != SCONST {
				l.Error("UESCAPE must be followed by a simple string literal")
				return 0
			}
			esc := escVal.str()
			if len(esc) != 1 || isUnicodeHexDigit(esc[0]) || strings.ContainsRune("+\"' \t\n\r\f\v", rune(esc[0])) {
				l.Error("invalid Unicode escape character")
				return 0
			}
			escape = esc[0]
		} else {
			l.haveLookahead = true
			l.lookaheadTok = nextTok
			l.lookaheadVal = nextVal
		}
		decoded, err := decodeUnicodeEscapes(lval.str(), escape)
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		if curTok == UIDENT && decoded == "" {
			l.Error("zero-length delimited identifier")
			return 0
		}
		if curTok == UIDENT {
			decoded = truncateIdentifier(decoded)
		}
		lval.setstr(decoded)
		if curTok == UIDENT {
			return IDENT
		}
		return SCONST
	}

	return curTok
}

func (l *Lexer) Error(s string) {
	l.errorAt(s, l.tokenStart)
}

func (l *Lexer) errorAt(message string, location int) {
	if l.err != nil {
		return
	}
	location = max(0, min(location, len(l.data)))
	filename, funcname := "gram.y", "parser_yyerror"
	if message == "syntax error" || strings.HasPrefix(message, "syntax error:") {
		filename, funcname = "scan.l", "scanner_yyerror"
		if location == len(l.data) {
			message = "syntax error at end of input"
		} else {
			end := max(location+1, min(l.tokenEnd, len(l.data)))
			message = fmt.Sprintf("syntax error at or near %q", l.data[location:end])
		}
	}
	l.err = &Error{
		Message: message,
		Funcname: funcname,
		Filename: filename,
		Cursorpos: utf8.RuneCountInString(l.data[:location]) + 1,
		Location: location,
	}
}

// Scan parses the SQL string into tokens matching PostgreSQL's scan.l and pg_query.Scan.
func Scan(input string) (*ScanResult, error) {
	l := NewLexer(input)
	l.scanMode = true

	data := l.data
	p := l.p
	pe := l.pe
	eof := l.eof
	cs := l.cs
	ts := l.ts
	te := l.te
	act := l.act

	var tokens []*ScanToken
	var lval *yySymType

	for p < pe {
		tok := 0
		var tokStart, tokEnd int32
		var keywordKind KeywordKind

		%% write exec;

		l.p = p
		l.cs = cs
		l.ts = ts
		l.te = te
		l.act = act

		if l.err != nil {
			return nil, l.err
		}
		if tok != 0 {
			tokens = append(tokens, &ScanToken{
				Start:       tokStart,
				End:         tokEnd,
				Token:       pg_query.Token(tok),
				KeywordKind: pg_query.KeywordKind(keywordKind),
			})
		}
	}

	return &ScanResult{
		Version: 180004,
		Tokens:  tokens,
	}, nil
}

func unquoteIdent(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
	}
	if len(s) >= 4 && (s[0] == 'u' || s[0] == 'U') && s[1] == '&' && s[2] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[3:len(s)-1], `""`, `"`)
	}
	return s
}

var (
	errInvalidUnicodeEscape = errors.New("invalid Unicode escape")
	errInvalidUnicodeValue = errors.New("invalid Unicode escape value")
	errInvalidUnicodePair = errors.New("invalid Unicode surrogate pair")
)

func isUnicodeHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func decodeUnicodeEscapes(s string, escape byte) (string, error) {
	if !utf8.ValidString(s) || strings.IndexByte(s, 0) >= 0 {
		return "", errInvalidUnicodeValue
	}
	if strings.IndexByte(s, escape) < 0 {
		return s, nil
	}
	var decoded strings.Builder
	decoded.Grow(len(s))
	var highSurrogate rune
	for i := 0; i < len(s); {
		if s[i] != escape {
			if highSurrogate != 0 {
				return "", errInvalidUnicodePair
			}
			decoded.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i < len(s) && s[i] == escape {
			if highSurrogate != 0 {
				return "", errInvalidUnicodePair
			}
			decoded.WriteByte(escape)
			i++
			continue
		}
		digits := 4
		if i < len(s) && s[i] == '+' {
			digits = 6
			i++
		}
		if len(s)-i < digits {
			return "", errInvalidUnicodeEscape
		}
		var value rune
		for j := range digits {
			c := s[i+j]
			if !isUnicodeHexDigit(c) {
				return "", errInvalidUnicodeEscape
			}
			value <<= 4
			switch {
			case c <= '9':
				value += rune(c - '0')
			case c <= 'F':
				value += rune(c - 'A' + 10)
			default:
				value += rune(c - 'a' + 10)
			}
		}
		i += digits
		if value == 0 || value > utf8.MaxRune {
			return "", errInvalidUnicodeValue
		}
		if highSurrogate != 0 {
			if value < 0xDC00 || value > 0xDFFF {
				return "", errInvalidUnicodePair
			}
			value = utf16.DecodeRune(highSurrogate, value)
			highSurrogate = 0
		} else if value >= 0xD800 && value <= 0xDBFF {
			highSurrogate = value
			continue
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return "", errInvalidUnicodePair
		}
		decoded.WriteRune(value)
	}
	if highSurrogate != 0 {
		return "", errInvalidUnicodePair
	}
	return decoded.String(), nil
}

type literalMode uint8

const (
	literalStandard literalMode = iota
	literalEscape
	literalBits
)

var (
	errUnterminatedString = errors.New("unterminated quoted string")
	errInvalidString = errors.New("invalid UTF-8 or zero byte in string")
)

func scanQuotedLiteral(data string, quote int, mode literalMode) (string, int, error) {
	var value strings.Builder
	segment := quote + 1
	for i := segment; i < len(data); {
		if data[i] == '\'' {
			if mode != literalBits && i+1 < len(data) && data[i+1] == '\'' {
				value.WriteString(data[segment:i+1])
				i += 2
				segment = i
				continue
			}
			if next := continuedQuote(data, i+1); next >= 0 {
				value.WriteString(data[segment:i])
				i = next + 1
				segment = i
				continue
			}
			text := data[segment:i]
			if value.Len() != 0 {
				value.WriteString(text)
				text = value.String()
			}
			if !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 {
				return "", i+1, errInvalidString
			}
			return text, i+1, nil
		}
		if mode == literalEscape && data[i] == '\\' {
			value.WriteString(data[segment:i])
			next, err := appendStringEscape(&value, data, i)
			if err != nil {
				return "", next, err
			}
			i = next
			segment = i
			continue
		}
		i++
	}
	return "", len(data), errUnterminatedString
}

func continuedQuote(data string, pos int) int {
	hasNewline := false
	for pos < len(data) {
		switch data[pos] {
		case ' ', '\t', '\f', '\v':
			pos++
		case '\n', '\r':
			hasNewline = true
			pos++
		case '-':
			if pos+1 >= len(data) || data[pos+1] != '-' {
				return -1
			}
			pos += 2
			for pos < len(data) && data[pos] != '\n' && data[pos] != '\r' {
				pos++
			}
		default:
			if hasNewline && data[pos] == '\'' {
				return pos
			}
			return -1
		}
	}
	return -1
}

func appendStringEscape(dst *strings.Builder, data string, slash int) (int, error) {
	pos := slash + 1
	if pos >= len(data) {
		return pos, errUnterminatedString
	}
	c := data[pos]
	pos++
	if c == 'u' || c == 'U' {
		value, next, err := stringUnicodeEscape(data, slash)
		if err != nil {
			return next, err
		}
		if value >= 0xD800 && value <= 0xDBFF {
			if next+1 >= len(data) || data[next] != '\\' || data[next+1] != 'u' && data[next+1] != 'U' {
				return next, errInvalidUnicodePair
			}
			second, end, err := stringUnicodeEscape(data, next)
			if err != nil {
				return end, err
			}
			if second < 0xDC00 || second > 0xDFFF {
				return end, errInvalidUnicodePair
			}
			value = utf16.DecodeRune(value, second)
			next = end
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return next, errInvalidUnicodePair
		}
		dst.WriteRune(value)
		return next, nil
	}
	if c >= '0' && c <= '7' {
		value := c - '0'
		for count := 1; count < 3 && pos < len(data) && data[pos] >= '0' && data[pos] <= '7'; count++ {
			value = value*8 + data[pos] - '0'
			pos++
		}
		dst.WriteByte(value)
		return pos, nil
	}
	if c == 'x' && pos < len(data) && isUnicodeHexDigit(data[pos]) {
		start := pos
		pos++
		if pos < len(data) && isUnicodeHexDigit(data[pos]) {
			pos++
		}
		value, err := strconv.ParseUint(data[start:pos], 16, 8)
		if err != nil {
			return pos, err
		}
		dst.WriteByte(byte(value))
		return pos, nil
	}
	switch c {
	case 'b':
		c = '\b'
	case 'f':
		c = '\f'
	case 'n':
		c = '\n'
	case 'r':
		c = '\r'
	case 't':
		c = '\t'
	}
	dst.WriteByte(c)
	return pos, nil
}

func stringUnicodeEscape(data string, slash int) (rune, int, error) {
	digits := 4
	if data[slash+1] == 'U' {
		digits = 8
	}
	end := slash + 2 + digits
	if end > len(data) {
		return 0, len(data), errInvalidUnicodeEscape
	}
	for i := slash+2; i < end; i++ {
		if !isUnicodeHexDigit(data[i]) {
			return 0, i, errInvalidUnicodeEscape
		}
	}
	value, err := strconv.ParseUint(data[slash+2:end], 16, 32)
	if err != nil || value == 0 || value > utf8.MaxRune {
		return 0, end, errInvalidUnicodeValue
	}
	return rune(value), end, nil
}

func truncateIdentifier(value string) string {
	if len(value) <= 63 {
		return value
	}
	end := 63
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func asciiDowncase(value string) string {
	first := strings.IndexFunc(value, func(c rune) bool { return c >= 'A' && c <= 'Z' })
	if first < 0 {
		return value
	}
	lower := []byte(value)
	for i := first; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower[i] += 'a' - 'A'
		}
	}
	return string(lower)
}
