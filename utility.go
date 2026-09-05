package pg_query

func IsUtilityStmt(input string) ([]bool, error) {
	tree, err := Parse(input)
	if err != nil {
		return nil, err
	}
	result := make([]bool, len(tree.Stmts))
	for i, stmt := range tree.Stmts {
		result[i] = utilityStatement(stmt.Stmt)
	}
	return result, nil
}

func utilityStatement(node *Node) bool {
	return node.GetSelectStmt() == nil && node.GetInsertStmt() == nil && node.GetUpdateStmt() == nil && node.GetDeleteStmt() == nil && node.GetMergeStmt() == nil
}
