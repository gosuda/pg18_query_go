package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

func (w *deparseWriter) xmlOption(option p.XmlOptionType) {
	switch option.String() {
	case "XMLOPTION_DOCUMENT":
		w.text("document ")
	case "XMLOPTION_CONTENT":
		w.text("content ")
	default:
		w.invalid("invalid XML option %v", option)
	}
}

func (w *deparseWriter) xmlAttributes(nodes []*p.Node, namespaces bool) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		target := node.GetResTarget()
		if target == nil || target.Val == nil {
			w.invalid("invalid XML attribute %T", node.GetNode())
			return
		}
		if namespaces && target.Name == "" {
			w.text("DEFAULT ")
		}
		if namespaces {
			w.exprIn(target.Val, deparseExpressionNone)
		} else {
			w.expr(target.Val)
		}
		if target.Name != "" {
			w.text(" AS ")
			w.ident(target.Name)
		}
	}
}

func (w *deparseWriter) xmlExpr(n *p.XmlExpr) { w.xmlExprIn(n, deparseExpressionA) }

func (w *deparseWriter) xmlExprIn(n *p.XmlExpr, context deparseExpressionContext) {
	switch n.Op.String() {
	case "IS_XMLCONCAT":
		w.text("xmlconcat(")
		w.exprs(n.Args)
		w.text(")")
	case "IS_XMLELEMENT":
		w.text("xmlelement(name ")
		w.ident(n.Name)
		if len(n.NamedArgs) > 0 {
			w.text(", xmlattributes(")
			w.xmlAttributes(n.NamedArgs, false)
			w.text(")")
		}
		if len(n.Args) > 0 {
			w.text(", ")
			w.exprs(n.Args)
		}
		w.text(")")
	case "IS_XMLFOREST":
		w.text("xmlforest(")
		w.xmlAttributes(n.NamedArgs, false)
		w.text(")")
	case "IS_XMLPARSE":
		if len(n.Args) != 2 {
			w.invalid("XMLPARSE requires two raw arguments")
			return
		}
		w.text("xmlparse(")
		w.xmlOption(n.Xmloption)
		w.expr(n.Args[0])
		w.text(")")
	case "IS_XMLPI":
		w.text("xmlpi(name ")
		w.ident(n.Name)
		if len(n.Args) > 0 {
			w.text(", ")
			w.expr(n.Args[0])
		}
		w.text(")")
	case "IS_XMLROOT":
		if len(n.Args) != 3 {
			w.invalid("XMLROOT requires three arguments")
			return
		}
		w.text("xmlroot(")
		w.expr(n.Args[0])
		w.text(", version ")
		if n.Args[1].GetAConst().GetIsnull() {
			w.text("no value")
		} else {
			w.expr(n.Args[1])
		}
		switch n.Args[2].GetAConst().GetIval().GetIval() {
		case 0:
			w.text(", standalone yes")
		case 1:
			w.text(", standalone no")
		case 2:
			w.text(", standalone no value")
		}
		w.text(")")
	case "IS_DOCUMENT":
		if len(n.Args) != 1 {
			w.invalid("IS DOCUMENT requires one argument")
			return
		}
		w.exprIn(n.Args[0], context)
		w.text(" IS DOCUMENT")
	default:
		w.invalid("unpermitted XML expression %v", n.Op)
	}
}

func (w *deparseWriter) xmlSerialize(n *p.XmlSerialize) {
	w.text("xmlserialize(")
	w.xmlOption(n.Xmloption)
	w.expr(n.Expr)
	w.text(" AS ")
	w.typeName(n.TypeName)
	if n.Indent {
		w.text(" INDENT")
	}
	w.text(")")
}

func (w *deparseWriter) xmlTable(n *p.RangeTableFunc) {
	if n.Lateral {
		w.text("LATERAL ")
	}
	w.text("xmltable(")
	if len(n.Namespaces) > 0 {
		w.text("xmlnamespaces(")
		w.xmlAttributes(n.Namespaces, true)
		w.text("), ")
	}
	w.text("(")
	w.exprIn(n.Rowexpr, deparseExpressionNone)
	w.text(") PASSING ")
	w.exprIn(n.Docexpr, deparseExpressionNone)
	w.text(" COLUMNS ")
	for i, node := range n.Columns {
		if i > 0 {
			w.text(", ")
		}
		c := node.GetRangeTableFuncCol()
		if c == nil {
			w.invalid("invalid XMLTABLE column %T", node.GetNode())
			return
		}
		w.ident(c.Colname)
		if c.ForOrdinality {
			w.text(" FOR ORDINALITY")
			continue
		}
		w.text(" ")
		w.typeName(c.TypeName)
		if c.Colexpr != nil {
			w.text(" PATH ")
			w.exprIn(c.Colexpr, deparseExpressionNone)
		}
		if c.Coldefexpr != nil {
			w.text(" DEFAULT ")
			w.exprIn(c.Coldefexpr, deparseExpressionNone)
		}
		if c.IsNotNull {
			w.text(" NOT NULL")
		}
	}
	w.text(")")
	if n.Alias != nil {
		w.text(" AS ")
		w.alias(n.Alias)
	}
}
