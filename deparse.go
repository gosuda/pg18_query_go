package pg_query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gosuda/pg18_query_go/internal/neoparse"
	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

// Deparse reconstructs SQL from a PostgreSQL raw parse tree.
func Deparse(tree *ParseResult) (string, error) {
	if tree == nil {
		return "", fmt.Errorf("deparse: nil parse result")
	}
	var w deparseWriter
	for i, raw := range tree.Stmts {
		if raw == nil || raw.Stmt == nil {
			return "", fmt.Errorf("deparse: statement %d is nil", i)
		}
		if i != 0 {
			w.text("; ")
		}
		w.stmt(raw.Stmt)
		if w.err != nil {
			return "", w.err
		}
	}
	return string(w.buf), nil
}

type deparseWriter struct {
	buf []byte
	err error
}

func (w *deparseWriter) text(s string)   { w.buf = append(w.buf, s...) }
func (w *deparseWriter) integer(n int64) { w.buf = strconv.AppendInt(w.buf, n, 10) }
func (w *deparseWriter) trimSpace() {
	if len(w.buf) > 0 && w.buf[len(w.buf)-1] == ' ' {
		w.buf = w.buf[:len(w.buf)-1]
	}
}
func (w *deparseWriter) invalid(format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf("deparse: "+format, args...)
	}
}

func (w *deparseWriter) ident(name string) {
	safe := len(name) > 0 && (name[0] >= 'a' && name[0] <= 'z' || name[0] == '_')
	for i := 1; safe && i < len(name); i++ {
		c := name[i]
		safe = c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_'
	}
	if keyword, ok := neoparse.ScanKeywords[name]; ok && keyword.KeywordKind != neoparse.KeywordKind_UNRESERVED_KEYWORD {
		safe = false
	}
	if safe {
		w.text(name)
		return
	}
	w.text("\"")
	for i := range len(name) {
		c := name[i]
		if c == '"' {
			w.text("\"")
		}
		w.buf = append(w.buf, c)
	}
	w.text("\"")
}

func (w *deparseWriter) literal(value string) {
	if strings.ContainsRune(value, '\\') {
		w.text("E")
	}
	w.text("'")
	for i := range len(value) {
		c := value[i]
		if c == '\'' || c == '\\' {
			w.buf = append(w.buf, c)
		}
		w.buf = append(w.buf, c)
	}
	w.text("'")
}

func deparseString(node *p.Node) string { return node.GetString_().GetSval() }

func (w *deparseWriter) name(parts []*p.Node) {
	for i, part := range parts {
		if i > 0 {
			w.text(".")
		}
		if part.GetString_() == nil {
			w.invalid("invalid name component %T", part.GetNode())
			return
		}
		w.ident(deparseString(part))
	}
}

func (w *deparseWriter) names(parts []*p.Node) {
	for i, part := range parts {
		if i > 0 {
			w.text(", ")
		}
		w.ident(deparseString(part))
	}
}

func (w *deparseWriter) exprs(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.expr(node)
	}
}

func (w *deparseWriter) parenthesized(node *p.Node, parens bool) {
	if parens {
		w.text("(")
	}
	w.expr(node)
	if parens {
		w.text(")")
	}
}

func deparseNeedsBParens(node *p.Node) bool {
	return node != nil && (node.GetBoolExpr() != nil || node.GetBooleanTest() != nil || node.GetNullTest() != nil || node.GetAExpr() != nil)
}

func (w *deparseWriter) qualifiedOperator(parts []*p.Node) {
	if len(parts) == 0 {
		w.invalid("operator has no name")
		return
	}
	if len(parts) == 1 {
		w.text(deparseString(parts[0]))
		return
	}
	w.text("OPERATOR(")
	w.operatorName(parts)
	w.text(")")
}

func (w *deparseWriter) operatorName(parts []*p.Node) {
	if len(parts) == 0 {
		w.invalid("operator has no name")
		return
	}
	if len(parts) > 1 {
		w.name(parts[:len(parts)-1])
		w.text(".")
	}
	w.text(deparseString(parts[len(parts)-1]))
}

func (w *deparseWriter) subqueryOperator(parts []*p.Node) {
	if len(parts) == 1 {
		switch deparseString(parts[0]) {
		case "~~":
			w.text("LIKE")
			return
		case "!~~":
			w.text("NOT LIKE")
			return
		case "~~*":
			w.text("ILIKE")
			return
		case "!~~*":
			w.text("NOT ILIKE")
			return
		}
	}
	w.qualifiedOperator(parts)
}

func (w *deparseWriter) constant(c *p.A_Const) {
	if c.Isnull {
		w.text("NULL")
		return
	}
	switch v := c.Val.(type) {
	case *p.A_Const_Ival:
		w.integer(int64(v.Ival.Ival))
	case *p.A_Const_Fval:
		w.text(v.Fval.Fval)
	case *p.A_Const_Boolval:
		if v.Boolval.Boolval {
			w.text("true")
		} else {
			w.text("false")
		}
	case *p.A_Const_Sval:
		w.literal(v.Sval.Sval)
	case *p.A_Const_Bsval:
		value := v.Bsval.Bsval
		if len(value) == 0 {
			w.invalid("empty bit string")
			return
		}
		w.text(value[:1])
		w.literal(value[1:])
	default:
		w.invalid("constant has no value")
	}
}
