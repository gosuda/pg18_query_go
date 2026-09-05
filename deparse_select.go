package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

type deparseRelationMode uint8

const (
	deparseRelationNormal deparseRelationMode = iota
	deparseRelationCreate
	deparseRelationInsert
)

func (w *deparseWriter) relation(n *p.RangeVar, mode deparseRelationMode) {
	if n == nil {
		w.invalid("nil relation")
		return
	}
	if !n.Inh && mode != deparseRelationCreate {
		w.text("ONLY ")
	}
	if n.Catalogname != "" {
		w.ident(n.Catalogname)
		w.text(".")
	}
	if n.Schemaname != "" {
		w.ident(n.Schemaname)
		w.text(".")
	}
	w.ident(n.Relname)
	if n.Alias != nil {
		if mode == deparseRelationInsert {
			w.text(" AS ")
		} else {
			w.text(" ")
		}
		w.alias(n.Alias)
	}
}

func (w *deparseWriter) alias(n *p.Alias) {
	w.ident(n.Aliasname)
	if len(n.Colnames) > 0 {
		w.text("(")
		w.names(n.Colnames)
		w.text(")")
	}
}

func (w *deparseWriter) targets(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		n := node.GetResTarget()
		if n == nil || n.Val == nil {
			w.invalid("target has no expression")
			return
		}
		w.expr(n.Val)
		if n.Name != "" {
			w.text(" AS ")
			w.ident(n.Name)
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) selectStmt(n *p.SelectStmt, setOperand bool) {
	if n == nil {
		w.invalid("nil SELECT operand")
		return
	}
	setop := n.Op.String()
	isSet := setop == "SETOP_UNION" || setop == "SETOP_INTERSECT" || setop == "SETOP_EXCEPT"
	parens := setOperand && (len(n.SortClause) > 0 || n.LimitOffset != nil || n.LimitCount != nil || len(n.LockingClause) > 0 || n.WithClause != nil || isSet)
	if parens {
		w.text("(")
	}
	w.with(n.WithClause)
	if isSet {
		w.selectStmt(n.Larg, true)
		switch setop {
		case "SETOP_UNION":
			w.text(" UNION")
		case "SETOP_INTERSECT":
			w.text(" INTERSECT")
		case "SETOP_EXCEPT":
			w.text(" EXCEPT")
		}
		if n.All {
			w.text(" ALL")
		}
		w.text(" ")
		w.selectStmt(n.Rarg, true)
	} else if len(n.ValuesLists) > 0 {
		w.text("VALUES ")
		for i, row := range n.ValuesLists {
			if i > 0 {
				w.text(", ")
			}
			w.text("(")
			w.exprs(row.GetList().GetItems())
			w.text(")")
		}
	} else {
		w.text("SELECT")
		if len(n.TargetList) > 0 {
			w.text(" ")
			if len(n.DistinctClause) > 0 {
				w.text("DISTINCT ")
				if n.DistinctClause[0] != nil && n.DistinctClause[0].Node != nil {
					w.text("ON (")
					w.exprs(n.DistinctClause)
					w.text(") ")
				}
			}
			w.targets(n.TargetList)
		}
		if n.IntoClause != nil {
			w.text(" INTO ")
			w.persistence(n.IntoClause.Rel.GetRelpersistence())
			w.into(n.IntoClause)
		}
		if len(n.FromClause) > 0 {
			w.text(" FROM ")
			w.tables(n.FromClause)
		}
		w.where(n.WhereClause)
		if len(n.GroupClause) > 0 {
			w.text(" GROUP BY ")
			if n.GroupDistinct {
				w.text("DISTINCT ")
			}
			w.groups(n.GroupClause)
		}
		if n.HavingClause != nil {
			w.text(" HAVING ")
			w.expr(n.HavingClause)
		}
		if len(n.WindowClause) > 0 {
			w.text(" WINDOW ")
			for i, node := range n.WindowClause {
				if i > 0 {
					w.text(", ")
				}
				window := node.GetWindowDef()
				if window == nil {
					w.invalid("invalid WINDOW clause %T", node.GetNode())
					return
				}
				w.text(window.Name)
				w.text(" AS ")
				w.window(window)
			}
		}
	}
	if len(n.SortClause) > 0 {
		w.text(" ORDER BY ")
		w.sorts(n.SortClause)
	}
	if n.LimitCount != nil {
		ties := n.LimitOption.String() == "LIMIT_OPTION_WITH_TIES"
		if ties {
			w.text(" FETCH FIRST ")
		} else {
			w.text(" LIMIT ")
		}
		if n.LimitCount.GetAConst().GetIsnull() {
			w.text("ALL")
		} else if ties {
			w.cExpr(n.LimitCount)
		} else {
			w.exprIn(n.LimitCount, deparseExpressionNone)
		}
		if ties {
			w.text(" ROWS WITH TIES")
		}
	}
	if n.LimitOffset != nil {
		w.text(" OFFSET ")
		w.expr(n.LimitOffset)
	}
	for _, node := range n.LockingClause {
		w.locking(node.GetLockingClause())
	}
	w.trimSpace()
	if parens {
		w.text(")")
	}
}

func (w *deparseWriter) persistence(value string) {
	switch value {
	case "t":
		w.text("TEMPORARY ")
	case "u":
		w.text("UNLOGGED ")
	}
}

func (w *deparseWriter) into(n *p.IntoClause) {
	w.relation(n.Rel, deparseRelationNormal)
	if len(n.ColNames) > 0 {
		w.text("(")
		w.names(n.ColNames)
		w.text(")")
	}
	if n.AccessMethod != "" {
		w.text(" USING ")
		w.ident(n.AccessMethod)
	}
	if len(n.Options) > 0 {
		w.text(" WITH ")
		w.relOptions(n.Options)
	}
	switch n.OnCommit.String() {
	case "ONCOMMIT_PRESERVE_ROWS":
		w.text(" ON COMMIT PRESERVE ROWS")
	case "ONCOMMIT_DELETE_ROWS":
		w.text(" ON COMMIT DELETE ROWS")
	case "ONCOMMIT_DROP":
		w.text(" ON COMMIT DROP")
	}
	if n.TableSpaceName != "" {
		w.text(" TABLESPACE ")
		w.ident(n.TableSpaceName)
	}
}

func (w *deparseWriter) with(n *p.WithClause) {
	if n == nil {
		return
	}
	w.text("WITH ")
	if n.Recursive {
		w.text("RECURSIVE ")
	}
	for i, node := range n.Ctes {
		if i > 0 {
			w.text(", ")
		}
		cte := node.GetCommonTableExpr()
		if cte == nil {
			w.invalid("invalid CTE %T", node.GetNode())
			return
		}
		w.ident(cte.Ctename)
		if len(cte.Aliascolnames) > 0 {
			w.text("(")
			w.names(cte.Aliascolnames)
			w.text(")")
		}
		w.text(" AS ")
		switch cte.Ctematerialized.String() {
		case "CTEMaterializeAlways":
			w.text("MATERIALIZED ")
		case "CTEMaterializeNever":
			w.text("NOT MATERIALIZED ")
		}
		w.text("(")
		w.stmt(cte.Ctequery)
		w.text(")")
		if search := cte.SearchClause; search != nil {
			w.text(" SEARCH ")
			if search.SearchBreadthFirst {
				w.text("BREADTH ")
			} else {
				w.text("DEPTH ")
			}
			w.text("FIRST BY ")
			w.names(search.SearchColList)
			w.text(" SET ")
			w.ident(search.SearchSeqColumn)
		}
		if cycle := cte.CycleClause; cycle != nil {
			w.text(" CYCLE ")
			w.names(cycle.CycleColList)
			w.text(" SET ")
			w.ident(cycle.CycleMarkColumn)
			if cycle.CycleMarkValue != nil {
				w.text(" TO ")
				w.expr(cycle.CycleMarkValue)
			}
			if cycle.CycleMarkDefault != nil {
				w.text(" DEFAULT ")
				w.expr(cycle.CycleMarkDefault)
			}
			w.text(" USING ")
			w.ident(cycle.CyclePathColumn)
		}
	}
	w.text(" ")
}

func (w *deparseWriter) tables(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.table(node)
	}
}

func (w *deparseWriter) table(node *p.Node) {
	if node == nil {
		w.invalid("nil table reference")
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_RangeVar:
		w.relation(n.RangeVar, deparseRelationNormal)
	case *p.Node_JoinExpr:
		w.join(n.JoinExpr)
	case *p.Node_RangeSubselect:
		if n.RangeSubselect.Lateral {
			w.text("LATERAL ")
		}
		w.text("(")
		w.stmt(n.RangeSubselect.Subquery)
		w.text(")")
		if n.RangeSubselect.Alias != nil {
			w.text(" ")
			w.alias(n.RangeSubselect.Alias)
		}
	case *p.Node_RangeFunction:
		w.rangeFunction(n.RangeFunction)
	case *p.Node_RangeTableSample:
		w.table(n.RangeTableSample.Relation)
		w.text(" TABLESAMPLE ")
		w.name(n.RangeTableSample.Method)
		w.text("(")
		w.exprs(n.RangeTableSample.Args)
		w.text(")")
		if n.RangeTableSample.Repeatable != nil {
			w.text(" REPEATABLE (")
			w.expr(n.RangeTableSample.Repeatable)
			w.text(")")
		}
	case *p.Node_RangeTableFunc:
		w.xmlTable(n.RangeTableFunc)
	case *p.Node_JsonTable:
		w.jsonTable(n.JsonTable)
	default:
		w.invalid("unpermitted table reference %T", node.Node)
	}
}

func (w *deparseWriter) join(n *p.JoinExpr) {
	if n.Alias != nil {
		w.text("(")
	}
	w.table(n.Larg)
	w.text(" ")
	if n.IsNatural {
		w.text("NATURAL ")
	}
	switch n.Jointype.String() {
	case "JOIN_INNER":
		if !n.IsNatural && n.Quals == nil && len(n.UsingClause) == 0 {
			w.text("CROSS ")
		}
	case "JOIN_LEFT":
		w.text("LEFT ")
	case "JOIN_FULL":
		w.text("FULL ")
	case "JOIN_RIGHT":
		w.text("RIGHT ")
	default:
		w.invalid("unpermitted raw join type %v", n.Jointype)
	}
	w.text("JOIN ")
	child := n.Rarg.GetJoinExpr()
	parens := child != nil && child.Alias == nil
	if parens {
		w.text("(")
	}
	w.table(n.Rarg)
	if parens {
		w.text(")")
	}
	if n.Quals != nil {
		w.text(" ON ")
		w.expr(n.Quals)
	}
	if len(n.UsingClause) > 0 {
		w.text(" USING (")
		w.names(n.UsingClause)
		w.text(")")
		if n.JoinUsingAlias != nil {
			w.text(" AS ")
			w.text(n.JoinUsingAlias.Aliasname)
		}
	}
	if n.Alias != nil {
		if n.JoinUsingAlias == nil {
			w.text(" ")
		}
		w.text(") ")
		w.alias(n.Alias)
	}
}

func (w *deparseWriter) rangeFunction(n *p.RangeFunction) {
	if n.Lateral {
		w.text("LATERAL ")
	}
	if n.IsRowsfrom {
		w.text("ROWS FROM (")
	}
	for i, node := range n.Functions {
		if !n.IsRowsfrom && i > 0 {
			w.invalid("multiple functions without ROWS FROM")
			return
		}
		if i > 0 {
			w.text(", ")
		}
		pair := node.GetList().GetItems()
		if len(pair) != 2 {
			w.invalid("range function is not an expression/column pair")
			return
		}
		if cast := pair[0].GetTypeCast(); cast != nil {
			w.text("CAST(")
			w.expr(cast.Arg)
			w.text(" AS ")
			w.typeName(cast.TypeName)
			w.text(")")
		} else {
			w.exprIn(pair[0], deparseExpressionNone)
		}
		if n.IsRowsfrom {
			w.text(" ")
			if cols := pair[1].GetList().GetItems(); len(cols) > 0 {
				w.text("AS (")
				w.columnDefs(cols)
				w.text(")")
			}
		}
	}
	if n.IsRowsfrom {
		w.text(")")
	}
	if n.Ordinality {
		w.text(" WITH ORDINALITY")
	}
	if n.Alias != nil {
		w.text(" ")
		w.alias(n.Alias)
	}
	if len(n.Coldeflist) > 0 {
		if n.Alias == nil {
			w.text(" AS ")
		} else {
			w.text(" ")
		}
		w.text("(")
		w.columnDefs(n.Coldeflist)
		w.text(")")
	}
}

func (w *deparseWriter) columnDefs(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.columnDef(node.GetColumnDef())
	}
}

func (w *deparseWriter) groups(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		group := node.GetGroupingSet()
		if group == nil {
			w.expr(node)
			continue
		}
		switch group.Kind.String() {
		case "GROUPING_SET_EMPTY":
			w.text("()")
			continue
		case "GROUPING_SET_SIMPLE":
		case "GROUPING_SET_ROLLUP":
			w.text("ROLLUP ")
		case "GROUPING_SET_CUBE":
			w.text("CUBE ")
		case "GROUPING_SET_SETS":
			w.text("GROUPING SETS ")
		default:
			w.invalid("invalid grouping set %v", group.Kind)
			return
		}
		w.text("(")
		w.groups(group.Content)
		w.text(")")
	}
}

func (w *deparseWriter) where(n *p.Node) {
	if n == nil {
		return
	}
	w.text(" WHERE ")
	if current := n.GetCurrentOfExpr(); current != nil {
		w.text("CURRENT OF ")
		w.ident(current.CursorName)
	} else {
		w.expr(n)
	}
}

func (w *deparseWriter) locking(n *p.LockingClause) {
	if n == nil {
		w.invalid("nil locking clause")
		return
	}
	switch n.Strength.String() {
	case "LCS_FORKEYSHARE":
		w.text(" FOR KEY SHARE")
	case "LCS_FORSHARE":
		w.text(" FOR SHARE")
	case "LCS_FORNOKEYUPDATE":
		w.text(" FOR NO KEY UPDATE")
	case "LCS_FORUPDATE":
		w.text(" FOR UPDATE")
	default:
		w.invalid("invalid locking strength %v", n.Strength)
		return
	}
	if len(n.LockedRels) > 0 {
		w.text(" OF ")
		for i, node := range n.LockedRels {
			if i > 0 {
				w.text(", ")
			}
			w.relation(node.GetRangeVar(), deparseRelationNormal)
		}
	}
	switch n.WaitPolicy.String() {
	case "LockWaitError":
		w.text(" NOWAIT")
	case "LockWaitSkip":
		w.text(" SKIP LOCKED")
	}
}
