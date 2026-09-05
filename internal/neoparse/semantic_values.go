package neoparse

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

type FloatVal struct{ Str string }
type BitStringVal string
type IntegerVal = pg_query.Integer
type ConstraintType = pg_query.ConstrType

const (
	JoinInner                = pg_query.JoinType_JOIN_INNER
	JoinLeft                 = pg_query.JoinType_JOIN_LEFT
	JoinFull                 = pg_query.JoinType_JOIN_FULL
	JoinRight                = pg_query.JoinType_JOIN_RIGHT
	JoinCross                = pg_query.JoinType_JOIN_INNER
	SortByDirDefault         = pg_query.SortByDir_SORTBY_DEFAULT
	SortByAsc                = pg_query.SortByDir_SORTBY_ASC
	SortByDesc               = pg_query.SortByDir_SORTBY_DESC
	SortByNullsDefault       = pg_query.SortByNulls_SORTBY_NULLS_DEFAULT
	SortByNullsFirst         = pg_query.SortByNulls_SORTBY_NULLS_FIRST
	SortByNullsLast          = pg_query.SortByNulls_SORTBY_NULLS_LAST
	ConstrNone               = pg_query.ConstrType_CONSTR_TYPE_UNDEFINED
	ConstrNotNull            = pg_query.ConstrType_CONSTR_NOTNULL
	ConstrNull               = pg_query.ConstrType_CONSTR_NULL
	ConstrDefault            = pg_query.ConstrType_CONSTR_DEFAULT
	ConstrPrimaryKey         = pg_query.ConstrType_CONSTR_PRIMARY
	ConstrUnique             = pg_query.ConstrType_CONSTR_UNIQUE
	ConstrCheck              = pg_query.ConstrType_CONSTR_CHECK
	ConstrForeignKey         = pg_query.ConstrType_CONSTR_FOREIGN
	ConstrIdentity           = pg_query.ConstrType_CONSTR_IDENTITY
	ConstrGenerated          = pg_query.ConstrType_CONSTR_GENERATED
	ConstrAttrDeferrable     = pg_query.ConstrType_CONSTR_ATTR_DEFERRABLE
	ConstrAttrNotDeferrable  = pg_query.ConstrType_CONSTR_ATTR_NOT_DEFERRABLE
	ConstrAttrDeferred       = pg_query.ConstrType_CONSTR_ATTR_DEFERRED
	ConstrAttrImmediate      = pg_query.ConstrType_CONSTR_ATTR_IMMEDIATE
	ConstrExclusion          = pg_query.ConstrType_CONSTR_EXCLUSION
	ConstrAttrEnforced       = pg_query.ConstrType_CONSTR_ATTR_ENFORCED
	ConstrAttrNotEnforced    = pg_query.ConstrType_CONSTR_ATTR_NOT_ENFORCED
	AExprOp                  = pg_query.A_Expr_Kind_AEXPR_OP
	AExprLike                = pg_query.A_Expr_Kind_AEXPR_LIKE
	AExprILike               = pg_query.A_Expr_Kind_AEXPR_ILIKE
	AExprIn                  = pg_query.A_Expr_Kind_AEXPR_IN
	AExprBetween             = pg_query.A_Expr_Kind_AEXPR_BETWEEN
	AExprNotBetween          = pg_query.A_Expr_Kind_AEXPR_NOT_BETWEEN
	AExprDistinct            = pg_query.A_Expr_Kind_AEXPR_DISTINCT
	AExprNotDistinct         = pg_query.A_Expr_Kind_AEXPR_NOT_DISTINCT
	AExprBetweenSymmetric    = pg_query.A_Expr_Kind_AEXPR_BETWEEN_SYM
	AExprNotBetweenSymmetric = pg_query.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM
	AExprOpAny               = pg_query.A_Expr_Kind_AEXPR_OP_ANY
	AExprOpAll               = pg_query.A_Expr_Kind_AEXPR_OP_ALL
	AExprNullIf              = pg_query.A_Expr_Kind_AEXPR_NULLIF
	AExprSimilar             = pg_query.A_Expr_Kind_AEXPR_SIMILAR
	SubLinkExists            = pg_query.SubLinkType_EXISTS_SUBLINK
	SubLinkAny               = pg_query.SubLinkType_ANY_SUBLINK
	SubLinkAll               = pg_query.SubLinkType_ALL_SUBLINK
	SubLinkExpr              = pg_query.SubLinkType_EXPR_SUBLINK
	SubLinkArray             = pg_query.SubLinkType_ARRAY_SUBLINK
	BoolTestIsTrue           = pg_query.BoolTestType_IS_TRUE
	BoolTestIsNotTrue        = pg_query.BoolTestType_IS_NOT_TRUE
	BoolTestIsFalse          = pg_query.BoolTestType_IS_FALSE
	BoolTestIsNotFalse       = pg_query.BoolTestType_IS_NOT_FALSE
	BoolTestIsUnknown        = pg_query.BoolTestType_IS_UNKNOWN
	BoolTestIsNotUnknown     = pg_query.BoolTestType_IS_NOT_UNKNOWN
)

func makeAConst(value any) *A_Const {
	result := &A_Const{Location: -1}
	switch value := value.(type) {
	case nil:
		result.Isnull = true
	case string:
		result.Val = &pg_query.A_Const_Sval{Sval: &String{Sval: value}}
	case bool:
		result.Val = &pg_query.A_Const_Boolval{Boolval: &Boolean{Boolval: value}}
	case int:
		return makeAConst(int64(value))
	case int32:
		result.Val = &pg_query.A_Const_Ival{Ival: &Integer{Ival: value}}
	case int64:
		if value >= math.MinInt32 && value <= math.MaxInt32 {
			result.Val = &pg_query.A_Const_Ival{Ival: &Integer{Ival: int32(value)}}
		} else {
			result.Val = &pg_query.A_Const_Fval{Fval: &Float{Fval: strconv.FormatInt(value, 10)}}
		}
	case float64:
		result.Val = &pg_query.A_Const_Fval{Fval: &Float{Fval: strconv.FormatFloat(value, 'g', -1, 64)}}
	case FloatVal:
		result.Val = &pg_query.A_Const_Fval{Fval: &Float{Fval: value.Str}}
	case *FloatVal:
		if value == nil {
			result.Isnull = true
		} else {
			result.Val = &pg_query.A_Const_Fval{Fval: &Float{Fval: value.Str}}
		}
	case BitStringVal:
		result.Val = &pg_query.A_Const_Bsval{Bsval: &BitString{Bsval: string(value)}}
	case *Integer:
		result.Val = &pg_query.A_Const_Ival{Ival: value}
	case *Float:
		result.Val = &pg_query.A_Const_Fval{Fval: value}
	case *String:
		result.Val = &pg_query.A_Const_Sval{Sval: value}
	case *Boolean:
		result.Val = &pg_query.A_Const_Boolval{Boolval: value}
	case *BitString:
		result.Val = &pg_query.A_Const_Bsval{Bsval: value}
	default:
		panic(fmt.Sprintf("invalid constant semantic value %T", value))
	}
	return result
}

func toNode(value any) Node {
	if value == nil {
		return nil
	}
	if node, ok := asNode(value); ok {
		return node
	}
	switch value := value.(type) {
	case string, bool, int, int32, int64, float64, BitStringVal:
		return wrapNode(makeAConst(value))
	case FloatVal:
		return wrapNode(&Float{Fval: value.Str})
	case *FloatVal:
		if value == nil {
			return nil
		}
		return wrapNode(&Float{Fval: value.Str})
	}
	panic(fmt.Sprintf("invalid node semantic value %T", value))
}

func toNodeSlice(value any) []Node {
	if nodes, ok := wrapNodeSlice(value); ok {
		return nodes
	}
	switch value := value.(type) {
	case []string:
		return toPgStringNodes(value)
	case []any:
		nodes := make([]Node, len(value))
		for i, item := range value {
			nodes[i] = toNode(item)
		}
		return nodes
	default:
		return []Node{toNode(value)}
	}
}

func toPgStringNodes(values []string) []Node {
	if values == nil {
		return nil
	}
	result := make([]Node, len(values))
	for i, value := range values {
		result[i] = wrapNode(&String{Sval: value})
	}
	return result
}

var enumSourceAliases = map[string][]string{
	"SET":                       {"VAR_SET_VALUE", "AP_SetObjects"},
	"VALUE":                     {"VAR_SET_VALUE"},
	"CURRENT":                   {"VAR_SET_CURRENT"},
	"MULTI":                     {"VAR_SET_MULTI"},
	"DEFAULT":                   {"VAR_SET_DEFAULT"},
	"RESET":                     {"VAR_RESET"},
	"RESET_ALL":                 {"VAR_RESET_ALL"},
	"ADD":                       {"AP_AddObjects"},
	"DROP":                      {"AP_DropObjects"},
	"USER":                      {"OBJECT_ROLE", "OVERRIDING_USER_VALUE"},
	"GROUP":                     {"OBJECT_ROLE"},
	"SYSTEM":                    {"OVERRIDING_SYSTEM_VALUE"},
	"SCHEMA":                    {"ACL_TARGET_ALL_IN_SCHEMA"},
	"TABLE":                     {"OBJECT_TABLE"},
	"LOCAL":                     {"LOCAL_CHECK_OPTION"},
	"CASCADED":                  {"CASCADED_CHECK_OPTION"},
	"NONE":                      {"NO_CHECK_OPTION", "OVERRIDING_NOT_SET"},
	"MATERIALIZED_VIEW":         {"OBJECT_MATVIEW"},
	"STATISTICS":                {"OBJECT_STATISTIC_EXT"},
	"FOREIGN_DATA_WRAPPER":      {"OBJECT_FDW"},
	"SERVER":                    {"OBJECT_FOREIGN_SERVER"},
	"TEXT_SEARCH_PARSER":        {"OBJECT_TSPARSER"},
	"TEXT_SEARCH_DICTIONARY":    {"OBJECT_TSDICTIONARY"},
	"TEXT_SEARCH_TEMPLATE":      {"OBJECT_TSTEMPLATE"},
	"TEXT_SEARCH_CONFIGURATION": {"OBJECT_TSCONFIGURATION"},
	"DOM_CONSTRAINT":            {"OBJECT_DOMCONSTRAINT"},
	"CONSTRAINT":                {"OBJECT_TABCONSTRAINT"},
	"OPERATOR_CLASS":            {"OBJECT_OPCLASS"},
	"OPERATOR_FAMILY":           {"OBJECT_OPFAMILY"},
	"LARGE_OBJECT":              {"OBJECT_LARGEOBJECT"},
	"AND":                       {"AND_EXPR"}, "OR": {"OR_EXPR"}, "NOT": {"NOT_EXPR"},
	"R": {"PARTITION_STRATEGY_RANGE"}, "L": {"PARTITION_STRATEGY_LIST"}, "H": {"PARTITION_STRATEGY_HASH"},
}

var enumDomainPrefixes = map[string][]string{
	"ObjectType":            {"OBJECT_"},
	"DropBehavior":          {"DROP_"},
	"OnCommitAction":        {"ONCOMMIT_"},
	"PartitionStrategy":     {"PARTITION_STRATEGY_"},
	"DefElemAction":         {"DEFELEM_"},
	"RoleSpecType":          {"ROLESPEC_"},
	"RoleStmtType":          {"ROLESTMT_"},
	"GrantTargetType":       {"ACL_TARGET_"},
	"FunctionParameterMode": {"FUNC_PARAM_"},
	"TransactionStmtKind":   {"TRANS_STMT_"},
	"OverridingKind":        {"OVERRIDING_"},
	"OnConflictAction":      {"ONCONFLICT_"},
	"ReturningOptionKind":   {"RETURNING_OPTION_"},
	"MergeMatchKind":        {"MERGE_WHEN_"},
	"CmdType":               {"CMD_"},
	"ReindexObjectType":     {"REINDEX_OBJECT_"},
	"FetchDirection":        {"FETCH_"},
	"DiscardMode":           {"DISCARD_"},
	"AlterSubscriptionType": {"ALTER_SUBSCRIPTION_"},
	"JoinType":              {"JOIN_"},
	"SetOperation":          {"SETOP_"},
	"ConstrType":            {"CONSTR_"},
}

var enumEmptyCanonical = map[string]string{
	"ObjectType":             "OBJECT_ACCESS_METHOD",
	"AlterPublicationAction": "AP_AddObjects",
	"ViewCheckOption":        "NO_CHECK_OPTION",
	"DropBehavior":           "DROP_RESTRICT",
	"FunctionParameterMode":  "FUNC_PARAM_DEFAULT",
	"GrantTargetType":        "ACL_TARGET_OBJECT",
}

func resolveEnum(domain, value string, values map[string]int32) int32 {
	if id, ok := values[value]; ok {
		return id
	}
	if value == "" {
		if canon, ok := enumEmptyCanonical[domain]; ok {
			if id, ok := values[canon]; ok {
				return id
			}
		}
		return 0
	}
	normalized := strings.ToUpper(strings.ReplaceAll(value, " ", "_"))
	if id, ok := values[normalized]; ok {
		return id
	}
	for _, alias := range enumSourceAliases[normalized] {
		if id, ok := values[alias]; ok {
			return id
		}
	}
	for _, prefix := range enumDomainPrefixes[domain] {
		if id, ok := values[prefix+normalized]; ok {
			return id
		}
	}
	panic("invalid enum semantic value: " + value)
}

func toPgObjectType(value string) ObjectType   { return enumObjectType(value) }
func toPgOverride(value string) OverridingKind { return enumOverridingKind(value) }

func toPgDropBehavior(cascade bool) DropBehavior {
	if cascade {
		return pg_query.DropBehavior_DROP_CASCADE
	}
	return pg_query.DropBehavior_DROP_RESTRICT
}

func toPgReferentialAction(action string) string {
	switch strings.ToUpper(action) {
	case "", "NO ACTION", "A":
		return "a"
	case "RESTRICT", "R":
		return "r"
	case "CASCADE", "C":
		return "c"
	case "SET NULL", "N":
		return "n"
	case "SET DEFAULT", "D":
		return "d"
	default:
		panic("invalid referential action: " + action)
	}
}
