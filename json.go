package pg_query

import (
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func ParseToJSON(input string) (string, error) {
	tree, err := Parse(input)
	if err != nil {
		return "", err
	}
	return string(appendMessageJSON(nil, tree.ProtoReflect(), "")), nil
}

func appendMessageJSON(out []byte, message protoreflect.Message, parent protoreflect.Name) []byte {
	if !message.IsValid() {
		return append(out, "null"...)
	}
	out = append(out, '{')
	descriptor := message.Descriptor()
	fields := descriptor.Fields()
	first := true
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		force := false
		switch descriptor.Name() {
		case "ParseResult":
			force = field.Name() == "stmts"
		case "A_Const":
			force = field.Name() == "location"
		case "String":
			force = field.Name() == "sval"
		case "Float":
			force = field.Name() == "fval"
		case "BitString":
			force = field.Name() == "bsval"
		case "Boolean":
			force = parent != "A_Const" && field.Name() == "boolval"
		}
		if !message.Has(field) && !force {
			continue
		}
		if !first {
			out = append(out, ',')
		}
		first = false
		out = appendJSONString(out, field.JSONName())
		out = append(out, ':')
		value := message.Get(field)
		if field.IsList() {
			list := value.List()
			out = append(out, '[')
			for j := 0; j < list.Len(); j++ {
				if j != 0 {
					out = append(out, ',')
				}
				out = appendValueJSON(out, field, list.Get(j), descriptor.Name())
			}
			out = append(out, ']')
			continue
		}
		out = appendValueJSON(out, field, value, descriptor.Name())
	}
	return append(out, '}')
}

func appendValueJSON(out []byte, field protoreflect.FieldDescriptor, value protoreflect.Value, parent protoreflect.Name) []byte {
	switch field.Kind() {
	case protoreflect.MessageKind:
		return appendMessageJSON(out, value.Message(), parent)
	case protoreflect.StringKind:
		return appendJSONString(out, value.String())
	case protoreflect.BoolKind:
		return strconv.AppendBool(out, value.Bool())
	case protoreflect.EnumKind:
		enum := field.Enum().Values().ByNumber(value.Enum())
		if enum == nil {
			return strconv.AppendInt(out, int64(value.Enum()), 10)
		}
		return appendJSONString(out, string(enum.Name()))
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.AppendInt(out, value.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.AppendUint(out, value.Uint(), 10)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return strconv.AppendFloat(out, value.Float(), 'f', 6, 64)
	default:
		panic("unsupported PostgreSQL JSON field kind: " + field.Kind().String())
	}
}

func appendJSONString(out []byte, value string) []byte {
	const hex = "0123456789abcdef"
	out = append(out, '"')
	start := 0
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 0x20 && c != '"' && c != '\\' && c != '<' && c != '>' {
			continue
		}
		out = append(out, value[start:i]...)
		switch c {
		case '"', '\\':
			out = append(out, '\\', c)
		case '\b':
			out = append(out, '\\', 'b')
		case '\f':
			out = append(out, '\\', 'f')
		case '\n':
			out = append(out, '\\', 'n')
		case '\r':
			out = append(out, '\\', 'r')
		case '\t':
			out = append(out, '\\', 't')
		// Upstream _outToken escapes angle brackets (see pg_query_json_helper.c).
		case '<':
			out = append(out, '\\', 'u', '0', '0', '3', 'c')
		case '>':
			out = append(out, '\\', 'u', '0', '0', '3', 'e')
		default:
			out = append(out, '\\', 'u', '0', '0', hex[c>>4], hex[c&15])
		}
		start = i + 1
	}
	out = append(out, value[start:]...)
	return append(out, '"')
}
