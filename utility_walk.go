package pg_query

import "google.golang.org/protobuf/reflect/protoreflect"

func utilityChildren(m protoreflect.Message, fields []string, visit func(protoreflect.Message)) {
	for _, name := range fields {
		f := utilityField(m, name)
		if f == nil || f.Kind() != protoreflect.MessageKind {
			continue
		}
		if f.IsList() {
			l := m.Get(f).List()
			for i := range l.Len() {
				visit(l.Get(i).Message())
			}
		} else if m.Has(f) {
			visit(m.Get(f).Message())
		}
	}
}

func utilityWalk(m protoreflect.Message, visit func(protoreflect.Message) bool) {
	if !m.IsValid() {
		return
	}
	m = utilityUnwrap(m)
	if !visit(m) {
		return
	}
	utilityChildren(m, utilityWalkFields[string(m.Descriptor().Name())], func(child protoreflect.Message) { utilityWalk(child, visit) })
}
