package pg_query

import (
	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func (w *deparseWriter) create(n *p.CreateStmt, foreign bool) {
	if n == nil {
		w.invalid("nil CREATE TABLE statement")
		return
	}
	w.text("CREATE ")
	if foreign {
		w.text("FOREIGN ")
	}
	rp := ""
	if n.Relation != nil {
		rp = n.Relation.Relpersistence
	}
	w.persistence(rp)
	w.text("TABLE ")
	if n.IfNotExists {
		w.text("IF NOT EXISTS ")
	}
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" ")
	if n.OfTypename != nil {
		w.text("OF ")
		w.typeName(n.OfTypename)
		w.text(" ")
	}
	if n.Partbound != nil {
		if len(n.InhRelations) != 1 {
			w.invalid("partitioned table requires one parent")
			return
		}
		w.text("PARTITION OF ")
		w.relation(n.InhRelations[0].GetRangeVar(), deparseRelationNormal)
		w.text(" ")
	}
	if len(n.TableElts) > 0 {
		w.text("(")
		for i, node := range n.TableElts {
			if i > 0 {
				w.text(", ")
			}
			switch t := node.Node.(type) {
			case *p.Node_ColumnDef:
				w.columnDef(t.ColumnDef)
			case *p.Node_TableLikeClause:
				w.tableLike(t.TableLikeClause)
			case *p.Node_Constraint:
				w.constraint(t.Constraint, false)
			default:
				w.invalid("invalid table element %T", node.Node)
				return
			}
		}
		w.text(") ")
	} else if n.Partbound == nil && n.OfTypename == nil {
		w.text("() ")
	}
	if n.Partbound != nil {
		w.partitionBound(n.Partbound)
		w.text(" ")
	} else {
		w.optInherit(n.InhRelations)
	}
	if n.Partspec != nil {
		w.partitionSpec(n.Partspec)
		w.text(" ")
	}
	if n.AccessMethod != "" {
		w.text("USING ")
		w.ident(n.AccessMethod)
		w.text(" ")
	}
	w.withOptions(n.Options)
	switch n.Oncommit.String() {
	case "ONCOMMIT_PRESERVE_ROWS":
		w.text("ON COMMIT PRESERVE ROWS ")
	case "ONCOMMIT_DELETE_ROWS":
		w.text("ON COMMIT DELETE ROWS ")
	case "ONCOMMIT_DROP":
		w.text("ON COMMIT DROP ")
	}
	if n.Tablespacename != "" {
		w.text("TABLESPACE ")
		w.ident(n.Tablespacename)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) tableLike(n *p.TableLikeClause) {
	if n == nil {
		w.invalid("nil LIKE clause")
		return
	}
	w.text("LIKE ")
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" ")
	const all = uint32(2147483647)
	if n.Options == all {
		w.text("INCLUDING ALL ")
	} else {
		if n.Options&1 != 0 {
			w.text("INCLUDING COMMENTS ")
		}
		if n.Options&2 != 0 {
			w.text("INCLUDING COMPRESSION ")
		}
		if n.Options&4 != 0 {
			w.text("INCLUDING CONSTRAINTS ")
		}
		if n.Options&8 != 0 {
			w.text("INCLUDING DEFAULTS ")
		}
		if n.Options&32 != 0 {
			w.text("INCLUDING IDENTITY ")
		}
		if n.Options&16 != 0 {
			w.text("INCLUDING GENERATED ")
		}
		if n.Options&64 != 0 {
			w.text("INCLUDING INDEXES ")
		}
		if n.Options&128 != 0 {
			w.text("INCLUDING STATISTICS ")
		}
		if n.Options&256 != 0 {
			w.text("INCLUDING STORAGE ")
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) optInherit(nodes []*p.Node) {
	if len(nodes) == 0 {
		return
	}
	w.text("INHERITS (")
	w.qualifiedNameList(nodes)
	w.text(") ")
}

func (w *deparseWriter) partitionElem(n *p.PartitionElem) {
	if n == nil {
		w.invalid("nil partition element")
		return
	}
	if n.Name != "" {
		w.ident(n.Name)
		w.text(" ")
	} else if n.Expr != nil {
		w.text("(")
		w.expr(n.Expr)
		w.text(") ")
	}
	if len(n.Collation) > 0 {
		w.text("COLLATE ")
		w.name(n.Collation)
		w.text(" ")
	}
	if len(n.Opclass) > 0 {
		w.name(n.Opclass)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) partitionSpec(n *p.PartitionSpec) {
	if n == nil {
		w.invalid("nil partition spec")
		return
	}
	w.text("PARTITION BY ")
	switch n.Strategy.String() {
	case "PARTITION_STRATEGY_LIST":
		w.text("LIST")
	case "PARTITION_STRATEGY_HASH":
		w.text("HASH")
	case "PARTITION_STRATEGY_RANGE":
		w.text("RANGE")
	default:
		w.invalid("invalid partition strategy %v", n.Strategy)
		return
	}
	w.text(" (")
	for i, node := range n.PartParams {
		if i > 0 {
			w.text(", ")
		}
		w.partitionElem(node.GetPartitionElem())
	}
	w.text(")")
}

func (w *deparseWriter) partitionBound(n *p.PartitionBoundSpec) {
	if n == nil {
		w.invalid("nil partition bound")
		return
	}
	if n.IsDefault {
		w.text("DEFAULT")
		return
	}
	w.text("FOR VALUES ")
	switch n.Strategy {
	case "h":
		w.text("WITH (MODULUS ")
		w.integer(int64(n.Modulus))
		w.text(", REMAINDER ")
		w.integer(int64(n.Remainder))
		w.text(")")
	case "l":
		w.text("IN (")
		w.exprs(n.Listdatums)
		w.text(")")
	case "r":
		w.text("FROM (")
		w.exprs(n.Lowerdatums)
		w.text(") TO (")
		w.exprs(n.Upperdatums)
		w.text(")")
	default:
		w.invalid("invalid partition bound strategy %q", n.Strategy)
	}
}

func (w *deparseWriter) partitionCmd(n *p.PartitionCmd) {
	if n == nil {
		w.invalid("nil partition command")
		return
	}
	w.relation(n.Name, deparseRelationNormal)
	if n.Bound != nil {
		w.text(" ")
		w.partitionBound(n.Bound)
	}
	if n.Concurrent {
		w.text(" CONCURRENTLY")
	}
}

func (w *deparseWriter) replicaIdentity(n *p.ReplicaIdentityStmt) {
	if n == nil {
		w.invalid("nil replica identity")
		return
	}
	switch n.IdentityType {
	case "n":
		w.text("NOTHING")
	case "f":
		w.text("FULL")
	case "d":
		w.text("DEFAULT")
	case "i":
		w.text("USING INDEX ")
		w.ident(n.Name)
	default:
		w.invalid("invalid replica identity %q", n.IdentityType)
	}
}

func (w *deparseWriter) alterTableCmd(n *p.AlterTableCmd, alterType bool) {
	if n == nil {
		w.invalid("nil ALTER TABLE command")
		return
	}
	var options string
	trailingMissingOk := false
	switch n.Subtype.String() {
	case "AT_AddColumn":
		if alterType {
			w.text("ADD ATTRIBUTE ")
		} else {
			w.text("ADD COLUMN ")
		}
	case "AT_ColumnDefault":
		w.text("ALTER COLUMN ")
		if n.Def != nil {
			options = "SET DEFAULT"
		} else {
			options = "DROP DEFAULT"
		}
	case "AT_DropNotNull":
		w.text("ALTER COLUMN ")
		options = "DROP NOT NULL"
	case "AT_SetNotNull":
		w.text("ALTER COLUMN ")
		options = "SET NOT NULL"
	case "AT_DropExpression":
		w.text("ALTER COLUMN ")
		options = "DROP EXPRESSION"
		trailingMissingOk = true
	case "AT_SetStatistics":
		w.text("ALTER COLUMN ")
		options = "SET STATISTICS"
	case "AT_SetOptions":
		w.text("ALTER COLUMN ")
		options = "SET"
	case "AT_ResetOptions":
		w.text("ALTER COLUMN ")
		options = "RESET"
	case "AT_SetStorage":
		w.text("ALTER COLUMN ")
		options = "SET STORAGE"
	case "AT_SetCompression":
		w.text("ALTER COLUMN ")
		options = "SET COMPRESSION"
	case "AT_DropColumn":
		if alterType {
			w.text("DROP ATTRIBUTE ")
		} else {
			w.text("DROP ")
		}
	case "AT_AddIndex":
		w.text("ADD INDEX ")
	case "AT_AddConstraint":
		w.text("ADD ")
	case "AT_AlterConstraint":
		w.text("ALTER ")
	case "AT_ValidateConstraint":
		w.text("VALIDATE CONSTRAINT ")
	case "AT_DropConstraint":
		w.text("DROP CONSTRAINT ")
	case "AT_AlterColumnType":
		if alterType {
			w.text("ALTER ATTRIBUTE ")
		} else {
			w.text("ALTER COLUMN ")
		}
		options = "TYPE"
	case "AT_AlterColumnGenericOptions":
		w.text("ALTER COLUMN ")
	case "AT_GenericOptions":
	case "AT_ChangeOwner":
		w.text("OWNER TO ")
		w.role(n.Newowner)
		w.trimSpace()
		return
	case "AT_ClusterOn":
		w.text("CLUSTER ON ")
	case "AT_DropCluster":
		w.text("SET WITHOUT CLUSTER")
		w.trimSpace()
		return
	case "AT_SetLogged":
		w.text("SET LOGGED")
		w.trimSpace()
		return
	case "AT_SetUnLogged":
		w.text("SET UNLOGGED")
		w.trimSpace()
		return
	case "AT_DropOids":
		w.text("SET WITHOUT OIDS")
		w.trimSpace()
		return
	case "AT_SetAccessMethod":
		w.text("SET ACCESS METHOD ")
	case "AT_SetTableSpace":
		w.text("SET TABLESPACE ")
	case "AT_SetRelOptions":
		w.text("SET ")
	case "AT_ResetRelOptions":
		w.text("RESET ")
	case "AT_EnableTrig":
		w.text("ENABLE TRIGGER ")
	case "AT_EnableAlwaysTrig":
		w.text("ENABLE ALWAYS TRIGGER ")
	case "AT_EnableReplicaTrig":
		w.text("ENABLE REPLICA TRIGGER ")
	case "AT_DisableTrig":
		w.text("DISABLE TRIGGER ")
	case "AT_EnableTrigAll":
		w.text("ENABLE TRIGGER ALL")
		w.trimSpace()
		return
	case "AT_DisableTrigAll":
		w.text("DISABLE TRIGGER ALL")
		w.trimSpace()
		return
	case "AT_EnableTrigUser":
		w.text("ENABLE TRIGGER USER")
		w.trimSpace()
		return
	case "AT_DisableTrigUser":
		w.text("DISABLE TRIGGER USER")
		w.trimSpace()
		return
	case "AT_EnableRule":
		w.text("ENABLE RULE ")
	case "AT_EnableAlwaysRule":
		w.text("ENABLE ALWAYS RULE ")
	case "AT_EnableReplicaRule":
		w.text("ENABLE REPLICA RULE ")
	case "AT_DisableRule":
		w.text("DISABLE RULE ")
	case "AT_AddInherit":
		w.text("INHERIT ")
	case "AT_DropInherit":
		w.text("NO INHERIT ")
	case "AT_AddOf":
		w.text("OF ")
	case "AT_DropOf":
		w.text("NOT OF")
		w.trimSpace()
		return
	case "AT_ReplicaIdentity":
		w.text("REPLICA IDENTITY ")
	case "AT_EnableRowSecurity":
		w.text("ENABLE ROW LEVEL SECURITY")
		w.trimSpace()
		return
	case "AT_DisableRowSecurity":
		w.text("DISABLE ROW LEVEL SECURITY")
		w.trimSpace()
		return
	case "AT_ForceRowSecurity":
		w.text("FORCE ROW LEVEL SECURITY")
		w.trimSpace()
		return
	case "AT_NoForceRowSecurity":
		w.text("NO FORCE ROW LEVEL SECURITY")
		w.trimSpace()
		return
	case "AT_AttachPartition":
		w.text("ATTACH PARTITION ")
	case "AT_DetachPartition":
		w.text("DETACH PARTITION ")
	case "AT_DetachPartitionFinalize":
		w.text("DETACH PARTITION ")
	case "AT_AddIdentity":
		w.text("ALTER ")
		options = "ADD"
	case "AT_SetIdentity":
		w.text("ALTER ")
	case "AT_DropIdentity":
		w.text("ALTER COLUMN ")
		options = "DROP IDENTITY"
		trailingMissingOk = true
	case "AT_SetExpression":
		w.text("ALTER COLUMN ")
	default:
		w.invalid("invalid ALTER TABLE command %v", n.Subtype)
		return
	}
	if n.MissingOk && !trailingMissingOk {
		if n.Subtype.String() == "AT_AddColumn" {
			w.text("IF NOT EXISTS ")
		} else {
			w.text("IF EXISTS ")
		}
	}
	if n.Name != "" {
		w.ident(n.Name)
		w.text(" ")
	} else if n.Subtype.String() == "AT_SetAccessMethod" {
		w.text(" DEFAULT")
	}
	if n.Num > 0 {
		w.integer(int64(n.Num))
		w.text(" ")
	}
	if options != "" {
		w.text(options)
		w.text(" ")
	}
	if n.MissingOk && trailingMissingOk {
		w.text("IF EXISTS ")
	}
	switch n.Subtype.String() {
	case "AT_AttachPartition", "AT_DetachPartition":
		w.partitionCmd(n.Def.GetPartitionCmd())
		w.text(" ")
	case "AT_DetachPartitionFinalize":
		w.partitionCmd(n.Def.GetPartitionCmd())
		w.text(" FINALIZE ")
	case "AT_AddColumn", "AT_AlterColumnType":
		w.columnDef(n.Def.GetColumnDef())
		w.text(" ")
	case "AT_ColumnDefault":
		if n.Def != nil {
			w.expr(n.Def)
			w.text(" ")
		}
	case "AT_SetStatistics":
		if n.Def != nil {
			if v := n.Def.GetInteger(); v != nil {
				w.integer(int64(v.Ival))
			} else {
				w.expr(n.Def)
			}
		} else {
			w.text("DEFAULT")
		}
		w.text(" ")
	case "AT_SetOptions", "AT_ResetOptions", "AT_SetRelOptions", "AT_ResetRelOptions":
		w.relOptions(n.Def.GetList().GetItems())
		w.text(" ")
	case "AT_SetStorage":
		w.ident(deparseString(n.Def))
		w.text(" ")
	case "AT_SetCompression":
		if deparseString(n.Def) == "default" {
			w.text("DEFAULT")
		} else {
			w.ident(deparseString(n.Def))
		}
		w.text(" ")
	case "AT_AddIdentity", "AT_AddConstraint":
		w.constraint(n.Def.GetConstraint(), false)
		w.text(" ")
	case "AT_AlterConstraint":
		w.atAlterConstraint(n.Def.GetAtalterConstraint())
		w.text(" ")
	case "AT_SetIdentity":
		w.alterIdentityOpts(n.Def.GetList().GetItems())
		w.text(" ")
	case "AT_AlterColumnGenericOptions", "AT_GenericOptions":
		w.genericOptions(n.Def.GetList().GetItems(), true)
		w.text("  ")
	case "AT_AddInherit", "AT_DropInherit":
		w.relation(n.Def.GetRangeVar(), deparseRelationNormal)
		w.text(" ")
	case "AT_AddOf":
		w.typeName(n.Def.GetTypeName())
		w.text(" ")
	case "AT_ReplicaIdentity":
		w.replicaIdentity(n.Def.GetReplicaIdentityStmt())
		w.text(" ")
	case "AT_SetExpression":
		w.text("SET EXPRESSION AS (")
		w.expr(n.Def)
		w.text(")")
	}
	w.dropBehavior(n.Behavior)
	w.trimSpace()
}

func (w *deparseWriter) alterTableObjType(t p.ObjectType) bool {
	switch t.String() {
	case "OBJECT_TABLE":
		w.text("TABLE ")
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
	case "OBJECT_INDEX":
		w.text("INDEX ")
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
	case "OBJECT_VIEW":
		w.text("VIEW ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	case "OBJECT_TYPE":
		w.text("TYPE ")
		return true
	default:
		w.invalid("invalid ALTER object type %v", t)
	}
	return false
}

func (w *deparseWriter) alterTable(n *p.AlterTableStmt) {
	if n == nil {
		w.invalid("nil ALTER TABLE statement")
		return
	}
	w.text("ALTER ")
	alterType := w.alterTableObjType(n.Objtype)
	if w.err != nil {
		return
	}
	if n.MissingOk {
		w.text("IF EXISTS ")
	}
	mode := deparseRelationNormal
	if alterType {
		mode = deparseRelationCreate
	}
	w.relation(n.Relation, mode)
	w.text(" ")
	for i, node := range n.Cmds {
		if i > 0 {
			w.text(", ")
		}
		w.alterTableCmd(node.GetAlterTableCmd(), alterType)
	}
	w.trimSpace()
}

func (w *deparseWriter) alterTableMoveAll(n *p.AlterTableMoveAllStmt) {
	if n == nil {
		w.invalid("nil ALTER ALL statement")
		return
	}
	w.text("ALTER ")
	w.alterTableObjType(n.Objtype)
	if w.err != nil {
		return
	}
	w.text("ALL IN TABLESPACE ")
	w.ident(n.OrigTablespacename)
	w.text(" ")
	if len(n.Roles) > 0 {
		w.text("OWNED BY ")
		w.roles(n.Roles)
		w.text(" ")
	}
	w.text("SET TABLESPACE ")
	w.ident(n.NewTablespacename)
	w.text(" ")
	if n.Nowait {
		w.text("NOWAIT")
	}
	w.trimSpace()
}

func (w *deparseWriter) alterDomain(n *p.AlterDomainStmt) {
	if n == nil {
		w.invalid("nil ALTER DOMAIN statement")
		return
	}
	w.text("ALTER DOMAIN ")
	w.name(n.TypeName)
	w.text(" ")
	switch n.Subtype {
	case "T":
		if n.Def != nil {
			w.text("SET DEFAULT ")
			w.expr(n.Def)
		} else {
			w.text("DROP DEFAULT")
		}
	case "N":
		w.text("DROP NOT NULL")
	case "O":
		w.text("SET NOT NULL")
	case "C":
		w.text("ADD ")
		w.constraint(n.Def.GetConstraint(), true)
	case "X":
		w.text("DROP CONSTRAINT ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.ident(n.Name)
		w.dropBehavior(n.Behavior)
		w.trimSpace()
		return
	case "V":
		w.text("VALIDATE CONSTRAINT ")
		w.ident(n.Name)
	default:
		w.invalid("invalid ALTER DOMAIN subtype %q", n.Subtype)
	}
}

func (w *deparseWriter) createFunction(n *p.CreateFunctionStmt) {
	if n == nil {
		w.invalid("nil CREATE FUNCTION statement")
		return
	}
	w.text("CREATE ")
	if n.Replace {
		w.text("OR REPLACE ")
	}
	if n.IsProcedure {
		w.text("PROCEDURE ")
	} else {
		w.text("FUNCTION ")
	}
	w.name(n.Funcname)
	w.text("(")
	tableFunc := false
	for _, node := range n.Parameters {
		if fp := node.GetFunctionParameter(); fp != nil && fp.Mode.String() == "FUNC_PARAM_TABLE" {
			tableFunc = true
		}
	}
	sep := false
	for _, node := range n.Parameters {
		fp := node.GetFunctionParameter()
		if fp == nil {
			w.invalid("invalid function parameter")
			return
		}
		if fp.Mode.String() == "FUNC_PARAM_TABLE" {
			continue
		}
		if sep {
			w.text(", ")
		}
		w.functionParameter(fp)
		sep = true
	}
	w.text(") ")
	if tableFunc {
		w.text("RETURNS TABLE (")
		sep = false
		for _, node := range n.Parameters {
			fp := node.GetFunctionParameter()
			if fp.Mode.String() != "FUNC_PARAM_TABLE" {
				continue
			}
			if sep {
				w.text(", ")
			}
			w.functionParameter(fp)
			sep = true
		}
		w.text(") ")
	} else if n.ReturnType != nil {
		w.text("RETURNS ")
		w.typeName(n.ReturnType)
		w.text(" ")
	}
	for _, node := range n.Options {
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid function option")
			return
		}
		w.funcOptItem(d, true)
		w.text(" ")
	}
	if n.SqlBody != nil {
		if ret := n.SqlBody.GetReturnStmt(); ret != nil {
			w.text("RETURN ")
			w.expr(ret.Returnval)
			w.text(" ")
		} else if body := n.SqlBody.GetList(); body != nil {
			w.text("BEGIN ATOMIC ")
			stmts := body.Items
			if len(stmts) > 0 {
				if inner := stmts[0].GetList(); inner != nil {
					stmts = inner.Items
				} else if stmts[0] == nil || stmts[0].Node == nil {
					stmts = nil
				}
			}
			for _, stmt := range stmts {
				if ret := stmt.GetReturnStmt(); ret != nil {
					w.text("RETURN ")
					w.expr(ret.Returnval)
					w.text("; ")
				} else {
					w.stmt(stmt)
					w.text("; ")
				}
			}
			w.text("END ")
		} else {
			w.stmt(n.SqlBody)
			w.text(" ")
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) createCast(n *p.CreateCastStmt) {
	if n == nil {
		w.invalid("nil CREATE CAST statement")
		return
	}
	w.text("CREATE CAST (")
	w.typeName(n.Sourcetype)
	w.text(" AS ")
	w.typeName(n.Targettype)
	w.text(") ")
	if n.Func != nil {
		w.text("WITH FUNCTION ")
		w.objectWithArgs(n.Func, false)
		w.text(" ")
	} else if n.Inout {
		w.text("WITH INOUT ")
	} else {
		w.text("WITHOUT FUNCTION ")
	}
	switch n.Context.String() {
	case "COERCION_IMPLICIT":
		w.text("AS IMPLICIT")
	case "COERCION_ASSIGNMENT":
		w.text("AS ASSIGNMENT")
	case "COERCION_EXPLICIT":
	default:
		w.invalid("invalid cast context %v", n.Context)
		return
	}
	w.trimSpace()
}

func (w *deparseWriter) createOpClass(n *p.CreateOpClassStmt) {
	if n == nil {
		w.invalid("nil CREATE OPERATOR CLASS statement")
		return
	}
	w.text("CREATE OPERATOR CLASS ")
	w.name(n.Opclassname)
	w.text(" ")
	if n.IsDefault {
		w.text("DEFAULT ")
	}
	w.text("FOR TYPE ")
	w.typeName(n.Datatype)
	w.text(" USING ")
	w.ident(n.Amname)
	w.text(" ")
	if len(n.Opfamilyname) > 0 {
		w.text("FAMILY ")
		w.name(n.Opfamilyname)
		w.text(" ")
	}
	w.text("AS ")
	w.opclassItems(n.Items)
}

func (w *deparseWriter) createTrig(n *p.CreateTrigStmt) {
	if n == nil {
		w.invalid("nil CREATE TRIGGER statement")
		return
	}
	w.text("CREATE ")
	if n.Replace {
		w.text("OR REPLACE ")
	}
	if n.Isconstraint {
		w.text("CONSTRAINT ")
	}
	w.text("TRIGGER ")
	w.ident(n.Trigname)
	w.text(" ")
	switch n.Timing {
	case 2:
		w.text("BEFORE ")
	case 0:
		w.text("AFTER ")
	case 64:
		w.text("INSTEAD OF ")
	default:
		w.invalid("invalid trigger timing %d", n.Timing)
		return
	}
	first := true
	if n.Events&4 != 0 {
		w.text("INSERT ")
		first = false
	}
	if n.Events&8 != 0 {
		if !first {
			w.text("OR ")
		}
		w.text("DELETE ")
		first = false
	}
	if n.Events&16 != 0 {
		if !first {
			w.text("OR ")
		}
		w.text("UPDATE ")
		if len(n.Columns) > 0 {
			w.text("OF ")
			w.columnList(n.Columns)
			w.text(" ")
		}
		first = false
	}
	if n.Events&32 != 0 {
		if !first {
			w.text("OR ")
		}
		w.text("TRUNCATE ")
	}
	w.text("ON ")
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" ")
	if len(n.TransitionRels) > 0 {
		w.text("REFERENCING ")
		for _, node := range n.TransitionRels {
			t := node.GetTriggerTransition()
			if t == nil {
				w.invalid("invalid trigger transition")
				return
			}
			if t.IsNew {
				w.text("NEW ")
			} else {
				w.text("OLD ")
			}
			if t.IsTable {
				w.text("TABLE ")
			} else {
				w.text("ROW ")
			}
			w.ident(t.Name)
			w.text(" ")
		}
	}
	if n.Constrrel != nil {
		w.text("FROM ")
		w.relation(n.Constrrel, deparseRelationNormal)
		w.text(" ")
	}
	if n.Deferrable {
		w.text("DEFERRABLE ")
	}
	if n.Initdeferred {
		w.text("INITIALLY DEFERRED ")
	}
	if n.Row {
		w.text("FOR EACH ROW ")
	}
	if n.WhenClause != nil {
		w.text("WHEN (")
		w.expr(n.WhenClause)
		w.text(") ")
	}
	w.text("EXECUTE FUNCTION ")
	w.name(n.Funcname)
	w.text("(")
	for i, node := range n.Args {
		if i > 0 {
			w.text(", ")
		}
		w.literal(deparseString(node))
	}
	w.text(")")
}

func (w *deparseWriter) alterEventTrig(n *p.AlterEventTrigStmt) {
	if n == nil {
		w.invalid("nil ALTER EVENT TRIGGER statement")
		return
	}
	w.text("ALTER EVENT TRIGGER ")
	w.ident(n.Trigname)
	w.text(" ")
	switch n.Tgenabled {
	case "O":
		w.text("ENABLE")
	case "R":
		w.text("ENABLE REPLICA")
	case "A":
		w.text("ENABLE ALWAYS")
	case "D":
		w.text("DISABLE")
	default:
		w.invalid("invalid event trigger state %q", n.Tgenabled)
	}
}

func (w *deparseWriter) alterExtensionContents(n *p.AlterExtensionContentsStmt) {
	if n == nil {
		w.invalid("nil ALTER EXTENSION statement")
		return
	}
	w.text("ALTER EXTENSION ")
	w.ident(n.Extname)
	w.text(" ")
	switch n.Action {
	case 1:
		w.text("ADD ")
	case -1:
		w.text("DROP ")
	default:
		w.invalid("invalid extension contents action %d", n.Action)
		return
	}
	switch n.Objtype.String() {
	case "OBJECT_ACCESS_METHOD":
		w.text("ACCESS METHOD ")
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
	case "OBJECT_CAST":
		w.text("CAST ")
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
	case "OBJECT_CONVERSION":
		w.text("CONVERSION ")
	case "OBJECT_DOMAIN":
		w.text("DOMAIN ")
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
	case "OBJECT_OPCLASS":
		w.text("OPERATOR CLASS ")
	case "OBJECT_OPFAMILY":
		w.text("OPERATOR FAMILY ")
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
	case "OBJECT_TABLE":
		w.text("TABLE ")
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
	case "OBJECT_VIEW":
		w.text("VIEW ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
	case "OBJECT_FDW":
		w.text("FOREIGN DATA WRAPPER ")
	case "OBJECT_FOREIGN_SERVER":
		w.text("SERVER ")
	case "OBJECT_TRANSFORM":
		w.text("TRANSFORM ")
	case "OBJECT_TYPE":
		w.text("TYPE ")
	default:
		w.invalid("invalid extension object type %v", n.Objtype)
		return
	}
	switch n.Objtype.String() {
	case "OBJECT_COLLATION", "OBJECT_CONVERSION", "OBJECT_TABLE", "OBJECT_TSPARSER",
		"OBJECT_TSDICTIONARY", "OBJECT_TSTEMPLATE", "OBJECT_TSCONFIGURATION",
		"OBJECT_SEQUENCE", "OBJECT_VIEW", "OBJECT_MATVIEW", "OBJECT_FOREIGN_TABLE":
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_ACCESS_METHOD", "OBJECT_LANGUAGE", "OBJECT_SCHEMA",
		"OBJECT_EVENT_TRIGGER", "OBJECT_FDW", "OBJECT_FOREIGN_SERVER":
		w.ident(deparseString(n.Object))
	case "OBJECT_AGGREGATE":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_CAST":
		items := n.Object.GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid CAST object")
			return
		}
		w.text("(")
		w.typeName(items[0].GetTypeName())
		w.text(" AS ")
		w.typeName(items[1].GetTypeName())
		w.text(")")
	case "OBJECT_DOMAIN", "OBJECT_TYPE":
		w.typeName(n.Object.GetTypeName())
	case "OBJECT_FUNCTION", "OBJECT_PROCEDURE", "OBJECT_ROUTINE":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_OPERATOR":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), true)
	case "OBJECT_OPCLASS", "OBJECT_OPFAMILY":
		items := n.Object.GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid operator class object")
			return
		}
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_TRANSFORM":
		items := n.Object.GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid TRANSFORM object")
			return
		}
		w.text("FOR ")
		w.typeName(items[0].GetTypeName())
		w.text(" LANGUAGE ")
		w.ident(deparseString(items[1]))
	}
}

func (w *deparseWriter) index(n *p.IndexStmt) {
	if n == nil {
		w.invalid("nil CREATE INDEX statement")
		return
	}
	w.text("CREATE ")
	if n.Unique {
		w.text("UNIQUE ")
	}
	w.text("INDEX ")
	if n.Concurrent {
		w.text("CONCURRENTLY ")
	}
	if n.IfNotExists {
		w.text("IF NOT EXISTS ")
	}
	if n.Idxname != "" {
		w.ident(n.Idxname)
		w.text(" ")
	}
	w.text("ON ")
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" ")
	if n.AccessMethod != "" {
		w.text("USING ")
		w.ident(n.AccessMethod)
		w.text(" ")
	}
	w.text("(")
	w.indexElems(n.IndexParams)
	w.text(") ")
	if len(n.IndexIncludingParams) > 0 {
		w.text("INCLUDE (")
		w.indexElems(n.IndexIncludingParams)
		w.text(") ")
	}
	if n.NullsNotDistinct {
		w.text("NULLS NOT DISTINCT ")
	}
	w.withOptions(n.Options)
	if n.TableSpace != "" {
		w.text("TABLESPACE ")
		w.ident(n.TableSpace)
		w.text(" ")
	}
	if n.WhereClause != nil {
		w.text("WHERE ")
		w.expr(n.WhereClause)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) createTableAs(n *p.CreateTableAsStmt) {
	if n == nil {
		w.invalid("nil CREATE TABLE AS statement")
		return
	}
	w.text("CREATE ")
	w.persistence(n.Into.Rel.GetRelpersistence())
	switch n.Objtype.String() {
	case "OBJECT_TABLE":
		w.text("TABLE ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	default:
		w.invalid("invalid CREATE TABLE AS object type %v", n.Objtype)
		return
	}
	if n.IfNotExists {
		w.text("IF NOT EXISTS ")
	}
	w.into(n.Into)
	w.text(" AS ")
	if exec := n.Query.GetExecuteStmt(); exec != nil {
		w.utilityStmt(n.Query)
	} else {
		w.selectStmt(n.Query.GetSelectStmt(), false)
	}
	w.text(" ")
	if n.Into.SkipData {
		if n.Query.GetExecuteStmt() == nil {
			w.text(" ")
		}
		w.text("WITH NO DATA ")
	}
	w.trimSpace()
}

func (w *deparseWriter) createRole(n *p.CreateRoleStmt) {
	if n == nil {
		w.invalid("nil CREATE ROLE statement")
		return
	}
	w.text("CREATE ")
	switch n.StmtType.String() {
	case "ROLESTMT_ROLE":
		w.text("ROLE ")
	case "ROLESTMT_USER":
		w.text("USER ")
	case "ROLESTMT_GROUP":
		w.text("GROUP ")
	default:
		w.invalid("invalid role statement type %v", n.StmtType)
		return
	}
	w.ident(n.Role)
	w.text(" ")
	if len(n.Options) > 0 {
		w.text("WITH ")
		for _, node := range n.Options {
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid role option")
				return
			}
			w.createRoleElem(d)
			w.text(" ")
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) createRoleElem(d *p.DefElem) {
	switch d.Defname {
	case "sysid":
		w.text("SYSID ")
		w.integer(ival(d.Arg))
	case "adminmembers":
		w.text("ADMIN ")
		w.roles(d.Arg.GetList().GetItems())
	case "rolemembers":
		w.text("ROLE ")
		w.roles(d.Arg.GetList().GetItems())
	case "addroleto":
		w.text("IN ROLE ")
		w.roles(d.Arg.GetList().GetItems())
	default:
		w.alterRoleElem(d)
	}
}

func (w *deparseWriter) alterRoleElem(d *p.DefElem) {
	switch d.Defname {
	case "password":
		w.text("PASSWORD ")
		if d.Arg == nil {
			w.text("NULL")
		} else if pr := d.Arg.GetParamRef(); pr != nil {
			if pr.Number == 0 {
				w.text("?")
			} else {
				w.text("$")
				w.integer(int64(pr.Number))
			}
		} else {
			w.literal(deparseString(d.Arg))
		}
	case "connectionlimit":
		w.text("CONNECTION LIMIT ")
		w.integer(ival(d.Arg))
	case "validUntil":
		w.text("VALID UNTIL ")
		w.literal(deparseString(d.Arg))
	case "superuser":
		if optBooleanValue(d.Arg) {
			w.text("SUPERUSER")
		} else {
			w.text("NOSUPERUSER")
		}
	case "createrole":
		if optBooleanValue(d.Arg) {
			w.text("CREATEROLE")
		} else {
			w.text("NOCREATEROLE")
		}
	case "isreplication":
		if optBooleanValue(d.Arg) {
			w.text("REPLICATION")
		} else {
			w.text("NOREPLICATION")
		}
	case "createdb":
		if optBooleanValue(d.Arg) {
			w.text("CREATEDB")
		} else {
			w.text("NOCREATEDB")
		}
	case "canlogin":
		if optBooleanValue(d.Arg) {
			w.text("LOGIN")
		} else {
			w.text("NOLOGIN")
		}
	case "bypassrls":
		if optBooleanValue(d.Arg) {
			w.text("BYPASSRLS")
		} else {
			w.text("NOBYPASSRLS")
		}
	case "inherit":
		if optBooleanValue(d.Arg) {
			w.text("INHERIT")
		} else {
			w.text("NOINHERIT")
		}
	default:
		w.invalid("invalid role option %q", d.Defname)
	}
}

func (w *deparseWriter) alterRole(n *p.AlterRoleStmt) {
	if n == nil {
		w.invalid("nil ALTER ROLE statement")
		return
	}
	w.text("ALTER ")
	if len(n.Options) == 1 {
		if d := n.Options[0].GetDefElem(); d != nil && d.Defname == "rolemembers" {
			w.text("GROUP ")
			w.role(n.Role)
			w.text(" ")
			switch n.Action {
			case 1:
				w.text("ADD USER ")
			case -1:
				w.text("DROP USER ")
			default:
				w.invalid("invalid ALTER GROUP action %d", n.Action)
				return
			}
			w.roles(d.Arg.GetList().GetItems())
			return
		}
	}
	w.text("ROLE ")
	w.role(n.Role)
	w.text(" WITH ")
	for _, node := range n.Options {
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid ALTER ROLE option")
			return
		}
		w.alterRoleElem(d)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) alterEnum(n *p.AlterEnumStmt) {
	if n == nil {
		w.invalid("nil ALTER TYPE statement")
		return
	}
	w.text("ALTER TYPE ")
	w.name(n.TypeName)
	w.text(" ")
	if n.OldVal == "" {
		w.text("ADD VALUE ")
		if n.SkipIfNewValExists {
			w.text("IF NOT EXISTS ")
		}
		w.literal(n.NewVal)
		w.text(" ")
		if n.NewValNeighbor != "" {
			if n.NewValIsAfter {
				w.text("AFTER ")
			} else {
				w.text("BEFORE ")
			}
			w.literal(n.NewValNeighbor)
		}
	} else {
		w.text("RENAME VALUE ")
		w.literal(n.OldVal)
		w.text(" TO ")
		w.literal(n.NewVal)
	}
	w.trimSpace()
}

func (w *deparseWriter) define(n *p.DefineStmt) {
	if n == nil {
		w.invalid("nil CREATE statement")
		return
	}
	w.text("CREATE ")
	if n.Replace {
		w.text("OR REPLACE ")
	}
	switch n.Kind.String() {
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
	case "OBJECT_TYPE":
		w.text("TYPE ")
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
	default:
		w.invalid("invalid DEFINE object type %v", n.Kind)
		return
	}
	if n.IfNotExists {
		w.text("IF NOT EXISTS ")
	}
	switch n.Kind.String() {
	case "OBJECT_AGGREGATE":
		w.name(n.Defnames)
	case "OBJECT_OPERATOR":
		w.operatorName(n.Defnames)
	case "OBJECT_TYPE", "OBJECT_TSPARSER", "OBJECT_TSDICTIONARY",
		"OBJECT_TSTEMPLATE", "OBJECT_TSCONFIGURATION", "OBJECT_COLLATION":
		w.name(n.Defnames)
	}
	w.text(" ")
	if !n.Oldstyle && n.Kind.String() == "OBJECT_AGGREGATE" {
		w.aggrArgs(n.Args)
		w.text(" ")
	}
	if n.Kind.String() == "OBJECT_COLLATION" && len(n.Definition) == 1 {
		if d := n.Definition[0].GetDefElem(); d != nil && d.Defname == "from" {
			w.text("FROM ")
			w.name(d.Arg.GetList().GetItems())
			return
		}
	}
	if len(n.Definition) > 0 {
		w.definition(n.Definition)
	}
	w.trimSpace()
}

func (w *deparseWriter) aggrArgs(nodes []*p.Node) {
	if len(nodes) != 2 {
		w.invalid("aggregate arguments require two elements")
		return
	}
	args := nodes[0].GetList()
	var orderPos int64 = -1
	if v := nodes[1].GetInteger(); v != nil {
		orderPos = int64(v.Ival)
	} else if a := nodes[1].GetAConst(); a != nil {
		orderPos = int64(a.GetIval().GetIval())
	}
	w.text("(")
	if args == nil {
		w.text("*")
		w.text(")")
		return
	}
	for i, item := range args.Items {
		if orderPos >= 0 && int64(i) == orderPos {
			if i > 0 {
				w.text(" ")
			}
			w.text("ORDER BY ")
		} else if i > 0 {
			w.text(", ")
		}
		w.functionParameter(item.GetFunctionParameter())
	}
	if orderPos == int64(len(args.Items)) && len(args.Items) > 0 {
		w.text(" ORDER BY ")
		w.functionParameter(args.Items[len(args.Items)-1].GetFunctionParameter())
	}
	w.text(")")
}
