package neoparse

import pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"

func grammarBoolExpr(op pg_query.BoolExprType, left, right Node) *pg_query.BoolExpr {
	if op != pg_query.BoolExprType_NOT_EXPR && left != nil {
		if existing := left.GetBoolExpr(); existing != nil && existing.Boolop == op {
			existing.Args = append(existing.Args, right)
			return existing
		}
	}
	args := []Node{right}
	if op != pg_query.BoolExprType_NOT_EXPR {
		args = []Node{left, right}
	}
	return &pg_query.BoolExpr{Boolop: op, Args: args, Location: -1}
}

func grammarNullTestType(isNotNull bool) pg_query.NullTestType {
	if isNotNull {
		return pg_query.NullTestType_IS_NOT_NULL
	}
	return pg_query.NullTestType_IS_NULL
}

func aConstValue(c *A_Const) any {
	if c == nil || c.Isnull {
		return nil
	}
	switch v := c.Val.(type) {
	case *pg_query.A_Const_Ival:
		return int64(v.Ival.Ival)
	case *pg_query.A_Const_Fval:
		return FloatVal{Str: v.Fval.Fval}
	case *pg_query.A_Const_Sval:
		return v.Sval.Sval
	case *pg_query.A_Const_Boolval:
		return v.Boolval.Boolval
	case *pg_query.A_Const_Bsval:
		return BitStringVal(v.Bsval.Bsval)
	}
	return nil
}

func grammarBoundValues(value any) []int32 {
	bounds, _ := value.([]int32)
	return bounds
}

func grammarAlias(value any) *pg_query.Alias {
	if value == nil {
		return nil
	}
	name, columns := toAliasParts(value)
	return &pg_query.Alias{Aliasname: name, Colnames: toPgStringNodes(columns)}
}
