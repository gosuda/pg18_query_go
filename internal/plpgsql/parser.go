package plpgsql

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

//go:generate ragel -Z scan.rl -o scan.go
//go:generate go tool goyacc -o y.go -p yy gram.y
//go:generate gofmt -w scan.go y.go

type Error struct {
	Message         string
	Line, Cursorpos int
}

func (e *Error) Error() string { return e.Message }
func failAt(source string, pos int, message string) {
	panic(&Error{Message: message, Line: 1 + strings.Count(source[:pos], "\n"), Cursorpos: utf8.RuneCountInString(source[:pos]) + 1})
}

type scope struct {
	label string
	loop  bool
	names map[string]*Datum
}
type caseElse struct {
	present bool
	body    []Statement
}
type parser struct {
	scan      scanner
	pending   []token
	last      token
	scopes    []*scope
	function  Function
	config    Config
	parseSQL  SQLParser
	result    *Block
	hasOutput bool
}

var keywords = map[string]int{
	"begin": BEGIN, "declare": DECLARE, "end": END, "if": IF, "then": THEN,
	"else": ELSE, "elsif": ELSIF, "elseif": ELSIF, "case": CASE, "when": WHEN,
	"loop": LOOP, "while": WHILE, "for": FOR, "foreach": FOREACH,
	"exception": EXCEPTION, "return": RETURN, "raise": RAISE, "assert": ASSERT,
	"perform": PERFORM, "execute": EXECUTE, "get": GET, "open": OPEN,
	"fetch": FETCH, "move": MOVE, "close": CLOSE, "exit": EXIT,
	"continue": CONTINUE, "null": NULL, "call": CALL, "do": DO,
	"commit": COMMIT, "rollback": ROLLBACK,
}

func Parse(source string, config Config, parseSQL SQLParser) (result *Function, err error) {
	defer func() {
		if value := recover(); value != nil {
			if e, ok := value.(*Error); ok {
				result, err = nil, e
				return
			}
			panic(value)
		}
	}()
	if i := strings.IndexByte(source, 0); i >= 0 {
		failAt(source, i, "invalid zero byte in function definition")
	}
	p := &parser{scan: scanner{source: source}, config: config, parseSQL: parseSQL}
	p.pushScope(config.Name, false)
	p.initialize()
	if yyParse(p) != 0 {
		p.Error("syntax error")
	}
	p.function.Action = p.result
	if config.Inline || config.Procedure || config.ReturnType == "void" || config.ReturnType == "event_trigger" || config.ReturnsSet || p.hasOutput {
		b := p.result
		if b.Label != "" || len(b.Exceptions) > 0 {
			b = &Block{Body: []Statement{b}}
			p.function.Action = b
		}
		if len(b.Body) == 0 {
			b.Body = append(b.Body, &Return{Kind: "return"})
		} else if r, ok := b.Body[len(b.Body)-1].(*Return); !ok || r.Kind != "return" {
			b.Body = append(b.Body, &Return{Kind: "return"})
		}
	}
	return &p.function, nil
}

func (p *parser) initialize() {
	p.config.ReturnType = catalogType(p.config.ReturnType)
	switch p.config.ReturnType {
	case "_record", "cstring", "internal", "any", "language_handler", "fdw_handler", "index_am_handler", "tsm_handler", "table_am_handler", "pg_ddl_command":
		p.fail("PL/pgSQL functions cannot return type " + p.config.ReturnType)
	}
	if p.config.ReturnType == "trigger" || p.config.ReturnType == "event_trigger" {
		if len(p.config.Parameters) > 0 {
			p.fail("trigger functions cannot have declared arguments")
		}
		if p.config.ReturnType == "trigger" {
			p.function.NewVarno = p.addDatum("new", "record", 0).Number
			p.function.OldVarno = p.addDatum("old", "record", 0).Number
			for _, v := range []struct{ name, typ string }{{"tg_name", "name"}, {"tg_when", "text"}, {"tg_level", "text"}, {"tg_op", "text"}, {"tg_relid", "oid"}, {"tg_relname", "name"}, {"tg_table_name", "name"}, {"tg_table_schema", "name"}, {"tg_nargs", "int4"}, {"tg_argv", "_text"}} {
				p.addDatum(v.name, v.typ, 0).Kind = Promise
			}
		} else {
			p.addDatum("tg_event", "text", 0).Kind = Promise
			p.addDatum("tg_tag", "text", 0).Kind = Promise
		}
	} else {
		var outputs []*Datum
		for i, arg := range p.config.Parameters {
			name := arg.Name
			if name == "" {
				name = fmt.Sprintf("$%d", i+1)
			}
			d := p.addDatum(name, polymorphicType(arg.Type), 0)
			p.currentScope().names[fmt.Sprintf("$%d", i+1)] = d
			if arg.Output {
				outputs = append(outputs, d)
				p.hasOutput = true
			}
		}
		if len(outputs) > 1 || len(outputs) == 1 && p.config.Procedure {
			p.makeRow(outputs, -1)
		}
		if polymorphicType(p.config.ReturnType) != p.config.ReturnType && !p.hasOutput {
			p.addDatum("$0", polymorphicType(p.config.ReturnType), 0)
		}
	}
	delete(p.currentScope().names, "found")
	p.addDatum("found", "bool", 0)
}

func polymorphicType(name string) string {
	switch name {
	case "anyarray", "anycompatiblearray":
		return "_int4"
	case "anyrange", "anycompatiblerange":
		return "int4range"
	case "anymultirange", "anycompatiblemultirange":
		return "int4multirange"
	case "anyelement", "anynonarray", "anycompatible", "anycompatiblenonarray", "anyenum":
		return "int4"
	}
	return name
}

func catalogType(name string) string {
	if builtinTypes[name] || strings.HasSuffix(name, "%TYPE") || strings.HasSuffix(name, "%rowtype") {
		return name
	}
	// libpg_query treats uncatalogued public types as records.
	if strings.HasPrefix(name, "_") {
		return "_record"
	}
	return "record"
}

func (p *parser) Error(message string) { failAt(p.scan.source, p.last.start, message) }
func (p *parser) fail(message string)  { p.Error(message) }
func (p *parser) next() token {
	if n := len(p.pending); n > 0 {
		t := p.pending[n-1]
		p.pending = p.pending[:n-1]
		p.last = t
		return t
	}
	t := p.scan.next()
	p.last = t
	return t
}
func (p *parser) unread(t token) { p.pending = append(p.pending, t) }
func (p *parser) peek() token    { t := p.next(); p.unread(t); return t }
func (p *parser) accept(text string) bool {
	t := p.next()
	if t.kind != STRING && !t.quoted && t.text == text {
		return true
	}
	p.unread(t)
	return false
}
func (p *parser) expect(text string) token {
	t := p.next()
	if t.kind == STRING || t.quoted || t.text != text {
		p.fail(fmt.Sprintf("expected %s at or near %q", text, t.text))
	}
	return t
}
func identifierToken(t token) bool {
	if t.kind == WORD {
		return true
	}
	switch t.kind {
	case ELSIF, EXCEPTION, RETURN, RAISE, ASSERT, PERFORM, GET, OPEN, FETCH, MOVE, CLOSE, EXIT, CONTINUE, CALL, DO, COMMIT, ROLLBACK:
		return true
	}
	return false
}
func (p *parser) identifier() token {
	t := p.next()
	if !identifierToken(t) {
		p.fail(fmt.Sprintf("expected identifier at or near %q", t.text))
	}
	return t
}
func (p *parser) Lex(value *yySymType) int {
	t := p.next()
	if t.kind != WORD && identifierToken(t) {
		next := p.peek()
		if next.text == "=" || next.text == ":=" || next.text == "[" || next.text == "." {
			t.kind = WORD
		}
	}
	if t.text == "<<" {
		name := p.identifier()
		p.expect(">>")
		t.kind = LABEL
		t.text = name.text
	}
	value.settok(t)
	return t.kind
}
func (p *parser) pushScope(label string, loop bool) *scope {
	s := &scope{label: label, loop: loop, names: make(map[string]*Datum)}
	p.scopes = append(p.scopes, s)
	return s
}
func (p *parser) currentScope() *scope { return p.scopes[len(p.scopes)-1] }
func (p *parser) endScope(s *scope, label string) {
	if label != "" && label != s.label {
		p.fail("end label does not match block or loop label")
	}
	p.scopes = p.scopes[:len(p.scopes)-1]
}
func (p *parser) lookup(name string) *Datum {
	for i := len(p.scopes) - 1; i >= 0; i-- {
		if d := p.scopes[i].names[name]; d != nil {
			return d
		}
	}
	return nil
}
func (p *parser) addDatum(name, typ string, line int) *Datum {
	if _, exists := p.currentScope().names[name]; exists {
		p.fail(fmt.Sprintf("duplicate declaration of variable %q", name))
	}
	typ = catalogType(typ)
	switch typ {
	case "_record", "void", "trigger", "event_trigger", "cstring", "internal", "any", "language_handler", "fdw_handler", "index_am_handler", "tsm_handler", "table_am_handler", "pg_ddl_command":
		p.fail("variable cannot have pseudo-type " + typ)
	}
	d := &Datum{Number: len(p.function.Datums), Name: name, Type: typ, Line: line}
	if typ == "record" || typ == "pg_type" || typ == "pg_class" || typ == "pg_attribute" || typ == "pg_proc" {
		d.Kind = Record
	}
	p.function.Datums = append(p.function.Datums, d)
	p.currentScope().names[name] = d
	return d
}
func (p *parser) makeRow(datums []*Datum, line int) *Datum {
	d := &Datum{Kind: Row, Number: len(p.function.Datums), Name: "(unnamed row)", Line: line}
	for _, v := range datums {
		d.Fields = append(d.Fields, Field{Name: v.Name, Varno: v.Number})
	}
	p.function.Datums = append(p.function.Datums, d)
	return d
}
func (p *parser) resolve(parts []string) *Datum {
	if len(parts) == 0 {
		return nil
	}
	d := p.lookup(parts[0])
	consumed := 1
	if len(parts) > 1 {
		for i := len(p.scopes) - 1; i >= 0; i-- {
			if p.scopes[i].label == parts[0] {
				if v := p.scopes[i].names[parts[1]]; v != nil {
					d = v
					consumed = 2
				}
				break
			}
		}
	}
	if d == nil {
		return nil
	}
	if consumed == len(parts) {
		return d
	}
	if d.Kind != Record {
		return nil
	}
	field := parts[consumed]
	for _, v := range p.function.Datums {
		if v.Kind == RecordField && v.Parent == d.Number && v.Name == field {
			return v
		}
	}
	v := &Datum{Kind: RecordField, Number: len(p.function.Datums), Parent: d.Number, Name: field}
	p.function.Datums = append(p.function.Datums, v)
	return v
}
func (p *parser) variable() (*Datum, token, int) {
	t := p.next()
	parts := []string{t.text}
	for len(parts) < 3 && p.accept(".") {
		parts = append(parts, p.identifier().text)
		if field := p.resolve(parts); field != nil && field.Kind == RecordField {
			break
		}
	}
	d := p.resolve(parts)
	if d == nil {
		p.fail(fmt.Sprintf("%q is not a known variable", strings.Join(parts, ".")))
	}
	t.text = strings.Join(parts, ".")
	return d, t, len(parts)
}
func (p *parser) assignable(d *Datum) {
	if d.Kind == RecordField {
		p.assignable(p.function.Datums[d.Parent])
		return
	}
	if d.Constant {
		p.fail(fmt.Sprintf("variable %q is declared CONSTANT", d.Name))
	}
}
func (p *parser) references(tokens []token) {
	for i := 0; i+2 < len(tokens); i++ {
		if !identifierToken(tokens[i]) || tokens[i+1].text != "." || !identifierToken(tokens[i+2]) {
			continue
		}
		parts := []string{tokens[i].text, tokens[i+2].text}
		if i+4 < len(tokens) && tokens[i+3].text == "." && identifierToken(tokens[i+4]) {
			parts = append(parts, tokens[i+4].text)
			i += 2
		}
		p.resolve(parts)
		i += 2
	}
}
func isTerm(t token, terms []string) bool {
	if t.kind == STRING || t.quoted {
		return false
	}
	for _, term := range terms {
		if t.text == term {
			return true
		}
	}
	return false
}
func (p *parser) readUntil(terms ...string) ([]token, token) {
	var tokens []token
	var delimiters []string
	for {
		t := p.next()
		if len(delimiters) == 0 && isTerm(t, terms) {
			p.references(tokens)
			return tokens, t
		}
		if t.kind == 0 || t.kind == ';' {
			p.fail("missing " + strings.Join(terms, " or ") + " at end of SQL construct")
		}
		if t.kind == '(' || t.kind == '[' {
			delimiters = append(delimiters, t.text)
		}
		if t.kind == ')' || t.kind == ']' {
			if len(delimiters) == 0 || t.text == ")" && delimiters[len(delimiters)-1] != "(" || t.text == "]" && delimiters[len(delimiters)-1] != "[" {
				p.fail("mismatched parentheses")
			}
			delimiters = delimiters[:len(delimiters)-1]
		}
		tokens = append(tokens, t)
	}
}
func (p *parser) text(tokens []token) string {
	if len(tokens) == 0 {
		p.fail("missing expression")
	}
	return p.scan.source[tokens[0].start:tokens[len(tokens)-1].end]
}
func (p *parser) checkedExpression(query string, mode int) *Expression {
	var tree *pg_query.ParseResult
	var err error
	if p.parseSQL != nil {
		tree, err = p.parseSQL(query, mode)
		if err != nil {
			p.fail(err.Error())
		}
	}
	return &Expression{Query: query, ParseMode: mode, SQL: tree}
}
func (p *parser) expression(mode int, until string) *Expression {
	tokens, _ := p.readUntil(until)
	return p.checkedExpression(p.text(tokens), mode)
}
func (p *parser) expressionUntil(mode int, terms ...string) (*Expression, token) {
	tokens, term := p.readUntil(terms...)
	return p.checkedExpression(p.text(tokens), mode), term
}
func (p *parser) integer(t token) int {
	value, err := strconv.Atoi(strings.ReplaceAll(t.text, "_", ""))
	if err != nil || value < 0 {
		p.fail("expected nonnegative integer")
	}
	return value
}
func (p *parser) directive(name token) {
	switch name.text {
	case "dump":
		return
	case "print_strict_params":
		t := p.next()
		if t.text != "on" && t.text != "off" {
			p.fail("unrecognized print_strict_params option")
		}
	case "variable_conflict":
		t := p.next()
		if t.text != "error" && t.text != "use_variable" && t.text != "use_column" {
			p.fail("unrecognized variable_conflict option")
		}
	default:
		p.fail("unrecognized PL/pgSQL compiler directive")
	}
}
