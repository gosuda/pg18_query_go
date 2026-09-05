package pg_query

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"sort"
	"strings"
	"unicode/utf8"
)

type summaryTruncation struct {
	message       protoreflect.Message
	field         string
	depth, length int
}

func summaryDummyColumn() *Node {
	return &Node{Node: &Node_ColumnRef{ColumnRef: &ColumnRef{Fields: []*Node{{Node: &Node_String_{String_: &String{Sval: "…"}}}}}}}
}
func summaryDummyTarget() *Node {
	return &Node{Node: &Node_ResTarget{ResTarget: &ResTarget{Name: "…", Val: summaryDummyColumn()}}}
}
func summarySelect(s *SelectStmt) *Node { return &Node{Node: &Node_SelectStmt{SelectStmt: s}} }
func summaryDeparseNode(n *Node) (string, error) {
	return Deparse(&ParseResult{Stmts: []*RawStmt{{Stmt: n}}})
}
func summaryTruncate(tree *ParseResult, limit int) (string, error) {
	out, err := Deparse(tree)
	if err != nil || len(out) <= limit {
		return out, err
	}
	var candidates []summaryTruncation
	var walk func(protoreflect.Message, int) error
	walk = func(m protoreflect.Message, depth int) error {
		if !m.IsValid() {
			return nil
		}
		m = utilityUnwrap(m)
		name := string(m.Descriptor().Name())
		if name == "RawStmt" {
			f := utilityField(m, "stmt")
			return walk(m.Get(f).Message(), depth)
		}
		fields := []string{}
		switch name {
		case "SelectStmt":
			fields = []string{"targetList", "whereClause", "valuesLists"}
		case "InsertStmt":
			fields = []string{"cols"}
		case "UpdateStmt", "OnConflictClause":
			fields = []string{"targetList", "whereClause"}
		case "DeleteStmt", "CopyStmt", "IndexStmt", "RuleStmt", "InferClause":
			fields = []string{"whereClause"}
		case "CommonTableExpr":
			fields = []string{"ctequery"}
		}
		for _, key := range fields {
			f := utilityField(m, key)
			if f == nil || !m.Has(f) {
				continue
			}
			v := m.Get(f)
			var dummy *Node
			subtract := 0
			switch key {
			case "whereClause":
				dummy = summarySelect(&SelectStmt{WhereClause: v.Message().Interface().(*Node)})
				subtract = 13
			case "ctequery":
				dummy = v.Message().Interface().(*Node)
			default:
				list := v.List()
				nodes := make([]*Node, list.Len())
				for i := range nodes {
					nodes[i] = list.Get(i).Message().Interface().(*Node)
				}
				switch key {
				case "targetList":
					if name == "SelectStmt" {
						dummy = summarySelect(&SelectStmt{TargetList: nodes})
						subtract = 7
					} else {
						dummy = &Node{Node: &Node_UpdateStmt{UpdateStmt: &UpdateStmt{Relation: &RangeVar{Relname: "x", Inh: true, Relpersistence: "p"}, TargetList: nodes}}}
						subtract = 13
					}
				case "valuesLists":
					dummy = summarySelect(&SelectStmt{ValuesLists: nodes})
					subtract = 9
				case "cols":
					dummy = &Node{Node: &Node_InsertStmt{InsertStmt: &InsertStmt{Relation: &RangeVar{Relname: "x", Inh: true, Relpersistence: "p"}, Cols: nodes, Override: OverridingKind_OVERRIDING_NOT_SET}}}
					subtract = 31
				}
			}
			sql, e := summaryDeparseNode(dummy)
			if e != nil {
				return e
			}
			length := len(sql) - subtract
			if length > 3 {
				candidates = append(candidates, summaryTruncation{m, key, depth, length})
			}
		}
		var childErr error
		utilityChildren(m, utilityWalkFields[name], func(child protoreflect.Message) {
			if childErr == nil {
				childErr = walk(child, depth+1)
			}
		})
		return childErr
	}
	for _, stmt := range tree.Stmts {
		if err = walk(stmt.ProtoReflect(), 0); err != nil {
			return "", err
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].depth != candidates[j].depth {
			return candidates[i].depth > candidates[j].depth
		}
		return candidates[i].length > candidates[j].length
	})
	for _, c := range candidates {
		f := utilityField(c.message, c.field)
		if f.IsList() {
			list := c.message.Mutable(f).List()
			list.Truncate(0)
			n := summaryDummyTarget()
			if c.field == "valuesLists" {
				n = &Node{Node: &Node_List{List: &List{Items: []*Node{summaryDummyColumn()}}}}
			}
			list.Append(protoreflect.ValueOfMessage(n.ProtoReflect()))
		} else {
			n := summaryDummyColumn()
			if c.field == "ctequery" {
				n = summarySelect(&SelectStmt{WhereClause: n})
			}
			c.message.Set(f, protoreflect.ValueOfMessage(n.ProtoReflect()))
		}
		out, err = Deparse(tree)
		if err != nil {
			return "", err
		}
		out = strings.ReplaceAll(out, "SELECT \"…\" AS \"…\"", "SELECT \"…\"")
		out = strings.ReplaceAll(out, "SELECT WHERE \"…\"", "\"…\"")
		out = strings.ReplaceAll(out, "\"…\"", "...")
		if len(out) <= limit {
			return out, nil
		}
	}
	if utf8.RuneCountInString(out) > limit {
		runes := []rune(out)
		out = string(runes[:max(0, limit-3)]) + "..."
	}
	return out, nil
}
