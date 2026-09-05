package neoparse

import (
	"fmt"
	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"reflect"
	"strconv"
	"strings"
)

func grammarUtilityRelation(relation, persistence any) *RangeVar {
	rel := toRangeVar(relation)
	switch strings.ToUpper(toStr(persistence)) {
	case "TEMP", "TEMPORARY", "LOCAL", "GLOBAL":
		rel.Relpersistence = "t"
	case "UNLOGGED":
		rel.Relpersistence = "u"
	default:
		rel.Relpersistence = "p"
	}
	return rel
}

func grammarView(relation, persistence, aliases, query, options, check any, replace, recursive bool) *ViewStmt {
	view := &ViewStmt{View: grammarUtilityRelation(relation, persistence), Aliases: toNodeSlice(toStrSlice(aliases)), Query: toNode(query), Options: toNodeSlice(toDefElemSlice(options)), WithCheckOption: enumViewCheckOption(toStr(check)), Replace: replace}
	if recursive {
		view.Query = grammarRecursiveViewQuery(view.View.Relname, toStrSlice(aliases), view.Query)
	}
	return view
}

func grammarRecursiveViewQuery(name string, aliases []string, query Node) Node {
	targets := make([]Node, len(aliases))
	for i, alias := range aliases {
		targets[i] = toNode(&ResTarget{Val: toNode(&ColumnRef{Fields: toNodeSlice([]string{alias}), Location: -1}), Location: -1})
	}
	cte := &CommonTableExpr{Ctename: name, Aliascolnames: toNodeSlice(aliases), Ctequery: query, Ctematerialized: pg_query.CTEMaterialize(1), Location: -1}
	return toNode(&SelectStmt{
		Op:          pg_query.SetOperation_SETOP_NONE,
		TargetList:  targets,
		FromClause:  []Node{toNode(&RangeVar{Relname: name, Inh: true, Relpersistence: "p", Location: -1})},
		WithClause:  &WithClause{Recursive: true, Ctes: []Node{toNode(cte)}, Location: -1},
		LimitOption: pg_query.LimitOption(1),
	})
}

// Statement slice helpers
func toStatement(v any) Statement { return toNode(v) }

func toStmtSlice(v any) []Statement {
	if v == nil {
		return nil
	}
	if sl, ok := v.([]Statement); ok {
		return sl
	}
	if s, ok := v.(Statement); ok {
		return []Statement{s}
	}
	return []Statement{toStatement(v)}
}

func singleStmtSlice(v any) []Statement {
	stmt := toStatement(v)
	if stmt == nil {
		return nil
	}
	return []Statement{stmt}
}

func appendToStmtSlice(slice, item any) []Statement {
	sl := toStmtSlice(slice)
	st := toStatement(item)
	if st == nil {
		return sl
	}
	return append(sl, st)
}

// Node helpers

func appendToNodeSlice(slice, item any) []Node {
	sl := toNodeSlice(slice)
	n := toNode(item)
	if n == nil {
		return sl
	}
	return append(sl, n)
}

// String helpers
func toStr(v any) string {
	if node, ok := v.(Node); ok {
		if node == nil {
			return ""
		}
		if s := node.GetString_(); s != nil {
			return s.Sval
		}
		if t := node.GetTypeName(); t != nil {
			return toStr(t)
		}
		if r := node.GetRoleSpec(); r != nil {
			return toStr(r)
		}
	}
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if s, ok := v.(*String); ok {
		if s == nil {
			return ""
		}
		return s.Sval
	}
	if role, ok := v.(*RoleSpec); ok {
		if role == nil {
			return ""
		}
		switch role.Roletype.String() {
		case "ROLESPEC_CURRENT_ROLE":
			return "current_role"
		case "ROLESPEC_CURRENT_USER":
			return "current_user"
		case "ROLESPEC_SESSION_USER":
			return "session_user"
		case "ROLESPEC_PUBLIC":
			return "public"
		}
		return role.Rolename
	}
	if tn, ok := v.(*TypeName); ok {
		if tn == nil {
			return ""
		}
		return strings.Join(toStrSlice(tn.Names), ".") + strings.Repeat("[]", len(tn.ArrayBounds))
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toStrSlice(v any) []string {
	if v == nil {
		return nil
	}
	if sl, ok := v.([]string); ok {
		return sl
	}
	if s, ok := v.(string); ok {
		return []string{s}
	}
	if nodes, ok := v.([]Node); ok {
		names := make([]string, len(nodes))
		for i, node := range nodes {
			names[i] = toStr(node)
		}
		return names
	}
	if values, ok := v.([]any); ok {
		names := make([]string, len(values))
		for i, value := range values {
			names[i] = toStr(value)
		}
		return names
	}
	return nil
}

func appendToStrSlice(slice, item any) []string {
	sl := toStrSlice(slice)
	s := toStr(item)
	if s == "" {
		return sl
	}
	return append(sl, s)
}

// ResTarget helpers
func toResTarget(v any) *ResTarget {
	if node, ok := v.(Node); ok {
		if target := node.GetResTarget(); target != nil {
			return target
		}
		if node != nil {
			return &ResTarget{Val: node}
		}
		return nil
	}
	if v == nil {
		return nil
	}
	if rt, ok := v.(*ResTarget); ok {
		return rt
	}
	if n, ok := v.(Node); ok {
		return &ResTarget{Val: toNode(n)}
	}
	return nil
}

func toResTargetSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToResTargetSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// RangeVar & Relation helpers
func toRangeVar(v any) *RangeVar {
	if node, ok := v.(Node); ok {
		return node.GetRangeVar()
	}
	if v == nil {
		return nil
	}
	if rv, ok := v.(*RangeVar); ok {
		return rv
	}
	if s, ok := v.(string); ok {
		return &RangeVar{Relname: s, Inh: true, Relpersistence: "p"}
	}
	if sl, ok := v.([]string); ok {
		if len(sl) == 1 {
			return &RangeVar{Relname: sl[0], Inh: true, Relpersistence: "p"}
		} else if len(sl) == 2 {
			return &RangeVar{Schemaname: sl[0], Relname: sl[1], Inh: true, Relpersistence: "p"}
		} else if len(sl) >= 3 {
			return &RangeVar{Catalogname: sl[0], Schemaname: sl[1], Relname: sl[2], Inh: true, Relpersistence: "p"}
		}
	}
	return nil
}

func toRangeVarSlice(v any) []Node {
	if _, ok := v.([]string); ok {
		return singleRangeVarSlice(v)
	}
	if _, ok := v.(string); ok {
		return singleRangeVarSlice(v)
	}
	nodes := toNodeSlice(v)
	result := nodes[:0]
	for _, node := range nodes {
		if node.GetRangeVar() != nil {
			result = append(result, node)
		}
	}
	return result
}

func appendToRangeVarSlice(slice, item any) []Node {
	nodes := toRangeVarSlice(slice)
	if relation := toRangeVar(item); relation != nil {
		return append(nodes, toNode(relation))
	}
	return nodes
}

func makeRangeVar(name any, inh bool) *RangeVar {
	rv := toRangeVar(name)
	if rv == nil {
		rv = &RangeVar{Relname: toStr(name), Relpersistence: "p"}
	}
	rv.Inh = inh
	return rv
}

func makeRangeVarWithIndirection(rel any, indir any) *RangeVar {
	first := toStr(rel)
	parts := toStrSlice(indir)
	if len(parts) == 0 {
		return &RangeVar{Relname: first, Inh: true, Relpersistence: "p"}
	}
	if len(parts) == 1 {
		return &RangeVar{Schemaname: first, Relname: parts[0], Inh: true, Relpersistence: "p"}
	}
	if len(parts) == 2 {
		return &RangeVar{Catalogname: first, Schemaname: parts[0], Relname: parts[1], Inh: true, Relpersistence: "p"}
	}
	return &RangeVar{Relname: parts[len(parts)-1], Inh: true, Relpersistence: "p"}
}

type AliasClause struct {
	Aliasname string
	Colnames  []string
}

func makeAliasClause(name any, colnames any) *AliasClause {
	return &AliasClause{
		Aliasname: toStr(name),
		Colnames:  toStrSlice(colnames),
	}
}

func toAliasParts(v any) (string, []string) {
	if v == nil {
		return "", nil
	}
	if a, ok := v.(*AliasClause); ok && a != nil {
		return a.Aliasname, a.Colnames
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return toStr(v), nil
}

func applyAliasToRangeVar(rv *RangeVar, aliasVal any) {
	if rv == nil || aliasVal == nil {
		return
	}
	name, cols := toAliasParts(aliasVal)
	rv.Alias = &pg_query.Alias{Aliasname: name, Colnames: toNodeSlice(cols)}
}

func makeRangeSubselect(subquery any, aliasVal any, lateral bool) *RangeSubselect {
	n := &RangeSubselect{Lateral: lateral, Subquery: toNode(subquery)}
	if aliasVal != nil {
		name, cols := toAliasParts(aliasVal)
		n.Alias = &pg_query.Alias{Aliasname: name, Colnames: toNodeSlice(cols)}
	}
	return n
}

func toAlias(v any) string {
	name, _ := toAliasParts(v)
	return name
}

// JoinExpr helpers
func toJoinExpr(v any) *JoinExpr {
	if node, ok := v.(Node); ok {
		return node.GetJoinExpr()
	}
	if v == nil {
		return nil
	}
	if j, ok := v.(*JoinExpr); ok {
		return j
	}
	return nil
}

type JoinQual struct {
	Alias string
	Quals Node
	Using []string
}

func makeJoinExpr(jtype JoinType, larg, rarg, qual any) *JoinExpr {
	je := &JoinExpr{Jointype: enumJoinType(toStr(jtype)), Larg: toNode(larg), Rarg: toNode(rarg)}
	if qual != nil {
		if jq, ok := qual.(*JoinQual); ok {
			je.Quals = jq.Quals
			je.UsingClause = toNodeSlice(jq.Using)
			if jq.Alias != "" {
				je.JoinUsingAlias = &pg_query.Alias{Aliasname: jq.Alias}
			}
		} else if n, ok := qual.(Node); ok {
			je.Quals = n
		}
	}
	return je
}

func applyRangeFunctionAlias(rf *RangeFunction, alias any) {
	if typed, ok := alias.(*RangeFunction); ok {
		rf.Alias, rf.Coldeflist = typed.Alias, typed.Coldeflist
		return
	}
	if alias != nil {
		name, cols := toAliasParts(alias)
		rf.Alias = &pg_query.Alias{Aliasname: name, Colnames: toNodeSlice(cols)}
	}
}

func makeNaturalJoinExpr(jtype JoinType, larg, rarg any) *JoinExpr {
	return &JoinExpr{Jointype: enumJoinType(toStr(jtype)), IsNatural: true, Larg: toNode(larg), Rarg: toNode(rarg)}
}

// SortBy helpers
func toSortBy(v any) *SortBy {
	if node, ok := v.(Node); ok {
		return node.GetSortBy()
	}
	if v == nil {
		return nil
	}
	if sb, ok := v.(*SortBy); ok {
		return sb
	}
	return nil
}

func toSortBySlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToSortBySlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

func toSortByDir(v any) SortByDir {
	if v == nil {
		return SortByDirDefault
	}
	if d, ok := v.(SortByDir); ok {
		return d
	}
	return SortByDirDefault
}

func toSortByNulls(v any) SortByNulls {
	if v == nil {
		return SortByNullsDefault
	}
	if n, ok := v.(SortByNulls); ok {
		return n
	}
	return SortByNullsDefault
}

// Limit & Offset helpers
type LimitInfo struct {
	Count  Node
	Offset Node
	Option int32
}

func combineLimitOffset(l1, l2 any) *LimitInfo {
	res := &LimitInfo{}
	if info1, ok := l1.(*LimitInfo); ok {
		if info1.Count != nil {
			res.Count = info1.Count
			res.Option = info1.Option
		}
		if info1.Offset != nil {
			res.Offset = info1.Offset
		}
	}
	if info2, ok := l2.(*LimitInfo); ok {
		if info2.Count != nil {
			res.Count = info2.Count
			res.Option = info2.Option
		}
		if info2.Offset != nil {
			res.Offset = info2.Offset
		}
	}
	return res
}

func applySelectOptions(sel *SelectStmt, sort, locking, limit, with any) string {
	if sort != nil {
		if len(sel.SortClause) != 0 {
			return "multiple ORDER BY clauses not allowed"
		}
		sel.SortClause = toSortBySlice(sort)
	}
	if limit != nil {
		info := limit.(*LimitInfo)
		if info.Count != nil && sel.LimitCount != nil {
			return "multiple LIMIT clauses not allowed"
		}
		if info.Offset != nil && sel.LimitOffset != nil {
			return "multiple OFFSET clauses not allowed"
		}
		applySelectLimit(sel, info)
	}
	sel.LockingClause = append(sel.LockingClause, toNodeSlice(locking)...)
	if with != nil {
		if sel.WithClause != nil {
			return "multiple WITH clauses not allowed"
		}
		sel.WithClause = toWithClause(with)
	}
	if sel.LimitOption == pg_query.LimitOption_LIMIT_OPTION_WITH_TIES {
		if len(sel.SortClause) == 0 {
			return "WITH TIES cannot be specified without ORDER BY"
		}
		for _, node := range sel.LockingClause {
			if node.GetLockingClause().WaitPolicy == pg_query.LockWaitPolicy_LockWaitSkip {
				return "WITH TIES cannot be specified with SKIP LOCKED"
			}
		}
	}
	return ""
}

func applySelectLimit(sel *SelectStmt, lim any) {
	if sel == nil || lim == nil {
		return
	}
	if info, ok := lim.(*LimitInfo); ok {
		if info.Count != nil {
			sel.LimitCount = info.Count
			sel.LimitOption = pg_query.LimitOption(info.Option + 1)
		}
		if info.Offset != nil {
			sel.LimitOffset = info.Offset
			if sel.LimitCount == nil {
				sel.LimitOption = pg_query.LimitOption_LIMIT_OPTION_COUNT
			}
		}
	} else if n, ok := lim.(Node); ok {
		sel.LimitCount = n
		sel.LimitOption = pg_query.LimitOption_LIMIT_OPTION_COUNT
	}
}

// SelectStmt helpers
func toSelectStmtPtr(v any) *SelectStmt {
	if node, ok := v.(Node); ok {
		return node.GetSelectStmt()
	}
	if v == nil {
		return nil
	}
	if sel, ok := v.(*SelectStmt); ok {
		return sel
	}
	return nil
}

type GroupClause struct {
	Items    []Node
	Distinct bool
}

func toSelectStmt(targetList, fromClause, whereClause, groupClause, havingClause, windowClause any) *SelectStmt {
	var group GroupClause
	if groupClause != nil {
		group = *groupClause.(*GroupClause)
	}
	return &SelectStmt{TargetList: toNodeSlice(toResTargetSlice(targetList)), FromClause: toNodeSlice(fromClause), WhereClause: toNode(whereClause), GroupClause: toNodeSlice(group.Items), GroupDistinct: group.Distinct, HavingClause: toNode(havingClause), WindowClause: toNodeSlice(toWindowDefSlice(windowClause)), LimitOption: pg_query.LimitOption(1), Op: pg_query.SetOperation(1)}
}

func makeSetOp(op pg_query.SetOperation, quant, larg, rarg any) *SelectStmt {
	return &SelectStmt{Op: op, All: strings.EqualFold(toStr(quant), "ALL"), Larg: toSelectStmtPtr(larg), Rarg: toSelectStmtPtr(rarg), LimitOption: pg_query.LimitOption_LIMIT_OPTION_DEFAULT}
}

// WindowDef helpers
func toWindowDef(v any) *WindowDef {
	if node, ok := v.(Node); ok {
		return node.GetWindowDef()
	}
	if v == nil {
		return nil
	}
	if wd, ok := v.(*WindowDef); ok {
		return wd
	}
	return nil
}

func toWindowDefSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToWindowDefSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// CTE helpers
func toWithClause(v any) *WithClause {
	if node, ok := v.(Node); ok {
		return node.GetWithClause()
	}
	if v == nil {
		return nil
	}
	if wc, ok := v.(*WithClause); ok {
		return wc
	}
	if ctes, ok := v.([]Node); ok {
		return &WithClause{Ctes: toNodeSlice(ctes)}
	}
	return nil
}

func toCommonTableExpr(v any) *CommonTableExpr {
	if node, ok := v.(Node); ok {
		return node.GetCommonTableExpr()
	}
	if v == nil {
		return nil
	}
	if cte, ok := v.(*CommonTableExpr); ok {
		return cte
	}
	return nil
}

func toCommonTableExprSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToCommonTableExprSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// ValuesList helpers
func toValuesList(v any) []Node {
	if rows, ok := v.([][]Node); ok {
		out := make([]Node, len(rows))
		for i, row := range rows {
			out[i] = toNode(&List{Items: toNodeSlice(row)})
		}
		return out
	}
	return toNodeSlice(v)
}

// ColumnDef & Constraint helpers
func toColDef(v any) *ColumnDef {
	if node, ok := v.(Node); ok {
		return node.GetColumnDef()
	}
	if v == nil {
		return nil
	}
	if cd, ok := v.(*ColumnDef); ok {
		return cd
	}
	return nil
}

func toColDefSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToColDefSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

func toConstraint(v any) *Constraint {
	if node, ok := v.(Node); ok {
		return node.GetConstraint()
	}
	if v == nil {
		return nil
	}
	if c, ok := v.(*Constraint); ok {
		return c
	}
	return nil
}

func toConstraintSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToConstraintSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// AlterTableCmd helpers
func toAlterCmd(v any) *AlterTableCmd {
	if node, ok := v.(Node); ok {
		return node.GetAlterTableCmd()
	}
	if v == nil {
		return nil
	}
	if cmd, ok := v.(*AlterTableCmd); ok {
		return cmd
	}
	return nil
}

func toAlterCmdSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToAlterCmdSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// CaseWhen helpers
func toCaseWhen(v any) *CaseWhen {
	if node, ok := v.(Node); ok {
		return node.GetCaseWhen()
	}
	if v == nil {
		return nil
	}
	if cw, ok := v.(*CaseWhen); ok {
		return cw
	}
	return nil
}

func toCaseWhenSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToCaseWhenSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

// Int helpers
func toInt(v any) int {
	return int(toInt64(v))
}

func toInt32(v any) int32 {
	return int32(toInt64(v))
}

func toInt64(v any) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case string:
		n, _ := strconv.ParseInt(val, 10, 64)
		return n
	}
	return 0
}

// Bool helpers
func toBool(v any) bool {
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		return strings.EqualFold(s, "cascade") || strings.EqualFold(s, "true") || strings.EqualFold(s, "no") || s == "1"
	}
	return false
}

// OnConflictClause helpers
func toOnConflictClause(v any) *OnConflictClause {
	if node, ok := v.(Node); ok {
		return node.GetOnConflictClause()
	}
	if v == nil {
		return nil
	}
	if oc, ok := v.(*OnConflictClause); ok {
		return oc
	}
	return nil
}

// FuncCall helpers
func toFuncCall(v any) *FuncCall {
	if node, ok := v.(Node); ok {
		return node.GetFuncCall()
	}
	if v == nil {
		return nil
	}
	if fc, ok := v.(*FuncCall); ok {
		return fc
	}
	return &FuncCall{Funcformat: pg_query.CoercionForm(1)}
}

// Jointype helpers
func toJoinType(v any) JoinType {
	if v == nil {
		return JoinInner
	}
	if jt, ok := v.(JoinType); ok {
		return jt
	}
	s := strings.ToUpper(toStr(v))
	switch {
	case strings.Contains(s, "LEFT"):
		return JoinLeft
	case strings.Contains(s, "RIGHT"):
		return JoinRight
	case strings.Contains(s, "FULL"):
		return JoinFull
	case strings.Contains(s, "CROSS"):
		return JoinCross
	default:
		return JoinInner
	}
}

// Insert helpers
type InsertRest struct {
	Cols   []Node
	Select *SelectStmt
}

func toInsertStmt(target, rest, onConflict, returning any) *InsertStmt {
	ins := &InsertStmt{Relation: toRangeVar(target), ReturningClause: grammarReturning(returning), Override: pg_query.OverridingKind(1)}
	if oc, ok := onConflict.(*OnConflictClause); ok {
		ins.OnConflictClause = oc
	}
	if r, ok := rest.(*InsertRest); ok {
		ins.Cols = r.Cols
		ins.SelectStmt = toNode(r.Select)
	} else if sel, ok := rest.(*SelectStmt); ok {
		ins.SelectStmt = toNode(sel)
	}
	return ins
}

func grammarReturning(value any) *ReturningClause {
	clause, _ := value.(*ReturningClause)
	return clause
}

type IndexElemOptions struct {
	Ordering       SortByDir
	NullsOrdering  SortByNulls
	Opclass        []string
	Collation      []string
	OpclassOptions []Node
}

func toIndexElem(name any, expr any, opts any) *IndexElem {
	elem := &IndexElem{Ordering: pg_query.SortByDir(1), NullsOrdering: pg_query.SortByNulls(1)}
	if name != nil {
		elem.Name = toStr(name)
	}
	if expr != nil {
		elem.Expr = toNode(expr)
	}
	if opt, ok := opts.(*IndexElemOptions); ok && opt != nil {
		elem.Ordering = opt.Ordering
		elem.NullsOrdering = opt.NullsOrdering
		elem.Opclass = toNodeSlice(opt.Opclass)
		elem.Collation = toNodeSlice(opt.Collation)
		elem.Opclassopts = opt.OpclassOptions
	}
	return elem
}

func toIndexElemSlice(v any) []Node {
	return toNodeSlice(v)
}

func appendToIndexElemSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

func toStrDefault(v any, def string) string {
	s := toStr(v)
	if s == "" {
		return def
	}
	return s
}

func toVariableSetStmt(v any) *VariableSetStmt {
	if node, ok := v.(Node); ok {
		return node.GetVariableSetStmt()
	}
	if v == nil {
		return nil
	}
	if s, ok := v.(*VariableSetStmt); ok {
		return s
	}
	return nil
}

func singleRangeVarSlice(v any) []Node {
	n := toRangeVar(v)
	if n == nil {
		return nil
	}
	return []Node{toNode(n)}
}

func toDefElem(v any) *DefElem {
	if node, ok := v.(Node); ok {
		return node.GetDefElem()
	}
	if v == nil {
		return nil
	}
	if d, ok := v.(*DefElem); ok {
		return d
	}
	if s, ok := v.(string); ok && s != "" {
		return &DefElem{Defname: s, Defaction: pg_query.DefElemAction(1)}
	}
	return nil
}

func toDefElemSlice(v any) []Node {
	if value, ok := v.(string); ok {
		return singleDefElemSlice(value)
	}
	return toNodeSlice(v)
}

func singleDefElemSlice(v any) []Node {
	n := toDefElem(v)
	if n == nil {
		return nil
	}
	return []Node{toNode(n)}
}

func appendToDefElemSlice(slice, item any) []Node {
	nodes := toDefElemSlice(slice)
	if element := toDefElem(item); element != nil {
		return append(nodes, toNode(element))
	}
	return nodes
}

func makeDefElem(name, arg any) *DefElem {
	n := toStr(name)
	if n == "" {
		return nil
	}
	return &DefElem{Defname: n, Arg: toNode(arg), Defaction: pg_query.DefElemAction(1)}
}

func toVacuumRelation(rel, cols any) *VacuumRelation {
	rv := toRangeVar(rel)
	if rv == nil {
		return nil
	}
	return &VacuumRelation{Relation: rv, VaCols: toNodeSlice(toStrSlice(cols))}
}

func toVacuumRelationSlice(v any) []Node {
	return toNodeSlice(v)
}

func singleVacuumRelationSlice(v any) []Node {
	n := toNode(v)
	if n == nil {
		return nil
	}
	return []Node{toNode(n)}
}

func appendToVacuumRelationSlice(slice, item any) []Node {
	return appendToNodeSlice(slice, item)
}

func makeDropOnObject(obj, onTable any) *List {
	items := toNodeSlice(toStrSlice(onTable))
	return &List{Items: toNodeSlice(append(items, toNode(&String{Sval: toStr(obj)})))}
}

func toIntDefault(v any, def int) int {
	if v == nil {
		return def
	}
	if i, ok := v.(int); ok {
		return i
	}
	if i64, ok := v.(int64); ok {
		return int(i64)
	}
	if s, ok := v.(string); ok {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
}

func makeIntoClause(rel, colNames any) *IntoClause {
	rv := toRangeVar(rel)
	if rv == nil {
		return nil
	}
	return &IntoClause{Rel: rv, ColNames: toNodeSlice(toStrSlice(colNames)), OnCommit: pg_query.OnCommitAction(1)}
}

func toCommentTarget(target any) Node {
	if target == nil {
		return nil
	}
	switch v := target.(type) {
	case []string:
		return toNode(&List{Items: toNodeSlice(v)})
	case []any:
		items := make([]Node, len(v))
		for i, item := range v {
			if n := toNode(item); n != nil {
				items[i] = n
			} else {
				items[i] = toNode(&String{Sval: toStr(item)})
			}
		}
		return toNode(&List{Items: toNodeSlice(items)})
	case string:
		return toNode(&String{Sval: v})
	default:
		return toNode(v)
	}
}

func makeCommentStmt(objType, target, comment any) *CommentStmt {
	return &CommentStmt{Objtype: enumObjectType(toStr(objType)), Object: toNode(toCommentTarget(target)), Comment: toStr(comment)}
}

func makeCreateTableAsStmt(optTemp, target, query, optWithData any, ifNotExists bool, objType string) *CreateTableAsStmt {
	into, _ := target.(*IntoClause)
	if into == nil {
		into = &IntoClause{OnCommit: pg_query.OnCommitAction(1)}
	}
	if into.Rel != nil {
		into.Rel = grammarUtilityRelation(into.Rel, optTemp)
	}
	withData := true
	if b, ok := optWithData.(bool); ok {
		withData = b
	}
	into.SkipData = !withData
	return &CreateTableAsStmt{Into: into, Query: toNode(toStatement(query)), Objtype: enumObjectType(toStr(objType)), IfNotExists: ifNotExists}
}

func makeVacuumStmt(full, freeze, verbose, analyze, rels any, isVacuum bool, locations [4]int) *VacuumStmt {
	var opts []Node
	if toStr(full) != "" {
		opts = append(opts, toNode(&DefElem{Defname: "full", Defaction: pg_query.DefElemAction(1), Location: int32(locations[0])}))
	}
	if toStr(freeze) != "" {
		opts = append(opts, toNode(&DefElem{Defname: "freeze", Defaction: pg_query.DefElemAction(1), Location: int32(locations[1])}))
	}
	if toStr(verbose) != "" {
		opts = append(opts, toNode(&DefElem{Defname: "verbose", Defaction: pg_query.DefElemAction(1), Location: int32(locations[2])}))
	}
	if toStr(analyze) != "" {
		opts = append(opts, toNode(&DefElem{Defname: "analyze", Defaction: pg_query.DefElemAction(1), Location: int32(locations[3])}))
	}
	return &VacuumStmt{Options: toNodeSlice(opts), Rels: toNodeSlice(toVacuumRelationSlice(rels)), IsVacuumcmd: isVacuum}
}

func makeDropRoleStmt(roles any, missingOk bool) *DropRoleStmt {
	return &DropRoleStmt{Roles: toNodeSlice(roles), MissingOk: missingOk}
}

func makeCreateRoleStmt(stmtType string, role any, optList any) *CreateRoleStmt {
	return &CreateRoleStmt{StmtType: enumRoleStmtType(toStr(stmtType)), Role: toStr(role), Options: toNodeSlice(toDefElemSlice(optList))}
}

func makeAlterRoleStmt(role any, optList any, action int) *AlterRoleStmt {
	return &AlterRoleStmt{Role: toRoleSpec(role), Options: toNodeSlice(toDefElemSlice(optList)), Action: int32(action)}
}

func makeGrantStmt(isGrant bool, targType, objType string, objects, privileges, grantees any, grantOption bool, behavior string) *GrantStmt {
	target := enumGrantTargetType(targType)
	object := enumObjectType(objType)
	if t, ok := objects.(*GrantStmt); ok {
		target, object, objects = t.Targtype, t.Objtype, t.Objects
	}
	return &GrantStmt{IsGrant: isGrant, Targtype: enumGrantTargetType(toStr(target)), Objtype: enumObjectType(toStr(object)), Objects: toNodeSlice(objects), Privileges: toNodeSlice(privileges), Grantees: toNodeSlice(toRoleSpecSlice(grantees)), GrantOption: grantOption, Behavior: enumDropBehavior(behavior)}
}

func makeFetchStmt(direction string, howMany any, portal any, isMove bool) *FetchStmt {
	var n int64 = 1
	if howMany != nil {
		if s, ok := howMany.(string); ok && strings.EqualFold(s, "All") {
			n = int64(^uint64(0) >> 1)
		} else if iv, ok := howMany.(int64); ok {
			n = iv
		} else {
			n = int64(toIntDefault(howMany, 1))
		}
	}
	return &FetchStmt{Direction: enumFetchDirection(toStr(direction)), HowMany: n, Portalname: toStr(portal), Ismove: isMove}
}

func makeDeclareCursorStmt(portal any, options int32, query any) *DeclareCursorStmt {
	return &DeclareCursorStmt{Portalname: toStr(portal), Options: int32(options), Query: toNode(toStatement(query))}
}

func makeClosePortalStmt(portal any) *ClosePortalStmt {
	return &ClosePortalStmt{Portalname: toStr(portal)}
}

func makeCallStmt(funcCall any) *CallStmt {
	var fc *FuncCall
	if f, ok := funcCall.(*FuncCall); ok {
		fc = f
	}
	return &CallStmt{Funccall: fc}
}

func makeDiscardStmt(target string) *DiscardStmt {
	return &DiscardStmt{Target: enumDiscardMode(toStr(target))}
}

func makeConstraintsSetStmt(constraints any, deferred bool) *ConstraintsSetStmt {
	return &ConstraintsSetStmt{Constraints: toNodeSlice(toRangeVarSlice(constraints)), Deferred: deferred}
}

func makeCreateSeqStmt(seq any, options any, ifNotExists bool) *CreateSeqStmt {
	return &CreateSeqStmt{Sequence: toRangeVar(seq), Options: toNodeSlice(toDefElemSlice(options)), IfNotExists: ifNotExists}
}

func makeAlterSeqStmt(seq any, options any, missingOk bool) *AlterSeqStmt {
	return &AlterSeqStmt{Sequence: toRangeVar(seq), Options: toNodeSlice(toDefElemSlice(options)), MissingOk: missingOk}
}

func makeClusterStmt(rel any, indexName any, params any) *ClusterStmt {
	return &ClusterStmt{Relation: toRangeVar(rel), Indexname: toStr(indexName), Params: toNodeSlice(toDefElemSlice(params))}
}

func makeReindexStmt(kind string, rel any, name any, params any) *ReindexStmt {
	return &ReindexStmt{Kind: enumReindexObjectType(toStr(kind)), Relation: toRangeVar(rel), Name: toStr(name), Params: toNodeSlice(toDefElemSlice(params))}
}

func makeRefreshMatViewStmt(concurrent bool, skipData bool, rel any) *RefreshMatViewStmt {
	return &RefreshMatViewStmt{Concurrent: concurrent, SkipData: skipData, Relation: toRangeVar(rel)}
}

func makeCopyStmt(rel any, query any, attlist any, isFrom bool, isProgram bool, filename any, options any, where any) *CopyStmt {
	return &CopyStmt{Relation: toRangeVar(rel), Query: toNode(toStatement(query)), Attlist: toNodeSlice(toStrSlice(attlist)), IsFrom: isFrom, IsProgram: isProgram, Filename: toStr(filename), Options: toNodeSlice(toDefElemSlice(options)), WhereClause: toNode(where)}
}

func makeDropOwnedStmt(roles any, behavior any) *DropOwnedStmt {
	return &DropOwnedStmt{Roles: toNodeSlice(roles), Behavior: enumDropBehavior(toStr(behavior))}
}

func makeReassignOwnedStmt(roles any, newRole any) *ReassignOwnedStmt {
	return &ReassignOwnedStmt{Roles: toNodeSlice(roles), Newrole: toRoleSpec(newRole)}
}

func toAnySlice(v any) []any {
	if v == nil {
		return nil
	}
	if sl, ok := v.([]any); ok {
		return sl
	}
	if sl, ok := v.([]Node); ok {
		res := make([]any, len(sl))
		for i, item := range sl {
			res[i] = item
		}
		return res
	}
	if sl, ok := v.([]string); ok {
		res := make([]any, len(sl))
		for i, item := range sl {
			res[i] = item
		}
		return res
	}
	return []any{v}
}

func makeCreateDomainStmt(name, typeName, colQualList any) *CreateDomainStmt {
	stmt := &CreateDomainStmt{Domainname: toNodeSlice(toStrSlice(name)), TypeName: toTypeNameNode(typeName)}
	for _, qualifier := range toNodeSlice(colQualList) {
		if collation := qualifier.GetCollateClause(); collation != nil {
			stmt.CollClause = collation
		} else {
			stmt.Constraints = append(stmt.Constraints, qualifier)
		}
	}
	return stmt
}

func makeAlterDomainStmt(subtype string, name any, def any, idName any, behavior any, missingOk bool) *AlterDomainStmt {
	return &AlterDomainStmt{Subtype: subtype,
		TypeName: toNodeSlice(toStrSlice(name)), Name: toStr(idName), Def: toNode(def), Behavior: enumDropBehavior(toStr(behavior)), MissingOk: missingOk}
}

func makeCreatePolicyStmt(name any, tbl any, cmdName any, permissive bool, roles any, qual any, withCheck any) *CreatePolicyStmt {
	return &CreatePolicyStmt{PolicyName: toStr(name), Table: toRangeVar(tbl), CmdName: toStr(cmdName), Permissive: permissive, Roles: toNodeSlice(roles), Qual: toNode(qual), WithCheck: toNode(withCheck)}
}

func makeCreateStatsStmt(defnames, statTypes, exprs, rels any, ifNotExists bool) *CreateStatsStmt {
	return &CreateStatsStmt{Defnames: toNodeSlice(toStrSlice(defnames)), StatTypes: toNodeSlice(toStrSlice(statTypes)), Exprs: toNodeSlice(exprs), Relations: toNodeSlice(rels), IfNotExists: ifNotExists}
}

func makeCreatePublicationStmt(name any, optList any, pubObjects any, forAllTables bool) *CreatePublicationStmt {
	return &CreatePublicationStmt{Pubname: toStr(name), Options: toNodeSlice(toDefElemSlice(optList)), Pubobjects: toNodeSlice(pubObjects), ForAllTables: forAllTables}
}

func makeAlterPublicationStmt(name any, optList any, pubObjects any, forAllTables bool, action string) *AlterPublicationStmt {
	return &AlterPublicationStmt{Pubname: toStr(name), Options: toNodeSlice(toDefElemSlice(optList)), Pubobjects: toNodeSlice(pubObjects), ForAllTables: forAllTables, Action: enumAlterPublicationAction(toStr(action))}
}

func makeRuleStmt(rel any, name any, where any, event string, instead bool, actions any, replace bool) *RuleStmt {
	return &RuleStmt{Relation: toRangeVar(rel), Rulename: toStr(name), WhereClause: toNode(where), Event: enumCmdType(toStr(event)), Instead: instead, Actions: toNodeSlice(actions), Replace: replace}
}

func makeRenameStmt(renameType, relType string, rel any, obj any, subname, newname any, behavior any, missingOk bool) *RenameStmt {
	var object Node
	switch value := obj.(type) {
	case []string, []Node:
		object = toNode(&List{Items: toNodeSlice(value)})
	default:
		object = toNode(value)
	}
	return &RenameStmt{RenameType: enumObjectType(toStr(renameType)), RelationType: enumObjectType(toStr(relType)), Relation: toRangeVar(rel), Object: toNode(object), Subname: toStr(subname), Newname: toStr(newname), Behavior: enumDropBehavior(toStr(behavior)), MissingOk: missingOk}
}

func makeAlterOwnerStmt(objType string, rel any, obj any, newOwner any) *AlterOwnerStmt {
	return &AlterOwnerStmt{ObjectType: enumObjectType(toStr(objType)), Relation: toRangeVar(rel), Object: toNode(toCommentTarget(obj)), Newowner: toRoleSpec(newOwner)}
}

func toRoleSpec(v any) *RoleSpec {
	if node, ok := v.(Node); ok {
		return node.GetRoleSpec()
	}
	if v == nil {
		return nil
	}
	if rs, ok := v.(*RoleSpec); ok {
		return rs
	}
	if s, ok := v.(string); ok {
		if strings.EqualFold(s, "current_user") {
			return &RoleSpec{Roletype: enumRoleSpecType(toStr("ROLESPEC_CURRENT_USER"))}
		}
		if strings.EqualFold(s, "session_user") {
			return &RoleSpec{Roletype: enumRoleSpecType(toStr("ROLESPEC_SESSION_USER"))}
		}
		if strings.EqualFold(s, "current_role") {
			return &RoleSpec{Roletype: enumRoleSpecType(toStr("ROLESPEC_CURRENT_ROLE"))}
		}
		if strings.EqualFold(s, "public") {
			return &RoleSpec{Roletype: enumRoleSpecType(toStr("ROLESPEC_PUBLIC"))}
		}
		return &RoleSpec{Roletype: enumRoleSpecType(toStr("ROLESPEC_CSTRING")), Rolename: s}
	}
	return nil
}

func makePublicationObjSpec(pubobjtype int, name string, pubtable any) *PublicationObjSpec {
	var pt *PublicationTable
	if p, ok := pubtable.(*PublicationTable); ok {
		pt = p
	}
	return &PublicationObjSpec{Pubobjtype: pg_query.PublicationObjSpecType(pubobjtype), Name: name, Pubtable: pt}
}

func makePublicationTable(rel any, where any, cols any) *PublicationTable {
	return &PublicationTable{Relation: toRangeVar(rel), WhereClause: toNode(where), Columns: toNodeSlice(cols)}
}

func preprocessPublicationObjects(l yyLexer, objects []Node) bool {
	previous := pg_query.PublicationObjSpecType_PUBLICATIONOBJ_CONTINUATION
	for _, node := range objects {
		object := node.GetPublicationObjSpec()
		if object.Pubobjtype == pg_query.PublicationObjSpecType_PUBLICATIONOBJ_CONTINUATION {
			if previous == pg_query.PublicationObjSpecType_PUBLICATIONOBJ_CONTINUATION {
				l.Error("invalid publication object list")
				return false
			}
			object.Pubobjtype = previous
		}
		if object.Pubobjtype == pg_query.PublicationObjSpecType_PUBLICATIONOBJ_TABLE {
			if object.Name == "" && object.Pubtable == nil {
				l.Error("invalid table name")
				return false
			}
			if object.Name != "" {
				object.Pubtable = &PublicationTable{Relation: &RangeVar{Relname: object.Name, Inh: true, Relpersistence: "p", Location: object.Location}}
				object.Name = ""
			}
			previous = object.Pubobjtype
			continue
		}
		if table := object.Pubtable; table != nil {
			if table.WhereClause != nil {
				l.Error("WHERE clause not allowed for schema")
				return false
			}
			if len(table.Columns) != 0 {
				l.Error("column specification not allowed for schema")
				return false
			}
			l.Error("invalid schema name")
			return false
		}
		object.Pubobjtype = pg_query.PublicationObjSpecType_PUBLICATIONOBJ_TABLES_IN_SCHEMA
		if object.Name == "" {
			object.Pubobjtype = pg_query.PublicationObjSpecType_PUBLICATIONOBJ_TABLES_IN_CUR_SCHEMA
		}
		previous = object.Pubobjtype
	}
	return true
}

func makeGrantRoleStmt(roles, grantees any, isGrant bool, options any, behavior any, grantor any) *GrantRoleStmt {
	return &GrantRoleStmt{GrantedRoles: toNodeSlice(roles), GranteeRoles: toNodeSlice(grantees), IsGrant: isGrant,
		Opt: toDefElemSlice(options), Behavior: enumDropBehavior(toStr(behavior)), Grantor: toRoleSpec(grantor)}
}

func makeCompositeTypeStmt(rel any, coldefs any) *CompositeTypeStmt {
	typevar := toRangeVar(rel)
	typevar.Inh = false
	return &CompositeTypeStmt{Typevar: typevar, Coldeflist: toNodeSlice(coldefs)}
}

func makeDropOpStmt(opType string, anyName, usingName any, ifExists bool, behavior any) *DropStmt {
	items := append([]Node{toNode(&String{Sval: toStr(usingName)})}, toNodeSlice(toStrSlice(anyName))...)
	return &DropStmt{RemoveType: enumObjectType(opType), Objects: toNodeSlice([]Node{toNode(&List{Items: toNodeSlice(items)})}), MissingOk: ifExists, Behavior: enumDropBehavior(toStr(behavior))}
}

func makeFuncWithArgs(name any, args any) *ObjectWithArgs {
	owa := &ObjectWithArgs{Objname: toNodeSlice(toStrSlice(name))}
	var parameters []Node
	if aggregate, ok := args.(*aggrArgsInfo); ok {
		parameters = aggregate.args
	} else {
		parameters = toNodeSlice(args)
	}
	owa.Objfuncargs = parameters
	for _, argument := range parameters {
		parameter := argument.GetFunctionParameter()
		if parameter.Mode != pg_query.FunctionParameterMode_FUNC_PARAM_OUT && parameter.Mode != pg_query.FunctionParameterMode_FUNC_PARAM_TABLE {
			owa.Objargs = append(owa.Objargs, toNode(parameter.ArgType))
		}
	}
	return owa
}

func makeFuncWithArgsUnspecified(name []string) *ObjectWithArgs {
	return &ObjectWithArgs{Objname: toNodeSlice(name), ArgsUnspecified: true}
}

func tableFunctionReturnType(columns any) *TypeName {
	parameters := toNodeSlice(columns)
	if len(parameters) == 1 {
		result := proto.Clone(parameters[0].GetFunctionParameter().ArgType).(*TypeName)
		result.Setof = true
		return result
	}
	return &TypeName{Names: toNodeSlice([]string{"pg_catalog", "record"}), Typemod: -1, Setof: true}
}

type triggerEventsInfo struct {
	events  int32
	columns []Node
}

type constrAttr struct {
	deferrable     bool
	initdeferred   bool
	skipValidation bool
	noInherit      bool
	notEnforced    bool
	seen           map[string]bool
}

func toConstrAttr(v any) constrAttr {
	if ca, ok := v.(constrAttr); ok {
		return ca
	}
	if ca, ok := v.(*constrAttr); ok && ca != nil {
		return *ca
	}
	return constrAttr{}
}

type aggrArgsInfo struct {
	args          []Node
	numDirectArgs int32
}

func makeOrderedAggregateArgs(direct, ordered []Node) (*aggrArgsInfo, error) {
	count := len(direct)
	last := direct[count-1].GetFunctionParameter()
	if last.Mode == pg_query.FunctionParameterMode_FUNC_PARAM_VARIADIC {
		first := ordered[0].GetFunctionParameter()
		if len(ordered) != 1 || first.Mode != pg_query.FunctionParameterMode_FUNC_PARAM_VARIADIC || !equalTypeSyntax(last.ArgType.ProtoReflect(), first.ArgType.ProtoReflect()) {
			return nil, fmt.Errorf("an ordered-set aggregate with a VARIADIC direct argument must have one VARIADIC aggregated argument of the same data type")
		}
		ordered = nil
	}
	return &aggrArgsInfo{args: append(direct, ordered...), numDirectArgs: int32(count)}, nil
}

// PostgreSQL's equal() ignores source positions when comparing type syntax.
func equalTypeSyntax(left, right protoreflect.Message) bool {
	if !left.IsValid() || !right.IsValid() {
		return left.IsValid() == right.IsValid()
	}
	if left.Descriptor() != right.Descriptor() {
		return false
	}
	fields := left.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		switch field.Name() {
		case "location", "name_location", "stmt_location", "stmt_len", "list_start", "list_end", "rexpr_list_start", "rexpr_list_end":
			continue
		}
		if left.Has(field) != right.Has(field) {
			return false
		}
		a, b := left.Get(field), right.Get(field)
		if field.IsList() {
			x, y := a.List(), b.List()
			if x.Len() != y.Len() {
				return false
			}
			for j := 0; j < x.Len(); j++ {
				if field.Kind() == protoreflect.MessageKind {
					if !equalTypeSyntax(x.Get(j).Message(), y.Get(j).Message()) {
						return false
					}
				} else if !x.Get(j).Equal(y.Get(j)) {
					return false
				}
			}
		} else if field.Kind() == protoreflect.MessageKind {
			if !equalTypeSyntax(a.Message(), b.Message()) {
				return false
			}
		} else if !a.Equal(b) {
			return false
		}
	}
	return true
}

func parseTypeName(s string) *TypeName {
	var bounds []Node
	for strings.HasSuffix(s, "[]") {
		bounds = append(bounds, toNode(&pg_query.Integer{Ival: -1}))
		s = strings.TrimSuffix(s, "[]")
	}
	parts := strings.Split(s, ".")
	tn := &TypeName{Names: toNodeSlice(parts), Typemod: -1}
	tn.ArrayBounds = bounds
	return tn
}

func makeDefElemOperatorArg(op any) Node {
	switch v := op.(type) {
	case string:
		return toNode(&List{Items: toNodeSlice([]Node{toNode(&String{Sval: v})})})
	case []string:
		var items []Node
		for _, s := range v {
			items = append(items, toNode(&String{Sval: s}))
		}
		return toNode(&List{Items: toNodeSlice(items)})
	case *List:
		return toNode(v)
	case Node:
		return toNode(&List{Items: toNodeSlice([]Node{toNode(v)})})
	default:
		return toNode(&List{Items: toNodeSlice([]Node{toNode(&String{Sval: toStr(op)})})})
	}
}

func makeTriggerFuncArg(v any) *String {
	return &String{Sval: fmt.Sprintf("%d", toInt64(v))}
}

func makeTypeNameNodes(types any) []Node {
	switch values := types.(type) {
	case []Node:
		res := make([]Node, len(values))
		for i, value := range values {
			res[i] = toNode(toTypeNameNode(value))
		}
		return res
	case []any:
		res := make([]Node, len(values))
		for i, value := range values {
			res[i] = toNode(toTypeNameNode(value))
		}
		return res
	default:
		strs := toStrSlice(types)
		res := make([]Node, len(strs))
		for i, s := range strs {
			res[i] = toNode(parseTypeName(s))
		}
		return res
	}
}

func makeTypeNameNode(s string) *TypeName {
	return parseTypeName(s)
}

func toObjectWithArgs(v any) *ObjectWithArgs {
	if node, ok := v.(Node); ok {
		return node.GetObjectWithArgs()
	}
	if o, ok := v.(*ObjectWithArgs); ok {
		return o
	}
	return nil
}

func makeDropSubscriptionStmt(name string, missingOk bool, behavior any) *DropSubscriptionStmt {
	return &DropSubscriptionStmt{Subname: name, MissingOk: missingOk, Behavior: enumDropBehavior(toStr(behavior))}
}

func toCollateClause(val any) *CollateClause {
	if node, ok := val.(Node); ok {
		return node.GetCollateClause()
	}
	if val == nil {
		return nil
	}
	if cc, ok := val.(*CollateClause); ok {
		return cc
	}
	return nil
}

func toAlterTableCmdSlice(v any) []Node {
	return toNodeSlice(v)
}

func toTypeNameNode(val any) *TypeName {
	if node, ok := val.(Node); ok {
		return node.GetTypeName()
	}
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case *TypeName:
		return v
	case string:
		return parseTypeName(v)
	case []string:
		return &TypeName{Names: toNodeSlice(v), Typemod: -1}
	case []any:
		names := make([]string, len(v))
		for i, name := range v {
			names[i] = toStr(name)
		}
		return &TypeName{Names: toNodeSlice(names), Typemod: -1}
	default:
		s := toStr(val)
		if s != "" {
			return parseTypeName(s)
		}
		return nil
	}
}

func toAnyNameNode(an any) Node {
	strs := toStrSlice(an)
	items := make([]Node, len(strs))
	for i, s := range strs {
		items[i] = toNode(&String{Sval: s})
	}
	return toNode(&List{Items: toNodeSlice(items)})
}

func toRoleSpecSlice(v any) []Node {
	return toNodeSlice(v)
}

type importQual struct {
	listType  int32
	tableList []Node
}

func toImportQual(v any) *importQual {
	if q, ok := v.(*importQual); ok && q != nil {
		return q
	}
	return &importQual{}
}

type transformElements struct {
	fromsql *ObjectWithArgs
	tosql   *ObjectWithArgs
}

func toTransformElements(v any) *transformElements {
	if te, ok := v.(*transformElements); ok && te != nil {
		return te
	}
	return &transformElements{}
}
