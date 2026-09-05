package pg_query

import p "github.com/gosuda/pg18_query_go/internal/pgproto"

func (w *deparseWriter) drop(n *p.DropStmt) {
	if n == nil {
		w.invalid("nil DROP statement")
		return
	}
	w.text("DROP ")
	switch n.RemoveType.String() {
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
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
	case "OBJECT_EXTENSION":
		w.text("EXTENSION ")
	case "OBJECT_FDW":
		w.text("FOREIGN DATA WRAPPER ")
	case "OBJECT_FOREIGN_SERVER":
		w.text("SERVER ")
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
	case "OBJECT_INDEX":
		w.text("INDEX ")
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	case "OBJECT_OPCLASS":
		w.text("OPERATOR CLASS ")
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
	case "OBJECT_OPFAMILY":
		w.text("OPERATOR FAMILY ")
	case "OBJECT_POLICY":
		w.text("POLICY ")
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
	case "OBJECT_PUBLICATION":
		w.text("PUBLICATION ")
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
	case "OBJECT_RULE":
		w.text("RULE ")
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
	case "OBJECT_STATISTIC_EXT":
		w.text("STATISTICS ")
	case "OBJECT_TABLE":
		w.text("TABLE ")
	case "OBJECT_TRANSFORM":
		w.text("TRANSFORM ")
	case "OBJECT_TRIGGER":
		w.text("TRIGGER ")
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
	case "OBJECT_TYPE":
		w.text("TYPE ")
	case "OBJECT_VIEW":
		w.text("VIEW ")
	default:
		w.invalid("invalid DROP object type %v", n.RemoveType)
		return
	}
	if n.Concurrent {
		w.text("CONCURRENTLY ")
	}
	if n.MissingOk {
		w.text("IF EXISTS ")
	}
	switch n.RemoveType.String() {
	case "OBJECT_TABLE", "OBJECT_SEQUENCE", "OBJECT_VIEW", "OBJECT_MATVIEW",
		"OBJECT_INDEX", "OBJECT_FOREIGN_TABLE", "OBJECT_COLLATION", "OBJECT_CONVERSION",
		"OBJECT_STATISTIC_EXT", "OBJECT_TSPARSER", "OBJECT_TSDICTIONARY",
		"OBJECT_TSTEMPLATE", "OBJECT_TSCONFIGURATION":
		w.anyNameList(n.Objects)
		w.text(" ")
	case "OBJECT_ACCESS_METHOD", "OBJECT_EVENT_TRIGGER", "OBJECT_EXTENSION", "OBJECT_FDW",
		"OBJECT_PUBLICATION", "OBJECT_SCHEMA", "OBJECT_FOREIGN_SERVER":
		for i, node := range n.Objects {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
	case "OBJECT_POLICY", "OBJECT_RULE", "OBJECT_TRIGGER":
		if len(n.Objects) != 1 {
			w.invalid("DROP %v requires one object", n.RemoveType)
			return
		}
		items := n.Objects[0].GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid DROP object")
			return
		}
		w.ident(deparseString(items[len(items)-1]))
		w.text(" ON ")
		w.name(items[:len(items)-1])
		w.text(" ")
	case "OBJECT_CAST":
		if len(n.Objects) != 1 {
			w.invalid("DROP CAST requires one object")
			return
		}
		items := n.Objects[0].GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid DROP CAST object")
			return
		}
		w.text("(")
		w.typeName(items[0].GetTypeName())
		w.text(" AS ")
		w.typeName(items[1].GetTypeName())
		w.text(") ")
	case "OBJECT_OPCLASS", "OBJECT_OPFAMILY":
		if len(n.Objects) != 1 {
			w.invalid("DROP OPERATOR requires one object")
			return
		}
		items := n.Objects[0].GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid DROP OPERATOR object")
			return
		}
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
		w.text(" ")
	case "OBJECT_TRANSFORM":
		if len(n.Objects) != 1 {
			w.invalid("DROP TRANSFORM requires one object")
			return
		}
		items := n.Objects[0].GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid DROP TRANSFORM object")
			return
		}
		w.text("FOR ")
		w.typeName(items[0].GetTypeName())
		w.text(" LANGUAGE ")
		w.ident(deparseString(items[1]))
		w.text(" ")
	case "OBJECT_LANGUAGE":
		for i, node := range n.Objects {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
	case "OBJECT_TYPE", "OBJECT_DOMAIN":
		for i, node := range n.Objects {
			if i > 0 {
				w.text(", ")
			}
			w.typeName(node.GetTypeName())
		}
		w.text(" ")
	case "OBJECT_AGGREGATE":
		for i, node := range n.Objects {
			if i > 0 {
				w.text(", ")
			}
			w.aggrWithArgs(node.GetObjectWithArgs())
		}
		w.text(" ")
	case "OBJECT_FUNCTION", "OBJECT_PROCEDURE", "OBJECT_ROUTINE":
		w.funcWithArgsList(n.Objects)
		w.text(" ")
	case "OBJECT_OPERATOR":
		for i, node := range n.Objects {
			if i > 0 {
				w.text(", ")
			}
			w.objectWithArgs(node.GetObjectWithArgs(), true)
		}
		w.text(" ")
	}
	w.dropBehavior(n.Behavior)
	w.trimSpace()
}

func (w *deparseWriter) aggrWithArgs(o *p.ObjectWithArgs) {
	if o == nil {
		w.invalid("nil aggregate reference")
		return
	}
	w.name(o.Objname)
	w.text("(")
	params := o.Objargs
	if len(o.Objfuncargs) > 0 {
		params = o.Objfuncargs
	}
	if len(params) == 0 && len(o.Objargs) == 0 && len(o.Objfuncargs) == 0 {
		w.text("*")
	} else {
		for i, node := range params {
			if i > 0 {
				w.text(", ")
			}
			if fp := node.GetFunctionParameter(); fp != nil {
				w.functionParameter(fp)
			} else {
				w.typeName(node.GetTypeName())
			}
		}
	}
	w.text(")")
}

func (w *deparseWriter) rename(n *p.RenameStmt) {
	if n == nil {
		w.invalid("nil ALTER RENAME statement")
		return
	}
	w.text("ALTER ")
	switch n.RenameType.String() {
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
	case "OBJECT_CONVERSION":
		w.text("CONVERSION ")
	case "OBJECT_DATABASE":
		w.text("DATABASE ")
	case "OBJECT_DOMAIN", "OBJECT_DOMCONSTRAINT":
		w.text("DOMAIN ")
	case "OBJECT_FDW":
		w.text("FOREIGN DATA WRAPPER ")
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
	case "OBJECT_ROLE":
		w.text("ROLE ")
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
	case "OBJECT_OPCLASS":
		w.text("OPERATOR CLASS ")
	case "OBJECT_OPFAMILY":
		w.text("OPERATOR FAMILY ")
	case "OBJECT_POLICY":
		w.text("POLICY ")
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
	case "OBJECT_PUBLICATION":
		w.text("PUBLICATION ")
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
	case "OBJECT_FOREIGN_SERVER":
		w.text("SERVER ")
	case "OBJECT_SUBSCRIPTION":
		w.text("SUBSCRIPTION ")
	case "OBJECT_TABLE", "OBJECT_TABCONSTRAINT":
		w.text("TABLE ")
	case "OBJECT_COLUMN":
		switch n.RelationType.String() {
		case "OBJECT_TABLE":
			w.text("TABLE ")
		case "OBJECT_FOREIGN_TABLE":
			w.text("FOREIGN TABLE ")
		case "OBJECT_VIEW":
			w.text("VIEW ")
		case "OBJECT_MATVIEW":
			w.text("MATERIALIZED VIEW ")
		default:
			w.invalid("invalid RENAME column parent %v", n.RelationType)
			return
		}
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
	case "OBJECT_VIEW":
		w.text("VIEW ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	case "OBJECT_INDEX":
		w.text("INDEX ")
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
	case "OBJECT_RULE":
		w.text("RULE ")
	case "OBJECT_TRIGGER":
		w.text("TRIGGER ")
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
	case "OBJECT_TABLESPACE":
		w.text("TABLESPACE ")
	case "OBJECT_STATISTIC_EXT":
		w.text("STATISTICS ")
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
	case "OBJECT_TYPE", "OBJECT_ATTRIBUTE":
		w.text("TYPE ")
	default:
		w.invalid("invalid RENAME object type %v", n.RenameType)
		return
	}
	if n.MissingOk {
		w.text("IF EXISTS ")
	}
	switch n.RenameType.String() {
	case "OBJECT_AGGREGATE":
		w.aggrWithArgs(n.Object.GetObjectWithArgs())
		w.text(" RENAME ")
	case "OBJECT_DOMCONSTRAINT":
		w.name(n.Object.GetList().GetItems())
		w.text(" RENAME CONSTRAINT ")
		w.ident(n.Subname)
		w.text(" ")
	case "OBJECT_OPCLASS", "OBJECT_OPFAMILY":
		items := n.Object.GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid RENAME operator object")
			return
		}
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
		w.text(" RENAME ")
	case "OBJECT_POLICY":
		w.ident(n.Subname)
		w.text(" ON ")
		w.relation(n.Relation, deparseRelationNormal)
		w.text(" RENAME ")
	case "OBJECT_FUNCTION", "OBJECT_PROCEDURE", "OBJECT_ROUTINE":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
		w.text(" RENAME ")
	case "OBJECT_SUBSCRIPTION":
		w.ident(deparseString(n.Object))
		w.text(" RENAME ")
	case "OBJECT_TABLE", "OBJECT_SEQUENCE", "OBJECT_VIEW", "OBJECT_MATVIEW",
		"OBJECT_INDEX", "OBJECT_FOREIGN_TABLE":
		w.relation(n.Relation, deparseRelationNormal)
		w.text(" RENAME ")
	case "OBJECT_COLUMN":
		w.relation(n.Relation, deparseRelationNormal)
		w.text(" RENAME COLUMN ")
		w.ident(n.Subname)
		w.text(" ")
	case "OBJECT_TABCONSTRAINT":
		w.relation(n.Relation, deparseRelationNormal)
		w.text(" RENAME CONSTRAINT ")
		w.ident(n.Subname)
		w.text(" ")
	case "OBJECT_RULE", "OBJECT_TRIGGER":
		w.ident(n.Subname)
		w.text(" ON ")
		w.relation(n.Relation, deparseRelationNormal)
		w.text(" RENAME ")
	case "OBJECT_FDW", "OBJECT_LANGUAGE", "OBJECT_PUBLICATION",
		"OBJECT_FOREIGN_SERVER", "OBJECT_EVENT_TRIGGER":
		w.ident(deparseString(n.Object))
		w.text(" RENAME ")
	case "OBJECT_DATABASE", "OBJECT_ROLE", "OBJECT_SCHEMA", "OBJECT_TABLESPACE":
		w.ident(n.Subname)
		w.text(" RENAME ")
	case "OBJECT_COLLATION", "OBJECT_CONVERSION", "OBJECT_DOMAIN", "OBJECT_STATISTIC_EXT",
		"OBJECT_TSPARSER", "OBJECT_TSDICTIONARY", "OBJECT_TSTEMPLATE",
		"OBJECT_TSCONFIGURATION", "OBJECT_TYPE":
		w.name(n.Object.GetList().GetItems())
		w.text(" RENAME ")
	case "OBJECT_ATTRIBUTE":
		w.relation(n.Relation, deparseRelationCreate)
		w.text(" RENAME ATTRIBUTE ")
		w.ident(n.Subname)
		w.text(" ")
	}
	w.text("TO ")
	w.ident(n.Newname)
	w.text(" ")
	w.dropBehavior(n.Behavior)
	w.trimSpace()
}

func (w *deparseWriter) alterObjectDepends(n *p.AlterObjectDependsStmt) {
	if n == nil {
		w.invalid("nil ALTER DEPENDS statement")
		return
	}
	w.text("ALTER ")
	switch n.ObjectType.String() {
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_TRIGGER":
		w.text("TRIGGER ")
		items := n.Object.GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid trigger reference")
			return
		}
		w.ident(deparseString(items[0]))
		w.text(" ON ")
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_INDEX":
		w.text("INDEX ")
		w.relation(n.Relation, deparseRelationNormal)
	default:
		w.invalid("invalid DEPENDS object type %v", n.ObjectType)
		return
	}
	w.text(" ")
	if n.Remove {
		w.text("NO ")
	}
	w.text("DEPENDS ON EXTENSION ")
	w.ident(n.Extname.GetSval())
}

func (w *deparseWriter) alterObjectSchema(n *p.AlterObjectSchemaStmt) {
	if n == nil {
		w.invalid("nil ALTER SET SCHEMA statement")
		return
	}
	w.text("ALTER ")
	switch n.ObjectType.String() {
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
		w.aggrWithArgs(n.Object.GetObjectWithArgs())
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_CONVERSION":
		w.text("CONVERSION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_DOMAIN":
		w.text("DOMAIN ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_EXTENSION":
		w.text("EXTENSION ")
		w.ident(deparseString(n.Object))
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), true)
	case "OBJECT_OPCLASS":
		items := n.Object.GetList().GetItems()
		w.text("OPERATOR CLASS ")
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_OPFAMILY":
		items := n.Object.GetList().GetItems()
		w.text("OPERATOR FAMILY ")
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_TABLE":
		w.text("TABLE ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_STATISTIC_EXT":
		w.text("STATISTICS ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_VIEW":
		w.text("VIEW ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.Relation, deparseRelationNormal)
	case "OBJECT_TYPE":
		w.text("TYPE ")
		w.name(n.Object.GetList().GetItems())
	default:
		w.invalid("invalid SET SCHEMA object type %v", n.ObjectType)
		return
	}
	w.text(" SET SCHEMA ")
	w.ident(n.Newschema)
}

func (w *deparseWriter) alterOwner(n *p.AlterOwnerStmt) {
	if n == nil {
		w.invalid("nil ALTER OWNER statement")
		return
	}
	w.text("ALTER ")
	switch n.ObjectType.String() {
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
		w.aggrWithArgs(n.Object.GetObjectWithArgs())
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_CONVERSION":
		w.text("CONVERSION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_DATABASE":
		w.text("DATABASE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_DOMAIN":
		w.text("DOMAIN ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_LARGEOBJECT":
		w.text("LARGE OBJECT ")
		w.numericOnly(n.Object)
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), true)
	case "OBJECT_OPCLASS":
		items := n.Object.GetList().GetItems()
		w.text("OPERATOR CLASS ")
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_OPFAMILY":
		items := n.Object.GetList().GetItems()
		w.text("OPERATOR FAMILY ")
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
		w.ident(deparseString(n.Object))
	case "OBJECT_TYPE":
		w.text("TYPE ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TABLESPACE":
		w.text("TABLESPACE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_STATISTIC_EXT":
		w.text("STATISTICS ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_FDW":
		w.text("FOREIGN DATA WRAPPER ")
		w.ident(deparseString(n.Object))
	case "OBJECT_FOREIGN_SERVER":
		w.text("SERVER ")
		w.ident(deparseString(n.Object))
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
		w.ident(deparseString(n.Object))
	case "OBJECT_PUBLICATION":
		w.text("PUBLICATION ")
		w.ident(deparseString(n.Object))
	case "OBJECT_SUBSCRIPTION":
		w.text("SUBSCRIPTION ")
		w.ident(deparseString(n.Object))
	default:
		w.invalid("invalid OWNER object type %v", n.ObjectType)
		return
	}
	w.text(" OWNER TO ")
	w.role(n.Newowner)
}

func (w *deparseWriter) comment(n *p.CommentStmt) {
	if n == nil {
		w.invalid("nil COMMENT statement")
		return
	}
	w.text("COMMENT ON ")
	switch n.Objtype.String() {
	case "OBJECT_COLUMN":
		w.text("COLUMN ")
	case "OBJECT_INDEX":
		w.text("INDEX ")
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
	case "OBJECT_STATISTIC_EXT":
		w.text("STATISTICS ")
	case "OBJECT_TABLE":
		w.text("TABLE ")
	case "OBJECT_VIEW":
		w.text("VIEW ")
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
	case "OBJECT_COLLATION":
		w.text("COLLATION ")
	case "OBJECT_CONVERSION":
		w.text("CONVERSION ")
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
	case "OBJECT_TSCONFIGURATION":
		w.text("TEXT SEARCH CONFIGURATION ")
	case "OBJECT_TSDICTIONARY":
		w.text("TEXT SEARCH DICTIONARY ")
	case "OBJECT_TSPARSER":
		w.text("TEXT SEARCH PARSER ")
	case "OBJECT_TSTEMPLATE":
		w.text("TEXT SEARCH TEMPLATE ")
	case "OBJECT_ACCESS_METHOD":
		w.text("ACCESS METHOD ")
	case "OBJECT_DATABASE":
		w.text("DATABASE ")
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
	case "OBJECT_EXTENSION":
		w.text("EXTENSION ")
	case "OBJECT_FDW":
		w.text("FOREIGN DATA WRAPPER ")
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
	case "OBJECT_PUBLICATION":
		w.text("PUBLICATION ")
	case "OBJECT_ROLE":
		w.text("ROLE ")
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
	case "OBJECT_FOREIGN_SERVER":
		w.text("SERVER ")
	case "OBJECT_SUBSCRIPTION":
		w.text("SUBSCRIPTION ")
	case "OBJECT_TABLESPACE":
		w.text("TABLESPACE ")
	case "OBJECT_TYPE":
		w.text("TYPE ")
	case "OBJECT_DOMAIN":
		w.text("DOMAIN ")
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
	case "OBJECT_OPERATOR":
		w.text("OPERATOR ")
	case "OBJECT_TABCONSTRAINT":
		w.text("CONSTRAINT ")
	case "OBJECT_DOMCONSTRAINT":
		w.text("CONSTRAINT ")
	case "OBJECT_POLICY":
		w.text("POLICY ")
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
	case "OBJECT_RULE":
		w.text("RULE ")
	case "OBJECT_TRANSFORM":
		w.text("TRANSFORM ")
	case "OBJECT_TRIGGER":
		w.text("TRIGGER ")
	case "OBJECT_OPCLASS":
		w.text("OPERATOR CLASS ")
	case "OBJECT_OPFAMILY":
		w.text("OPERATOR FAMILY ")
	case "OBJECT_LARGEOBJECT":
		w.text("LARGE OBJECT ")
	case "OBJECT_CAST":
		w.text("CAST ")
	default:
		w.invalid("invalid COMMENT object type %v", n.Objtype)
		return
	}
	switch n.Objtype.String() {
	case "OBJECT_COLUMN", "OBJECT_INDEX", "OBJECT_SEQUENCE", "OBJECT_STATISTIC_EXT",
		"OBJECT_TABLE", "OBJECT_VIEW", "OBJECT_MATVIEW", "OBJECT_COLLATION",
		"OBJECT_CONVERSION", "OBJECT_FOREIGN_TABLE", "OBJECT_TSCONFIGURATION",
		"OBJECT_TSDICTIONARY", "OBJECT_TSPARSER", "OBJECT_TSTEMPLATE":
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_ACCESS_METHOD", "OBJECT_DATABASE", "OBJECT_EVENT_TRIGGER",
		"OBJECT_EXTENSION", "OBJECT_FDW", "OBJECT_LANGUAGE", "OBJECT_PUBLICATION",
		"OBJECT_ROLE", "OBJECT_SCHEMA", "OBJECT_FOREIGN_SERVER",
		"OBJECT_SUBSCRIPTION", "OBJECT_TABLESPACE":
		w.ident(deparseString(n.Object))
	case "OBJECT_TYPE", "OBJECT_DOMAIN":
		w.typeName(n.Object.GetTypeName())
	case "OBJECT_AGGREGATE":
		w.aggrWithArgs(n.Object.GetObjectWithArgs())
	case "OBJECT_FUNCTION", "OBJECT_PROCEDURE", "OBJECT_ROUTINE":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_OPERATOR":
		w.objectWithArgs(n.Object.GetObjectWithArgs(), true)
	case "OBJECT_TABCONSTRAINT", "OBJECT_POLICY", "OBJECT_RULE", "OBJECT_TRIGGER":
		items := n.Object.GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid COMMENT object")
			return
		}
		w.ident(deparseString(items[len(items)-1]))
		w.text(" ON ")
		w.name(items[:len(items)-1])
	case "OBJECT_DOMCONSTRAINT":
		items := n.Object.GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid domain constraint reference")
			return
		}
		w.ident(deparseString(items[1]))
		w.text(" ON DOMAIN ")
		w.typeName(items[0].GetTypeName())
	case "OBJECT_TRANSFORM":
		items := n.Object.GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid TRANSFORM reference")
			return
		}
		w.text("FOR ")
		w.typeName(items[0].GetTypeName())
		w.text(" LANGUAGE ")
		w.ident(deparseString(items[1]))
	case "OBJECT_OPCLASS", "OBJECT_OPFAMILY":
		items := n.Object.GetList().GetItems()
		if len(items) == 0 {
			w.invalid("invalid operator object")
			return
		}
		w.name(items[1:])
		w.text(" USING ")
		w.ident(deparseString(items[0]))
	case "OBJECT_LARGEOBJECT":
		w.numericOnly(n.Object)
	case "OBJECT_CAST":
		items := n.Object.GetList().GetItems()
		if len(items) != 2 {
			w.invalid("invalid CAST reference")
			return
		}
		w.text("(")
		w.typeName(items[0].GetTypeName())
		w.text(" AS ")
		w.typeName(items[1].GetTypeName())
		w.text(")")
	}
	w.text(" IS ")
	if n.Comment != "" {
		w.literal(n.Comment)
	} else {
		w.text("NULL")
	}
}

func (w *deparseWriter) secLabel(n *p.SecLabelStmt) {
	if n == nil {
		w.invalid("nil SECURITY LABEL statement")
		return
	}
	w.text("SECURITY LABEL ")
	if n.Provider != "" {
		w.text("FOR ")
		w.ident(n.Provider)
		w.text(" ")
	}
	w.text("ON ")
	switch n.Objtype.String() {
	case "OBJECT_COLUMN":
		w.text("COLUMN ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_FOREIGN_TABLE":
		w.text("FOREIGN TABLE ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_SEQUENCE":
		w.text("SEQUENCE ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_TABLE":
		w.text("TABLE ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_VIEW":
		w.text("VIEW ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_MATVIEW":
		w.text("MATERIALIZED VIEW ")
		w.name(n.Object.GetList().GetItems())
	case "OBJECT_DATABASE":
		w.text("DATABASE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_EVENT_TRIGGER":
		w.text("EVENT TRIGGER ")
		w.ident(deparseString(n.Object))
	case "OBJECT_LANGUAGE":
		w.text("LANGUAGE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_PUBLICATION":
		w.text("PUBLICATION ")
		w.ident(deparseString(n.Object))
	case "OBJECT_ROLE":
		w.text("ROLE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_SCHEMA":
		w.text("SCHEMA ")
		w.ident(deparseString(n.Object))
	case "OBJECT_SUBSCRIPTION":
		w.text("SUBSCRIPTION ")
		w.ident(deparseString(n.Object))
	case "OBJECT_TABLESPACE":
		w.text("TABLESPACE ")
		w.ident(deparseString(n.Object))
	case "OBJECT_TYPE":
		w.text("TYPE ")
		w.typeName(n.Object.GetTypeName())
	case "OBJECT_DOMAIN":
		w.text("DOMAIN ")
		w.typeName(n.Object.GetTypeName())
	case "OBJECT_AGGREGATE":
		w.text("AGGREGATE ")
		w.aggrWithArgs(n.Object.GetObjectWithArgs())
	case "OBJECT_FUNCTION":
		w.text("FUNCTION ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_LARGEOBJECT":
		w.text("LARGE OBJECT ")
		w.numericOnly(n.Object)
	case "OBJECT_PROCEDURE":
		w.text("PROCEDURE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	case "OBJECT_ROUTINE":
		w.text("ROUTINE ")
		w.objectWithArgs(n.Object.GetObjectWithArgs(), false)
	default:
		w.invalid("invalid SECURITY LABEL object type %v", n.Objtype)
		return
	}
	w.text(" IS ")
	if n.Label != "" {
		w.literal(n.Label)
	} else {
		w.text("NULL")
	}
}

func (w *deparseWriter) alterDefaultPrivileges(n *p.AlterDefaultPrivilegesStmt) {
	if n == nil {
		w.invalid("nil ALTER DEFAULT PRIVILEGES statement")
		return
	}
	w.text("ALTER DEFAULT PRIVILEGES ")
	for _, node := range n.Options {
		d := node.GetDefElem()
		if d == nil {
			w.invalid("invalid default privilege option")
			return
		}
		switch d.Defname {
		case "schemas":
			w.text("IN SCHEMA ")
			w.names(d.Arg.GetList().GetItems())
			w.text(" ")
		case "roles":
			w.text("FOR ROLE ")
			w.roles(d.Arg.GetList().GetItems())
			w.text(" ")
		default:
			w.invalid("invalid default privilege option %q", d.Defname)
			return
		}
	}
	w.grant(n.Action)
}

func (w *deparseWriter) alterSubscription(n *p.AlterSubscriptionStmt) {
	if n == nil {
		w.invalid("nil ALTER SUBSCRIPTION statement")
		return
	}
	w.text("ALTER SUBSCRIPTION ")
	w.ident(n.Subname)
	w.text(" ")
	switch n.Kind.String() {
	case "ALTER_SUBSCRIPTION_OPTIONS":
		w.text("SET ")
		w.definition(n.Options)
	case "ALTER_SUBSCRIPTION_SKIP":
		w.text("SKIP ")
		w.definition(n.Options)
	case "ALTER_SUBSCRIPTION_CONNECTION":
		w.text("CONNECTION ")
		w.literal(n.Conninfo)
	case "ALTER_SUBSCRIPTION_REFRESH":
		w.text("REFRESH PUBLICATION ")
		if len(n.Options) > 0 {
			w.text("WITH ")
			w.definition(n.Options)
		}
		w.trimSpace()
		return
	case "ALTER_SUBSCRIPTION_ADD_PUBLICATION":
		w.text("ADD PUBLICATION ")
		for i, node := range n.Publication {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
		if len(n.Options) > 0 {
			w.text("WITH ")
			w.definition(n.Options)
		}
		w.trimSpace()
		return
	case "ALTER_SUBSCRIPTION_DROP_PUBLICATION":
		w.text("DROP PUBLICATION ")
		for i, node := range n.Publication {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
		if len(n.Options) > 0 {
			w.text("WITH ")
			w.definition(n.Options)
		}
		w.trimSpace()
		return
	case "ALTER_SUBSCRIPTION_SET_PUBLICATION":
		w.text("SET PUBLICATION ")
		for i, node := range n.Publication {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
		if len(n.Options) > 0 {
			w.text("WITH ")
			w.definition(n.Options)
		}
		w.trimSpace()
		return
	case "ALTER_SUBSCRIPTION_ENABLED":
		w.text(" ")
		if len(n.Options) != 1 {
			w.invalid("invalid ALTER SUBSCRIPTION ENABLED option")
			return
		}
		d := n.Options[0].GetDefElem()
		if d == nil || d.Defname != "enabled" {
			w.invalid("invalid ALTER SUBSCRIPTION ENABLED option")
			return
		}
		if optBooleanValue(d.Arg) {
			w.text("ENABLE")
		} else {
			w.text("DISABLE")
		}
	default:
		w.invalid("invalid ALTER SUBSCRIPTION kind %v", n.Kind)
	}
}

func (w *deparseWriter) alterPublication(n *p.AlterPublicationStmt) {
	if n == nil {
		w.invalid("nil ALTER PUBLICATION statement")
		return
	}
	w.text("ALTER PUBLICATION ")
	w.ident(n.Pubname)
	w.text(" ")
	if len(n.Pubobjects) > 0 {
		switch n.Action.String() {
		case "AP_SetObjects":
			w.text("SET ")
		case "AP_AddObjects":
			w.text("ADD ")
		case "AP_DropObjects":
			w.text("DROP ")
		default:
			w.invalid("invalid ALTER PUBLICATION action %v", n.Action)
			return
		}
		w.publicationObjects(n.Pubobjects)
	} else if len(n.Options) > 0 {
		w.text("SET ")
		w.definition(n.Options)
	} else {
		w.invalid("ALTER PUBLICATION requires objects or options")
	}
}

func (w *deparseWriter) alterTSConfiguration(n *p.AlterTSConfigurationStmt) {
	if n == nil {
		w.invalid("nil ALTER TEXT SEARCH CONFIGURATION statement")
		return
	}
	w.text("ALTER TEXT SEARCH CONFIGURATION ")
	w.name(n.Cfgname)
	w.text(" ")
	switch n.Kind.String() {
	case "ALTER_TSCONFIG_ADD_MAPPING":
		w.text("ADD MAPPING FOR ")
		w.names(n.Tokentype)
		w.text(" WITH ")
		w.anyNameList(n.Dicts)
	case "ALTER_TSCONFIG_ALTER_MAPPING_FOR_TOKEN":
		w.text("ALTER MAPPING FOR ")
		w.names(n.Tokentype)
		w.text(" WITH ")
		w.anyNameList(n.Dicts)
	case "ALTER_TSCONFIG_REPLACE_DICT":
		if len(n.Dicts) == 0 {
			w.invalid("missing dictionaries")
			return
		}
		w.text("ALTER MAPPING REPLACE ")
		w.name(n.Dicts[0].GetList().GetItems())
		w.text(" WITH ")
		w.name(n.Dicts[1].GetList().GetItems())
	case "ALTER_TSCONFIG_REPLACE_DICT_FOR_TOKEN":
		w.text("ALTER MAPPING FOR ")
		w.names(n.Tokentype)
		w.text(" REPLACE ")
		w.name(n.Dicts[0].GetList().GetItems())
		w.text(" WITH ")
		w.name(n.Dicts[1].GetList().GetItems())
	case "ALTER_TSCONFIG_DROP_MAPPING":
		w.text("DROP MAPPING ")
		if n.MissingOk {
			w.text("IF EXISTS ")
		}
		w.text("FOR ")
		w.names(n.Tokentype)
	default:
		w.invalid("invalid TS configuration kind %v", n.Kind)
	}
}

func (w *deparseWriter) rule(n *p.RuleStmt) {
	if n == nil {
		w.invalid("nil RULE statement")
		return
	}
	w.text("CREATE ")
	if n.Replace {
		w.text("OR REPLACE ")
	}
	w.text("RULE ")
	w.ident(n.Rulename)
	w.text(" AS ON ")
	switch n.Event.String() {
	case "CMD_SELECT":
		w.text("SELECT ")
	case "CMD_UPDATE":
		w.text("UPDATE ")
	case "CMD_INSERT":
		w.text("INSERT ")
	case "CMD_DELETE":
		w.text("DELETE ")
	case "CMD_MERGE":
		w.text("MERGE ")
	default:
		w.invalid("invalid RULE event %v", n.Event)
		return
	}
	w.text("TO ")
	w.relation(n.Relation, deparseRelationNormal)
	w.text(" ")
	if n.WhereClause != nil {
		w.text("WHERE ")
		w.expr(n.WhereClause)
		w.text(" ")
	}
	w.text("DO ")
	if n.Instead {
		w.text("INSTEAD ")
	}
	if len(n.Actions) == 0 {
		w.text("NOTHING")
	} else if len(n.Actions) == 1 {
		w.ruleAction(n.Actions[0])
	} else {
		w.text("(")
		for i, action := range n.Actions {
			if i > 0 {
				w.text("; ")
			}
			w.ruleAction(action)
		}
		w.text(")")
	}
}

func (w *deparseWriter) ruleAction(node *p.Node) {
	if node == nil {
		w.invalid("nil RULE action")
		return
	}
	switch node.Node.(type) {
	case *p.Node_SelectStmt, *p.Node_InsertStmt, *p.Node_UpdateStmt, *p.Node_DeleteStmt:
		w.stmt(node)
	case *p.Node_NotifyStmt:
		w.utilityStmt(node)
	default:
		w.invalid("unpermitted RULE action %T", node.Node)
	}
}
