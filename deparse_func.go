package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

func (w *deparseWriter) funcCall(n *p.FuncCall) { w.funcCallIn(n, deparseExpressionA) }

func (w *deparseWriter) funcCallIn(n *p.FuncCall, context deparseExpressionContext) {
	if len(n.Funcname) == 0 {
		w.invalid("function has no name")
		return
	}
	if len(n.Funcname) == 2 && deparseString(n.Funcname[0]) == "pg_catalog" {
		name := deparseString(n.Funcname[1])
		args := n.Args
		if name == "overlay" && len(args) == 4 {
			w.text("OVERLAY(")
			w.expr(args[0])
			w.text(" PLACING ")
			w.expr(args[1])
			w.text(" FROM ")
			w.expr(args[2])
			w.text(" FOR ")
			w.expr(args[3])
			w.text(")")
			return
		}
		if n.Funcformat.String() == "COERCE_SQL_SYNTAX" {
			switch name {
			case "substring":
				if len(args) < 2 || len(args) > 3 {
					w.invalid("SUBSTRING requires two or three arguments")
					return
				}
				w.text("SUBSTRING(")
				w.expr(args[0])
				w.text(" FROM ")
				w.expr(args[1])
				if len(args) == 3 {
					w.text(" FOR ")
					w.expr(args[2])
				}
				w.text(")")
				return
			case "position":
				if len(args) != 2 {
					w.invalid("POSITION requires two arguments")
					return
				}
				w.text("POSITION(")
				w.bExpr(args[1])
				w.text(" IN ")
				w.bExpr(args[0])
				w.text(")")
				return
			case "overlay":
				if len(args) != 3 {
					w.invalid("OVERLAY requires three or four arguments")
					return
				}
				w.text("overlay(")
				w.expr(args[0])
				w.text(" placing ")
				w.expr(args[1])
				w.text(" from ")
				w.expr(args[2])
				w.text(")")
				return
			case "pg_collation_for":
				if len(args) != 1 {
					w.invalid("COLLATION FOR requires one argument")
					return
				}
				w.text("collation for (")
				w.expr(args[0])
				w.text(")")
				return
			case "extract":
				if len(args) != 2 {
					w.invalid("EXTRACT requires two arguments")
					return
				}
				w.text("extract (")
				w.expr(args[0])
				w.text(" FROM ")
				w.expr(args[1])
				w.text(")")
				return
			case "overlaps":
				if len(args) != 4 {
					w.invalid("OVERLAPS requires four arguments")
					return
				}
				w.text("(")
				w.exprs(args[:2])
				w.text(") overlaps (")
				w.exprs(args[2:])
				w.text(") ")
				return
			case "ltrim", "btrim", "rtrim":
				if len(args) < 1 || len(args) > 2 {
					w.invalid("TRIM requires one or two arguments")
					return
				}
				w.text("TRIM (")
				switch name {
				case "ltrim":
					w.text("LEADING ")
				case "btrim":
					w.text("BOTH ")
				case "rtrim":
					w.text("TRAILING ")
				}
				if len(args) == 2 {
					w.expr(args[1])
				}
				w.text(" FROM ")
				w.expr(args[0])
				w.text(")")
				return
			case "timezone":
				if len(args) < 1 || len(args) > 2 {
					w.invalid("AT TIME ZONE requires one or two arguments")
					return
				}
				arg := args[len(args)-1]
				if context != deparseExpressionA {
					w.text("(")
				}
				w.parenthesized(arg, arg.GetAExpr() != nil)
				if len(args) == 1 {
					w.text(" AT LOCAL")
				} else {
					w.text(" AT TIME ZONE ")
					w.expr(args[0])
				}
				if context != deparseExpressionA {
					w.text(")")
				}
				return
			case "normalize", "is_normalized":
				if len(args) < 1 || len(args) > 2 {
					w.invalid("NORMALIZE requires one or two arguments")
					return
				}
				if name == "normalize" {
					w.text("normalize (")
				}
				w.expr(args[0])
				if name == "is_normalized" {
					w.text(" IS ")
				}
				if len(args) == 2 {
					if name == "normalize" {
						w.text(", ")
					}
					w.text(args[1].GetAConst().GetSval().GetSval())
				}
				if name == "normalize" {
					w.text(")")
				} else {
					w.text(" NORMALIZED ")
				}
				return
			case "xmlexists":
				if len(args) != 2 {
					w.invalid("XMLEXISTS requires two arguments")
					return
				}
				w.text("xmlexists (")
				w.exprIn(args[0], deparseExpressionNone)
				w.text(" PASSING ")
				w.exprIn(args[1], deparseExpressionNone)
				w.text(")")
				return
			case "system_user":
				w.text("SYSTEM_USER")
				return
			}
		}
	}
	w.name(n.Funcname)
	w.text("(")
	if n.AggDistinct {
		w.text("DISTINCT ")
	}
	if n.AggStar {
		w.text("*")
	} else {
		for i, arg := range n.Args {
			if i > 0 {
				w.text(", ")
			}
			if n.FuncVariadic && i == len(n.Args)-1 {
				w.text("VARIADIC ")
			}
			w.expr(arg)
		}
	}
	if len(n.AggOrder) > 0 && !n.AggWithinGroup {
		w.text(" ORDER BY ")
		w.sorts(n.AggOrder)
	}
	w.text(")")
	if len(n.AggOrder) > 0 && n.AggWithinGroup {
		w.text(" WITHIN GROUP (ORDER BY ")
		w.sorts(n.AggOrder)
		w.text(")")
	}
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

func (w *deparseWriter) sorts(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		sort := node.GetSortBy()
		if sort == nil {
			w.invalid("invalid sort expression %T", node.GetNode())
			return
		}
		w.expr(sort.Node)
		switch sort.SortbyDir.String() {
		case "SORTBY_ASC":
			w.text(" ASC")
		case "SORTBY_DESC":
			w.text(" DESC")
		case "SORTBY_USING":
			w.text(" USING ")
			w.qualifiedOperator(sort.UseOp)
		}
		switch sort.SortbyNulls.String() {
		case "SORTBY_NULLS_FIRST":
			w.text(" NULLS FIRST")
		case "SORTBY_NULLS_LAST":
			w.text(" NULLS LAST")
		}
	}
}

const (
	deparseFrameNondefault              = 0x00001
	deparseFrameRange                   = 0x00002
	deparseFrameRows                    = 0x00004
	deparseFrameGroups                  = 0x00008
	deparseFrameBetween                 = 0x00010
	deparseFrameStartUnboundedPreceding = 0x00020
	deparseFrameEndUnboundedPreceding   = 0x00040
	deparseFrameStartUnboundedFollowing = 0x00080
	deparseFrameEndUnboundedFollowing   = 0x00100
	deparseFrameStartCurrentRow         = 0x00200
	deparseFrameEndCurrentRow           = 0x00400
	deparseFrameStartOffsetPreceding    = 0x00800
	deparseFrameEndOffsetPreceding      = 0x01000
	deparseFrameStartOffsetFollowing    = 0x02000
	deparseFrameEndOffsetFollowing      = 0x04000
	deparseFrameExcludeCurrentRow       = 0x08000
	deparseFrameExcludeGroup            = 0x10000
	deparseFrameExcludeTies             = 0x20000
)

func (w *deparseWriter) window(n *p.WindowDef) {
	w.text("(")
	space := false
	if n.Refname != "" {
		w.ident(n.Refname)
		space = true
	}
	if len(n.PartitionClause) > 0 {
		if space {
			w.text(" ")
		}
		w.text("PARTITION BY ")
		w.exprs(n.PartitionClause)
		space = true
	}
	if len(n.OrderClause) > 0 {
		if space {
			w.text(" ")
		}
		w.text("ORDER BY ")
		w.sorts(n.OrderClause)
		space = true
	}
	f := n.FrameOptions
	if f&deparseFrameNondefault != 0 {
		if space {
			w.text(" ")
		}
		switch {
		case f&deparseFrameRange != 0:
			w.text("RANGE ")
		case f&deparseFrameRows != 0:
			w.text("ROWS ")
		case f&deparseFrameGroups != 0:
			w.text("GROUPS ")
		}
		if f&deparseFrameBetween != 0 {
			w.text("BETWEEN ")
		}
		switch {
		case f&deparseFrameStartUnboundedPreceding != 0:
			w.text("UNBOUNDED PRECEDING")
		case f&deparseFrameStartUnboundedFollowing != 0:
			w.invalid("frame starts with UNBOUNDED FOLLOWING")
		case f&deparseFrameStartCurrentRow != 0:
			w.text("CURRENT ROW")
		case f&deparseFrameStartOffsetPreceding != 0:
			w.expr(n.StartOffset)
			w.text(" PRECEDING")
		case f&deparseFrameStartOffsetFollowing != 0:
			w.expr(n.StartOffset)
			w.text(" FOLLOWING")
		}
		if f&deparseFrameBetween != 0 {
			w.text(" AND ")
			switch {
			case f&deparseFrameEndUnboundedPreceding != 0:
				w.invalid("frame ends with UNBOUNDED PRECEDING")
			case f&deparseFrameEndUnboundedFollowing != 0:
				w.text("UNBOUNDED FOLLOWING")
			case f&deparseFrameEndCurrentRow != 0:
				w.text("CURRENT ROW")
			case f&deparseFrameEndOffsetPreceding != 0:
				w.expr(n.EndOffset)
				w.text(" PRECEDING")
			case f&deparseFrameEndOffsetFollowing != 0:
				w.expr(n.EndOffset)
				w.text(" FOLLOWING")
			}
		}
		switch {
		case f&deparseFrameExcludeCurrentRow != 0:
			w.text(" EXCLUDE CURRENT ROW")
		case f&deparseFrameExcludeGroup != 0:
			w.text(" EXCLUDE GROUP")
		case f&deparseFrameExcludeTies != 0:
			w.text(" EXCLUDE TIES")
		}
	}
	w.text(")")
}
