package pg_query

import (
	"strings"

	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func (w *deparseWriter) fdwOptions(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(" ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid FDW option")
			return
		}
		switch {
		case d.Defname == "handler" && d.Arg != nil:
			w.text("HANDLER ")
			w.name(d.Arg.GetList().GetItems())
		case d.Defname == "handler":
			w.text("NO HANDLER")
		case d.Defname == "validator" && d.Arg != nil:
			w.text("VALIDATOR ")
			w.name(d.Arg.GetList().GetItems())
		case d.Defname == "validator":
			w.text("NO VALIDATOR")
		default:
			w.invalid("invalid FDW option %q", d.Defname)
			return
		}
	}
}

func (w *deparseWriter) funcAs(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		s := deparseString(node)
		if strings.Contains(s, "$$") {
			w.literal(s)
		} else {
			w.text("$$")
			w.text(s)
			w.text("$$")
		}
	}
}

func (w *deparseWriter) funcOptItem(d *p.DefElem, create bool) {
	if d == nil {
		w.invalid("nil function option")
		return
	}
	switch d.Defname {
	case "as":
		w.text("AS ")
		w.funcAs(d.Arg.GetList().GetItems())
	case "language":
		w.text("LANGUAGE ")
		w.nonReservedWordOrSconst(deparseString(d.Arg))
	case "transform":
		w.text("TRANSFORM ")
		items := d.Arg.GetList().GetItems()
		for i, item := range items {
			if i > 0 {
				w.text(", ")
			}
			w.text("FOR TYPE ")
			w.typeName(item.GetTypeName())
		}
	case "window":
		w.text("WINDOW")
	case "strict":
		if optBooleanValue(d.Arg) {
			w.text("RETURNS NULL ON NULL INPUT")
		} else {
			w.text("CALLED ON NULL INPUT")
		}
	case "volatility":
		switch deparseString(d.Arg) {
		case "immutable":
			w.text("IMMUTABLE")
		case "stable":
			w.text("STABLE")
		case "volatile":
			w.text("VOLATILE")
		default:
			w.invalid("invalid volatility %q", deparseString(d.Arg))
		}
	case "security":
		if optBooleanValue(d.Arg) {
			w.text("SECURITY DEFINER")
		} else {
			w.text("SECURITY INVOKER")
		}
	case "leakproof":
		if optBooleanValue(d.Arg) {
			w.text("LEAKPROOF")
		} else {
			w.text("NOT LEAKPROOF")
		}
	case "cost":
		w.text("COST ")
		w.ddlValue(d.Arg)
	case "rows":
		w.text("ROWS ")
		w.ddlValue(d.Arg)
	case "support":
		w.text("SUPPORT ")
		w.name(d.Arg.GetList().GetItems())
	case "set":
		w.variableSet(d.Arg.GetVariableSetStmt())
	case "parallel":
		w.text("PARALLEL ")
		w.ident(deparseString(d.Arg))
	default:
		w.invalid("invalid function option %q", d.Defname)
	}
}

func (w *deparseWriter) createdbOpts(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(" ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid database option")
			return
		}
		if d.Defname == "connection_limit" {
			w.text("CONNECTION LIMIT")
		} else {
			w.text(strings.ToUpper(d.Defname))
		}
		w.text(" ")
		if d.Arg == nil {
			w.text("DEFAULT")
		} else if v := d.Arg.GetInteger(); v != nil {
			w.integer(int64(v.Ival))
		} else {
			w.optBooleanOrString(deparseString(d.Arg))
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) utilityOpts(nodes []*p.Node) {
	if len(nodes) == 0 {
		return
	}
	w.text("(")
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid utility option")
			return
		}
		w.text(strings.ToUpper(d.Defname))
		if d.Arg != nil {
			w.text(" ")
			switch n := d.Arg.Node.(type) {
			case *p.Node_Integer:
				w.integer(int64(n.Integer.Ival))
			case *p.Node_Float:
				w.text(n.Float.Fval)
			case *p.Node_String_:
				w.optBooleanOrString(n.String_.Sval)
			case *p.Node_AConst:
				a := n.AConst
				switch v := a.Val.(type) {
				case *p.A_Const_Ival:
					w.integer(int64(v.Ival.Ival))
				case *p.A_Const_Fval:
					w.text(v.Fval.Fval)
				case *p.A_Const_Sval:
					w.optBooleanOrString(v.Sval.Sval)
				case *p.A_Const_Boolval:
					if v.Boolval.Boolval {
						w.text("TRUE")
					} else {
						w.text("FALSE")
					}
				default:
					w.invalid("invalid utility option value %T", a.Val)
					return
				}
			default:
				w.invalid("invalid utility option value %T", d.Arg.Node)
				return
			}
		}
	}
	w.text(") ")
}

func (w *deparseWriter) preparable(node *p.Node) {
	if node == nil {
		w.invalid("nil preparable statement")
		return
	}
	switch n := node.Node.(type) {
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
		w.invalid("unpermitted preparable statement %T", node.Node)
	}
}

func (w *deparseWriter) explainable(node *p.Node) {
	if node == nil {
		w.invalid("nil explainable statement")
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_SelectStmt:
		w.selectStmt(n.SelectStmt, false)
	case *p.Node_InsertStmt:
		w.insert(n.InsertStmt)
	case *p.Node_UpdateStmt:
		w.update(n.UpdateStmt)
	case *p.Node_DeleteStmt:
		w.delete(n.DeleteStmt)
	case *p.Node_DeclareCursorStmt:
		w.utilityStmt(node)
	case *p.Node_CreateTableAsStmt:
		w.createTableAs(n.CreateTableAsStmt)
	case *p.Node_RefreshMatViewStmt:
		w.utilityStmt(node)
	case *p.Node_ExecuteStmt:
		w.utilityStmt(node)
	case *p.Node_MergeStmt:
		w.merge(n.MergeStmt)
	default:
		w.invalid("unpermitted explainable statement %T", node.Node)
	}
}

func (w *deparseWriter) publicationObjects(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		obj := node.GetPublicationObjSpec()
		if obj == nil {
			w.invalid("invalid publication object %T", node.GetNode())
			return
		}
		switch obj.Pubobjtype.String() {
		case "PUBLICATIONOBJ_TABLE":
			w.text("TABLE ")
			w.relation(obj.Pubtable.Relation, deparseRelationNormal)
			if len(obj.Pubtable.Columns) > 0 {
				w.text("(")
				w.columnList(obj.Pubtable.Columns)
				w.text(")")
			}
			if obj.Pubtable.WhereClause != nil {
				w.text(" WHERE (")
				w.expr(obj.Pubtable.WhereClause)
				w.text(")")
			}
		case "PUBLICATIONOBJ_TABLES_IN_SCHEMA":
			w.text("TABLES IN SCHEMA ")
			w.ident(obj.Name)
		case "PUBLICATIONOBJ_TABLES_IN_CUR_SCHEMA":
			w.text("TABLES IN SCHEMA CURRENT_SCHEMA")
		default:
			w.invalid("invalid publication object type %v", obj.Pubobjtype)
			return
		}
	}
}

func (w *deparseWriter) opclassItems(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		item := node.GetCreateOpClassItem()
		if item == nil {
			w.invalid("invalid operator class item %T", node.GetNode())
			return
		}
		switch item.Itemtype {
		case 1:
			w.text("OPERATOR ")
			w.integer(int64(item.Number))
			w.text(" ")
			if item.Name != nil {
				if len(item.Name.Objargs) > 0 {
					w.objectWithArgs(item.Name, true)
				} else {
					w.operatorName(item.Name.Objname)
				}
				w.text(" ")
			}
			if len(item.OrderFamily) > 0 {
				w.text("FOR ORDER BY ")
				w.name(item.OrderFamily)
				w.text(" ")
			}
			if len(item.ClassArgs) > 0 {
				w.text("(")
				w.typeList(item.ClassArgs)
				w.text(")")
				w.text(" ")
			}
			w.trimSpace()
		case 2:
			w.text("FUNCTION ")
			w.integer(int64(item.Number))
			w.text(" ")
			if len(item.ClassArgs) > 0 {
				w.text("(")
				w.typeList(item.ClassArgs)
				w.text(") ")
			}
			if item.Name != nil {
				w.objectWithArgs(item.Name, false)
			}
			w.trimSpace()
		case 3:
			w.text("STORAGE ")
			w.typeName(item.Storedtype)
		default:
			w.invalid("invalid operator class item type %d", item.Itemtype)
			return
		}
	}
}

func (w *deparseWriter) operatorDefList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid operator definition")
			return
		}
		w.ident(d.Defname)
		w.text(" = ")
		if d.Arg != nil {
			w.defArg(d.Arg, true)
		} else {
			w.text("NONE")
		}
	}
}

func (w *deparseWriter) statsElem(n *p.StatsElem) {
	if n == nil {
		w.invalid("nil statistics element")
		return
	}
	if n.Name != "" {
		w.ident(n.Name)
	} else if n.Expr != nil {
		w.text("(")
		w.expr(n.Expr)
		w.text(")")
	}
}

func (w *deparseWriter) accessPriv(n *p.AccessPriv) {
	if n == nil {
		w.invalid("nil access privilege")
		return
	}
	if n.PrivName != "" {
		switch n.PrivName {
		case "select", "references", "create":
			w.text(n.PrivName)
		default:
			w.ident(n.PrivName)
		}
	} else {
		w.text("ALL")
	}
	w.text(" ")
	if len(n.Cols) > 0 {
		w.text("(")
		w.columnList(n.Cols)
		w.text(")")
	}
	w.trimSpace()
}

func (w *deparseWriter) privilegeTarget(targtype p.GrantTargetType, objtype p.ObjectType, objs []*p.Node) {
	switch targtype.String() {
	case "ACL_TARGET_OBJECT":
		switch objtype.String() {
		case "OBJECT_TABLE":
			w.qualifiedNameList(objs)
		case "OBJECT_SEQUENCE":
			w.text("SEQUENCE ")
			w.qualifiedNameList(objs)
		case "OBJECT_FDW":
			w.text("FOREIGN DATA WRAPPER ")
			w.names(objs)
		case "OBJECT_FOREIGN_SERVER":
			w.text("FOREIGN SERVER ")
			w.names(objs)
		case "OBJECT_FUNCTION":
			w.text("FUNCTION ")
			w.funcWithArgsList(objs)
		case "OBJECT_PROCEDURE":
			w.text("PROCEDURE ")
			w.funcWithArgsList(objs)
		case "OBJECT_ROUTINE":
			w.text("ROUTINE ")
			w.funcWithArgsList(objs)
		case "OBJECT_DATABASE":
			w.text("DATABASE ")
			w.names(objs)
		case "OBJECT_DOMAIN":
			w.text("DOMAIN ")
			w.anyNameList(objs)
		case "OBJECT_LANGUAGE":
			w.text("LANGUAGE ")
			w.names(objs)
		case "OBJECT_LARGEOBJECT":
			w.text("LARGE OBJECT ")
			w.numericList(objs)
		case "OBJECT_SCHEMA":
			w.text("SCHEMA ")
			w.names(objs)
		case "OBJECT_TABLESPACE":
			w.text("TABLESPACE ")
			w.names(objs)
		case "OBJECT_TYPE":
			w.text("TYPE ")
			w.anyNameList(objs)
		default:
			w.invalid("invalid privilege target type %v", objtype)
		}
	case "ACL_TARGET_ALL_IN_SCHEMA":
		switch objtype.String() {
		case "OBJECT_TABLE":
			w.text("ALL TABLES IN SCHEMA ")
		case "OBJECT_SEQUENCE":
			w.text("ALL SEQUENCES IN SCHEMA ")
		case "OBJECT_FUNCTION":
			w.text("ALL FUNCTIONS IN SCHEMA ")
		case "OBJECT_PROCEDURE":
			w.text("ALL PROCEDURES IN SCHEMA ")
		case "OBJECT_ROUTINE":
			w.text("ALL ROUTINES IN SCHEMA ")
		default:
			w.invalid("invalid privilege schema target %v", objtype)
			return
		}
		w.names(objs)
	case "ACL_TARGET_DEFAULTS":
		switch objtype.String() {
		case "OBJECT_TABLE":
			w.text("TABLES")
		case "OBJECT_FUNCTION":
			w.text("FUNCTIONS")
		case "OBJECT_SEQUENCE":
			w.text("SEQUENCES")
		case "OBJECT_TYPE":
			w.text("TYPES")
		case "OBJECT_SCHEMA":
			w.text("SCHEMAS")
		case "OBJECT_LARGEOBJECT":
			w.text("LARGE OBJECTS")
		default:
			w.invalid("invalid default privilege target %v", objtype)
		}
	default:
		w.invalid("invalid privilege target %v", targtype)
	}
}
func (w *deparseWriter) anyNameList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		if rv := node.GetRangeVar(); rv != nil {
			parts := rangeVarToStrings(rv)
			w.name(stringsToNodes(parts))
			continue
		}
		w.name(node.GetList().GetItems())
	}
}

func rangeVarToStrings(rv *p.RangeVar) []string {
	if rv == nil {
		return nil
	}
	var parts []string
	if rv.Catalogname != "" {
		parts = append(parts, rv.Catalogname)
	}
	if rv.Schemaname != "" {
		parts = append(parts, rv.Schemaname)
	}
	parts = append(parts, rv.Relname)
	return parts
}

func stringsToNodes(parts []string) []*p.Node {
	nodes := make([]*p.Node, len(parts))
	for i, s := range parts {
		nodes[i] = p.MakeStrNode(s)
	}
	return nodes
}
func (w *deparseWriter) funcWithArgsList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.objectWithArgs(node.GetObjectWithArgs(), false)
	}
}
func (w *deparseWriter) numericList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.numericOnly(node)
	}
}

func (w *deparseWriter) grant(n *p.GrantStmt) {
	if n == nil {
		w.invalid("nil GRANT statement")
		return
	}
	if n.IsGrant {
		w.text("GRANT ")
	} else {
		w.text("REVOKE ")
	}
	if !n.IsGrant && n.GrantOption {
		w.text("GRANT OPTION FOR ")
	}
	if len(n.Privileges) > 0 {
		for i, node := range n.Privileges {
			if i > 0 {
				w.text(", ")
			}
			w.accessPriv(node.GetAccessPriv())
		}
		w.text(" ")
	} else {
		w.text("ALL ")
	}
	w.text("ON ")
	w.privilegeTarget(n.Targtype, n.Objtype, n.Objects)
	w.text(" ")
	if n.IsGrant {
		w.text("TO ")
	} else {
		w.text("FROM ")
	}
	for i, node := range n.Grantees {
		if i > 0 {
			w.text(", ")
		}
		w.role(node.GetRoleSpec())
	}
	w.trimSpace()
	if n.IsGrant && n.GrantOption {
		w.text(" WITH GRANT OPTION ")
	}
	w.dropBehavior(n.Behavior)
	if n.Grantor != nil {
		w.trimSpace()
		w.text(" GRANTED BY ")
		w.role(n.Grantor)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) grantRole(n *p.GrantRoleStmt) {
	if n == nil {
		w.invalid("nil role GRANT statement")
		return
	}
	if n.IsGrant {
		w.text("GRANT ")
	} else {
		w.text("REVOKE ")
	}
	if !n.IsGrant && len(n.Opt) > 0 {
		d := n.Opt[0].GetDefElem()
		if d == nil {
			w.invalid("invalid REVOKE role option")
			return
		}
		switch d.Defname {
		case "admin":
			w.text("ADMIN ")
		case "inherit":
			w.text("INHERIT ")
		case "set":
			w.text("SET ")
		default:
			w.invalid("invalid REVOKE role option %q", d.Defname)
			return
		}
		w.text("OPTION FOR ")
	}
	for i, node := range n.GrantedRoles {
		if i > 0 {
			w.text(", ")
		}
		w.accessPriv(node.GetAccessPriv())
	}
	w.trimSpace()
	if n.IsGrant {
		w.text(" TO ")
	} else {
		w.text(" FROM ")
	}
	w.roles(n.GranteeRoles)
	w.text(" ")
	if n.IsGrant {
		if len(n.Opt) > 0 {
			w.text("WITH ")
		}
		for i, node := range n.Opt {
			if i > 0 {
				w.text(", ")
			}
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid GRANT role option")
				return
			}
			val := "FALSE"
			if optBooleanValue(d.Arg) {
				val = "OPTION"
			}
			switch d.Defname {
			case "admin":
				w.text("ADMIN ")
			case "inherit":
				w.text("INHERIT ")
			case "set":
				w.text("SET ")
			default:
				w.invalid("invalid GRANT role option %q", d.Defname)
				return
			}
			w.text(val)
		}
		if len(n.Opt) > 0 {
			w.text(" ")
		}
	}
	if n.Grantor != nil {
		w.text("GRANTED BY ")
		w.role(n.Grantor)
		w.text(" ")
	}
	if n.Behavior == p.DropBehavior_DROP_CASCADE {
		w.text("CASCADE ")
	}
	w.trimSpace()
}

func (w *deparseWriter) fetch(n *p.FetchStmt) {
	if n == nil {
		w.invalid("nil FETCH statement")
		return
	}
	if n.Ismove {
		w.text("MOVE ")
	} else {
		w.text("FETCH ")
	}
	all := n.HowMany == int64(^uint64(0)>>1)
	switch n.Direction.String() {
	case "FETCH_FORWARD":
		if all {
			w.text("ALL ")
		} else if n.HowMany != 1 {
			w.text("FORWARD ")
			w.integer(n.HowMany)
			w.text(" ")
		}
	case "FETCH_BACKWARD":
		if all {
			w.text("BACKWARD ALL ")
		} else if n.HowMany == 1 {
			w.text("PRIOR ")
		} else {
			w.text("BACKWARD ")
			w.integer(n.HowMany)
			w.text(" ")
		}
	case "FETCH_ABSOLUTE":
		if all || n.HowMany == 1 {
			w.text("FIRST ")
		} else if n.HowMany == -1 {
			w.text("LAST ")
		} else {
			w.text("ABSOLUTE ")
			w.integer(n.HowMany)
			w.text(" ")
		}
	case "FETCH_RELATIVE":
		w.text("RELATIVE ")
		w.integer(n.HowMany)
		w.text(" ")
	default:
		w.invalid("invalid FETCH direction %v", n.Direction)
		return
	}
	w.ident(n.Portalname)
}

func (w *deparseWriter) vacuum(n *p.VacuumStmt) {
	if n == nil {
		w.invalid("nil VACUUM statement")
		return
	}
	if n.IsVacuumcmd {
		w.text("VACUUM ")
	} else {
		w.text("ANALYZE ")
	}
	w.utilityOpts(n.Options)
	for i, node := range n.Rels {
		if i > 0 {
			w.text(", ")
		}
		rel := node.GetVacuumRelation()
		if rel == nil {
			w.invalid("invalid VACUUM relation %T", node.GetNode())
			return
		}
		w.relation(rel.Relation, deparseRelationNormal)
		if len(rel.VaCols) > 0 {
			w.text("(")
			w.columnList(rel.VaCols)
			w.text(")")
		}
	}
	w.trimSpace()
}

func (w *deparseWriter) copy(n *p.CopyStmt) {
	if n == nil {
		w.invalid("nil COPY statement")
		return
	}
	w.text("COPY ")
	if n.Relation != nil {
		w.relation(n.Relation, deparseRelationNormal)
		if len(n.Attlist) > 0 {
			w.text("(")
			w.columnList(n.Attlist)
			w.text(")")
		}
		w.text(" ")
	}
	if n.Query != nil {
		w.text("(")
		w.preparable(n.Query)
		w.text(") ")
	}
	if n.IsFrom {
		w.text("FROM ")
	} else {
		w.text("TO ")
	}
	if n.IsProgram {
		w.text("PROGRAM ")
	}
	if n.Filename != "" {
		w.literal(n.Filename)
		w.text(" ")
	} else {
		if n.IsFrom {
			w.text("STDIN ")
		} else {
			w.text("STDOUT ")
		}
	}
	if len(n.Options) > 0 {
		oldFmt := true
		for _, node := range n.Options {
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid COPY option")
				return
			}
			switch d.Defname {
			case "freeze":
				if !(d.Arg != nil && optBooleanValue(d.Arg)) {
					oldFmt = false
				}
			case "header":
				if !(d.Arg != nil && optBooleanValue(d.Arg)) {
					oldFmt = false
				}
			case "format":
				if deparseString(d.Arg) != "csv" {
					oldFmt = false
				}
			case "force_quote":
				if d.Arg.GetList() == nil {
					oldFmt = false
				}
			default:
				oldFmt = false
			}
		}
		if oldFmt {
			for _, node := range n.Options {
				d := node.GetDefElem()
				switch d.Defname {
				case "freeze":
					w.text("FREEZE ")
				case "header":
					w.text("HEADER ")
				case "format":
					w.text("CSV ")
				case "force_quote":
					w.text("FORCE QUOTE ")
					w.columnList(d.Arg.GetList().GetItems())
					w.text(" ")
				}
			}
		} else {
			w.text("WITH (")
			for i, node := range n.Options {
				if i > 0 {
					w.text(", ")
				}
				d := node.GetDefElem()
				switch d.Defname {
				case "format":
					w.text("FORMAT ")
					switch deparseString(d.Arg) {
					case "binary":
						w.text("BINARY")
					case "csv":
						w.text("CSV")
					case "text":
						w.text("TEXT")
					default:
						w.invalid("invalid COPY format %q", deparseString(d.Arg))
						return
					}
				case "freeze":
					w.text("FREEZE")
					if d.Arg != nil {
						w.optBoolean(d.Arg)
					}
				case "delimiter":
					w.text("DELIMITER ")
					w.literal(deparseString(d.Arg))
				case "null":
					w.text("NULL ")
					w.literal(deparseString(d.Arg))
				case "header":
					w.text("HEADER")
					if d.Arg != nil {
						w.optBoolean(d.Arg)
					}
				case "quote":
					w.text("QUOTE ")
					w.literal(deparseString(d.Arg))
				case "escape":
					w.text("ESCAPE ")
					w.literal(deparseString(d.Arg))
				case "force_quote":
					w.text("FORCE_QUOTE ")
					if d.Arg.GetAStar() != nil {
						w.text("*")
					} else if d.Arg.GetList() != nil {
						w.text("(")
						w.columnList(d.Arg.GetList().GetItems())
						w.text(")")
					} else {
						w.invalid("invalid FORCE_QUOTE argument")
						return
					}
				case "force_not_null":
					w.text("FORCE_NOT_NULL ")
					if d.Arg.GetAStar() != nil {
						w.text("*")
					} else {
						w.text("(")
						w.columnList(d.Arg.GetList().GetItems())
						w.text(")")
					}
				case "force_null":
					w.text("FORCE_NULL ")
					if d.Arg.GetAStar() != nil {
						w.text("*")
					} else {
						w.text("(")
						w.columnList(d.Arg.GetList().GetItems())
						w.text(")")
					}
				case "encoding":
					w.text("ENCODING ")
					w.literal(deparseString(d.Arg))
				default:
					w.ident(d.Defname)
					if d.Arg == nil {
						continue
					}
					w.text(" ")
					switch a := d.Arg.Node.(type) {
					case *p.Node_String_:
						w.optBooleanOrString(a.String_.Sval)
					case *p.Node_Integer:
						w.integer(int64(a.Integer.Ival))
					case *p.Node_Float:
						w.text(a.Float.Fval)
					case *p.Node_AStar:
						w.text("*")
					case *p.Node_List:
						w.text("(")
						for j, item := range a.List.Items {
							if j > 0 {
								w.text(", ")
							}
							w.optBooleanOrString(deparseString(item))
						}
						w.text(")")
					case *p.Node_AConst:
						switch v := a.AConst.Val.(type) {
						case *p.A_Const_Sval:
							w.optBooleanOrString(v.Sval.Sval)
						case *p.A_Const_Ival:
							w.integer(int64(v.Ival.Ival))
						case *p.A_Const_Fval:
							w.text(v.Fval.Fval)
						case *p.A_Const_Boolval:
							if v.Boolval.Boolval {
								w.text("TRUE")
							} else {
								w.text("FALSE")
							}
						default:
							w.invalid("invalid COPY option value %T", a.AConst.Val)
							return
						}
					default:
						w.invalid("invalid COPY option value %T", d.Arg.Node)
						return
					}
				}
			}
			w.text(") ")
		}
	}
	if n.WhereClause != nil {
		w.text("WHERE ")
		w.expr(n.WhereClause)
		w.text(" ")
	}
	w.trimSpace()
}

func (w *deparseWriter) optBoolean(node *p.Node) {
	if node == nil {
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_String_:
		w.text(" ")
		w.text(n.String_.Sval)
	case *p.Node_Integer:
		w.text(" ")
		w.integer(int64(n.Integer.Ival))
	case *p.Node_Boolean:
		if n.Boolean.Boolval {
			w.text(" TRUE")
		} else {
			w.text(" FALSE")
		}
	case *p.Node_AConst:
		a := n.AConst
		switch v := a.Val.(type) {
		case *p.A_Const_Sval:
			w.text(" ")
			w.text(v.Sval.Sval)
		case *p.A_Const_Ival:
			w.text(" ")
			w.integer(int64(v.Ival.Ival))
		case *p.A_Const_Boolval:
			if v.Boolval.Boolval {
				w.text(" TRUE")
			} else {
				w.text(" FALSE")
			}
		}
	}
}

func (w *deparseWriter) variableSet(n *p.VariableSetStmt) {
	if n == nil {
		w.invalid("nil SET statement")
		return
	}
	switch n.Kind.String() {
	case "VAR_SET_VALUE":
		w.text("SET ")
		if n.IsLocal {
			w.text("LOCAL ")
		}
		if n.Name == "timezone" && n.JumbleArgs {
			w.text("TIME ZONE ")
			if len(n.Args) == 1 && n.Args[0].GetTypeCast() != nil {
				cast := n.Args[0].GetTypeCast()
				w.text("interval ")
				w.expr(cast.Arg)
				w.intervalMods(cast.TypeName.Typmods)
			} else {
				w.varList(n.Args)
			}
			return
		}
		if n.Name == "xmloption" && n.JumbleArgs && len(n.Args) > 0 {
			opt := ""
			if a := n.Args[0].GetAConst(); a != nil {
				opt = a.GetSval().GetSval()
			}
			w.text("XML OPTION ")
			if opt == "DOCUMENT" {
				w.text("DOCUMENT")
				return
			}
			if opt == "CONTENT" {
				w.text("CONTENT")
				return
			}
			w.varList(n.Args)
			return
		}
		w.ident(n.Name)
		w.text(" TO ")
		w.varList(n.Args)
	case "VAR_SET_DEFAULT":
		w.text("SET ")
		if n.IsLocal {
			w.text("LOCAL ")
		}
		if n.Name == "timezone" && n.JumbleArgs {
			w.text("TIME ZONE DEFAULT")
			return
		}
		w.ident(n.Name)
		w.text(" TO DEFAULT")
	case "VAR_SET_CURRENT":
		w.text("SET ")
		if n.IsLocal {
			w.text("LOCAL ")
		}
		w.ident(n.Name)
		w.text(" FROM CURRENT")
	case "VAR_SET_MULTI":
		w.text("SET ")
		if n.IsLocal {
			w.text("LOCAL ")
		}
		switch n.Name {
		case "TRANSACTION":
			w.text("TRANSACTION ")
			w.transactionModes(n.Args)
		case "SESSION CHARACTERISTICS":
			w.text("SESSION CHARACTERISTICS AS TRANSACTION ")
			w.transactionModes(n.Args)
		case "TRANSACTION SNAPSHOT":
			w.text("TRANSACTION SNAPSHOT ")
			if len(n.Args) > 0 {
				w.literal(n.Args[0].GetAConst().GetSval().GetSval())
			}
		default:
			w.invalid("invalid multi-variable %q", n.Name)
		}
	case "VAR_RESET":
		w.text("RESET ")
		w.ident(n.Name)
	case "VAR_RESET_ALL":
		w.text("RESET ALL")
	default:
		w.invalid("invalid SET kind %v", n.Kind)
	}
}

func (w *deparseWriter) varList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		switch n := node.Node.(type) {
		case *p.Node_ParamRef:
			if n.ParamRef.Number == 0 {
				w.text("?")
			} else {
				w.text("$")
				w.integer(int64(n.ParamRef.Number))
			}
		case *p.Node_AConst:
			a := n.AConst
			switch v := a.Val.(type) {
			case *p.A_Const_Ival:
				w.integer(int64(v.Ival.Ival))
			case *p.A_Const_Fval:
				w.text(v.Fval.Fval)
			case *p.A_Const_Sval:
				w.optBooleanOrString(v.Sval.Sval)
			case *p.A_Const_Boolval:
				if v.Boolval.Boolval {
					w.text("TRUE")
				} else {
					w.text("FALSE")
				}
			default:
				w.invalid("invalid SET value")
				return
			}
		case *p.Node_TypeCast:
			w.typeCast(n.TypeCast)
		default:
			w.invalid("invalid SET value %T", node.Node)
			return
		}
	}
}

func (w *deparseWriter) transactionModes(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid transaction mode")
			return
		}
		switch d.Defname {
		case "transaction_isolation":
			w.text("ISOLATION LEVEL ")
			s := ""
			if a := d.Arg.GetAConst(); a != nil {
				s = a.GetSval().GetSval()
			}
			switch s {
			case "read uncommitted":
				w.text("READ UNCOMMITTED")
			case "read committed":
				w.text("READ COMMITTED")
			case "repeatable read":
				w.text("REPEATABLE READ")
			case "serializable":
				w.text("SERIALIZABLE")
			default:
				w.invalid("invalid isolation level %q", s)
				return
			}
		case "transaction_read_only":
			if ival(d.Arg) == 1 {
				w.text("READ ONLY")
			} else {
				w.text("READ WRITE")
			}
		case "transaction_deferrable":
			if ival(d.Arg) == 1 {
				w.text("DEFERRABLE")
			} else {
				w.text("NOT DEFERRABLE")
			}
		default:
			w.invalid("invalid transaction mode %q", d.Defname)
			return
		}
	}
}

func ival(node *p.Node) int64 {
	if node == nil {
		return 0
	}
	if v := node.GetInteger(); v != nil {
		return int64(v.Ival)
	}
	if a := node.GetAConst(); a != nil {
		if v := a.GetIval(); v != nil {
			return int64(v.Ival)
		}
	}
	return 0
}

func (w *deparseWriter) transaction(n *p.TransactionStmt) {
	if n == nil {
		w.invalid("nil transaction statement")
		return
	}
	switch n.Kind.String() {
	case "TRANS_STMT_BEGIN":
		w.text("BEGIN ")
		w.transactionModes(n.Options)
		w.trimSpace()
	case "TRANS_STMT_START":
		w.text("START TRANSACTION ")
		w.transactionModes(n.Options)
		w.trimSpace()
	case "TRANS_STMT_COMMIT":
		w.text("COMMIT")
		if n.Chain {
			w.text(" AND CHAIN")
		}
	case "TRANS_STMT_ROLLBACK":
		w.text("ROLLBACK")
		if n.Chain {
			w.text(" AND CHAIN")
		}
	case "TRANS_STMT_SAVEPOINT":
		w.text("SAVEPOINT ")
		w.ident(n.SavepointName)
	case "TRANS_STMT_RELEASE":
		w.text("RELEASE ")
		w.ident(n.SavepointName)
	case "TRANS_STMT_ROLLBACK_TO":
		w.text("ROLLBACK TO SAVEPOINT ")
		w.ident(n.SavepointName)
	case "TRANS_STMT_PREPARE":
		w.text("PREPARE TRANSACTION ")
		w.literal(n.Gid)
	case "TRANS_STMT_COMMIT_PREPARED":
		w.text("COMMIT PREPARED ")
		w.literal(n.Gid)
	case "TRANS_STMT_ROLLBACK_PREPARED":
		w.text("ROLLBACK PREPARED ")
		w.literal(n.Gid)
	default:
		w.invalid("invalid transaction statement %v", n.Kind)
	}
}

func (w *deparseWriter) alterIdentityOpts(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(" ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid identity option")
			return
		}
		switch d.Defname {
		case "restart":
			if d.Arg == nil {
				w.text("RESTART")
			} else {
				w.text("RESTART ")
				w.numericOnly(d.Arg)
			}
		case "generated":
			w.text("SET GENERATED ")
			switch ival(d.Arg) {
			case 97:
				w.text("ALWAYS")
			case 100:
				w.text("BY DEFAULT")
			default:
				w.invalid("invalid identity generation %d", ival(d.Arg))
				return
			}
		default:
			w.text("SET ")
			w.seqOptElem(d)
		}
	}
}
