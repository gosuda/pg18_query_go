package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

func (w *deparseWriter) stmt(node *p.Node) {
	if node == nil {
		w.invalid("nil statement")
		return
	}
	if w.err != nil {
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_RawStmt:
		w.stmt(n.RawStmt.Stmt)
	case *p.Node_SelectStmt:
		w.selectStmt(n.SelectStmt, false)
	case *p.Node_InsertStmt:
		w.insert(n.InsertStmt)
	case *p.Node_UpdateStmt:
		w.update(n.UpdateStmt)
	case *p.Node_DeleteStmt:
		w.delete(n.DeleteStmt)
	case *p.Node_MergeStmt:
		w.merge(n.MergeStmt)
	default:
		if !w.utilityStmt(node) {
			w.invalid("unpermitted statement type %T", node.Node)
		}
		w.trimSpace()
	}
}

func (w *deparseWriter) insertColumns(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		n := node.GetResTarget()
		if n == nil {
			w.invalid("invalid INSERT column %T", node.GetNode())
			return
		}
		w.ident(n.Name)
		w.indirection(n.Indirection)
	}
}

func (w *deparseWriter) override(kind p.OverridingKind) {
	switch kind.String() {
	case "OVERRIDING_USER_VALUE":
		w.text("OVERRIDING USER VALUE ")
	case "OVERRIDING_SYSTEM_VALUE":
		w.text("OVERRIDING SYSTEM VALUE ")
	}
}

func (w *deparseWriter) insert(n *p.InsertStmt) {
	w.with(n.WithClause)
	w.text("INSERT INTO ")
	w.relation(n.Relation, deparseRelationInsert)
	w.text(" ")
	if len(n.Cols) > 0 {
		w.text("(")
		w.insertColumns(n.Cols)
		w.text(") ")
	}
	w.override(n.Override)
	if n.SelectStmt != nil {
		w.stmt(n.SelectStmt)
	} else {
		w.text("DEFAULT VALUES")
	}
	if n.OnConflictClause != nil {
		w.conflict(n.OnConflictClause)
	}
	w.returning(n.ReturningClause)
}

func (w *deparseWriter) conflict(n *p.OnConflictClause) {
	w.text(" ON CONFLICT ")
	if infer := n.Infer; infer != nil {
		if len(infer.IndexElems) > 0 {
			w.text("(")
			for i, node := range infer.IndexElems {
				if i > 0 {
					w.text(", ")
				}
				w.indexElem(node.GetIndexElem())
			}
			w.text(") ")
		}
		if infer.Conname != "" {
			w.text("ON CONSTRAINT ")
			w.ident(infer.Conname)
			w.text(" ")
		}
		if infer.WhereClause != nil {
			w.trimSpace()
			w.where(infer.WhereClause)
			w.text(" ")
		}
	}
	switch n.Action.String() {
	case "ONCONFLICT_NOTHING":
		w.text("DO NOTHING")
	case "ONCONFLICT_UPDATE":
		w.text("DO UPDATE")
	default:
		w.invalid("invalid ON CONFLICT action %v", n.Action)
		return
	}
	if len(n.TargetList) > 0 {
		w.text(" SET ")
		w.assignments(n.TargetList)
	}
	w.where(n.WhereClause)
}

func (w *deparseWriter) assignments(nodes []*p.Node) {
	for i := 0; i < len(nodes); {
		if i > 0 {
			w.text(", ")
		}
		n := nodes[i].GetResTarget()
		if n == nil {
			w.invalid("invalid assignment target %T", nodes[i].GetNode())
			return
		}
		if multi := n.Val.GetMultiAssignRef(); multi != nil {
			count := int(multi.Ncolumns)
			if count <= 0 || count > len(nodes)-i {
				w.invalid("invalid multiple assignment arity %d", count)
				return
			}
			w.text("(")
			for j := range count {
				if j > 0 {
					w.text(", ")
				}
				target := nodes[i+j].GetResTarget()
				if target == nil {
					w.invalid("invalid multiple assignment target")
					return
				}
				w.ident(target.Name)
				w.indirection(target.Indirection)
			}
			w.text(") = ")
			w.expr(multi.Source)
			i += count
		} else {
			w.ident(n.Name)
			w.indirection(n.Indirection)
			w.text(" = ")
			w.expr(n.Val)
			i++
		}
	}
}

func (w *deparseWriter) returning(n *p.ReturningClause) {
	if n == nil {
		return
	}
	w.text(" RETURNING ")
	if len(n.Options) > 0 {
		w.text("WITH (")
		for i, node := range n.Options {
			if i > 0 {
				w.text(", ")
			}
			option := node.GetReturningOption()
			if option == nil {
				w.invalid("invalid RETURNING option %T", node.GetNode())
				return
			}
			switch option.Option.String() {
			case "RETURNING_OPTION_OLD":
				w.text("OLD AS ")
			case "RETURNING_OPTION_NEW":
				w.text("NEW AS ")
			default:
				w.invalid("invalid RETURNING option %v", option.Option)
				return
			}
			w.ident(option.Value)
		}
		w.text(") ")
	}
	w.targets(n.Exprs)
}

func (w *deparseWriter) update(n *p.UpdateStmt) {
	w.with(n.WithClause)
	w.text("UPDATE ")
	w.relation(n.Relation, deparseRelationNormal)
	if len(n.TargetList) > 0 {
		w.text(" SET ")
		w.assignments(n.TargetList)
	}
	if len(n.FromClause) > 0 {
		w.text(" FROM ")
		w.tables(n.FromClause)
	}
	w.where(n.WhereClause)
	w.returning(n.ReturningClause)
}

func (w *deparseWriter) delete(n *p.DeleteStmt) {
	w.with(n.WithClause)
	w.text("DELETE FROM ")
	w.relation(n.Relation, deparseRelationNormal)
	if len(n.UsingClause) > 0 {
		w.text(" USING ")
		w.tables(n.UsingClause)
	}
	w.where(n.WhereClause)
	w.returning(n.ReturningClause)
}

func (w *deparseWriter) merge(n *p.MergeStmt) {
	w.with(n.WithClause)
	w.text("MERGE INTO ")
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" USING ")
	w.table(n.SourceRelation)
	w.text(" ON ")
	w.expr(n.JoinCondition)
	for _, node := range n.MergeWhenClauses {
		clause := node.GetMergeWhenClause()
		if clause == nil {
			w.invalid("invalid MERGE clause %T", node.GetNode())
			return
		}
		w.text(" WHEN ")
		switch clause.MatchKind.String() {
		case "MERGE_WHEN_MATCHED":
			w.text("MATCHED")
		case "MERGE_WHEN_NOT_MATCHED_BY_SOURCE":
			w.text("NOT MATCHED BY SOURCE")
		case "MERGE_WHEN_NOT_MATCHED_BY_TARGET":
			w.text("NOT MATCHED")
		default:
			w.invalid("invalid MERGE match kind %v", clause.MatchKind)
			return
		}
		if clause.Condition != nil {
			w.text(" AND ")
			w.expr(clause.Condition)
		}
		w.text(" THEN ")
		switch clause.CommandType.String() {
		case "CMD_INSERT":
			w.text("INSERT ")
			if len(clause.TargetList) > 0 {
				w.text("(")
				w.insertColumns(clause.TargetList)
				w.text(") ")
			}
			w.override(clause.Override)
			if len(clause.Values) > 0 {
				w.text("VALUES (")
				w.exprs(clause.Values)
				w.text(")")
			} else {
				w.text("DEFAULT VALUES ")
			}
		case "CMD_UPDATE":
			w.text("UPDATE SET ")
			w.assignments(clause.TargetList)
		case "CMD_DELETE":
			w.text("DELETE")
		case "CMD_NOTHING":
			w.text("DO NOTHING")
		default:
			w.invalid("unpermitted MERGE command type %v", clause.CommandType)
			return
		}
	}
	w.returning(n.ReturningClause)
	w.trimSpace()
}
