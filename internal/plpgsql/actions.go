package plpgsql

import (
	"fmt"
	"strings"
)

func (p *parser) declaration(name token) {
	if _, exists := p.currentScope().names[name.text]; exists {
		p.fail(fmt.Sprintf("duplicate declaration of variable %q", name.text))
	}
	if p.accept("alias") {
		p.expect("for")
		d, _, _ := p.variable()
		p.expect(";")
		p.currentScope().names[name.text] = d
		return
	}
	options := 256
	if p.accept("no") {
		p.expect("scroll")
		options |= 4
	} else if p.accept("scroll") {
		options |= 2
	}
	if p.accept("cursor") {
		p.pushScope(name.text, false)
		argrow := -1
		if open := p.next(); open.text == "(" {
			var args []*Datum
			for {
				arg := p.identifier()
				tokens, term := p.readUntil(",", ")")
				args = append(args, p.addDatum(arg.text, p.datatype(tokens), arg.line))
				if term.text == ")" {
					break
				}
			}
			argrow = p.makeRow(args, open.line).Number
		} else {
			p.unread(open)
		}
		if !p.accept("for") {
			p.expect("is")
		}
		query := p.expression(0, ";")
		p.scopes = p.scopes[:len(p.scopes)-1]
		d := p.addDatum(name.text, "refcursor", name.line)
		d.CursorQuery, d.CursorArgRow, d.CursorOptions = query, argrow, options
		return
	}
	if options != 256 {
		p.fail("expected CURSOR after SCROLL")
	}
	constant := p.accept("constant")
	tokens, term := p.readUntil("collate", "not", "default", ":=", "=", ";")
	typ := p.datatype(tokens)
	if term.text == "collate" {
		coll := p.identifier()
		_ = coll
		for p.accept(".") {
			p.identifier()
		}
		if typ != "text" && typ != "varchar" && typ != "bpchar" && typ != "name" && !strings.HasPrefix(typ, "_") {
			p.fail("collations are not supported by type " + typ)
		}
		term = p.next()
	}
	notnull := term.text == "not"
	if notnull {
		p.expect("null")
		term = p.next()
	}
	var initial *Expression
	switch term.text {
	case "default", ":=", "=":
		initial = p.expression(2, ";")
	case ";":
	default:
		p.fail("expected default value or semicolon")
	}
	if notnull && initial == nil {
		p.fail(fmt.Sprintf("variable %q must have a default value, since it's declared NOT NULL", name.text))
	}
	d := p.addDatum(name.text, typ, name.line)
	d.Constant, d.NotNull, d.Default = constant, notnull, initial
}

func (p *parser) datatype(tokens []token) string {
	if len(tokens) == 0 {
		p.fail("missing data type declaration")
	}
	if len(tokens) >= 3 && tokens[len(tokens)-2].text == "%" {
		var parts []string
		for i := 0; i < len(tokens)-2; i++ {
			if tokens[i].text != "." {
				parts = append(parts, tokens[i].text)
			}
		}
		if tokens[len(tokens)-1].text == "type" {
			return strings.Join(parts, ".") + "%TYPE"
		}
		return strings.Join(parts, ".") + "%rowtype"
	}
	query := "SELECT NULL::" + p.text(tokens)
	if p.parseSQL == nil {
		p.fail("SQL parser is required for data type declarations")
	}
	tree, err := p.parseSQL(query, 0)
	if err != nil {
		p.fail(err.Error())
	}
	if len(tree.Stmts) != 1 {
		p.fail("invalid data type declaration")
	}
	selectStmt := tree.Stmts[0].Stmt.GetSelectStmt()
	if selectStmt == nil || len(selectStmt.TargetList) != 1 {
		p.fail("invalid data type declaration")
	}
	cast := selectStmt.TargetList[0].GetResTarget().GetVal().GetTypeCast()
	if cast == nil || cast.TypeName == nil || len(cast.TypeName.Names) == 0 {
		p.fail("invalid data type declaration")
	}
	typeName := cast.TypeName.Names[len(cast.TypeName.Names)-1].GetString_().GetSval()
	if len(cast.TypeName.ArrayBounds) > 0 {
		typeName = "_" + typeName
	}
	return catalogType(typeName)
}

func (p *parser) exceptionVariables(at token) {
	for _, name := range []string{"sqlstate", "sqlerrm"} {
		d := &Datum{Kind: Scalar, Number: len(p.function.Datums), Name: name, Type: "text", Line: at.line, Constant: true}
		p.function.Datums = append(p.function.Datums, d)
		p.currentScope().names[name] = d
	}
}
func (p *parser) condition() string {
	t := p.next()
	if t.text == "sqlstate" && !t.quoted {
		t = p.next()
		if t.kind != STRING || len(t.text) != 5 {
			p.fail("invalid SQLSTATE code")
		}
		for _, c := range t.text {
			if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z') {
				p.fail("invalid SQLSTATE code")
			}
		}
		return t.text
	}
	if conditionCounts[t.text] == 0 {
		p.fail(fmt.Sprintf("unrecognized exception condition %q", t.text))
	}
	return t.text
}
func (p *parser) conditions() []string {
	var conditions []string
	for {
		name := p.condition()
		count := conditionCounts[name]
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			conditions = append(conditions, name)
		}
		if !p.accept("or") {
			break
		}
	}
	p.expect("then")
	return conditions
}
func (p *parser) caseExpression() *Expression {
	if p.peek().text == "when" {
		return nil
	}
	tokens, when := p.readUntil("when")
	p.unread(when)
	return p.checkedExpression(p.text(tokens), 2)
}
func (p *parser) caseStatement(at token, expr *Expression, whens []When, otherwise caseElse) *Case {
	s := &Case{Base: Base{at.line}, Test: expr, Whens: whens, HaveElse: otherwise.present, Else: otherwise.body}
	if expr != nil {
		name := fmt.Sprintf("__Case__Variable_%d__", len(p.function.Datums))
		s.Varno = p.addDatum(name, "int4", at.line).Number
		for i := range s.Whens {
			s.Whens[i].Expr = p.checkedExpression("\""+name+"\" IN ("+s.Whens[i].Expr.Query+")", 2)
		}
	}
	return s
}
func (p *parser) loopTargets() ([]string, []*Datum, int) {
	var names []string
	var datums []*Datum
	line := p.peek().line
	for {
		t := p.next()
		parts := []string{t.text}
		for p.accept(".") {
			parts = append(parts, p.identifier().text)
		}
		names = append(names, strings.Join(parts, "."))
		datums = append(datums, p.resolve(parts))
		if !p.accept(",") {
			break
		}
	}
	return names, datums, line
}
func (p *parser) rowTarget(names []string, datums []*Datum, line int) *Datum {
	for _, d := range datums {
		if d == nil {
			p.fail("loop variable of loop over rows must be a record variable or list of scalar variables")
		}
		p.assignable(d)
	}
	if len(datums) == 1 && (datums[0].Kind == Record || datums[0].Kind == Row) {
		return datums[0]
	}
	for _, d := range datums {
		if d.Kind == Record || d.Kind == Row {
			p.fail("record variable cannot be part of multiple-item INTO list")
		}
	}
	row := p.makeRow(datums, line)
	for i, name := range names {
		row.Fields[i].Name = name
	}
	return row
}
func (p *parser) forHeader() *Loop {
	p.currentScope().loop = true
	names, targets, line := p.loopTargets()
	p.expect("in")
	if p.accept("execute") {
		query, term := p.expressionUntil(2, "loop", "using")
		s := &Loop{Kind: "dynfors", Query: query, Var: p.rowTarget(names, targets, line)}
		if term.text == "using" {
			s.Params = p.parameters("loop")
		}
		return s
	}
	t := p.peek()
	if cursor := p.lookup(t.text); cursor != nil && cursor.Type == "refcursor" {
		p.next()
		if len(names) != 1 {
			p.fail("cursor FOR loop must have only one target variable")
		}
		if cursor.CursorQuery == nil {
			p.fail("cursor FOR loop must use a bound cursor variable")
		}
		args := p.cursorArgs(cursor, "loop")
		return &Loop{Kind: "forc", Curvar: cursor.Number, Argquery: args, Var: p.addDatum(names[0], "record", line)}
	}
	reverse := p.accept("reverse")
	tokens, term := p.readUntil("..", "loop")
	if term.text == "loop" {
		if reverse {
			p.fail("cannot specify REVERSE in query FOR loop")
		}
		return &Loop{Kind: "fors", Query: p.checkedExpression(p.text(tokens), 0), Var: p.rowTarget(names, targets, line)}
	}
	if len(names) != 1 {
		p.fail("integer FOR loop must have only one target variable")
	}
	lower := p.checkedExpression(p.text(tokens), 2)
	upper, term := p.expressionUntil(2, "loop", "by")
	var step *Expression
	if term.text == "by" {
		step = p.expression(2, "loop")
	}
	return &Loop{Kind: "fori", Var: p.addDatum(names[0], "int4", line), Lower: lower, Upper: upper, Step: step, Reverse: reverse}
}
func (p *parser) foreachHeader() *Loop {
	p.currentScope().loop = true
	names, targets, line := p.loopTargets()
	slice := 0
	if p.accept("slice") {
		slice = p.integer(p.next())
	}
	p.expect("in")
	p.expect("array")
	expr := p.expression(2, "loop")
	var d *Datum
	if len(targets) == 1 {
		d = targets[0]
		if d == nil {
			p.fail("loop variable of FOREACH must be a known variable")
		}
		p.assignable(d)
	} else {
		d = p.rowTarget(names, targets, line)
	}
	return &Loop{Kind: "foreach_a", Varno: d.Number, Slice: slice, Query: expr}
}
func (p *parser) exitStatement(at token, isExit bool) *Exit {
	s := &Exit{Base: Base{at.line}, IsExit: isExit}
	if t := p.peek(); t.text != "when" && t.text != ";" {
		s.Label = p.identifier().text
	}
	valid := false
	for i := len(p.scopes) - 1; i >= 0; i-- {
		c := p.scopes[i]
		if s.Label != "" && s.Label == c.label {
			if !isExit && !c.loop {
				p.fail("block label cannot be used in CONTINUE")
			}
			valid = true
			break
		}
		if s.Label == "" && c.loop {
			valid = true
			break
		}
	}
	if !valid {
		p.fail("EXIT or CONTINUE does not reference an enclosing loop or block")
	}
	if p.accept("when") {
		s.Cond = p.expression(2, ";")
	} else {
		p.expect(";")
	}
	return s
}
