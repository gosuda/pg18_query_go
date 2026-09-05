package plpgsql

import (
	"fmt"
	"math"
	"strings"
)

func (p *parser) cursor() *Datum {
	d, _, _ := p.variable()
	if d.Kind != Scalar || p.peek().text == "[" {
		p.fail("cursor variable must be a simple variable")
	}
	if d.Type != "refcursor" {
		p.fail(fmt.Sprintf("variable %q must be of type cursor or refcursor", d.Name))
	}
	return d
}
func (p *parser) cursorArgs(cursor *Datum, until string) *Expression {
	if cursor.CursorArgRow < 0 {
		if p.peek().text == "(" {
			p.fail(fmt.Sprintf("cursor %q has no arguments", cursor.Name))
		}
		p.expect(until)
		return nil
	}
	p.expect("(")
	row := p.function.Datums[cursor.CursorArgRow]
	args := make([]string, len(row.Fields))
	anyNamed := false
	for i := range args {
		position := i
		first := p.next()
		second := p.next()
		if first.kind == WORD && (second.text == ":=" || second.text == "=>") {
			position = -1
			for j, field := range row.Fields {
				if field.Name == first.text {
					position = j
					break
				}
			}
			if position < 0 {
				p.fail(fmt.Sprintf("cursor %q has no argument named %q", cursor.Name, first.text))
			}
			anyNamed = true
		} else {
			p.unread(second)
			p.unread(first)
		}
		if args[position] != "" {
			p.fail("cursor argument specified more than once")
		}
		expr, term := p.expressionUntil(2, ",", ")")
		args[position] = expr.Query
		if term.text == ")" && i != len(args)-1 {
			p.fail("not enough arguments for cursor")
		}
		if term.text == "," && i == len(args)-1 {
			p.fail("too many arguments for cursor")
		}
	}
	if anyNamed {
		for i, field := range row.Fields {
			args[i] += " AS " + quoteIdentifier(field.Name)
		}
	}
	p.expect(until)
	return p.checkedExpression(strings.Join(args, ", "), 2)
}
func quoteIdentifier(name string) string {
	plain := name != ""
	for i, c := range name {
		if !(c >= 'a' && c <= 'z' || c == '_' || i > 0 && c >= '0' && c <= '9') {
			plain = false
			break
		}
	}
	if _, keyword := keywords[name]; keyword {
		plain = false
	}
	if plain {
		return name
	}
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}
func (p *parser) openStatement(at token) *Open {
	cursor := p.cursor()
	s := &Open{Base: Base{at.line}, Curvar: cursor.Number, Options: 256}
	if cursor.CursorQuery != nil {
		s.Args = p.cursorArgs(cursor, ";")
		return s
	}
	if p.accept("no") {
		p.expect("scroll")
		s.Options |= 4
	} else if p.accept("scroll") {
		s.Options |= 2
	}
	p.expect("for")
	if p.accept("execute") {
		var term token
		s.Dynamic, term = p.expressionUntil(2, "using", ";")
		if term.text == "using" {
			s.Params = p.parameters(";")
		}
	} else {
		s.Query = p.expression(0, ";")
	}
	return s
}
func (p *parser) closeStatement(at token) *Close {
	cursor := p.cursor()
	p.expect(";")
	return &Close{Base: Base{at.line}, Curvar: cursor.Number}
}
func (p *parser) fetchStatement(at token, move bool) *Fetch {
	s := &Fetch{Base: Base{at.line}, Count: 1, Move: move}
	checkFrom := true
	t := p.next()
	switch t.text {
	case "next":
	case "prior":
		s.Direction = 1
	case "first":
		s.Direction = 2
	case "last":
		s.Direction, s.Count = 2, -1
	case "absolute", "relative":
		s.Direction = 2
		if t.text == "relative" {
			s.Direction = 3
		}
		s.Expr, _ = p.expressionUntil(2, "from", "in")
		checkFrom = false
	case "all":
		s.Count, s.Multiple = math.MaxInt64, true
	case "forward", "backward":
		if t.text == "backward" {
			s.Direction = 1
		}
		if p.accept("from") || p.accept("in") {
			checkFrom = false
		} else if p.accept("all") {
			s.Count, s.Multiple = math.MaxInt64, true
		} else {
			s.Expr, _ = p.expressionUntil(2, "from", "in")
			s.Multiple = true
			checkFrom = false
		}
	case "from", "in":
		checkFrom = false
	default:
		p.unread(t)
		if p.lookup(t.text) != nil {
			checkFrom = false
		} else {
			s.Expr, _ = p.expressionUntil(2, "from", "in")
			s.Multiple = true
			checkFrom = false
		}
	}
	if checkFrom && !p.accept("from") {
		p.expect("in")
	}
	s.Curvar = p.cursor().Number
	if !move {
		p.expect("into")
		s.Target, _ = p.intoTarget(false)
		if s.Multiple {
			p.fail("FETCH statement cannot return multiple rows")
		}
	}
	p.expect(";")
	return s
}
