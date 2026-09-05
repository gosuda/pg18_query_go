package pg_query

import (
	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func (w *deparseWriter) functionParameter(n *p.FunctionParameter) {
	if n == nil {
		w.invalid("nil function parameter")
		return
	}
	switch n.Mode.String() {
	case "FUNC_PARAM_IN":
		w.text("IN ")
	case "FUNC_PARAM_OUT":
		w.text("OUT ")
	case "FUNC_PARAM_INOUT":
		w.text("INOUT ")
	case "FUNC_PARAM_VARIADIC":
		w.text("VARIADIC ")
	case "FUNC_PARAM_TABLE":
	case "FUNC_PARAM_DEFAULT":
	default:
		w.invalid("invalid function parameter mode %v", n.Mode)
		return
	}
	if n.Name != "" {
		w.ident(n.Name)
		w.text(" ")
	}
	w.typeName(n.ArgType)
	if n.Defexpr != nil {
		w.text(" = ")
		w.expr(n.Defexpr)
	}
	w.trimSpace()
}

func (w *deparseWriter) constraint(c *p.Constraint, alterDomain bool) {
	if c == nil {
		w.invalid("nil constraint")
		return
	}
	if c.Conname != "" {
		w.text("CONSTRAINT ")
		w.ident(c.Conname)
		w.text(" ")
	}
	switch c.Contype.String() {
	case "CONSTR_NULL":
		w.text("NULL ")
	case "CONSTR_NOTNULL":
		w.text("NOT NULL ")
	case "CONSTR_DEFAULT":
		w.text("DEFAULT ")
		w.bExpr(c.RawExpr)
	case "CONSTR_IDENTITY":
		w.text("GENERATED ")
		switch c.GeneratedWhen {
		case "a":
			w.text("ALWAYS ")
		case "d":
			w.text("BY DEFAULT ")
		default:
			w.invalid("invalid identity generation %q", c.GeneratedWhen)
			return
		}
		w.text("AS IDENTITY ")
		if len(c.Options) > 0 {
			w.text("(")
			for i, node := range c.Options {
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
			w.text(" ) ")
		}
	case "CONSTR_GENERATED":
		w.text("GENERATED ALWAYS AS (")
		w.expr(c.RawExpr)
		w.text(") ")
		if c.GeneratedKind == "s" {
			w.text("STORED ")
		} else {
			w.text("VIRTUAL ")
		}
	case "CONSTR_CHECK":
		w.text("CHECK (")
		w.expr(c.RawExpr)
		w.text(") ")
	case "CONSTR_PRIMARY":
		w.text("PRIMARY KEY ")
	case "CONSTR_UNIQUE":
		w.text("UNIQUE ")
		if c.NullsNotDistinct {
			w.text("NULLS NOT DISTINCT ")
		}
	case "CONSTR_EXCLUSION":
		w.text("EXCLUDE ")
		if c.AccessMethod != "" && c.AccessMethod != "btree" {
			w.text("USING ")
			w.ident(c.AccessMethod)
			w.text(" ")
		}
		w.text("(")
		for i, node := range c.Exclusions {
			if i > 0 {
				w.text(", ")
			}
			pair := node.GetList().GetItems()
			if len(pair) != 2 {
				w.invalid("exclusion element is not an index/operator pair")
				return
			}
			w.indexElem(pair[0].GetIndexElem())
			w.text(" WITH ")
			w.qualifiedOperator(pair[1].GetList().GetItems())
		}
		w.text(") ")
		if c.WhereClause != nil {
			w.text("WHERE (")
			w.expr(c.WhereClause)
			w.text(") ")
		}
	case "CONSTR_FOREIGN":
		if len(c.FkAttrs) > 0 {
			w.text("FOREIGN KEY ")
		}
	case "CONSTR_ATTR_DEFERRABLE":
		w.text("DEFERRABLE ")
	case "CONSTR_ATTR_NOT_DEFERRABLE":
		w.text("NOT DEFERRABLE ")
	case "CONSTR_ATTR_DEFERRED":
		w.text("INITIALLY DEFERRED ")
	case "CONSTR_ATTR_IMMEDIATE":
		w.text("INITIALLY IMMEDIATE ")
	case "CONSTR_ATTR_ENFORCED":
		w.text("ENFORCED ")
	case "CONSTR_ATTR_NOT_ENFORCED":
		w.text("NOT ENFORCED ")
	default:
		w.invalid("invalid constraint type %v", c.Contype)
		return
	}
	if len(c.Keys) > 0 && !(alterDomain && len(c.Keys) == 1 && deparseString(c.Keys[0]) == "value") {
		parens := len(c.Keys) != 1 || (c.Contype != p.ConstrType_CONSTR_NULL && c.Contype != p.ConstrType_CONSTR_NOTNULL)
		if parens {
			w.text("(")
		}
		w.columnList(c.Keys)
		if c.WithoutOverlaps {
			w.text(" WITHOUT OVERLAPS")
		}
		if parens {
			w.text(")")
		}
		w.text(" ")
	}
	if len(c.FkAttrs) > 0 {
		w.text("(")
		w.periodColumns(c.FkAttrs, c.FkWithPeriod)
		w.text(") ")
	}
	if c.Pktable != nil {
		w.text("REFERENCES ")
		w.relation(c.Pktable, deparseRelationNormal)
		w.text(" ")
		if len(c.PkAttrs) > 0 {
			w.text("(")
			w.periodColumns(c.PkAttrs, c.PkWithPeriod)
			w.text(") ")
		}
	}
	switch c.FkMatchtype {
	case "", "s":
	case "f":
		w.text("MATCH FULL ")
	default:
		w.invalid("invalid foreign key match type %q", c.FkMatchtype)
		return
	}
	switch c.FkUpdAction {
	case "", "a":
	case "r":
		w.text("ON UPDATE RESTRICT ")
	case "c":
		w.text("ON UPDATE CASCADE ")
	case "n":
		w.text("ON UPDATE SET NULL ")
	case "d":
		w.text("ON UPDATE SET DEFAULT ")
	default:
		w.invalid("invalid foreign key update action %q", c.FkUpdAction)
		return
	}
	switch c.FkDelAction {
	case "", "a":
	case "r":
		w.text("ON DELETE RESTRICT ")
	case "c":
		w.text("ON DELETE CASCADE ")
	case "n", "d":
		w.text("ON DELETE SET ")
		if c.FkDelAction == "d" {
			w.text("DEFAULT ")
		} else {
			w.text("NULL ")
		}
		if len(c.FkDelSetCols) > 0 {
			w.text("(")
			for i, col := range c.FkDelSetCols {
				if i > 0 {
					w.text(", ")
				}
				w.ident(deparseString(col))
			}
			w.text(")")
			w.text(" ")
		}
	default:
		w.invalid("invalid foreign key delete action %q", c.FkDelAction)
		return
	}
	if len(c.Including) > 0 {
		w.text("INCLUDE (")
		w.columnList(c.Including)
		w.text(") ")
	}
	switch c.Contype.String() {
	case "CONSTR_PRIMARY", "CONSTR_UNIQUE", "CONSTR_EXCLUSION":
		w.withOptions(c.Options)
	}
	if c.Indexname != "" {
		w.text("USING INDEX ")
		w.ident(c.Indexname)
		w.text(" ")
	}
	if c.Indexspace != "" {
		w.text("USING INDEX TABLESPACE ")
		w.ident(c.Indexspace)
		w.text(" ")
	}
	if c.Deferrable {
		w.text("DEFERRABLE ")
	}
	if c.Initdeferred {
		w.text("INITIALLY DEFERRED ")
	}
	if c.IsNoInherit {
		w.text("NO INHERIT ")
	}
	if (c.Contype.String() == "CONSTR_FOREIGN" || c.Contype.String() == "CONSTR_CHECK") && !c.IsEnforced {
		w.text("NOT ENFORCED ")
	}
	if c.SkipValidation {
		w.text("NOT VALID ")
	}
	w.trimSpace()
}

func (w *deparseWriter) atAlterConstraint(c *p.ATAlterConstraint) {
	if c == nil {
		w.invalid("nil ALTER CONSTRAINT specification")
		return
	}
	if c.Conname != "" {
		w.text("CONSTRAINT ")
		w.ident(c.Conname)
		w.text(" ")
	}
	if c.AlterEnforceability {
		if c.IsEnforced {
			w.text("ENFORCED ")
		} else {
			w.text("NOT ENFORCED ")
		}
	}
	if c.AlterDeferrability {
		if c.Initdeferred {
			w.text("INITIALLY DEFERRED ")
		} else if c.Deferrable {
			w.text("DEFERRABLE ")
		} else {
			w.text("NOT DEFERRABLE ")
		}
	}
	if c.AlterInheritability {
		if c.Noinherit {
			w.text("NO INHERIT ")
		} else {
			w.text("INHERIT ")
		}
	}
	w.trimSpace()
}
