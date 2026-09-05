package pg_query

import (
	"strings"

	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

type deparseExpressionContext uint8

const (
	deparseExpressionA deparseExpressionContext = iota
	deparseExpressionNone
)

func (w *deparseWriter) expr(node *p.Node) { w.exprIn(node, deparseExpressionA) }

func (w *deparseWriter) exprIn(node *p.Node, context deparseExpressionContext) {
	if node == nil || w.err != nil {
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_AConst:
		w.constant(n.AConst)
	case *p.Node_Integer:
		w.integer(int64(n.Integer.Ival))
	case *p.Node_Float:
		w.text(n.Float.Fval)
	case *p.Node_String_:
		w.literal(n.String_.Sval)
	case *p.Node_Boolean:
		if n.Boolean.Boolval {
			w.text("true")
		} else {
			w.text("false")
		}
	case *p.Node_List:
		w.exprs(n.List.Items)
	case *p.Node_AStar:
		w.text("*")
	case *p.Node_ColumnRef:
		fields := n.ColumnRef.Fields
		if len(fields) == 0 {
			w.invalid("empty column reference")
			return
		}
		if fields[0].GetAStar() != nil {
			w.text("*")
		} else {
			w.ident(deparseString(fields[0]))
		}
		w.indirection(fields[1:])
	case *p.Node_ParamRef:
		if n.ParamRef.Number == 0 {
			w.text("?")
		} else {
			w.text("$")
			w.integer(int64(n.ParamRef.Number))
		}
	case *p.Node_AExpr:
		w.operatorExpr(n.AExpr, context)
	case *p.Node_BoolExpr:
		w.boolExpr(n.BoolExpr)
	case *p.Node_TypeCast:
		w.typeCast(n.TypeCast)
	case *p.Node_TypeName:
		w.typeName(n.TypeName)
	case *p.Node_CollateClause:
		if n.CollateClause.Arg != nil {
			w.parenthesized(n.CollateClause.Arg, n.CollateClause.Arg.GetAExpr() != nil)
			w.text(" ")
		}
		w.text("COLLATE ")
		w.name(n.CollateClause.Collname)
	case *p.Node_FuncCall:
		w.funcCallIn(n.FuncCall, context)
	case *p.Node_NamedArgExpr:
		w.ident(n.NamedArgExpr.Name)
		w.text(" := ")
		w.expr(n.NamedArgExpr.Arg)
	case *p.Node_SubLink:
		w.subLink(n.SubLink)
	case *p.Node_AArrayExpr:
		w.array(n.AArrayExpr)
	case *p.Node_RowExpr:
		if n.RowExpr.RowFormat.String() == "COERCE_EXPLICIT_CALL" {
			w.text("ROW")
		}
		w.text("(")
		w.exprs(n.RowExpr.Args)
		w.text(")")
	case *p.Node_GroupingFunc:
		w.text("GROUPING(")
		w.exprs(n.GroupingFunc.Args)
		w.text(")")
	case *p.Node_CoalesceExpr:
		w.text("COALESCE(")
		w.exprs(n.CoalesceExpr.Args)
		w.text(")")
	case *p.Node_MinMaxExpr:
		switch n.MinMaxExpr.Op.String() {
		case "IS_GREATEST":
			w.text("GREATEST(")
		case "IS_LEAST":
			w.text("LEAST(")
		default:
			w.invalid("invalid min/max operator %v", n.MinMaxExpr.Op)
			return
		}
		w.exprs(n.MinMaxExpr.Args)
		w.text(")")
	case *p.Node_NullTest:
		w.expr(n.NullTest.Arg)
		switch n.NullTest.Nulltesttype.String() {
		case "IS_NULL":
			w.text(" IS NULL")
		case "IS_NOT_NULL":
			w.text(" IS NOT NULL")
		default:
			w.invalid("invalid null test %v", n.NullTest.Nulltesttype)
		}
	case *p.Node_BooleanTest:
		w.parenthesized(n.BooleanTest.Arg, n.BooleanTest.Arg.GetBoolExpr() != nil)
		switch n.BooleanTest.Booltesttype.String() {
		case "IS_TRUE":
			w.text(" IS TRUE")
		case "IS_NOT_TRUE":
			w.text(" IS NOT TRUE")
		case "IS_FALSE":
			w.text(" IS FALSE")
		case "IS_NOT_FALSE":
			w.text(" IS NOT FALSE")
		case "IS_UNKNOWN":
			w.text(" IS UNKNOWN")
		case "IS_NOT_UNKNOWN":
			w.text(" IS NOT UNKNOWN")
		default:
			w.invalid("invalid boolean test %v", n.BooleanTest.Booltesttype)
		}
	case *p.Node_CaseExpr:
		w.text("CASE")
		if n.CaseExpr.Arg != nil {
			w.text(" ")
			w.expr(n.CaseExpr.Arg)
			w.trimSpace()
		}
		for _, arg := range n.CaseExpr.Args {
			when := arg.GetCaseWhen()
			if when == nil {
				w.invalid("invalid CASE arm %T", arg.GetNode())
				return
			}
			w.text(" WHEN ")
			w.expr(when.Expr)
			w.text(" THEN ")
			w.expr(when.Result)
		}
		if n.CaseExpr.Defresult != nil {
			w.text(" ELSE ")
			w.expr(n.CaseExpr.Defresult)
		}
		w.text(" END")
	case *p.Node_AIndirection:
		arg := n.AIndirection.Arg
		parens := arg.GetAIndirection() != nil || arg.GetFuncCall() != nil || arg.GetAExpr() != nil || arg.GetTypeCast() != nil || arg.GetRowExpr() != nil || arg.GetAArrayExpr() != nil || arg.GetJsonFuncExpr() != nil
		if arg.GetColumnRef() != nil && len(n.AIndirection.Indirection) > 0 && n.AIndirection.Indirection[0].GetAIndices() == nil {
			parens = true
		}
		w.parenthesized(arg, parens)
		w.indirection(n.AIndirection.Indirection)
	case *p.Node_AIndices:
		w.indices(n.AIndices)
	case *p.Node_SqlvalueFunction:
		w.sqlValueFunction(n.SqlvalueFunction)
	case *p.Node_SetToDefault:
		w.text("DEFAULT")
	case *p.Node_MergeSupportFunc:
		w.text("merge_action() ")
	case *p.Node_XmlExpr:
		w.xmlExprIn(n.XmlExpr, context)
	case *p.Node_XmlSerialize:
		w.xmlSerialize(n.XmlSerialize)
	case *p.Node_JsonIsPredicate, *p.Node_JsonValueExpr, *p.Node_JsonObjectAgg,
		*p.Node_JsonArrayAgg, *p.Node_JsonObjectConstructor, *p.Node_JsonArrayConstructor,
		*p.Node_JsonArrayQueryConstructor, *p.Node_JsonParseExpr, *p.Node_JsonScalarExpr,
		*p.Node_JsonSerializeExpr, *p.Node_JsonFuncExpr:
		w.jsonExpr(node)
	default:
		w.invalid("unpermitted node type in expression: %T", node.Node)
	}
}

func (w *deparseWriter) operatorExpr(n *p.A_Expr, context deparseExpressionContext) {
	if len(n.Name) == 0 {
		w.invalid("operator expression has no name")
		return
	}
	name := deparseString(n.Name[0])
	switch n.Kind.String() {
	case "AEXPR_OP":
		if n.Lexpr != nil {
			w.parenthesizedIn(n.Lexpr, deparseNeedsBParens(n.Lexpr), context)
			w.text(" ")
		}
		w.qualifiedOperator(n.Name)
		if n.Rexpr != nil {
			w.text(" ")
			w.parenthesizedIn(n.Rexpr, deparseNeedsBParens(n.Rexpr), context)
		}
	case "AEXPR_OP_ANY", "AEXPR_OP_ALL":
		w.exprIn(n.Lexpr, context)
		w.text(" ")
		w.subqueryOperator(n.Name)
		if n.Kind.String() == "AEXPR_OP_ANY" {
			w.text(" ANY(")
		} else {
			w.text(" ALL(")
		}
		w.exprIn(n.Rexpr, context)
		w.text(")")
	case "AEXPR_DISTINCT":
		w.parenthesizedIn(n.Lexpr, deparseNeedsBParens(n.Lexpr), context)
		w.text(" IS DISTINCT FROM ")
		w.parenthesizedIn(n.Rexpr, deparseNeedsBParens(n.Rexpr), context)
	case "AEXPR_NOT_DISTINCT":
		w.exprIn(n.Lexpr, context)
		w.text(" IS NOT DISTINCT FROM ")
		w.exprIn(n.Rexpr, context)
	case "AEXPR_NULLIF":
		w.text("NULLIF(")
		w.exprIn(n.Lexpr, context)
		w.text(", ")
		w.exprIn(n.Rexpr, context)
		w.text(")")
	case "AEXPR_IN":
		w.exprIn(n.Lexpr, context)
		switch name {
		case "=":
			w.text(" IN (")
		case "<>":
			w.text(" NOT IN (")
		default:
			w.invalid("invalid IN operator %q", name)
			return
		}
		w.expr(n.Rexpr)
		w.text(")")
	case "AEXPR_LIKE", "AEXPR_ILIKE", "AEXPR_SIMILAR":
		w.exprIn(n.Lexpr, context)
		switch name {
		case "~~":
			w.text(" LIKE ")
		case "!~~":
			w.text(" NOT LIKE ")
		case "~~*":
			w.text(" ILIKE ")
		case "!~~*":
			w.text(" NOT ILIKE ")
		case "~":
			w.text(" SIMILAR TO ")
		case "!~":
			w.text(" NOT SIMILAR TO ")
		default:
			w.invalid("invalid pattern operator %q", name)
			return
		}
		if n.Kind.String() == "AEXPR_SIMILAR" {
			call := n.Rexpr.GetFuncCall()
			if call == nil || len(call.Args) < 1 || len(call.Args) > 2 {
				w.invalid("invalid SIMILAR TO escape expression")
				return
			}
			w.exprIn(call.Args[0], context)
			if len(call.Args) == 2 {
				w.text(" ESCAPE ")
				w.exprIn(call.Args[1], context)
			}
		} else {
			w.exprIn(n.Rexpr, context)
		}
	case "AEXPR_BETWEEN", "AEXPR_NOT_BETWEEN", "AEXPR_BETWEEN_SYM", "AEXPR_NOT_BETWEEN_SYM":
		w.exprIn(n.Lexpr, context)
		w.text(" ")
		w.text(name)
		w.text(" ")
		bounds := n.Rexpr.GetList().GetItems()
		if len(bounds) != 2 {
			w.invalid("BETWEEN requires two bounds")
			return
		}
		w.exprIn(bounds[0], context)
		w.text(" AND ")
		w.exprIn(bounds[1], context)
	default:
		w.invalid("invalid operator expression kind %v", n.Kind)
	}
}

func (w *deparseWriter) boolExpr(n *p.BoolExpr) {
	kind := n.Boolop.String()
	if kind == "NOT_EXPR" {
		if len(n.Args) != 1 {
			w.invalid("NOT requires one operand")
			return
		}
		w.text("NOT ")
	} else if kind != "AND_EXPR" && kind != "OR_EXPR" {
		w.invalid("invalid boolean operator %v", n.Boolop)
		return
	}
	for i, arg := range n.Args {
		if i > 0 {
			if kind == "AND_EXPR" {
				w.text(" AND ")
			} else {
				w.text(" OR ")
			}
		}
		b := arg.GetBoolExpr()
		w.parenthesized(arg, b != nil && (b.Boolop.String() == "AND_EXPR" || b.Boolop.String() == "OR_EXPR"))
	}
}

func (w *deparseWriter) subLink(n *p.SubLink) {
	switch n.SubLinkType.String() {
	case "EXISTS_SUBLINK":
		w.text("EXISTS (")
	case "ALL_SUBLINK":
		w.expr(n.Testexpr)
		w.text(" ")
		w.subqueryOperator(n.OperName)
		w.text(" ALL (")
	case "ANY_SUBLINK":
		w.expr(n.Testexpr)
		if len(n.OperName) == 0 {
			w.text(" IN (")
		} else {
			w.text(" ")
			w.subqueryOperator(n.OperName)
			w.text(" ANY (")
		}
	case "EXPR_SUBLINK":
		w.text("(")
	case "ARRAY_SUBLINK":
		w.text("ARRAY(")
	default:
		w.invalid("unpermitted sublink type %v", n.SubLinkType)
		return
	}
	w.stmt(n.Subselect)
	w.text(")")
}

func (w *deparseWriter) array(n *p.A_ArrayExpr) {
	w.text("ARRAY[")
	for i, element := range n.Elements {
		if i > 0 {
			w.text(", ")
		}
		w.expr(element)
	}
	w.text("]")
}

func (w *deparseWriter) indices(n *p.A_Indices) {
	w.text("[")
	w.expr(n.Lidx)
	if n.IsSlice {
		w.text(":")
	}
	w.expr(n.Uidx)
	w.text("]")
}

func (w *deparseWriter) indirection(nodes []*p.Node) {
	for _, node := range nodes {
		switch n := node.Node.(type) {
		case *p.Node_String_:
			w.text(".")
			w.ident(n.String_.Sval)
		case *p.Node_AStar:
			w.text(".*")
		case *p.Node_AIndices:
			w.indices(n.AIndices)
		default:
			w.invalid("invalid indirection node %T", node.Node)
		}
	}
}

func (w *deparseWriter) sqlValueFunction(n *p.SQLValueFunction) {
	name := strings.TrimPrefix(n.Op.String(), "SVFOP_")
	precision := n.Typmod != -1
	name = strings.TrimSuffix(name, "_N")
	switch name {
	case "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP", "LOCALTIME", "LOCALTIMESTAMP", "CURRENT_ROLE", "CURRENT_USER", "USER", "SESSION_USER", "CURRENT_CATALOG", "CURRENT_SCHEMA":
		w.text(strings.ToLower(name))
	default:
		w.invalid("invalid SQL value function %v", n.Op)
		return
	}
	if precision {
		w.text("(")
		w.integer(int64(n.Typmod))
		w.text(")")
	}
}

func (w *deparseWriter) typeCast(n *p.TypeCast) {
	if n.TypeName == nil {
		w.invalid("cast has no type")
		return
	}
	parens := deparseNeedsBParens(n.Arg)
	if c := n.Arg.GetAConst(); c != nil {
		if len(n.TypeName.Names) == 2 && deparseString(n.TypeName.Names[0]) == "pg_catalog" {
			switch deparseString(n.TypeName.Names[1]) {
			case "bpchar":
				if len(n.TypeName.Typmods) == 0 {
					w.text("char ")
					w.constant(c)
					return
				}
			case "bool":
				if s := c.GetSval(); s != nil {
					if s.Sval == "t" {
						w.text("true")
						return
					}
					if s.Sval == "f" {
						w.text("false")
						return
					}
				}
			}
		}
		if c.GetFval() != nil || c.GetIval().GetIval() < 0 {
			parens = true
		}
		if len(n.TypeName.Names) == 1 && deparseString(n.TypeName.Names[0]) == "point" && c.Location > n.TypeName.Location {
			w.text(" point ")
			w.constant(c)
			return
		}
	}
	w.parenthesizedIn(n.Arg, parens, deparseExpressionNone)
	w.text("::")
	w.typeName(n.TypeName)
}

func (w *deparseWriter) typeName(n *p.TypeName) {
	if n == nil {
		w.invalid("nil type name")
		return
	}
	if n.Setof {
		w.text("SETOF ")
	}
	skipMods := false
	if len(n.Names) == 2 && deparseString(n.Names[0]) == "pg_catalog" {
		name := deparseString(n.Names[1])
		switch name {
		case "bpchar":
			w.text("char")
		case "varchar", "numeric", "time", "timestamp":
			w.text(name)
		case "bool":
			w.text("boolean")
		case "int2":
			w.text("smallint")
		case "int4":
			w.text("int")
		case "int8":
			w.text("bigint")
		case "real", "float4":
			w.text("real")
		case "float8":
			w.text("double precision")
		case "timetz", "timestamptz":
			if name == "timetz" {
				w.text("time ")
			} else {
				w.text("timestamp ")
			}
			if len(n.Typmods) > 0 {
				w.text("(")
				w.exprs(n.Typmods)
				w.text(") ")
			}
			w.text("with time zone")
			skipMods = true
		case "interval":
			w.text("interval")
			if len(n.Typmods) > 0 {
				w.intervalMods(n.Typmods)
				skipMods = true
			}
		default:
			w.text("pg_catalog.")
			w.text(name)
		}
	} else {
		w.name(n.Names)
	}
	if !skipMods && len(n.Typmods) > 0 {
		w.text("(")
		w.exprs(n.Typmods)
		w.text(")")
	}
	for _, bound := range n.ArrayBounds {
		w.text("[")
		if i := bound.GetInteger(); i != nil && i.Ival != -1 {
			w.integer(int64(i.Ival))
		}
		w.text("]")
	}
	if n.PctType {
		w.text("%type")
	}
}

func (w *deparseWriter) intervalMods(mods []*p.Node) {
	if len(mods) == 0 {
		return
	}
	mask := mods[0].GetAConst().GetIval().GetIval()
	const year = 1 << 2
	const month = 1 << 1
	const day = 1 << 3
	const hour = 1 << 10
	const minute = 1 << 11
	const second = 1 << 12
	switch mask {
	case year:
		w.text(" year")
	case month:
		w.text(" month")
	case day:
		w.text(" day")
	case hour:
		w.text(" hour")
	case minute:
		w.text(" minute")
	case second:
		w.text(" second")
	case year | month:
		w.text(" year to month")
	case day | hour:
		w.text(" day to hour")
	case day | hour | minute:
		w.text(" day to minute")
	case day | hour | minute | second:
		w.text(" day to second")
	case hour | minute:
		w.text(" hour to minute")
	case hour | minute | second:
		w.text(" hour to second")
	case minute | second:
		w.text(" minute to second")
	case 0x7fff:
	default:
		w.invalid("invalid interval field mask %d", mask)
	}
	if len(mods) == 2 {
		precision := mods[1].GetAConst().GetIval().GetIval()
		if precision != 0xffff {
			w.text("(")
			w.integer(int64(precision))
			w.text(")")
		}
	}
}
