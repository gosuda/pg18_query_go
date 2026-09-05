package plpgsql

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type jsonWriter struct{ bytes.Buffer }
type jsonObject struct {
	writer *jsonWriter
	first  bool
}

func (w *jsonWriter) string(value string) {
	encoder := json.NewEncoder(&w.Buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	w.Truncate(w.Len() - 1)
}
func (w *jsonWriter) object(kind string) jsonObject {
	w.WriteByte('{')
	w.string("PLpgSQL_" + kind)
	w.WriteString(":{")
	return jsonObject{writer: w, first: true}
}
func (o *jsonObject) field(name string) {
	if !o.first {
		o.writer.WriteByte(',')
	}
	o.first = false
	o.writer.string(name)
	o.writer.WriteByte(':')
}
func (o *jsonObject) end() { o.writer.WriteString("}}") }
func (o *jsonObject) integer(name string, value int) {
	if value != 0 {
		o.field(name)
		o.writer.WriteString(strconv.Itoa(value))
	}
}
func (o *jsonObject) enum(name string, value int) {
	o.field(name)
	o.writer.WriteString(strconv.Itoa(value))
}
func (o *jsonObject) boolean(name string, value bool) {
	if value {
		o.field(name)
		o.writer.WriteString("true")
	}
}
func (o *jsonObject) text(name string, value string) {
	if value != "" {
		o.field(name)
		o.writer.string(value)
	}
}
func (o *jsonObject) expr(name string, value *Expression) {
	if value == nil {
		return
	}
	o.field(name)
	o.writer.expression(value)
}
func (o *jsonObject) expressions(name string, values []*Expression) {
	if len(values) == 0 {
		return
	}
	o.field(name)
	o.writer.WriteByte('[')
	for i, expr := range values {
		if i > 0 {
			o.writer.WriteByte(',')
		}
		o.writer.expression(expr)
	}
	o.writer.WriteByte(']')
}
func (o *jsonObject) statements(name string, values []Statement) {
	if len(values) == 0 {
		return
	}
	o.field(name)
	o.writer.WriteByte('[')
	for i, stmt := range values {
		if i > 0 {
			o.writer.WriteByte(',')
		}
		o.writer.statement(stmt)
	}
	o.writer.WriteByte(']')
}
func (o *jsonObject) datum(name string, value *Datum) {
	if value != nil {
		o.field(name)
		o.writer.datum(value)
	}
}
func (w *jsonWriter) expression(expr *Expression) {
	o := w.object("expr")
	o.field("query")
	w.string(expr.Query)
	o.enum("parseMode", expr.ParseMode)
	o.end()
}
func (w *jsonWriter) datum(d *Datum) {
	switch d.Kind {
	case Scalar, Promise:
		o := w.object("var")
		o.text("refname", d.Name)
		o.integer("lineno", d.Line)
		o.field("datatype")
		t := w.object("type")
		t.text("typname", d.Type)
		t.end()
		o.boolean("isconst", d.Constant)
		o.boolean("notnull", d.NotNull)
		o.expr("default_val", d.Default)
		o.expr("cursor_explicit_expr", d.CursorQuery)
		o.integer("cursor_explicit_argrow", d.CursorArgRow)
		o.integer("cursor_options", d.CursorOptions)
		o.end()
	case Record:
		o := w.object("rec")
		o.text("refname", d.Name)
		o.integer("dno", d.Number)
		o.integer("lineno", d.Line)
		o.end()
	case RecordField:
		o := w.object("recfield")
		o.text("fieldname", d.Name)
		o.integer("recparentno", d.Parent)
		o.end()
	case Row:
		o := w.object("row")
		o.text("refname", d.Name)
		o.integer("lineno", d.Line)
		o.field("fields")
		w.WriteByte('[')
		for i, field := range d.Fields {
			if i > 0 {
				w.WriteByte(',')
			}
			if field.Name == "" {
				w.WriteString("null")
				continue
			}
			w.WriteString("{\"name\":")
			w.string(field.Name)
			if field.Varno != 0 {
				w.WriteString(",\"varno\":")
				w.WriteString(strconv.Itoa(field.Varno))
			}
			w.WriteByte('}')
		}
		w.WriteByte(']')
		o.end()
	}
}
func (w *jsonWriter) block(b *Block) {
	o := w.object("stmt_block")
	o.integer("lineno", b.Line)
	o.text("label", b.Label)
	o.statements("body", b.Body)
	if len(b.Exceptions) > 0 {
		o.field("exceptions")
		e := w.object("exception_block")
		e.field("exc_list")
		w.WriteByte('[')
		for i, handler := range b.Exceptions {
			if i > 0 {
				w.WriteByte(',')
			}
			h := w.object("exception")
			h.field("conditions")
			w.WriteByte('[')
			for j, name := range handler.Conditions {
				if j > 0 {
					w.WriteByte(',')
				}
				c := w.object("condition")
				c.text("condname", name)
				c.end()
			}
			w.WriteByte(']')
			h.statements("action", handler.Action)
			h.end()
		}
		w.WriteByte(']')
		e.end()
	}
	o.end()
}
func (w *jsonWriter) statement(statement Statement) {
	switch s := statement.(type) {
	case *Block:
		w.block(s)
	case *Assign:
		o := w.object("stmt_assign")
		o.integer("lineno", s.Line)
		o.integer("varno", s.Varno)
		o.expr("expr", s.Expr)
		o.end()
	case *If:
		o := w.object("stmt_if")
		o.integer("lineno", s.Line)
		o.expr("cond", s.Cond)
		o.statements("then_body", s.Then)
		if len(s.Elsifs) > 0 {
			o.field("elsif_list")
			w.WriteByte('[')
			for i, branch := range s.Elsifs {
				if i > 0 {
					w.WriteByte(',')
				}
				e := w.object("if_elsif")
				e.integer("lineno", branch.Line)
				e.expr("cond", branch.Cond)
				e.statements("stmts", branch.Body)
				e.end()
			}
			w.WriteByte(']')
		}
		o.statements("else_body", s.Else)
		o.end()
	case *Case:
		o := w.object("stmt_case")
		o.integer("lineno", s.Line)
		o.expr("t_expr", s.Test)
		o.integer("t_varno", s.Varno)
		if len(s.Whens) > 0 {
			o.field("case_when_list")
			w.WriteByte('[')
			for i, branch := range s.Whens {
				if i > 0 {
					w.WriteByte(',')
				}
				e := w.object("case_when")
				e.integer("lineno", branch.Line)
				e.expr("expr", branch.Expr)
				e.statements("stmts", branch.Body)
				e.end()
			}
			w.WriteByte(']')
		}
		o.boolean("have_else", s.HaveElse)
		o.statements("else_stmts", s.Else)
		o.end()
	case *Loop:
		o := w.object("stmt_" + s.Kind)
		o.integer("lineno", s.Line)
		o.text("label", s.Label)
		switch s.Kind {
		case "while":
			o.expr("cond", s.Cond)
			o.statements("body", s.Body)
		case "fori":
			o.datum("var", s.Var)
			o.expr("lower", s.Lower)
			o.expr("upper", s.Upper)
			o.expr("step", s.Step)
			o.boolean("reverse", s.Reverse)
			o.statements("body", s.Body)
		case "fors", "dynfors":
			o.datum("var", s.Var)
			o.statements("body", s.Body)
			o.expr("query", s.Query)
			o.expressions("params", s.Params)
		case "forc":
			o.datum("var", s.Var)
			o.statements("body", s.Body)
			o.integer("curvar", s.Curvar)
			o.expr("argquery", s.Argquery)
		case "foreach_a":
			o.integer("varno", s.Varno)
			o.integer("slice", s.Slice)
			o.expr("expr", s.Query)
			o.statements("body", s.Body)
		default:
			o.statements("body", s.Body)
		}
		o.end()
	case *Exit:
		o := w.object("stmt_exit")
		o.integer("lineno", s.Line)
		o.boolean("is_exit", s.IsExit)
		o.text("label", s.Label)
		o.expr("cond", s.Cond)
		o.end()
	case *Return:
		o := w.object("stmt_" + s.Kind)
		o.integer("lineno", s.Line)
		if s.Kind == "return_query" {
			o.expr("query", s.Query)
			o.expr("dynquery", s.Dynquery)
			o.expressions("params", s.Params)
		} else {
			o.expr("expr", s.Expr)
		}
		o.end()
	case *Raise:
		o := w.object("stmt_raise")
		o.integer("lineno", s.Line)
		o.integer("elog_level", s.Level)
		o.text("condname", s.Condition)
		if s.Message != nil {
			o.field("message")
			w.string(*s.Message)
		}
		o.expressions("params", s.Params)
		if len(s.Options) > 0 {
			o.field("options")
			w.WriteByte('[')
			for i, option := range s.Options {
				if i > 0 {
					w.WriteByte(',')
				}
				r := w.object("raise_option")
				r.enum("opt_type", option.Kind)
				r.expr("expr", option.Expr)
				r.end()
			}
			w.WriteByte(']')
		}
		o.end()
	case *Assert:
		o := w.object("stmt_assert")
		o.integer("lineno", s.Line)
		o.expr("cond", s.Cond)
		o.expr("message", s.Message)
		o.end()
	case *Execute:
		kind := "execsql"
		if s.Dynamic {
			kind = "dynexecute"
		}
		o := w.object("stmt_" + kind)
		o.integer("lineno", s.Line)
		if s.Dynamic {
			o.expr("query", s.Query)
		} else {
			o.expr("sqlstmt", s.Query)
		}
		o.boolean("into", s.Into)
		o.boolean("strict", s.Strict)
		o.datum("target", s.Target)
		o.expressions("params", s.Params)
		o.end()
	case *GetDiagnostics:
		o := w.object("stmt_getdiag")
		o.integer("lineno", s.Line)
		o.boolean("is_stacked", s.Stacked)
		if len(s.Items) > 0 {
			o.field("diag_items")
			w.WriteByte('[')
			for i, item := range s.Items {
				if i > 0 {
					w.WriteByte(',')
				}
				d := w.object("diag_item")
				d.text("kind", item.Kind)
				d.integer("target", item.Target)
				d.end()
			}
			w.WriteByte(']')
		}
		o.end()
	case *Open:
		o := w.object("stmt_open")
		o.integer("lineno", s.Line)
		o.integer("curvar", s.Curvar)
		o.integer("cursor_options", s.Options)
		o.expr("argquery", s.Args)
		o.expr("query", s.Query)
		o.expr("dynquery", s.Dynamic)
		o.expressions("params", s.Params)
		o.end()
	case *Fetch:
		o := w.object("stmt_fetch")
		o.integer("lineno", s.Line)
		o.datum("target", s.Target)
		o.integer("curvar", s.Curvar)
		o.enum("direction", s.Direction)
		if s.Count != 0 {
			o.field("how_many")
			w.WriteString(strconv.FormatInt(s.Count, 10))
		}
		o.expr("expr", s.Expr)
		o.boolean("is_move", s.Move)
		o.boolean("returns_multiple_rows", s.Multiple)
		o.end()
	case *Close:
		o := w.object("stmt_close")
		o.integer("lineno", s.Line)
		o.integer("curvar", s.Curvar)
		o.end()
	case *Perform:
		o := w.object("stmt_perform")
		o.integer("lineno", s.Line)
		o.expr("expr", s.Expr)
		o.end()
	case *Call:
		o := w.object("stmt_call")
		o.integer("lineno", s.Line)
		o.expr("expr", s.Expr)
		o.boolean("is_call", s.IsCall)
		o.datum("target", s.Target)
		o.end()
	case *Transaction:
		kind := "commit"
		if s.Rollback {
			kind = "rollback"
		}
		o := w.object("stmt_" + kind)
		o.integer("lineno", s.Line)
		o.boolean("chain", s.Chain)
		o.end()
	default:
		panic("unknown PL/pgSQL statement")
	}
}

// ToJSON renders the upstream PL/pgSQL wire contract after compilation.
func ToJSON(functions []*Function) string {
	if len(functions) == 0 {
		return "[]"
	}
	w := &jsonWriter{}
	w.WriteString("[\n")
	for i, f := range functions {
		if i > 0 {
			w.WriteString(",\n")
		}
		o := w.object("function")
		o.integer("new_varno", f.NewVarno)
		o.integer("old_varno", f.OldVarno)
		o.field("datums")
		w.WriteByte('[')
		for j, d := range f.Datums {
			if j > 0 {
				w.WriteByte(',')
			}
			w.datum(d)
		}
		w.WriteByte(']')
		if f.Action != nil {
			o.field("action")
			w.block(f.Action)
		}
		o.end()
	}
	w.WriteString("\n]")
	return w.String()
}
