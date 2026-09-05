package plpgsql

import (
	"fmt"
	"strings"
)

func (p *parser) returnStatement(at token) *Return {
	s := &Return{Base: Base{at.line}, Kind: "return"}
	if p.accept("query") {
		if !p.config.ReturnsSet {
			p.fail("cannot use RETURN QUERY in a non-SETOF function")
		}
		s.Kind = "return_query"
		if p.accept("execute") {
			var term token
			s.Dynquery, term = p.expressionUntil(2, ";", "using")
			if term.text == "using" {
				s.Params = p.parameters(";")
			}
		} else {
			s.Query = p.expression(0, ";")
		}
		return s
	}
	if p.accept("next") {
		if !p.config.ReturnsSet {
			p.fail("cannot use RETURN NEXT in a non-SETOF function")
		}
		s.Kind = "return_next"
	}
	bare := p.hasOutput || s.Kind == "return" && (p.config.ReturnsSet || p.config.ReturnType == "void" || p.config.Procedure || p.config.Inline || p.config.ReturnType == "event_trigger")
	if bare {
		if !p.accept(";") {
			if s.Kind == "return_next" {
				p.fail("RETURN NEXT cannot have a parameter in function with OUT parameters")
			} else if p.config.ReturnsSet {
				p.fail("RETURN cannot have a parameter in function returning set")
			} else if p.config.Procedure {
				p.fail("RETURN cannot have a parameter in a procedure")
			} else if p.config.ReturnType == "void" {
				p.fail("RETURN cannot have a parameter in function returning void")
			} else {
				p.fail("RETURN cannot have a parameter in function with OUT parameters")
			}
		}
		return s
	}
	tokens, _ := p.readUntil(";")
	if len(tokens) > 0 {
		parts := []string{tokens[0].text}
		valid := identifierToken(tokens[0])
		for i := 1; i < len(tokens); i += 2 {
			if i+1 >= len(tokens) || tokens[i].text != "." || !identifierToken(tokens[i+1]) {
				valid = false
				break
			}
			parts = append(parts, tokens[i+1].text)
		}
		if valid {
			if d := p.resolve(parts); d != nil && d.Kind != RecordField {
				return s
			}
		}
	}
	s.Expr = p.checkedExpression(p.text(tokens), 2)
	return s
}

func (p *parser) parameters(until string) []*Expression {
	var params []*Expression
	for {
		expr, term := p.expressionUntil(2, ",", until)
		params = append(params, expr)
		if term.text == until {
			return params
		}
	}
}

func (p *parser) raiseStatement(at token) *Raise {
	s := &Raise{Base: Base{at.line}, Level: 21}
	if p.accept(";") {
		return s
	}
	levels := map[string]int{"exception": 21, "warning": 19, "notice": 18, "info": 17, "log": 15, "debug": 14}
	if level, ok := levels[p.peek().text]; ok {
		p.next()
		s.Level = level
	}
	t := p.peek()
	if t.kind == STRING {
		p.next()
		s.Message = &t.text
		for p.accept(",") {
			expr, term := p.expressionUntil(2, ",", ";", "using")
			s.Params = append(s.Params, expr)
			p.unread(term)
		}
	} else if t.text != "using" {
		s.Condition = p.condition()
	}
	if p.accept("using") {
		kinds := map[string]int{"errcode": 0, "message": 1, "detail": 2, "hint": 3, "column": 4, "constraint": 5, "datatype": 6, "table": 7, "schema": 8}
		for {
			name := p.next()
			kind, ok := kinds[name.text]
			if !ok {
				p.fail("unrecognized RAISE statement option")
			}
			if !p.accept("=") {
				p.expect(":=")
			}
			expr, term := p.expressionUntil(2, ",", ";")
			s.Options = append(s.Options, RaiseOption{Kind: kind, Expr: expr})
			if term.text == ";" {
				break
			}
		}
	} else {
		p.expect(";")
	}
	if s.Message != nil {
		expected := 0
		for i := 0; i < len(*s.Message); i++ {
			if (*s.Message)[i] == '%' {
				if i+1 < len(*s.Message) && (*s.Message)[i+1] == '%' {
					i++
				} else {
					expected++
				}
			}
		}
		if expected < len(s.Params) {
			p.fail("too many parameters specified for RAISE")
		}
		if expected > len(s.Params) {
			p.fail("too few parameters specified for RAISE")
		}
	}
	return s
}
func (p *parser) assertStatement(at token) *Assert {
	cond, term := p.expressionUntil(2, ",", ";")
	s := &Assert{Base: Base{at.line}, Cond: cond}
	if term.text == "," {
		s.Message = p.expression(2, ";")
	}
	return s
}
func (p *parser) performStatement(at token) *Perform {
	tokens, _ := p.readUntil(";")
	query := "SELECT" + p.scan.source[at.end:tokensEnd(p, tokens)]
	return &Perform{Base: Base{at.line}, Expr: p.checkedExpression(query, 0)}
}
func tokensEnd(p *parser, tokens []token) int {
	if len(tokens) == 0 {
		p.fail("missing expression")
	}
	return tokens[len(tokens)-1].end
}
func (p *parser) callStatement(at token, isCall bool) *Call {
	p.unread(at)
	return &Call{Base: Base{at.line}, Expr: p.expression(0, ";"), IsCall: isCall}
}
func (p *parser) transaction(at token, rollback bool) *Transaction {
	chain := false
	if p.accept("and") {
		no := p.accept("no")
		p.expect("chain")
		chain = !no
	}
	p.expect(";")
	return &Transaction{Base: Base{at.line}, Rollback: rollback, Chain: chain}
}
func (p *parser) intoTarget(allowStrict bool) (*Datum, bool) {
	strict := false
	if allowStrict {
		strict = p.accept("strict")
	}
	d, first, _ := p.variable()
	p.assignable(d)
	if d.Kind == Record || d.Kind == Row {
		if p.peek().text == "," {
			p.fail("record variable cannot be part of multiple-item INTO list")
		}
		return d, strict
	}
	if p.peek().text == "[" {
		p.fail("INTO target must not be an array element")
	}
	values := []*Datum{d}
	names := []string{first.text}
	for p.accept(",") {
		v, name, _ := p.variable()
		p.assignable(v)
		if v.Kind == Row || v.Kind == Record || p.peek().text == "[" {
			p.fail("INTO target is not a scalar variable")
		}
		values = append(values, v)
		names = append(names, name.text)
	}
	return p.rowTarget(names, values, first.line), strict
}
func (p *parser) executeStatement(at token) *Execute {
	query, term := p.expressionUntil(2, "into", "using", ";")
	s := &Execute{Base: Base{at.line}, Dynamic: true, Query: query}
	for term.text != ";" {
		switch term.text {
		case "into":
			if s.Into {
				p.fail("INTO specified more than once")
			}
			s.Into = true
			s.Target, s.Strict = p.intoTarget(true)
			term = p.next()
		case "using":
			if len(s.Params) > 0 {
				p.fail("USING specified more than once")
			}
			for {
				var expr *Expression
				expr, term = p.expressionUntil(2, ",", ";", "into")
				s.Params = append(s.Params, expr)
				if term.text != "," {
					break
				}
			}
		default:
			p.fail("expected INTO, USING, or semicolon")
		}
	}
	return s
}
func (p *parser) sqlStatement(at token) Statement {
	p.unread(at)
	first, after := p.next(), p.peek()
	if after.text == ":=" || after.text == "=" || after.text == "[" || after.text == "." {
		p.unread(first)
		d, _, names := p.variable()
		p.assignable(d)
		for p.accept("[") {
			p.subscript()
		}
		for p.accept(".") {
			p.identifier()
			for p.accept("[") {
				p.subscript()
			}
		}
		if !p.accept(":=") {
			p.expect("=")
		}
		tokens, _ := p.readUntil(";")
		expr := p.checkedExpression(p.scan.source[at.start:tokensEnd(p, tokens)], 2+names)
		return &Assign{Base: Base{at.line}, Varno: d.Number, Expr: expr}
	}
	s := &Execute{Base: Base{at.line}}
	depth, blocks := 0, 0
	intoStart, intoEnd := -1, -1
	previous := first
	prefix := first.text
	routine := false
	var observed []token
	for {
		t := p.next()
		if t.kind == 0 {
			p.fail("unexpected end of function definition")
		}
		if t.text == ";" && depth == 0 && blocks == 0 {
			query := p.scan.source[at.start:t.start]
			if intoStart >= 0 {
				query = p.scan.source[at.start:intoStart] + strings.Repeat(" ", intoEnd-intoStart) + p.scan.source[intoEnd:t.start]
			}
			query = strings.TrimRight(query, " \t\r\n\f\v")
			p.references(observed)
			s.Query = p.checkedExpression(query, 0)
			return s
		}
		if len(prefix) < 40 {
			prefix += " " + t.text
			if prefix == "create function" || prefix == "create procedure" || prefix == "create or replace function" || prefix == "create or replace procedure" {
				routine = true
			}
		}
		if t.text == "(" {
			depth++
		} else if t.text == ")" {
			if depth == 0 {
				p.fail("mismatched parentheses")
			}
			depth--
		}
		if routine && depth == 0 {
			if t.text == "begin" || t.text == "case" {
				blocks++
			} else if t.text == "end" && blocks > 0 {
				blocks--
			}
		}
		if t.text == "into" && !t.quoted && previous.text != "insert" && previous.text != "merge" && first.text != "import" {
			if s.Into {
				p.fail("INTO specified more than once")
			}
			s.Into = true
			intoStart = t.start
			p.references(observed)
			s.Target, s.Strict = p.intoTarget(true)
			intoEnd = p.peek().start
		} else {
			observed = append(observed, t)
		}
		previous = t
	}
}
func (p *parser) diagnostics(at token) *GetDiagnostics {
	s := &GetDiagnostics{Base: Base{at.line}, Stacked: p.accept("stacked")}
	if !s.Stacked {
		p.accept("current")
	}
	p.expect("diagnostics")
	for {
		d, _, _ := p.variable()
		p.assignable(d)
		if d.Kind == Row || d.Kind == Record || p.peek().text == "[" {
			p.fail("diagnostics target is not a scalar variable")
		}
		if !p.accept("=") {
			p.expect(":=")
		}
		kind := p.next().text
		switch kind {
		case "row_count", "pg_routine_oid":
			if s.Stacked {
				p.fail("diagnostics item " + kind + " is not allowed in GET STACKED DIAGNOSTICS")
			}
		case "pg_context":
		case "pg_exception_context", "pg_exception_detail", "pg_exception_hint", "returned_sqlstate", "column_name", "constraint_name", "pg_datatype_name", "message_text", "table_name", "schema_name":
			if !s.Stacked {
				p.fail("diagnostics item " + kind + " is not allowed in GET CURRENT DIAGNOSTICS")
			}
		default:
			p.fail(fmt.Sprintf("unrecognized GET DIAGNOSTICS item %q", kind))
		}
		s.Items = append(s.Items, Diagnostic{Kind: strings.ToUpper(kind), Target: d.Number})
		if !p.accept(",") {
			break
		}
	}
	p.expect(";")
	return s
}

func (p *parser) subscript() {
	p.readUntil("]")
}
