package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

func (w *deparseWriter) jsonFormat(n *p.JsonFormat) {
	if n == nil || n.FormatType.String() == "JS_FORMAT_DEFAULT" {
		return
	}
	w.text("FORMAT JSON ")
	switch n.Encoding.String() {
	case "JS_ENC_UTF8":
		w.text("ENCODING utf8 ")
	case "JS_ENC_UTF16":
		w.text("ENCODING utf16 ")
	case "JS_ENC_UTF32":
		w.text("ENCODING utf32 ")
	}
}

func (w *deparseWriter) jsonValue(n *p.JsonValueExpr) {
	if n == nil {
		w.invalid("nil JSON value expression")
		return
	}
	w.expr(n.RawExpr)
	w.text(" ")
	w.jsonFormat(n.Format)
}

func (w *deparseWriter) jsonOutput(n *p.JsonOutput) {
	if n == nil {
		return
	}
	if n.Returning == nil {
		w.invalid("JSON output has no returning specification")
		return
	}
	w.text("RETURNING ")
	w.typeName(n.TypeName)
	w.text(" ")
	w.jsonFormat(n.Returning.Format)
}

func (w *deparseWriter) jsonKeyValue(n *p.JsonKeyValue) {
	if n == nil {
		w.invalid("nil JSON key/value")
		return
	}
	w.expr(n.Key)
	w.text(": ")
	w.jsonValue(n.Value)
}

func (w *deparseWriter) jsonAggregateSuffix(n *p.JsonAggConstructor) {
	if n.AggFilter != nil {
		w.text(" FILTER (WHERE ")
		w.expr(n.AggFilter)
		w.text(")")
	}
	if n.Over != nil {
		w.text(" OVER ")
		if n.Over.Name != "" {
			w.text(n.Over.Name)
		} else {
			w.window(n.Over)
		}
	}
}

func (w *deparseWriter) jsonExpr(node *p.Node) {
	switch v := node.Node.(type) {
	case *p.Node_JsonIsPredicate:
		n := v.JsonIsPredicate
		w.expr(n.Expr)
		w.text(" ")
		w.jsonFormat(n.Format)
		w.text("IS JSON ")
		switch n.ItemType.String() {
		case "JS_TYPE_ARRAY":
			w.text("ARRAY ")
		case "JS_TYPE_OBJECT":
			w.text("OBJECT ")
		case "JS_TYPE_SCALAR":
			w.text("SCALAR ")
		}
		if n.UniqueKeys {
			w.text("WITH UNIQUE ")
		}
		w.trimSpace()
	case *p.Node_JsonValueExpr:
		w.jsonValue(v.JsonValueExpr)
		w.trimSpace()
	case *p.Node_JsonObjectAgg:
		n := v.JsonObjectAgg
		if n.Constructor == nil {
			w.invalid("JSON_OBJECTAGG has no constructor")
			return
		}
		w.text("JSON_OBJECTAGG(")
		w.jsonKeyValue(n.Arg)
		if n.AbsentOnNull {
			w.text("ABSENT ON NULL ")
		}
		if n.Unique {
			w.text("WITH UNIQUE ")
		}
		w.jsonOutput(n.Constructor.Output)
		w.trimSpace()
		w.text(")")
		w.jsonAggregateSuffix(n.Constructor)
	case *p.Node_JsonArrayAgg:
		n := v.JsonArrayAgg
		if n.Constructor == nil {
			w.invalid("JSON_ARRAYAGG has no constructor")
			return
		}
		w.text("JSON_ARRAYAGG(")
		w.jsonValue(n.Arg)
		if len(n.Constructor.AggOrder) > 0 {
			w.text("ORDER BY ")
			w.sorts(n.Constructor.AggOrder)
			w.text(" ")
		}
		if !n.AbsentOnNull {
			w.text("NULL ON NULL ")
		}
		w.jsonOutput(n.Constructor.Output)
		w.trimSpace()
		w.text(")")
		w.jsonAggregateSuffix(n.Constructor)
	case *p.Node_JsonObjectConstructor:
		n := v.JsonObjectConstructor
		w.text("JSON_OBJECT(")
		for i, arg := range n.Exprs {
			if i > 0 {
				w.text(", ")
			}
			w.jsonKeyValue(arg.GetJsonKeyValue())
			w.trimSpace()
		}
		w.text(" ")
		if n.AbsentOnNull {
			w.text("ABSENT ON NULL ")
		}
		if n.Unique {
			w.text("WITH UNIQUE ")
		}
		w.jsonOutput(n.Output)
		w.trimSpace()
		w.text(")")
	case *p.Node_JsonArrayConstructor:
		n := v.JsonArrayConstructor
		w.text("JSON_ARRAY(")
		for i, arg := range n.Exprs {
			if i > 0 {
				w.text(", ")
			}
			w.jsonValue(arg.GetJsonValueExpr())
			w.trimSpace()
		}
		w.text(" ")
		if !n.AbsentOnNull {
			w.text("NULL ON NULL ")
		}
		w.jsonOutput(n.Output)
		w.trimSpace()
		w.text(")")
	case *p.Node_JsonArrayQueryConstructor:
		n := v.JsonArrayQueryConstructor
		w.text("JSON_ARRAY(")
		w.stmt(n.Query)
		w.text(" ")
		w.jsonFormat(n.Format)
		w.jsonOutput(n.Output)
		w.trimSpace()
		w.text(")")
	case *p.Node_JsonParseExpr:
		n := v.JsonParseExpr
		w.text("JSON(")
		w.jsonValue(n.Expr)
		if n.UniqueKeys {
			w.text(" WITH UNIQUE KEYS")
		}
		w.text(")")
	case *p.Node_JsonScalarExpr:
		w.text("JSON_SCALAR(")
		w.expr(v.JsonScalarExpr.Expr)
		w.text(")")
	case *p.Node_JsonSerializeExpr:
		n := v.JsonSerializeExpr
		w.text("JSON_SERIALIZE(")
		w.jsonValue(n.Expr)
		w.jsonOutput(n.Output)
		w.text(")")
	case *p.Node_JsonFuncExpr:
		w.jsonFunction(v.JsonFuncExpr)
	default:
		w.invalid("unpermitted JSON expression %T", node.Node)
	}
}

func (w *deparseWriter) jsonPassing(nodes []*p.Node) {
	if len(nodes) == 0 {
		return
	}
	w.text(" PASSING ")
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		n := node.GetJsonArgument()
		if n == nil {
			w.invalid("invalid JSON PASSING argument %T", node.GetNode())
			return
		}
		w.jsonValue(n.Val)
		w.text(" AS ")
		w.ident(n.Name)
	}
}

func (w *deparseWriter) jsonWrapper(wrapper p.JsonWrapper, allowNone bool) {
	switch wrapper.String() {
	case "JSW_NONE":
		if allowNone {
			w.text(" WITHOUT WRAPPER")
		}
	case "JSW_CONDITIONAL":
		w.text(" WITH CONDITIONAL WRAPPER")
	case "JSW_UNCONDITIONAL":
		w.text(" WITH UNCONDITIONAL WRAPPER")
	}
}

func (w *deparseWriter) jsonQuotes(quotes p.JsonQuotes) {
	switch quotes.String() {
	case "JS_QUOTES_KEEP":
		w.text(" KEEP QUOTES")
	case "JS_QUOTES_OMIT":
		w.text(" OMIT QUOTES")
	}
}

func (w *deparseWriter) jsonBehavior(n *p.JsonBehavior) {
	if n == nil {
		w.invalid("nil JSON behavior")
		return
	}
	switch n.Btype.String() {
	case "JSON_BEHAVIOR_NULL":
		w.text("NULL")
	case "JSON_BEHAVIOR_ERROR":
		w.text("ERROR")
	case "JSON_BEHAVIOR_EMPTY":
		w.text("EMPTY")
	case "JSON_BEHAVIOR_TRUE":
		w.text("TRUE")
	case "JSON_BEHAVIOR_FALSE":
		w.text("FALSE")
	case "JSON_BEHAVIOR_EMPTY_ARRAY":
		w.text("EMPTY ARRAY")
	case "JSON_BEHAVIOR_EMPTY_OBJECT":
		w.text("EMPTY OBJECT")
	case "JSON_BEHAVIOR_DEFAULT":
		w.text("DEFAULT ")
		w.expr(n.Expr)
	case "JSON_BEHAVIOR_UNKNOWN":
		w.text("UNKNOWN")
	default:
		w.invalid("invalid JSON behavior %v", n.Btype)
	}
}

func (w *deparseWriter) jsonBehaviors(empty, onError *p.JsonBehavior) {
	if empty != nil {
		w.text(" ")
		w.jsonBehavior(empty)
		w.text(" ON EMPTY")
	}
	if onError != nil {
		w.text(" ")
		w.jsonBehavior(onError)
		w.text(" ON ERROR")
	}
}

func (w *deparseWriter) jsonFunction(n *p.JsonFuncExpr) {
	switch n.Op.String() {
	case "JSON_EXISTS_OP":
		w.text("JSON_EXISTS(")
	case "JSON_QUERY_OP":
		w.text("JSON_QUERY(")
	case "JSON_VALUE_OP":
		w.text("JSON_VALUE(")
	case "JSON_TABLE_OP":
		w.text("JSON_TABLE(")
	default:
		w.invalid("invalid JSON function %v", n.Op)
		return
	}
	w.jsonValue(n.ContextItem)
	w.text(", ")
	w.expr(n.Pathspec)
	w.jsonPassing(n.Passing)
	if n.Output != nil {
		w.text(" ")
		w.jsonOutput(n.Output)
	}
	w.jsonWrapper(n.Wrapper, true)
	w.jsonQuotes(n.Quotes)
	w.jsonBehaviors(n.OnEmpty, n.OnError)
	w.text(")")
}

func (w *deparseWriter) jsonPath(n *p.JsonTablePathSpec) {
	if n == nil {
		w.invalid("nil JSON path")
		return
	}
	w.literal(n.String_.GetAConst().GetSval().GetSval())
	if n.Name != "" {
		w.text(" AS ")
		w.ident(n.Name)
	}
}

func (w *deparseWriter) jsonColumn(n *p.JsonTableColumn) {
	if n == nil {
		w.invalid("nil JSON_TABLE column")
		return
	}
	kind := n.Coltype.String()
	if kind == "JTC_NESTED" {
		w.text("NESTED PATH ")
		w.jsonPath(n.Pathspec)
		w.jsonColumns(n.Columns)
		return
	}
	w.ident(n.Name)
	w.text(" ")
	switch kind {
	case "JTC_FOR_ORDINALITY":
		w.text(" FOR ORDINALITY")
	case "JTC_EXISTS", "JTC_FORMATTED", "JTC_REGULAR":
		w.typeName(n.TypeName)
		if kind == "JTC_EXISTS" {
			w.text(" EXISTS ")
		} else {
			w.text(" ")
		}
		w.jsonFormat(n.Format)
		if n.Pathspec != nil {
			w.text("PATH ")
			w.jsonPath(n.Pathspec)
		}
	default:
		w.invalid("invalid JSON_TABLE column type %v", n.Coltype)
	}
	w.jsonWrapper(n.Wrapper, kind == "JTC_REGULAR" || kind == "JTC_FORMATTED")
	w.jsonQuotes(n.Quotes)
	w.jsonBehaviors(n.OnEmpty, n.OnError)
}

func (w *deparseWriter) jsonColumns(nodes []*p.Node) {
	w.text(" COLUMNS (")
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.jsonColumn(node.GetJsonTableColumn())
	}
	w.text(")")
}

func (w *deparseWriter) jsonTable(n *p.JsonTable) {
	w.text("JSON_TABLE(")
	w.jsonValue(n.ContextItem)
	w.text(", ")
	w.jsonPath(n.Pathspec)
	w.jsonPassing(n.Passing)
	w.jsonColumns(n.Columns)
	if n.OnError != nil {
		w.jsonBehavior(n.OnError)
		w.text(" ON ERROR")
	}
	w.text(")")
	if n.Alias != nil {
		w.text(" ")
		w.alias(n.Alias)
	}
}
