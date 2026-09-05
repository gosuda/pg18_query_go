package pg_query

import (
	"errors"
	"fmt"

	"github.com/gosuda/pg18_query_go/internal/neoparse"
	pgproto "github.com/gosuda/pg18_query_go/internal/pgproto"
	"github.com/gosuda/pg18_query_go/internal/plpgsql"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ParsePlPgSqlToJSON parses CREATE FUNCTION, CREATE PROCEDURE, and DO bodies.
func ParsePlPgSqlToJSON(input string) (string, error) {
	tree, err := Parse(input)
	if err != nil {
		return "", err
	}
	var functions []*plpgsql.Function
	visit := func(message protoreflect.Message) bool {
		if err != nil {
			return false
		}
		var function *plpgsql.Function
		switch node := message.Interface().(type) {
		case *pgproto.CreateFunctionStmt:
			function, err = parsePlpgsqlFunction(node)
		case *pgproto.DoStmt:
			body, language, present := plpgsqlOptions(node.Args)
			if language != "plpgsql" {
				function = &plpgsql.Function{}
			} else if !present {
				err = fmt.Errorf("no inline code specified")
			} else {
				function, err = plpgsql.Parse(body, plpgsql.Config{Inline: true, ReturnType: "void", Name: "inline_code_block"}, neoparse.ParseMode)
			}
		default:
			return true
		}
		if err == nil {
			functions = append(functions, function)
		}
		return err == nil
	}
	for _, statement := range tree.Stmts {
		walkPlpgsqlTree(statement.Stmt.ProtoReflect(), visit)
		if err != nil {
			break
		}
	}
	if err != nil {
		var parseError *plpgsql.Error
		if errors.As(err, &parseError) {
			return "", &Error{Message: parseError.Message, Lineno: parseError.Line, Cursorpos: parseError.Cursorpos}
		}
		return "", err
	}
	return plpgsql.ToJSON(functions), nil
}

func plpgsqlOptions(options []*pgproto.Node) (body, language string, present bool) {
	language = "plpgsql"
	for _, option := range options {
		def := option.GetDefElem()
		if def == nil {
			continue
		}
		switch def.Defname {
		case "language":
			language = def.Arg.GetString_().GetSval()
		case "as":
			if list := def.Arg.GetList(); list != nil {
				for _, item := range list.Items {
					body = item.GetString_().GetSval()
					present = true
				}
			} else if value := def.Arg.GetString_(); value != nil {
				body, present = value.Sval, true
			}
		}
	}
	return body, language, present
}
func plpgsqlTypeName(typ *pgproto.TypeName) string {
	if typ == nil || len(typ.Names) == 0 {
		return ""
	}
	name := typ.Names[len(typ.Names)-1].GetString_().GetSval()
	if len(typ.ArrayBounds) > 0 {
		name = "_" + name
	}
	return name
}
func parsePlpgsqlFunction(node *pgproto.CreateFunctionStmt) (*plpgsql.Function, error) {
	body, language, present := plpgsqlOptions(node.Options)
	if language != "plpgsql" {
		return &plpgsql.Function{}, nil
	}
	if !present {
		return nil, fmt.Errorf("no function body specified")
	}
	if node.ReturnType.GetPctType() {
		return nil, fmt.Errorf("function return %%TYPE references require a database catalog")
	}
	config := plpgsql.Config{Procedure: node.IsProcedure, ReturnType: plpgsqlTypeName(node.ReturnType), ReturnsSet: node.ReturnType.GetSetof()}
	if len(node.Funcname) > 0 {
		config.Name = node.Funcname[0].GetString_().GetSval()
	}
	outputs := 0
	outputType := ""
	for _, parameter := range node.Parameters {
		arg := parameter.GetFunctionParameter()
		if arg == nil {
			return nil, fmt.Errorf("invalid function parameter")
		}
		if arg.ArgType.GetPctType() {
			return nil, fmt.Errorf("function parameter %%TYPE references require a database catalog")
		}
		output := arg.Mode == pgproto.FunctionParameterMode_FUNC_PARAM_OUT || arg.Mode == pgproto.FunctionParameterMode_FUNC_PARAM_INOUT || arg.Mode == pgproto.FunctionParameterMode_FUNC_PARAM_TABLE
		input := arg.Mode != pgproto.FunctionParameterMode_FUNC_PARAM_OUT && arg.Mode != pgproto.FunctionParameterMode_FUNC_PARAM_TABLE
		typ := plpgsqlTypeName(arg.ArgType)
		if arg.Mode == pgproto.FunctionParameterMode_FUNC_PARAM_VARIADIC && typ != "anyarray" && typ != "anycompatiblearray" && typ != "any" {
			return nil, fmt.Errorf("VARIADIC parameter must be an array")
		}
		if output {
			outputs++
			outputType = typ
		}
		if arg.Mode == pgproto.FunctionParameterMode_FUNC_PARAM_TABLE {
			config.ReturnsSet = true
		}
		config.Parameters = append(config.Parameters, plpgsql.Parameter{Name: arg.Name, Type: typ, Input: input, Output: output})
	}
	if outputs > 1 || outputs > 0 && node.IsProcedure {
		outputType = "record"
	}
	if config.ReturnType == "" {
		config.ReturnType = outputType
		if config.ReturnType == "" && node.IsProcedure {
			config.ReturnType = "void"
		}
		if config.ReturnType == "" {
			return nil, fmt.Errorf("function result type must be specified")
		}
	} else if outputs > 0 && config.ReturnType != outputType {
		return nil, fmt.Errorf("function result type must be %s because of OUT parameters", outputType)
	}
	return plpgsql.Parse(body, config, neoparse.ParseMode)
}
func walkPlpgsqlTree(m protoreflect.Message, visit func(protoreflect.Message) bool) {
	if m == nil || !m.IsValid() {
		return
	}
	m = utilityUnwrap(m)
	name := string(m.Descriptor().Name())
	if name == "RawStmt" {
		if f := utilityField(m, "stmt"); f != nil && m.Has(f) {
			walkPlpgsqlTree(m.Get(f).Message(), visit)
		}
		return
	}
	visit(m)
	if name == "CreateFunctionStmt" || name == "DoStmt" {
		return
	}
	fields := m.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		if f.IsMap() {
			continue
		}
		if f.Kind() != protoreflect.MessageKind && f.Kind() != protoreflect.GroupKind {
			continue
		}
		if f.IsList() {
			l := m.Get(f).List()
			for j := range l.Len() {
				if mm, ok := l.Get(j).Interface().(protoreflect.Message); ok {
					walkPlpgsqlTree(mm, visit)
				}
			}
		} else if m.Has(f) {
			walkPlpgsqlTree(m.Get(f).Message(), visit)
		}
	}
}
