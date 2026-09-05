package plpgsql

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type token struct { kind int; text string; start, end, line int; quoted bool }
type scanner struct { source string; pos, line int }

%%{
 machine plpgsql_scanner;
 alphtype byte;
 ident = ([a-zA-Z_] | 0x80..0xff) ([a-zA-Z_0-9$] | 0x80..0xff)*;
 number = [0-9] [a-zA-Z_0-9]*;
 whitespace = [ \t\r\n\f\v]+;
 main := |*
  whitespace => {};
  '--' [^\n\r]* => {};
  '/*' => {
   depth, i := 1, ts+2
   for i < len(data) && depth > 0 {
    if strings.HasPrefix(data[i:], "/*") { depth++; i += 2 } else if strings.HasPrefix(data[i:], "*/") { depth--; i += 2 } else { i++ }
   }
   if depth != 0 { failAt(data, ts, "unterminated block comment") }
   p = i-1
  };
  [uU] "&'" => { tok = s.unicodeToken(ts, '\''); p = tok.end-1; fbreak; };
  [uU] '&"' => { tok = s.unicodeToken(ts, '"'); p = tok.end-1; fbreak; };
  [eE] "'" => { tok = s.quotedToken(ts, ts+1, '\'', true); p = tok.end-1; fbreak; };
  "'" => { tok = s.quotedToken(ts, ts, '\'', false); p = tok.end-1; fbreak; };
  '"' => { tok = s.quotedToken(ts, ts, '"', false); p = tok.end-1; fbreak; };
  '$' => {
   i := ts+1
   for i < len(data) && (data[i] == '_' || data[i] >= 'a' && data[i] <= 'z' || data[i] >= 'A' && data[i] <= 'Z' || i > ts+1 && data[i] >= '0' && data[i] <= '9' || data[i] >= 128) { i++ }
   if i < len(data) && data[i] == '$' {
    delim := data[ts:i+1]
    end := strings.Index(data[i+1:], delim)
    if end < 0 { failAt(data, ts, "unterminated dollar-quoted string") }
    tok = token{kind: STRING, text: data[i+1:i+1+end], start: ts, end: i+1+end+len(delim)}
   } else {
    i = ts+1
    for i < len(data) && data[i] >= '0' && data[i] <= '9' { i++ }
    tok = token{kind: WORD, text: data[ts:i], start: ts, end: i}
   }
   p = tok.end-1
   fbreak;
  };
  ident => { tok = token{kind: WORD, text: foldIdentifier(data[ts:te]), start: ts, end: te}; fbreak; };
  number => { tok = token{kind: NUMBER, text: data[ts:te], start: ts, end: te}; fbreak; };
  ':=' | '=>' | '..' | '<<' | '>>' | '::' => { tok = token{kind: SYMBOL, text: data[ts:te], start: ts, end: te}; fbreak; };
  any => { tok = token{kind: int(data[ts]), text: data[ts:te], start: ts, end: te}; fbreak; };
 *|;
}%%

%% write data;

func (s *scanner) next() token {
 data := s.source
 p, pe, eof := s.pos, len(data), len(data)
 cs, ts, te, act := 0, 0, 0, 0
 tok := token{start: len(data), end: len(data)}
 %% write init;
 %% write exec;
 _ = cs; _ = ts; _ = te; _ = act; _ = eof
 tok.line = s.line + 1 + strings.Count(data[s.pos:tok.start], "\n")
 s.line += strings.Count(data[s.pos:p], "\n")
 s.pos = p
 if tok.kind == WORD && !tok.quoted {
  if k, ok := keywords[tok.text]; ok { tok.kind = k }
 }
 return tok
}

func (s *scanner) quotedToken(start, quoteAt int, quote byte, escape bool) token {
 var b strings.Builder
 data := s.source
 for i := quoteAt+1; i < len(data); {
  c := data[i]
  i++
  if c == quote {
   if i < len(data) && data[i] == quote { b.WriteByte(quote); i++; continue }
   if quote == '\'' { if next := continuedQuote(data, i); next >= 0 { i = next+1; continue } }
   kind, text := STRING, b.String()
   if !utf8.ValidString(text) { failAt(data, start, "invalid UTF-8 in string") }
   if quote == '"' { kind = WORD; if b.Len() == 0 { failAt(data, start, "zero-length delimited identifier") }; if quoteAt == start { text = truncateIdentifier(text) } }
   return token{kind: kind, text: text, start: start, end: i, quoted: quote == '"'}
  }
  if c == '\\' && escape {
   if i == len(data) { break }
   c = data[i]; i++
   switch c {
   case 'b': c = '\b'
   case 'f': c = '\f'
   case 'n': c = '\n'
   case 'r': c = '\r'
   case 't': c = '\t'
   case 'u', 'U':
    value, end := s.escapeRune(i, c)
    b.WriteRune(value); i = end; continue
   case 'x':
    j := i
    for i < len(data) && i < j+2 && strings.ContainsRune("0123456789abcdefABCDEF", rune(data[i])) { i++ }
    if i > j { v, _ := strconv.ParseUint(data[j:i], 16, 8); c = byte(v) }
   default:
    if c >= '0' && c <= '7' {
     j := i-1
     for i < len(data) && i < j+3 && data[i] >= '0' && data[i] <= '7' { i++ }
     v, _ := strconv.ParseUint(data[j:i], 8, 16); c = byte(v)
    }
   }
  }
  if c == 0 { failAt(data, start, "invalid zero byte in string") }
  b.WriteByte(c)
 }
 failAt(data, start, "unterminated quoted string")
 return token{}
}

func foldIdentifier(value string) string {
 var result []byte
 for i := range len(value) {
  if value[i] >= 'A' && value[i] <= 'Z' {
   if result == nil { result = []byte(value) }
   result[i] += 'a'-'A'
  }
 }
 if result != nil { value = string(result) }
 return truncateIdentifier(value)
}
func truncateIdentifier(value string) string {
 if len(value) <= 63 { return value }
 end := 63
 for !utf8.RuneStart(value[end]) { end-- }
 return value[:end]
}
func continuedQuote(data string, pos int) int {
 newline := false
 for pos < len(data) {
  switch data[pos] {
  case ' ', '\t', '\f', '\v': pos++
  case '\n', '\r': newline = true; pos++
  case '-':
   if !strings.HasPrefix(data[pos:], "--") { return -1 }
   for pos < len(data) && data[pos] != '\n' && data[pos] != '\r' { pos++ }
  default:
   if newline && data[pos] == '\'' { return pos }
   return -1
  }
 }
 return -1
}
