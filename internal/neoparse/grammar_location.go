package neoparse

import pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"

func grammarSetLocation(value any, location int) {
	switch n := value.(type) {
	case *pg_query.A_Const:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RangeVar:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.TableFunc:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.Var:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.Param:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.Aggref:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.GroupingFunc:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.WindowFunc:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.MergeSupportFunc:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.FuncExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.NamedArgExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.OpExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.DistinctExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.NullIfExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ScalarArrayOpExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.BoolExpr:
		if n != nil && n.Location < 0 {
			n.Location = int32(location)
		}
	case *pg_query.SubLink:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RelabelType:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CoerceViaIO:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ArrayCoerceExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ConvertRowtypeExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CollateExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CaseExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CaseWhen:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ArrayExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RowExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CoalesceExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.MinMaxExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.SQLValueFunction:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.XmlExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonFormat:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonConstructorExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonIsPredicate:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonBehavior:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.NullTest:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.BooleanTest:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CoerceToDomain:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CoerceToDomainValue:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.SetToDefault:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.TypeName:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ColumnRef:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ParamRef:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.A_Expr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.TypeCast:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CollateClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RoleSpec:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.FuncCall:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.A_ArrayExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ResTarget:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.SortBy:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.WindowDef:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RangeTableFunc:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RangeTableFuncCol:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.RangeTableSample:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ColumnDef:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.DefElem:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.XmlSerialize:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PartitionElem:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PartitionSpec:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PartitionBoundSpec:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PartitionRangeDatum:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.GroupingSet:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.WithClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.InferClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.OnConflictClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CTESearchClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CTECycleClause:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.CommonTableExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.ReturningOption:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonFuncExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonTablePathSpec:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonTable:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonTableColumn:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonParseExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonScalarExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonSerializeExpr:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonObjectConstructor:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonArrayConstructor:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonArrayQueryConstructor:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.JsonAggConstructor:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PLAssignStmt:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.VariableSetStmt:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.Constraint:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.FunctionParameter:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.TransactionStmt:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.DeallocateStmt:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.PublicationObjSpec:
		if n != nil {
			n.Location = int32(location)
		}
	case *pg_query.Node:
		if n == nil {
			return
		}
		switch w := n.Node.(type) {
		case *pg_query.Node_RangeVar:
			if w.RangeVar != nil {
				w.RangeVar.Location = int32(location)
			}
		case *pg_query.Node_TableFunc:
			if w.TableFunc != nil {
				w.TableFunc.Location = int32(location)
			}
		case *pg_query.Node_Var:
			if w.Var != nil {
				w.Var.Location = int32(location)
			}
		case *pg_query.Node_Param:
			if w.Param != nil {
				w.Param.Location = int32(location)
			}
		case *pg_query.Node_Aggref:
			if w.Aggref != nil {
				w.Aggref.Location = int32(location)
			}
		case *pg_query.Node_GroupingFunc:
			if w.GroupingFunc != nil {
				w.GroupingFunc.Location = int32(location)
			}
		case *pg_query.Node_WindowFunc:
			if w.WindowFunc != nil {
				w.WindowFunc.Location = int32(location)
			}
		case *pg_query.Node_MergeSupportFunc:
			if w.MergeSupportFunc != nil {
				w.MergeSupportFunc.Location = int32(location)
			}
		case *pg_query.Node_FuncExpr:
			if w.FuncExpr != nil {
				w.FuncExpr.Location = int32(location)
			}
		case *pg_query.Node_NamedArgExpr:
			if w.NamedArgExpr != nil {
				w.NamedArgExpr.Location = int32(location)
			}
		case *pg_query.Node_OpExpr:
			if w.OpExpr != nil {
				w.OpExpr.Location = int32(location)
			}
		case *pg_query.Node_DistinctExpr:
			if w.DistinctExpr != nil {
				w.DistinctExpr.Location = int32(location)
			}
		case *pg_query.Node_NullIfExpr:
			if w.NullIfExpr != nil {
				w.NullIfExpr.Location = int32(location)
			}
		case *pg_query.Node_ScalarArrayOpExpr:
			if w.ScalarArrayOpExpr != nil {
				w.ScalarArrayOpExpr.Location = int32(location)
			}
		case *pg_query.Node_BoolExpr:
			if w.BoolExpr != nil && w.BoolExpr.Location < 0 {
				w.BoolExpr.Location = int32(location)
			}
		case *pg_query.Node_SubLink:
			if w.SubLink != nil {
				w.SubLink.Location = int32(location)
			}
		case *pg_query.Node_RelabelType:
			if w.RelabelType != nil {
				w.RelabelType.Location = int32(location)
			}
		case *pg_query.Node_CoerceViaIo:
			if w.CoerceViaIo != nil {
				w.CoerceViaIo.Location = int32(location)
			}
		case *pg_query.Node_ArrayCoerceExpr:
			if w.ArrayCoerceExpr != nil {
				w.ArrayCoerceExpr.Location = int32(location)
			}
		case *pg_query.Node_ConvertRowtypeExpr:
			if w.ConvertRowtypeExpr != nil {
				w.ConvertRowtypeExpr.Location = int32(location)
			}
		case *pg_query.Node_CollateExpr:
			if w.CollateExpr != nil {
				w.CollateExpr.Location = int32(location)
			}
		case *pg_query.Node_CaseExpr:
			if w.CaseExpr != nil {
				w.CaseExpr.Location = int32(location)
			}
		case *pg_query.Node_CaseWhen:
			if w.CaseWhen != nil {
				w.CaseWhen.Location = int32(location)
			}
		case *pg_query.Node_ArrayExpr:
			if w.ArrayExpr != nil {
				w.ArrayExpr.Location = int32(location)
			}
		case *pg_query.Node_RowExpr:
			if w.RowExpr != nil {
				w.RowExpr.Location = int32(location)
			}
		case *pg_query.Node_CoalesceExpr:
			if w.CoalesceExpr != nil {
				w.CoalesceExpr.Location = int32(location)
			}
		case *pg_query.Node_MinMaxExpr:
			if w.MinMaxExpr != nil {
				w.MinMaxExpr.Location = int32(location)
			}
		case *pg_query.Node_SqlvalueFunction:
			if w.SqlvalueFunction != nil {
				w.SqlvalueFunction.Location = int32(location)
			}
		case *pg_query.Node_XmlExpr:
			if w.XmlExpr != nil {
				w.XmlExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonFormat:
			if w.JsonFormat != nil {
				w.JsonFormat.Location = int32(location)
			}
		case *pg_query.Node_JsonConstructorExpr:
			if w.JsonConstructorExpr != nil {
				w.JsonConstructorExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonIsPredicate:
			if w.JsonIsPredicate != nil {
				w.JsonIsPredicate.Location = int32(location)
			}
		case *pg_query.Node_JsonBehavior:
			if w.JsonBehavior != nil {
				w.JsonBehavior.Location = int32(location)
			}
		case *pg_query.Node_JsonExpr:
			if w.JsonExpr != nil {
				w.JsonExpr.Location = int32(location)
			}
		case *pg_query.Node_NullTest:
			if w.NullTest != nil {
				w.NullTest.Location = int32(location)
			}
		case *pg_query.Node_BooleanTest:
			if w.BooleanTest != nil {
				w.BooleanTest.Location = int32(location)
			}
		case *pg_query.Node_CoerceToDomain:
			if w.CoerceToDomain != nil {
				w.CoerceToDomain.Location = int32(location)
			}
		case *pg_query.Node_CoerceToDomainValue:
			if w.CoerceToDomainValue != nil {
				w.CoerceToDomainValue.Location = int32(location)
			}
		case *pg_query.Node_SetToDefault:
			if w.SetToDefault != nil {
				w.SetToDefault.Location = int32(location)
			}
		case *pg_query.Node_TypeName:
			if w.TypeName != nil {
				w.TypeName.Location = int32(location)
			}
		case *pg_query.Node_ColumnRef:
			if w.ColumnRef != nil {
				w.ColumnRef.Location = int32(location)
			}
		case *pg_query.Node_ParamRef:
			if w.ParamRef != nil {
				w.ParamRef.Location = int32(location)
			}
		case *pg_query.Node_AExpr:
			if w.AExpr != nil {
				w.AExpr.Location = int32(location)
			}
		case *pg_query.Node_TypeCast:
			if w.TypeCast != nil {
				w.TypeCast.Location = int32(location)
			}
		case *pg_query.Node_CollateClause:
			if w.CollateClause != nil {
				w.CollateClause.Location = int32(location)
			}
		case *pg_query.Node_RoleSpec:
			if w.RoleSpec != nil {
				w.RoleSpec.Location = int32(location)
			}
		case *pg_query.Node_FuncCall:
			if w.FuncCall != nil {
				w.FuncCall.Location = int32(location)
			}
		case *pg_query.Node_AArrayExpr:
			if w.AArrayExpr != nil {
				w.AArrayExpr.Location = int32(location)
			}
		case *pg_query.Node_ResTarget:
			if w.ResTarget != nil {
				w.ResTarget.Location = int32(location)
			}
		case *pg_query.Node_SortBy:
			if w.SortBy != nil {
				w.SortBy.Location = int32(location)
			}
		case *pg_query.Node_WindowDef:
			if w.WindowDef != nil {
				w.WindowDef.Location = int32(location)
			}
		case *pg_query.Node_RangeTableFunc:
			if w.RangeTableFunc != nil {
				w.RangeTableFunc.Location = int32(location)
			}
		case *pg_query.Node_RangeTableFuncCol:
			if w.RangeTableFuncCol != nil {
				w.RangeTableFuncCol.Location = int32(location)
			}
		case *pg_query.Node_RangeTableSample:
			if w.RangeTableSample != nil {
				w.RangeTableSample.Location = int32(location)
			}
		case *pg_query.Node_ColumnDef:
			if w.ColumnDef != nil {
				w.ColumnDef.Location = int32(location)
			}
		case *pg_query.Node_DefElem:
			if w.DefElem != nil {
				w.DefElem.Location = int32(location)
			}
		case *pg_query.Node_XmlSerialize:
			if w.XmlSerialize != nil {
				w.XmlSerialize.Location = int32(location)
			}
		case *pg_query.Node_PartitionElem:
			if w.PartitionElem != nil {
				w.PartitionElem.Location = int32(location)
			}
		case *pg_query.Node_PartitionSpec:
			if w.PartitionSpec != nil {
				w.PartitionSpec.Location = int32(location)
			}
		case *pg_query.Node_PartitionBoundSpec:
			if w.PartitionBoundSpec != nil {
				w.PartitionBoundSpec.Location = int32(location)
			}
		case *pg_query.Node_PartitionRangeDatum:
			if w.PartitionRangeDatum != nil {
				w.PartitionRangeDatum.Location = int32(location)
			}
		case *pg_query.Node_GroupingSet:
			if w.GroupingSet != nil {
				w.GroupingSet.Location = int32(location)
			}
		case *pg_query.Node_WithClause:
			if w.WithClause != nil {
				w.WithClause.Location = int32(location)
			}
		case *pg_query.Node_InferClause:
			if w.InferClause != nil {
				w.InferClause.Location = int32(location)
			}
		case *pg_query.Node_OnConflictClause:
			if w.OnConflictClause != nil {
				w.OnConflictClause.Location = int32(location)
			}
		case *pg_query.Node_CtesearchClause:
			if w.CtesearchClause != nil {
				w.CtesearchClause.Location = int32(location)
			}
		case *pg_query.Node_CtecycleClause:
			if w.CtecycleClause != nil {
				w.CtecycleClause.Location = int32(location)
			}
		case *pg_query.Node_CommonTableExpr:
			if w.CommonTableExpr != nil {
				w.CommonTableExpr.Location = int32(location)
			}
		case *pg_query.Node_ReturningOption:
			if w.ReturningOption != nil {
				w.ReturningOption.Location = int32(location)
			}
		case *pg_query.Node_JsonFuncExpr:
			if w.JsonFuncExpr != nil {
				w.JsonFuncExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonTablePathSpec:
			if w.JsonTablePathSpec != nil {
				w.JsonTablePathSpec.Location = int32(location)
			}
		case *pg_query.Node_JsonTable:
			if w.JsonTable != nil {
				w.JsonTable.Location = int32(location)
			}
		case *pg_query.Node_JsonTableColumn:
			if w.JsonTableColumn != nil {
				w.JsonTableColumn.Location = int32(location)
			}
		case *pg_query.Node_JsonParseExpr:
			if w.JsonParseExpr != nil {
				w.JsonParseExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonScalarExpr:
			if w.JsonScalarExpr != nil {
				w.JsonScalarExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonSerializeExpr:
			if w.JsonSerializeExpr != nil {
				w.JsonSerializeExpr.Location = int32(location)
			}
		case *pg_query.Node_JsonObjectConstructor:
			if w.JsonObjectConstructor != nil {
				w.JsonObjectConstructor.Location = int32(location)
			}
		case *pg_query.Node_JsonArrayConstructor:
			if w.JsonArrayConstructor != nil {
				w.JsonArrayConstructor.Location = int32(location)
			}
		case *pg_query.Node_JsonArrayQueryConstructor:
			if w.JsonArrayQueryConstructor != nil {
				w.JsonArrayQueryConstructor.Location = int32(location)
			}
		case *pg_query.Node_JsonAggConstructor:
			if w.JsonAggConstructor != nil {
				w.JsonAggConstructor.Location = int32(location)
			}
		case *pg_query.Node_PlassignStmt:
			if w.PlassignStmt != nil {
				w.PlassignStmt.Location = int32(location)
			}
		case *pg_query.Node_VariableSetStmt:
			if w.VariableSetStmt != nil {
				w.VariableSetStmt.Location = int32(location)
			}
		case *pg_query.Node_Constraint:
			if w.Constraint != nil {
				w.Constraint.Location = int32(location)
			}
		case *pg_query.Node_FunctionParameter:
			if w.FunctionParameter != nil {
				w.FunctionParameter.Location = int32(location)
			}
		case *pg_query.Node_TransactionStmt:
			if w.TransactionStmt != nil {
				w.TransactionStmt.Location = int32(location)
			}
		case *pg_query.Node_DeallocateStmt:
			if w.DeallocateStmt != nil {
				w.DeallocateStmt.Location = int32(location)
			}
		case *pg_query.Node_PublicationObjSpec:
			if w.PublicationObjSpec != nil {
				w.PublicationObjSpec.Location = int32(location)
			}
		case *pg_query.Node_AConst:
			if w.AConst != nil {
				w.AConst.Location = int32(location)
			}
		}
	}
}
