package pg_query

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"sort"
	"strconv"
	"strings"
)

type normalizeLocation struct{ start, param int }
type normalizeState struct {
	input          string
	utility        bool
	next, external int
	locations      []normalizeLocation
	refs           *[]int
}

func Normalize(input string) (string, error)        { return normalizeSQL(input, false) }
func NormalizeUtility(input string) (string, error) { return normalizeSQL(input, true) }
func normalizeSQL(input string, utility bool) (string, error) {
	tree, err := Parse(input)
	if err != nil {
		return "", err
	}
	state := normalizeState{input: input, utility: utility, next: 1}
	for _, stmt := range tree.Stmts {
		state.walk(stmt.ProtoReflect())
	}
	scan, err := Scan(input)
	if err != nil {
		return "", err
	}
	sort.SliceStable(state.locations, func(i, j int) bool { return state.locations[i].start < state.locations[j].start })
	var out strings.Builder
	last, token := 0, 0
	for _, loc := range state.locations {
		if loc.start < last || loc.start >= len(input) {
			continue
		}
		for token < len(scan.Tokens) && int(scan.Tokens[token].Start) < loc.start {
			token++
		}
		if token == len(scan.Tokens) {
			break
		}
		if input[loc.start] == '-' {
			token++
			if token == len(scan.Tokens) {
				break
			}
		}
		end := int(scan.Tokens[token].End)
		if end < loc.start || end > len(input) {
			continue
		}
		out.WriteString(input[last:loc.start])
		out.WriteByte('$')
		n := loc.param
		if n < 0 {
			n = state.external - n
		}
		out.WriteString(strconv.Itoa(n))
		last = end
	}
	out.WriteString(input[last:])
	return out.String(), nil
}
func (s *normalizeState) record(location int) {
	if location < 0 {
		return
	}
	id := -s.next
	s.next++
	s.locations = append(s.locations, normalizeLocation{location, id})
	if s.refs != nil {
		*s.refs = append(*s.refs, id)
	}
}
func (s *normalizeState) quoted(location int) {
	for i := max(0, location); i < len(s.input); i++ {
		if s.input[i] != '\'' && s.input[i] != '$' {
			continue
		}
		if i-2 >= location && s.input[i-1] == '&' && (s.input[i-2] == 'u' || s.input[i-2] == 'U') {
			i -= 2
		} else if i-1 >= location && strings.ContainsRune("bBxXnNeE", rune(s.input[i-1])) {
			i--
		}
		s.record(i)
		return
	}
}
func (s *normalizeState) walk(m protoreflect.Message) {
	if !m.IsValid() {
		return
	}
	m = utilityUnwrap(m)
	name := string(m.Descriptor().Name())
	fields := utilityWalkFields[name]
	switch n := m.Interface().(type) {
	case *A_Const:
		s.record(int(n.Location))
		return
	case *ParamRef:
		s.external = max(s.external, int(n.Number))
		if s.refs != nil {
			*s.refs = append(*s.refs, int(n.Number))
		}
		return
	case *TypeName:
		return
	case *DefElem:
		if n.Arg.GetString_() != nil || n.Arg.GetList() != nil && len(n.Arg.GetList().Items) == 1 && n.Arg.GetList().Items[0].GetString_() != nil {
			s.quoted(int(n.Location))
		}
		fields = []string{"arg"}
	case *SelectStmt:
		if !s.utility {
			s.selectStmt(n)
		}
		return
	case *NotifyStmt:
		if n.Payload != "" {
			if i := strings.IndexByte(s.input, ','); i >= 0 {
				s.quoted(i + 1)
			}
		}
		return
	case *CreateSubscriptionStmt:
		if n.Conninfo != "" {
			if i := strings.Index(s.input, n.Conninfo); i >= 0 {
				s.record(i - 1)
			}
		}
		return
	case *AlterSubscriptionStmt:
		if n.Conninfo != "" {
			if i := strings.Index(s.input, n.Conninfo); i >= 0 {
				s.record(i - 1)
			}
		}
		return
	}
	switch name {
	case "RawStmt":
		fields = []string{"stmt"}
	case "VariableSetStmt":
		if s.utility {
			return
		}
		fields = []string{"args"}
	case "CopyStmt":
		if s.utility {
			return
		}
		fields = []string{"query"}
	case "CallStmt":
		if s.utility {
			return
		}
		fields = []string{"funccall"}
	case "ExplainStmt":
		fields = []string{"query"}
	case "CreateRoleStmt", "AlterRoleStmt", "CreateUserMappingStmt", "AlterUserMappingStmt":
		fields = []string{"options"}
	case "DeclareCursorStmt":
		if s.utility {
			return
		}
		fields = []string{"query"}
	case "CreateFunctionStmt":
		if s.utility {
			return
		}
		fields = []string{"options"}
	case "DoStmt":
		if s.utility {
			return
		}
		fields = []string{"args"}
	case "InsertStmt", "UpdateStmt", "DeleteStmt", "MergeStmt":
		if s.utility {
			return
		}
	}
	utilityChildren(m, fields, s.walk)
}
func (s *normalizeState) selectStmt(n *SelectStmt) {
	walkNodes := func(nodes []*Node) {
		for _, node := range nodes {
			if node != nil {
				s.walk(node.ProtoReflect())
			}
		}
	}
	utilityChildren(n.ProtoReflect(), []string{"distinctClause", "intoClause"}, s.walk)
	type target struct {
		hash uint64
		refs []int
		used bool
	}
	targets := make([]target, 0, len(n.TargetList))
	for _, node := range n.TargetList {
		var refs []int
		s.refs = &refs
		s.walk(node.ProtoReflect())
		s.refs = nil
		var hash uint64
		if r := node.GetResTarget(); r != nil && r.Val != nil {
			hash = HashXXH3_64(fingerprintMessage(nil, r.Val.ProtoReflect(), "", "", 0, FingerprintDefault), 3)
		}
		targets = append(targets, target{hash: hash, refs: refs})
	}
	utilityChildren(n.ProtoReflect(), []string{"fromClause", "whereClause"}, s.walk)
	for _, node := range n.GroupClause {
		if c := node.GetAConst(); c != nil && c.GetIval() != nil {
			continue
		}
		hash := HashXXH3_64(fingerprintMessage(nil, node.ProtoReflect(), "", "", 0, FingerprintDefault), 3)
		match := -1
		for i := range targets {
			if !targets[i].used && targets[i].hash == hash {
				match = i
				targets[i].used = true
				break
			}
		}
		start := len(s.locations)
		s.walk(node.ProtoReflect())
		if match >= 0 && len(targets[match].refs) == len(s.locations)-start {
			for i, id := range targets[match].refs {
				s.locations[start+i].param = id
			}
			s.next -= len(targets[match].refs)
		}
	}
	for _, node := range n.SortClause {
		if b := node.GetSortBy(); b != nil {
			if c := b.Node.GetAConst(); c != nil && c.GetIval() != nil {
				continue
			}
		}
		walkNodes([]*Node{node})
	}
	utilityChildren(n.ProtoReflect(), []string{"havingClause", "windowClause", "valuesLists", "limitOffset", "limitCount", "lockingClause", "withClause", "larg", "rarg"}, s.walk)
}
