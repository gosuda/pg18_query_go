package pg_query

import (
	"github.com/gosuda/pg18_query_go/internal/neoparse"
	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func (w *deparseWriter) ddlValue(node *p.Node) {
	if node == nil {
		w.text("NULL")
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_Integer:
		w.integer(int64(n.Integer.Ival))
	case *p.Node_Float:
		w.text(n.Float.Fval)
	case *p.Node_Boolean:
		if n.Boolean.Boolval {
			w.text("true")
		} else {
			w.text("false")
		}
	case *p.Node_String_:
		w.literal(n.String_.Sval)
	case *p.Node_TypeName:
		w.typeName(n.TypeName)
	default:
		w.expr(node)
	}
}
func (w *deparseWriter) defArg(node *p.Node, operator bool) {
	if node == nil {
		return
	}
	if list := node.GetList(); list != nil {
		w.qualifiedOperator(list.Items)
		return
	}
	if value := node.GetString_(); value != nil {
		if value.Sval == "none" && !operator {
			w.text("NONE")
			return
		}
		if keyword, ok := neoparse.ScanKeywords[value.Sval]; ok && keyword.KeywordKind == neoparse.KeywordKind_RESERVED_KEYWORD {
			w.text(value.Sval)
		} else {
			w.literal(value.Sval)
		}
		return
	}
	w.ddlValue(node)
}
func (w *deparseWriter) definition(options []*p.Node) {
	w.text("(")
	for i, node := range options {
		if i > 0 {
			w.text(", ")
		}
		option := node.GetDefElem()
		if option == nil {
			w.invalid("expected definition option")
			return
		}
		w.ident(option.Defname)
		if option.Arg != nil {
			w.text(" = ")
			w.defArg(option.Arg, false)
		}
	}
	w.text(")")
}
func (w *deparseWriter) relOptions(options []*p.Node) {
	w.text("(")
	for i, node := range options {
		if i > 0 {
			w.text(", ")
		}
		option := node.GetDefElem()
		if option == nil {
			w.invalid("expected relation option")
			return
		}
		if option.Defnamespace != "" {
			w.ident(option.Defnamespace)
			w.text(".")
		}
		if option.Defname != "" {
			w.ident(option.Defname)
			if option.Arg != nil {
				w.text("=")
			}
		}
		w.defArg(option.Arg, false)
	}
	w.text(")")
}
func (w *deparseWriter) withOptions(options []*p.Node) {
	if len(options) > 0 {
		w.trimSpace()
		w.text(" WITH ")
		w.relOptions(options)
	}
}
func (w *deparseWriter) genericOptions(options []*p.Node, alter bool) {
	if len(options) == 0 {
		return
	}
	w.text("OPTIONS (")
	for i, node := range options {
		if i > 0 {
			w.text(", ")
		}
		option := node.GetDefElem()
		if option == nil {
			w.invalid("expected generic option")
			return
		}
		if alter {
			switch option.Defaction {
			case p.DefElemAction_DEFELEM_ADD:
				w.text("ADD ")
			case p.DefElemAction_DEFELEM_SET:
				w.text("SET ")
			case p.DefElemAction_DEFELEM_DROP:
				w.text("DROP ")
			}
		}
		w.ident(option.Defname)
		if option.Arg != nil {
			w.text(" ")
			w.literal(deparseString(option.Arg))
		}
	}
	w.text(")")
}
func (w *deparseWriter) seqOptElem(d *p.DefElem) {
	if d == nil {
		w.invalid("expected sequence option")
		return
	}
	switch d.Defname {
	case "as":
		w.text("AS ")
		w.typeName(d.Arg.GetTypeName())
	case "cache", "increment", "start", "restart":
		w.text(seqOptKeyword(d.Defname))
		if d.Arg != nil {
			w.text(" ")
			w.numericOnly(d.Arg)
		}
	case "cycle":
		if optBooleanValue(d.Arg) {
			w.text("CYCLE")
		} else {
			w.text("NO CYCLE")
		}
	case "owned_by":
		w.text("OWNED BY ")
		w.name(d.Arg.GetList().GetItems())
	case "sequence_name":
		w.text("SEQUENCE NAME ")
		w.name(d.Arg.GetList().GetItems())
	case "minvalue", "maxvalue":
		if d.Arg == nil {
			w.text("NO ")
		}
		w.text(seqOptKeyword(d.Defname))
		if d.Arg != nil {
			w.text(" ")
			w.numericOnly(d.Arg)
		}
	case "logged", "unlogged":
		w.text(seqOptKeyword(d.Defname))
	default:
		w.invalid("unrecognized sequence option %q", d.Defname)
	}
}

func seqOptKeyword(name string) string {
	switch name {
	case "cache":
		return "CACHE"
	case "increment":
		return "INCREMENT"
	case "start":
		return "START"
	case "restart":
		return "RESTART"
	case "minvalue":
		return "MINVALUE"
	case "maxvalue":
		return "MAXVALUE"
	case "logged":
		return "LOGGED"
	case "unlogged":
		return "UNLOGGED"
	}
	return name
}

func (w *deparseWriter) sequenceOptions(options []*p.Node) {
	for i, node := range options {
		if i > 0 {
			w.text(" ")
		}
		d := node.GetDefElem()
		if d == nil {
			w.invalid("expected sequence option")
			return
		}
		w.seqOptElem(d)
	}
}

func (w *deparseWriter) numericOnly(node *p.Node) {
	if node == nil {
		w.invalid("nil numeric value")
		return
	}
	switch n := node.Node.(type) {
	case *p.Node_Integer:
		w.integer(int64(n.Integer.Ival))
	case *p.Node_Float:
		w.text(n.Float.Fval)
	case *p.Node_AConst:
		a := n.AConst
		if a.Isnull {
			w.invalid("null numeric value")
			return
		}
		switch v := a.Val.(type) {
		case *p.A_Const_Ival:
			w.integer(int64(v.Ival.Ival))
		case *p.A_Const_Fval:
			w.text(v.Fval.Fval)
		default:
			w.invalid("invalid numeric value %T", a.Val)
		}
	default:
		w.invalid("invalid numeric value %T", node.Node)
	}
}

func optBooleanValue(node *p.Node) bool {
	if node == nil {
		return true
	}
	switch n := node.Node.(type) {
	case *p.Node_Boolean:
		return n.Boolean.Boolval
	case *p.Node_Integer:
		return n.Integer.Ival != 0
	case *p.Node_String_:
		s := n.String_.Sval
		return len(s) >= 2 && (s[0] == 'o' || s[0] == 'O') && (s[1] == 'n' || s[1] == 'N')
	case *p.Node_AConst:
		a := n.AConst
		if a.Isnull {
			return true
		}
		switch v := a.Val.(type) {
		case *p.A_Const_Boolval:
			return v.Boolval.Boolval
		case *p.A_Const_Ival:
			return v.Ival.Ival != 0
		case *p.A_Const_Sval:
			s := v.Sval.Sval
			if len(s) >= 3 && (s[0] == 'o' || s[0] == 'O') {
				return s[1] == 'n' || s[1] == 'N'
			}
			return false
		}
	}
	return false
}

func (w *deparseWriter) columnList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.ident(deparseString(node))
	}
}

func (w *deparseWriter) qualifiedNameList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.relation(node.GetRangeVar(), deparseRelationNormal)
	}
}

func (w *deparseWriter) relationExprList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.relation(node.GetRangeVar(), deparseRelationNormal)
	}
}

func (w *deparseWriter) optBooleanOrString(s string) {
	switch s {
	case "true":
		w.text("TRUE")
	case "false":
		w.text("FALSE")
	case "on":
		w.text("ON")
	case "off":
		w.text("OFF")
	default:
		w.nonReservedWordOrSconst(s)
	}
}

func (w *deparseWriter) nonReservedWordOrSconst(s string) {
	if s == "" {
		w.text("''")
		return
	}
	if len(s) >= 64 {
		w.literal(s)
		return
	}
	w.ident(s)
}
func (w *deparseWriter) dropBehavior(behavior p.DropBehavior) {
	if behavior == p.DropBehavior_DROP_CASCADE {
		w.trimSpace()
		w.text(" CASCADE")
	}
}
func (w *deparseWriter) role(role *p.RoleSpec) {
	if role == nil {
		w.invalid("nil role specification")
		return
	}
	switch role.Roletype {
	case p.RoleSpecType_ROLESPEC_CSTRING:
		w.ident(role.Rolename)
	case p.RoleSpecType_ROLESPEC_CURRENT_ROLE:
		w.text("CURRENT_ROLE")
	case p.RoleSpecType_ROLESPEC_CURRENT_USER:
		w.text("CURRENT_USER")
	case p.RoleSpecType_ROLESPEC_SESSION_USER:
		w.text("SESSION_USER")
	case p.RoleSpecType_ROLESPEC_PUBLIC:
		w.text("public")
	default:
		w.invalid("unrecognized role specification %s", role.Roletype)
	}
}
func (w *deparseWriter) roles(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.role(node.GetRoleSpec())
	}
}
func (w *deparseWriter) typeList(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.typeName(node.GetTypeName())
	}
}
func (w *deparseWriter) objectWithArgs(object *p.ObjectWithArgs, operator bool) {
	if object == nil {
		w.invalid("nil object with arguments")
		return
	}
	if operator {
		w.operatorName(object.Objname)
	} else {
		w.name(object.Objname)
	}
	if object.ArgsUnspecified {
		return
	}
	w.text("(")
	if len(object.Objfuncargs) > 0 {
		for i, node := range object.Objfuncargs {
			if i > 0 {
				w.text(", ")
			}
			w.functionParameter(node.GetFunctionParameter())
		}
	} else {
		for i, node := range object.Objargs {
			if i > 0 {
				w.text(", ")
			}
			if node == nil || node.Node == nil {
				w.text("NONE")
			} else {
				w.typeName(node.GetTypeName())
			}
		}
	}
	w.text(")")
}
func (w *deparseWriter) columnDef(column *p.ColumnDef) {
	if column == nil {
		w.invalid("nil column definition")
		return
	}
	if column.Colname != "" {
		w.ident(column.Colname)
		w.text(" ")
	}
	if column.TypeName != nil {
		w.typeName(column.TypeName)
		w.text(" ")
	}
	if column.StorageName != "" {
		w.text("STORAGE ")
		w.text(column.StorageName)
		w.text(" ")
	}
	if column.RawDefault != nil {
		w.text("USING ")
		w.expr(column.RawDefault)
		w.text(" ")
	}
	if column.Compression != "" {
		w.text("COMPRESSION ")
		w.text(column.Compression)
		w.text(" ")
	}
	if len(column.Fdwoptions) > 0 {
		w.genericOptions(column.Fdwoptions, false)
		w.text(" ")
	}
	for _, node := range column.Constraints {
		w.constraint(node.GetConstraint(), false)
		w.text(" ")
	}
	if column.CollClause != nil {
		w.text("COLLATE ")
		w.name(column.CollClause.Collname)
	}
	w.trimSpace()
}
func (w *deparseWriter) indexElem(elem *p.IndexElem) {
	if elem == nil {
		w.invalid("nil index element")
		return
	}
	if elem.Name != "" {
		w.ident(elem.Name)
	} else if elem.Expr != nil {
		switch elem.Expr.Node.(type) {
		case *p.Node_FuncCall, *p.Node_SqlvalueFunction, *p.Node_CoalesceExpr, *p.Node_MinMaxExpr, *p.Node_XmlExpr, *p.Node_XmlSerialize:
			w.windowless(elem.Expr)
		default:
			w.text("(")
			w.expr(elem.Expr)
			w.text(")")
		}
	} else {
		w.invalid("index element has neither name nor expression")
		return
	}
	if len(elem.Collation) > 0 {
		w.text(" COLLATE ")
		w.name(elem.Collation)
	}
	if len(elem.Opclass) > 0 {
		w.text(" ")
		w.name(elem.Opclass)
		if len(elem.Opclassopts) > 0 {
			w.relOptions(elem.Opclassopts)
		}
	}
	switch elem.Ordering {
	case p.SortByDir_SORTBY_ASC:
		w.text(" ASC")
	case p.SortByDir_SORTBY_DESC:
		w.text(" DESC")
	case p.SortByDir_SORTBY_USING:
		w.invalid("USING is not an index ordering")
	}
	switch elem.NullsOrdering {
	case p.SortByNulls_SORTBY_NULLS_FIRST:
		w.text(" NULLS FIRST")
	case p.SortByNulls_SORTBY_NULLS_LAST:
		w.text(" NULLS LAST")
	}
}
func (w *deparseWriter) indexElems(nodes []*p.Node) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		w.indexElem(node.GetIndexElem())
	}
}
func (w *deparseWriter) periodColumns(nodes []*p.Node, period bool) {
	for i, node := range nodes {
		if i > 0 {
			w.text(", ")
		}
		if period && i == len(nodes)-1 {
			w.text("PERIOD ")
		}
		w.ident(deparseString(node))
	}
}
