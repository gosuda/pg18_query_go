package pg_query

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"slices"
	"strings"
)

type summaryRange struct {
	node    *RangeVar
	context SummaryResult_Context
}
type summaryState struct {
	result *SummaryResult
	ranges []summaryRange
	filter bool
}

func Summary(input string, truncateLimit int) (*SummaryResult, error) {
	tree, err := Parse(input)
	if err != nil {
		return nil, err
	}
	s := summaryState{result: &SummaryResult{}}
	for _, stmt := range tree.Stmts {
		s.walk(stmt.ProtoReflect())
	}
	for _, rv := range s.ranges {
		n := rv.node
		name := summaryRelationName(n)
		if !slices.Contains(s.result.CteNames, n.Relname) {
			s.result.Tables = append(s.result.Tables, &SummaryResult_Table{Name: name, SchemaName: n.Schemaname, TableName: n.Relname, Context: rv.context})
		}
		if n.Alias != nil {
			if s.result.Aliases == nil {
				s.result.Aliases = make(map[string]string)
			}
			s.result.Aliases[n.Alias.Aliasname] = name
		}
	}
	var statementWalk func(protoreflect.Message) bool
	statementWalk = func(m protoreflect.Message) bool {
		name := string(m.Descriptor().Name())
		if summaryStatementTypes[name] && !slices.Contains(s.result.StatementTypes, name) {
			s.result.StatementTypes = append(s.result.StatementTypes, name)
		}
		if name == "ExplainStmt" || name == "PrepareStmt" || name == "ViewStmt" || name == "CreateTableAsStmt" {
			utilityChildren(m, []string{"query"}, func(child protoreflect.Message) { utilityWalk(child, statementWalk) })
		}
		return true
	}
	for _, stmt := range tree.Stmts {
		utilityWalk(stmt.Stmt.ProtoReflect(), statementWalk)
	}
	if truncateLimit != -1 {
		s.result.TruncatedQuery, err = summaryTruncate(tree, truncateLimit)
		if err != nil {
			return nil, err
		}
	}
	return s.result, nil
}
func summaryQuote(name string) string {
	scan, err := Scan(name)
	if err == nil && len(scan.Tokens) == 1 && int(scan.Tokens[0].Start) == 0 && int(scan.Tokens[0].End) == len(name) && (scan.Tokens[0].KeywordKind == KeywordKind_NO_KEYWORD || scan.Tokens[0].KeywordKind == KeywordKind_UNRESERVED_KEYWORD) {
		safe := name != ""
		for i, c := range name {
			if !(c == '_' || c >= 'a' && c <= 'z' || i > 0 && (c >= '0' && c <= '9' || c == '$')) {
				safe = false
				break
			}
		}
		if safe {
			return name
		}
	}
	return "\"" + strings.ReplaceAll(name, "\"", "\"\"") + "\""
}
func summaryRelationName(n *RangeVar) string {
	if n.Schemaname != "" {
		return summaryQuote(n.Schemaname) + "." + summaryQuote(n.Relname)
	}
	return summaryQuote(n.Relname)
}
func (s *summaryState) addRange(n *Node, c SummaryResult_Context) {
	if n == nil {
		return
	}
	switch {
	case n.GetRangeVar() != nil:
		s.ranges = append(s.ranges, summaryRange{n.GetRangeVar(), c})
	case n.GetJoinExpr() != nil:
		j := n.GetJoinExpr()
		s.addRange(j.Larg, c)
		s.addRange(j.Rarg, c)
	case n.GetRangeTableSample() != nil:
		s.addRange(n.GetRangeTableSample().Relation, c)
	}
}
func (s *summaryState) addRelation(n *RangeVar, c SummaryResult_Context) {
	if n != nil {
		s.ranges = append(s.ranges, summaryRange{n, c})
	}
}
func (s *summaryState) addRanges(nodes []*Node, c SummaryResult_Context) {
	for _, n := range nodes {
		s.addRange(n, c)
	}
}
func summaryNames(nodes []*Node) []string {
	parts := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if v := n.GetString_(); v != nil {
			parts = append(parts, v.Sval)
		}
	}
	return parts
}
func (s *summaryState) addFunction(nodes []*Node, c SummaryResult_Context) {
	parts := summaryNames(nodes)
	if len(parts) == 0 {
		return
	}
	f := &SummaryResult_Function{Name: strings.Join(parts, "."), FunctionName: parts[len(parts)-1], Context: c}
	if len(parts) == 2 || len(parts) == 3 {
		f.SchemaName = parts[len(parts)-2]
	}
	s.result.Functions = append(s.result.Functions, f)
}
func (s *summaryState) walk(m protoreflect.Message) {
	if !m.IsValid() {
		return
	}
	m = utilityUnwrap(m)
	name := string(m.Descriptor().Name())
	fields := utilityWalkFields[name]
	switch n := m.Interface().(type) {
	case *RawStmt:
		fields = []string{"stmt"}
	case *ExplainStmt, *PrepareStmt:
		fields = []string{"query"}
	case *CallStmt:
		fields = []string{"funccall"}
	case *CommonTableExpr:
		s.result.CteNames = append(s.result.CteNames, n.Ctename)
	case *SelectStmt:
		if s.filter {
			return
		}
		if n.Op == SetOperation_SETOP_NONE {
			s.addRanges(n.FromClause, SummaryResult_Select)
		}
		if n.IntoClause != nil {
			s.addRelation(n.IntoClause.Rel, SummaryResult_DDL)
		}
		if n.WhereClause != nil {
			s.filter = true
			s.walk(n.WhereClause.ProtoReflect())
			s.filter = false
		}
	case *InsertStmt:
		s.addRelation(n.Relation, SummaryResult_DML)
	case *UpdateStmt:
		s.addRelation(n.Relation, SummaryResult_DML)
		s.addRanges(n.FromClause, SummaryResult_Select)
	case *MergeStmt:
		s.addRelation(n.Relation, SummaryResult_DML)
	case *DeleteStmt:
		s.addRelation(n.Relation, SummaryResult_DML)
		s.addRanges(n.UsingClause, SummaryResult_Select)
	case *CopyStmt:
		s.addRelation(n.Relation, SummaryResult_Select)
		if n.Query != nil {
			s.walk(n.Query.ProtoReflect())
		}
	case *AlterTableStmt:
		if n.Objtype != ObjectType_OBJECT_INDEX {
			s.addRelation(n.Relation, SummaryResult_DDL)
		}
	case *CreateStmt:
		s.addRelation(n.Relation, SummaryResult_DDL)
	case *CreateTableAsStmt:
		if n.Into != nil {
			s.addRelation(n.Into.Rel, SummaryResult_DDL)
		}
		if n.Query != nil {
			s.walk(n.Query.ProtoReflect())
		}
	case *TruncateStmt:
		s.addRanges(n.Relations, SummaryResult_DDL)
	case *ViewStmt:
		s.addRelation(n.View, SummaryResult_DDL)
		if n.Query != nil {
			s.walk(n.Query.ProtoReflect())
		}
	case *IndexStmt:
		s.addRelation(n.Relation, SummaryResult_DDL)
		for _, p := range n.IndexParams {
			if e := p.GetIndexElem(); e != nil && e.Expr != nil {
				s.walk(e.Expr.ProtoReflect())
			}
		}
		if n.WhereClause != nil {
			s.walk(n.WhereClause.ProtoReflect())
		}
	case *CreateTrigStmt:
		s.addRelation(n.Relation, SummaryResult_DDL)
	case *RuleStmt:
		s.addRelation(n.Relation, SummaryResult_DDL)
	case *RefreshMatViewStmt:
		s.addRelation(n.Relation, SummaryResult_DDL)
	case *VacuumStmt:
		for _, r := range n.Rels {
			if v := r.GetVacuumRelation(); v != nil {
				s.addRelation(v.Relation, SummaryResult_DDL)
			}
		}
	case *LockStmt:
		s.addRanges(n.Relations, SummaryResult_DDL)
	case *CreateFunctionStmt:
		s.addFunction(n.Funcname, SummaryResult_DDL)
	case *FuncCall:
		s.addFunction(n.Funcname, SummaryResult_Call)
	case *GrantStmt:
		if n.Objtype == ObjectType_OBJECT_TABLE {
			s.addRanges(n.Objects, SummaryResult_DDL)
		} else if n.Objtype == ObjectType_OBJECT_FUNCTION || n.Objtype == ObjectType_OBJECT_PROCEDURE || n.Objtype == ObjectType_OBJECT_ROUTINE {
			for _, o := range n.Objects {
				if a := o.GetObjectWithArgs(); a != nil {
					s.addFunction(a.Objname, SummaryResult_DDL)
				}
			}
		}
	case *DropStmt:
		for i, o := range n.Objects {
			if n.RemoveType == ObjectType_OBJECT_FUNCTION && i == 0 {
				if a := o.GetObjectWithArgs(); a != nil {
					s.addFunction(a.Objname, SummaryResult_DDL)
				}
			}
			if n.RemoveType != ObjectType_OBJECT_TABLE && n.RemoveType != ObjectType_OBJECT_RULE && n.RemoveType != ObjectType_OBJECT_TRIGGER {
				continue
			}
			if l := o.GetList(); l != nil {
				parts := summaryNames(l.Items)
				if n.RemoveType != ObjectType_OBJECT_TABLE && len(parts) > 0 {
					parts = parts[:len(parts)-1]
				}
				if len(parts) > 0 {
					rv := &RangeVar{Relname: parts[len(parts)-1]}
					if len(parts) > 1 {
						rv.Schemaname = parts[len(parts)-2]
					}
					s.addRelation(rv, SummaryResult_DDL)
				}
			}
		}
	case *RenameStmt:
		if n.RenameType == ObjectType_OBJECT_TABLE && n.Relation != nil {
			s.addRelation(n.Relation, SummaryResult_DDL)
			s.addRelation(&RangeVar{Schemaname: n.Relation.Schemaname, Relname: n.Newname}, SummaryResult_DDL)
		} else if n.RenameType == ObjectType_OBJECT_FUNCTION || n.RenameType == ObjectType_OBJECT_PROCEDURE || n.RenameType == ObjectType_OBJECT_ROUTINE {
			if a := n.Object.GetObjectWithArgs(); a != nil {
				s.addFunction(a.Objname, SummaryResult_DDL)
				s.result.Functions = append(s.result.Functions, &SummaryResult_Function{Name: n.Newname, FunctionName: n.Newname, Context: SummaryResult_DDL})
			}
		}
	case *ColumnRef:
		if s.filter && len(n.Fields) > 0 && n.Fields[len(n.Fields)-1].GetString_() != nil {
			parts := summaryNames(n.Fields)
			fc := &SummaryResult_FilterColumn{Column: parts[len(parts)-1]}
			if len(parts) == 2 {
				fc.TableName = parts[0]
			} else if len(parts) == 3 {
				fc.SchemaName = parts[0]
				fc.TableName = parts[1]
			}
			s.result.FilterColumns = append(s.result.FilterColumns, fc)
		}
	}
	utilityChildren(m, fields, s.walk)
}
