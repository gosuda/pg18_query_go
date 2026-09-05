package pg_query

import (
	"strings"

	p "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func (w *deparseWriter) utilityStmt(node *p.Node) bool {
	if node == nil {
		return false
	}
	switch n := node.Node.(type) {
	case *p.Node_AlterEventTrigStmt:
		w.alterEventTrig(n.AlterEventTrigStmt)
	case *p.Node_AlterCollationStmt:
		w.text("ALTER COLLATION ")
		w.name(n.AlterCollationStmt.Collname)
		w.text(" REFRESH VERSION")
	case *p.Node_AlterDatabaseStmt:
		w.text("ALTER DATABASE ")
		w.ident(n.AlterDatabaseStmt.Dbname)
		if len(n.AlterDatabaseStmt.Options) == 0 {
			w.text(" REFRESH COLLATION VERSION")
			break
		}
		w.text(" ")
		w.createdbOpts(n.AlterDatabaseStmt.Options)
		w.trimSpace()
	case *p.Node_AlterDatabaseSetStmt:
		w.text("ALTER DATABASE ")
		w.ident(n.AlterDatabaseSetStmt.Dbname)
		w.text(" ")
		w.variableSet(n.AlterDatabaseSetStmt.Setstmt)
	case *p.Node_AlterDatabaseRefreshCollStmt:
		w.text("ALTER DATABASE ")
		w.ident(n.AlterDatabaseRefreshCollStmt.Dbname)
		w.text(" REFRESH COLLATION VERSION")
	case *p.Node_AlterDefaultPrivilegesStmt:
		w.alterDefaultPrivileges(n.AlterDefaultPrivilegesStmt)
	case *p.Node_AlterDomainStmt:
		w.alterDomain(n.AlterDomainStmt)
	case *p.Node_AlterEnumStmt:
		w.alterEnum(n.AlterEnumStmt)
	case *p.Node_AlterExtensionStmt:
		w.text("ALTER EXTENSION ")
		w.ident(n.AlterExtensionStmt.Extname)
		w.text(" UPDATE ")
		for _, node := range n.AlterExtensionStmt.Options {
			d := node.GetDefElem()
			if d == nil || d.Defname != "new_version" {
				w.invalid("invalid ALTER EXTENSION option")
				return true
			}
			w.text("TO ")
			w.nonReservedWordOrSconst(deparseString(d.Arg))
			w.text(" ")
		}
		w.trimSpace()
	case *p.Node_AlterExtensionContentsStmt:
		w.alterExtensionContents(n.AlterExtensionContentsStmt)
	case *p.Node_AlterFdwStmt:
		w.text("ALTER FOREIGN DATA WRAPPER ")
		w.ident(n.AlterFdwStmt.Fdwname)
		w.text(" ")
		if len(n.AlterFdwStmt.FuncOptions) > 0 {
			w.fdwOptions(n.AlterFdwStmt.FuncOptions)
			w.text(" ")
		}
		if len(n.AlterFdwStmt.Options) > 0 {
			w.genericOptions(n.AlterFdwStmt.Options, true)
		}
		w.trimSpace()
	case *p.Node_AlterForeignServerStmt:
		w.text("ALTER SERVER ")
		w.ident(n.AlterForeignServerStmt.Servername)
		w.text(" ")
		if n.AlterForeignServerStmt.HasVersion {
			w.text("VERSION ")
			if n.AlterForeignServerStmt.Version != "" {
				w.literal(n.AlterForeignServerStmt.Version)
			} else {
				w.text("NULL")
			}
			w.text(" ")
		}
		if len(n.AlterForeignServerStmt.Options) > 0 {
			w.genericOptions(n.AlterForeignServerStmt.Options, true)
		}
		w.trimSpace()
	case *p.Node_AlterFunctionStmt:
		w.text("ALTER ")
		switch n.AlterFunctionStmt.Objtype.String() {
		case "OBJECT_FUNCTION":
			w.text("FUNCTION ")
		case "OBJECT_PROCEDURE":
			w.text("PROCEDURE ")
		case "OBJECT_ROUTINE":
			w.text("ROUTINE ")
		default:
			w.invalid("invalid ALTER FUNCTION object type %v", n.AlterFunctionStmt.Objtype)
			return true
		}
		w.objectWithArgs(n.AlterFunctionStmt.Func, false)
		w.text(" ")
		for i, node := range n.AlterFunctionStmt.Actions {
			if i > 0 {
				w.text(" ")
			}
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid ALTER FUNCTION action")
				return true
			}
			w.funcOptItem(d, false)
		}
		w.trimSpace()
	case *p.Node_AlterObjectDependsStmt:
		w.alterObjectDepends(n.AlterObjectDependsStmt)
	case *p.Node_AlterObjectSchemaStmt:
		w.alterObjectSchema(n.AlterObjectSchemaStmt)
	case *p.Node_AlterOwnerStmt:
		w.alterOwner(n.AlterOwnerStmt)
	case *p.Node_AlterOperatorStmt:
		w.text("ALTER OPERATOR ")
		w.objectWithArgs(n.AlterOperatorStmt.Opername, true)
		w.text(" SET (")
		w.operatorDefList(n.AlterOperatorStmt.Options)
		w.text(")")
	case *p.Node_AlterTypeStmt:
		w.text("ALTER TYPE ")
		w.name(n.AlterTypeStmt.TypeName)
		w.text(" SET (")
		w.operatorDefList(n.AlterTypeStmt.Options)
		w.text(")")
	case *p.Node_AlterPolicyStmt:
		w.text("ALTER POLICY ")
		w.ident(n.AlterPolicyStmt.PolicyName)
		w.text(" ON ")
		w.relation(n.AlterPolicyStmt.Table, deparseRelationNormal)
		w.text(" ")
		if len(n.AlterPolicyStmt.Roles) > 0 {
			w.text("TO ")
			w.roles(n.AlterPolicyStmt.Roles)
			w.text(" ")
		}
		if n.AlterPolicyStmt.Qual != nil {
			w.text("USING (")
			w.expr(n.AlterPolicyStmt.Qual)
			w.text(") ")
		}
		if n.AlterPolicyStmt.WithCheck != nil {
			w.text("WITH CHECK (")
			w.expr(n.AlterPolicyStmt.WithCheck)
			w.text(") ")
		}
		w.trimSpace()
	case *p.Node_AlterSeqStmt:
		w.text("ALTER SEQUENCE ")
		if n.AlterSeqStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.relation(n.AlterSeqStmt.Sequence, deparseRelationNormal)
		w.text(" ")
		w.sequenceOptions(n.AlterSeqStmt.Options)
		w.trimSpace()
	case *p.Node_AlterSystemStmt:
		w.text("ALTER SYSTEM ")
		w.variableSet(n.AlterSystemStmt.Setstmt)
	case *p.Node_AlterTableMoveAllStmt:
		w.alterTableMoveAll(n.AlterTableMoveAllStmt)
	case *p.Node_AlterTableStmt:
		w.alterTable(n.AlterTableStmt)
	case *p.Node_AlterTableSpaceOptionsStmt:
		w.text("ALTER TABLESPACE ")
		w.ident(n.AlterTableSpaceOptionsStmt.Tablespacename)
		w.text(" ")
		if n.AlterTableSpaceOptionsStmt.IsReset {
			w.text("RESET ")
		} else {
			w.text("SET ")
		}
		w.relOptions(n.AlterTableSpaceOptionsStmt.Options)
	case *p.Node_AlterPublicationStmt:
		w.alterPublication(n.AlterPublicationStmt)
	case *p.Node_AlterRoleSetStmt:
		w.text("ALTER ROLE ")
		if n.AlterRoleSetStmt.Role == nil {
			w.text("ALL")
		} else {
			w.role(n.AlterRoleSetStmt.Role)
		}
		w.text(" ")
		if n.AlterRoleSetStmt.Database != "" {
			w.text("IN DATABASE ")
			w.ident(n.AlterRoleSetStmt.Database)
			w.text(" ")
		}
		w.variableSet(n.AlterRoleSetStmt.Setstmt)
	case *p.Node_AlterRoleStmt:
		w.alterRole(n.AlterRoleStmt)
	case *p.Node_AlterSubscriptionStmt:
		w.alterSubscription(n.AlterSubscriptionStmt)
	case *p.Node_AlterStatsStmt:
		w.text("ALTER STATISTICS ")
		if n.AlterStatsStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.name(n.AlterStatsStmt.Defnames)
		w.text(" ")
		if target := n.AlterStatsStmt.Stxstattarget; target != nil {
			if i := target.GetInteger(); i != nil {
				w.text("SET STATISTICS ")
				w.integer(int64(i.Ival))
			} else {
				w.text("SET STATISTICS DEFAULT")
			}
		} else {
			w.text("SET STATISTICS DEFAULT")
		}
	case *p.Node_AlterTsconfigurationStmt:
		w.alterTSConfiguration(n.AlterTsconfigurationStmt)
	case *p.Node_AlterTsdictionaryStmt:
		w.text("ALTER TEXT SEARCH DICTIONARY ")
		w.name(n.AlterTsdictionaryStmt.Dictname)
		w.text(" ")
		w.definition(n.AlterTsdictionaryStmt.Options)
	case *p.Node_AlterUserMappingStmt:
		w.text("ALTER USER MAPPING FOR ")
		w.role(n.AlterUserMappingStmt.User)
		w.text(" SERVER ")
		w.ident(n.AlterUserMappingStmt.Servername)
		w.text(" ")
		w.genericOptions(n.AlterUserMappingStmt.Options, true)
		w.trimSpace()
	case *p.Node_CallStmt:
		w.text("CALL ")
		w.funcCall(n.CallStmt.Funccall)
	case *p.Node_CheckPointStmt:
		w.text("CHECKPOINT")
	case *p.Node_ClosePortalStmt:
		w.text("CLOSE ")
		if n.ClosePortalStmt.Portalname != "" {
			w.ident(n.ClosePortalStmt.Portalname)
		} else {
			w.text("ALL")
		}
	case *p.Node_ClusterStmt:
		w.text("CLUSTER ")
		w.utilityOpts(n.ClusterStmt.Params)
		if n.ClusterStmt.Relation != nil {
			w.relation(n.ClusterStmt.Relation, deparseRelationNormal)
			w.text(" ")
		}
		if n.ClusterStmt.Indexname != "" {
			w.text("USING ")
			w.ident(n.ClusterStmt.Indexname)
			w.text(" ")
		}
		w.trimSpace()
	case *p.Node_CommentStmt:
		w.comment(n.CommentStmt)
	case *p.Node_ConstraintsSetStmt:
		w.text("SET CONSTRAINTS ")
		if len(n.ConstraintsSetStmt.Constraints) > 0 {
			w.qualifiedNameList(n.ConstraintsSetStmt.Constraints)
			w.text(" ")
		} else {
			w.text("ALL ")
		}
		if n.ConstraintsSetStmt.Deferred {
			w.text("DEFERRED")
		} else {
			w.text("IMMEDIATE")
		}
	case *p.Node_CopyStmt:
		w.copy(n.CopyStmt)
	case *p.Node_CreateAmStmt:
		w.text("CREATE ACCESS METHOD ")
		w.ident(n.CreateAmStmt.Amname)
		w.text(" TYPE ")
		switch strings.ToLower(n.CreateAmStmt.Amtype) {
		case "i":
			w.text("INDEX ")
		case "t":
			w.text("TABLE ")
		default:
			w.invalid("invalid access method type %q", n.CreateAmStmt.Amtype)
			return true
		}
		w.text("HANDLER ")
		w.name(n.CreateAmStmt.HandlerName)
	case *p.Node_CreateTableAsStmt:
		w.createTableAs(n.CreateTableAsStmt)
	case *p.Node_CreateCastStmt:
		w.createCast(n.CreateCastStmt)
	case *p.Node_CreateConversionStmt:
		w.text("CREATE ")
		if n.CreateConversionStmt.Def {
			w.text("DEFAULT ")
		}
		w.text("CONVERSION ")
		w.name(n.CreateConversionStmt.ConversionName)
		w.text(" FOR ")
		w.literal(n.CreateConversionStmt.ForEncodingName)
		w.text(" TO ")
		w.literal(n.CreateConversionStmt.ToEncodingName)
		w.text("FROM ")
		w.name(n.CreateConversionStmt.FuncName)
	case *p.Node_CreateDomainStmt:
		w.text("CREATE DOMAIN ")
		w.name(n.CreateDomainStmt.Domainname)
		w.text(" AS ")
		w.typeName(n.CreateDomainStmt.TypeName)
		w.text(" ")
		if n.CreateDomainStmt.CollClause != nil {
			w.text("COLLATE ")
			w.name(n.CreateDomainStmt.CollClause.Collname)
			w.text(" ")
		}
		for _, node := range n.CreateDomainStmt.Constraints {
			w.constraint(node.GetConstraint(), false)
			w.text(" ")
		}
		w.trimSpace()
	case *p.Node_CreateExtensionStmt:
		w.text("CREATE EXTENSION ")
		if n.CreateExtensionStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		w.ident(n.CreateExtensionStmt.Extname)
		w.text(" ")
		for _, node := range n.CreateExtensionStmt.Options {
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid CREATE EXTENSION option")
				return true
			}
			switch d.Defname {
			case "schema":
				w.text("SCHEMA ")
				w.ident(deparseString(d.Arg))
			case "new_version":
				w.text("VERSION ")
				w.nonReservedWordOrSconst(deparseString(d.Arg))
			case "cascade":
				w.text("CASCADE")
			default:
				w.invalid("invalid CREATE EXTENSION option %q", d.Defname)
				return true
			}
			w.text(" ")
		}
		w.trimSpace()
	case *p.Node_CreateFdwStmt:
		w.text("CREATE FOREIGN DATA WRAPPER ")
		w.ident(n.CreateFdwStmt.Fdwname)
		w.text(" ")
		if len(n.CreateFdwStmt.FuncOptions) > 0 {
			w.fdwOptions(n.CreateFdwStmt.FuncOptions)
			w.text(" ")
		}
		w.genericOptions(n.CreateFdwStmt.Options, false)
		w.trimSpace()
	case *p.Node_CreateForeignServerStmt:
		w.text("CREATE SERVER ")
		if n.CreateForeignServerStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		w.ident(n.CreateForeignServerStmt.Servername)
		w.text(" ")
		if n.CreateForeignServerStmt.Servertype != "" {
			w.text("TYPE ")
			w.literal(n.CreateForeignServerStmt.Servertype)
			w.text(" ")
		}
		if n.CreateForeignServerStmt.Version != "" {
			w.text("VERSION ")
			w.literal(n.CreateForeignServerStmt.Version)
			w.text(" ")
		}
		w.text("FOREIGN DATA WRAPPER ")
		w.ident(n.CreateForeignServerStmt.Fdwname)
		w.text(" ")
		w.genericOptions(n.CreateForeignServerStmt.Options, false)
		w.trimSpace()
	case *p.Node_CreateForeignTableStmt:
		w.create(n.CreateForeignTableStmt.BaseStmt, true)
		w.text(" SERVER ")
		w.ident(n.CreateForeignTableStmt.Servername)
		w.text(" ")
		if len(n.CreateForeignTableStmt.Options) > 0 {
			w.genericOptions(n.CreateForeignTableStmt.Options, true)
		}
		w.trimSpace()
	case *p.Node_CreateFunctionStmt:
		w.createFunction(n.CreateFunctionStmt)
	case *p.Node_CreateOpClassStmt:
		w.createOpClass(n.CreateOpClassStmt)
	case *p.Node_CreateOpFamilyStmt:
		w.text("CREATE OPERATOR FAMILY ")
		w.name(n.CreateOpFamilyStmt.Opfamilyname)
		w.text(" USING ")
		w.ident(n.CreateOpFamilyStmt.Amname)
	case *p.Node_CreatePublicationStmt:
		w.text("CREATE PUBLICATION ")
		w.ident(n.CreatePublicationStmt.Pubname)
		w.text(" ")
		if len(n.CreatePublicationStmt.Pubobjects) > 0 {
			w.text("FOR ")
			w.publicationObjects(n.CreatePublicationStmt.Pubobjects)
			w.text(" ")
		} else if n.CreatePublicationStmt.ForAllTables {
			w.text("FOR ALL TABLES ")
		}
		if len(n.CreatePublicationStmt.Options) > 0 {
			w.text("WITH ")
			w.definition(n.CreatePublicationStmt.Options)
		}
		w.trimSpace()
	case *p.Node_AlterOpFamilyStmt:
		w.text("ALTER OPERATOR FAMILY ")
		w.name(n.AlterOpFamilyStmt.Opfamilyname)
		w.text(" USING ")
		w.ident(n.AlterOpFamilyStmt.Amname)
		w.text(" ")
		if n.AlterOpFamilyStmt.IsDrop {
			w.text("DROP ")
		} else {
			w.text("ADD ")
		}
		w.opclassItems(n.AlterOpFamilyStmt.Items)
	case *p.Node_CreatePolicyStmt:
		w.text("CREATE POLICY ")
		w.ident(n.CreatePolicyStmt.PolicyName)
		w.text(" ON ")
		w.relation(n.CreatePolicyStmt.Table, deparseRelationNormal)
		w.text(" ")
		if !n.CreatePolicyStmt.Permissive {
			w.text("AS RESTRICTIVE ")
		}
		switch strings.ToLower(n.CreatePolicyStmt.CmdName) {
		case "all":
		case "select":
			w.text("FOR SELECT ")
		case "insert":
			w.text("FOR INSERT ")
		case "update":
			w.text("FOR UPDATE ")
		case "delete":
			w.text("FOR DELETE ")
		default:
			w.invalid("invalid policy command %q", n.CreatePolicyStmt.CmdName)
			return true
		}
		w.text("TO ")
		w.roles(n.CreatePolicyStmt.Roles)
		w.text(" ")
		if n.CreatePolicyStmt.Qual != nil {
			w.text("USING (")
			w.expr(n.CreatePolicyStmt.Qual)
			w.text(") ")
		}
		if n.CreatePolicyStmt.WithCheck != nil {
			w.text("WITH CHECK (")
			w.expr(n.CreatePolicyStmt.WithCheck)
			w.text(") ")
		}
		w.trimSpace()
	case *p.Node_CreatePlangStmt:
		w.text("CREATE ")
		if n.CreatePlangStmt.Replace {
			w.text("OR REPLACE ")
		}
		if n.CreatePlangStmt.Pltrusted {
			w.text("TRUSTED ")
		}
		w.text("LANGUAGE ")
		w.nonReservedWordOrSconst(n.CreatePlangStmt.Plname)
		w.text(" HANDLER ")
		w.name(n.CreatePlangStmt.Plhandler)
		w.text(" ")
		if len(n.CreatePlangStmt.Plinline) > 0 {
			w.text("INLINE ")
			w.name(n.CreatePlangStmt.Plinline)
			w.text(" ")
		}
		if len(n.CreatePlangStmt.Plvalidator) > 0 {
			w.text("VALIDATOR ")
			w.name(n.CreatePlangStmt.Plvalidator)
			w.text(" ")
		}
		w.trimSpace()
	case *p.Node_CreateSchemaStmt:
		w.text("CREATE SCHEMA ")
		if n.CreateSchemaStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		if n.CreateSchemaStmt.Schemaname != "" {
			w.ident(n.CreateSchemaStmt.Schemaname)
			w.text(" ")
		}
		if n.CreateSchemaStmt.Authrole != nil {
			w.text("AUTHORIZATION ")
			w.role(n.CreateSchemaStmt.Authrole)
			w.text(" ")
		}
		for i, node := range n.CreateSchemaStmt.SchemaElts {
			if i > 0 {
				w.text(" ")
			}
			if !w.utilityStmt(node) {
				w.invalid("unpermitted schema element %T", node.GetNode())
				return true
			}
		}
		w.trimSpace()
	case *p.Node_CreateSeqStmt:
		w.text("CREATE ")
		rp := ""
		if n.CreateSeqStmt.Sequence != nil {
			rp = n.CreateSeqStmt.Sequence.Relpersistence
		}
		w.persistence(rp)
		w.text("SEQUENCE ")
		if n.CreateSeqStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		w.relation(n.CreateSeqStmt.Sequence, deparseRelationNormal)
		w.text(" ")
		w.sequenceOptions(n.CreateSeqStmt.Options)
		w.trimSpace()
	case *p.Node_CreateStmt:
		w.create(n.CreateStmt, false)
	case *p.Node_CreateSubscriptionStmt:
		w.text("CREATE SUBSCRIPTION ")
		w.ident(n.CreateSubscriptionStmt.Subname)
		w.text(" CONNECTION ")
		if n.CreateSubscriptionStmt.Conninfo != "" {
			w.literal(n.CreateSubscriptionStmt.Conninfo)
		} else {
			w.text("''")
		}
		w.text(" PUBLICATION ")
		for i, node := range n.CreateSubscriptionStmt.Publication {
			if i > 0 {
				w.text(", ")
			}
			w.ident(deparseString(node))
		}
		w.text(" ")
		if len(n.CreateSubscriptionStmt.Options) > 0 {
			w.text("WITH ")
			w.definition(n.CreateSubscriptionStmt.Options)
		}
		w.trimSpace()
	case *p.Node_CreateStatsStmt:
		w.text("CREATE STATISTICS ")
		if n.CreateStatsStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		w.name(n.CreateStatsStmt.Defnames)
		w.text(" ")
		if len(n.CreateStatsStmt.StatTypes) > 0 {
			w.text("(")
			w.names(n.CreateStatsStmt.StatTypes)
			w.text(") ")
		}
		w.text("ON ")
		for i, node := range n.CreateStatsStmt.Exprs {
			if i > 0 {
				w.text(", ")
			}
			w.statsElem(node.GetStatsElem())
		}
		w.text(" FROM ")
		w.tables(n.CreateStatsStmt.Relations)
	case *p.Node_CreateTableSpaceStmt:
		w.text("CREATE TABLESPACE ")
		w.ident(n.CreateTableSpaceStmt.Tablespacename)
		w.text(" ")
		if n.CreateTableSpaceStmt.Owner != nil {
			w.text("OWNER ")
			w.role(n.CreateTableSpaceStmt.Owner)
			w.text(" ")
		}
		w.text("LOCATION ")
		if n.CreateTableSpaceStmt.Location != "" {
			w.literal(n.CreateTableSpaceStmt.Location)
		} else {
			w.text("''")
		}
		w.text(" ")
		w.withOptions(n.CreateTableSpaceStmt.Options)
		w.trimSpace()
	case *p.Node_CreateTransformStmt:
		w.text("CREATE ")
		if n.CreateTransformStmt.Replace {
			w.text("OR REPLACE ")
		}
		w.text("TRANSFORM FOR ")
		w.typeName(n.CreateTransformStmt.TypeName)
		w.text(" LANGUAGE ")
		w.ident(n.CreateTransformStmt.Lang)
		w.text(" (")
		if n.CreateTransformStmt.Fromsql != nil {
			w.text("FROM SQL WITH FUNCTION ")
			w.objectWithArgs(n.CreateTransformStmt.Fromsql, false)
		}
		if n.CreateTransformStmt.Fromsql != nil && n.CreateTransformStmt.Tosql != nil {
			w.text(", ")
		}
		if n.CreateTransformStmt.Tosql != nil {
			w.text("TO SQL WITH FUNCTION ")
			w.objectWithArgs(n.CreateTransformStmt.Tosql, false)
		}
		w.text(")")
	case *p.Node_CreateTrigStmt:
		w.createTrig(n.CreateTrigStmt)
	case *p.Node_CreateEventTrigStmt:
		w.text("CREATE EVENT TRIGGER ")
		w.ident(n.CreateEventTrigStmt.Trigname)
		w.text(" ON ")
		w.ident(n.CreateEventTrigStmt.Eventname)
		w.text(" ")
		if len(n.CreateEventTrigStmt.Whenclause) > 0 {
			w.text("WHEN ")
			for i, node := range n.CreateEventTrigStmt.Whenclause {
				if i > 0 {
					w.text(" AND ")
				}
				d := node.GetDefElem()
				if d == nil {
					w.invalid("invalid event trigger WHEN clause")
					return true
				}
				w.ident(d.Defname)
				w.text(" IN (")
				for j, item := range d.Arg.GetList().GetItems() {
					if j > 0 {
						w.text(", ")
					}
					w.literal(deparseString(item))
				}
				w.text(")")
			}
			w.text(" ")
		}
		w.text("EXECUTE FUNCTION ")
		w.name(n.CreateEventTrigStmt.Funcname)
		w.text("()")
	case *p.Node_CreateRoleStmt:
		w.createRole(n.CreateRoleStmt)
	case *p.Node_CreateUserMappingStmt:
		w.text("CREATE USER MAPPING ")
		if n.CreateUserMappingStmt.IfNotExists {
			w.text("IF NOT EXISTS ")
		}
		w.text("FOR ")
		w.role(n.CreateUserMappingStmt.User)
		w.text(" SERVER ")
		w.ident(n.CreateUserMappingStmt.Servername)
		w.text(" ")
		w.genericOptions(n.CreateUserMappingStmt.Options, false)
		w.trimSpace()
	case *p.Node_CreatedbStmt:
		w.text("CREATE DATABASE ")
		w.ident(n.CreatedbStmt.Dbname)
		w.text(" ")
		w.createdbOpts(n.CreatedbStmt.Options)
		w.trimSpace()
	case *p.Node_DeallocateStmt:
		w.text("DEALLOCATE ")
		if n.DeallocateStmt.Name != "" {
			w.ident(n.DeallocateStmt.Name)
		} else {
			w.text("ALL")
		}
	case *p.Node_DeclareCursorStmt:
		w.text("DECLARE ")
		w.ident(n.DeclareCursorStmt.Portalname)
		w.text(" ")
		if n.DeclareCursorStmt.Options&1 != 0 {
			w.text("BINARY ")
		}
		if n.DeclareCursorStmt.Options&2 != 0 {
			w.text("SCROLL ")
		}
		if n.DeclareCursorStmt.Options&4 != 0 {
			w.text("NO SCROLL ")
		}
		if n.DeclareCursorStmt.Options&8 != 0 {
			w.text("INSENSITIVE ")
		}
		w.text("CURSOR ")
		if n.DeclareCursorStmt.Options&32 != 0 {
			w.text("WITH HOLD ")
		}
		w.text("FOR ")
		w.selectStmt(n.DeclareCursorStmt.Query.GetSelectStmt(), false)
	case *p.Node_DefineStmt:
		w.define(n.DefineStmt)
	case *p.Node_DiscardStmt:
		w.text("DISCARD ")
		switch n.DiscardStmt.Target.String() {
		case "DISCARD_ALL":
			w.text("ALL")
		case "DISCARD_PLANS":
			w.text("PLANS")
		case "DISCARD_SEQUENCES":
			w.text("SEQUENCES")
		case "DISCARD_TEMP":
			w.text("TEMP")
		default:
			w.invalid("invalid DISCARD target %v", n.DiscardStmt.Target)
			return true
		}
	case *p.Node_DoStmt:
		w.text("DO ")
		for _, node := range n.DoStmt.Args {
			d := node.GetDefElem()
			if d == nil {
				w.invalid("invalid DO option")
				return true
			}
			switch d.Defname {
			case "language":
				w.text("LANGUAGE ")
				w.ident(deparseString(d.Arg))
				w.text(" ")
			case "as":
				body := deparseString(d.Arg)
				delim := "$$"
				if strings.Contains(body, "$$") {
					delim = "$outer$"
				}
				w.text(delim)
				w.text(body)
				w.text(delim)
				w.text(" ")
			default:
				w.invalid("invalid DO option %q", d.Defname)
				return true
			}
		}
		w.trimSpace()
	case *p.Node_DropOwnedStmt:
		w.text("DROP OWNED BY ")
		w.roles(n.DropOwnedStmt.Roles)
		w.text(" ")
		w.dropBehavior(n.DropOwnedStmt.Behavior)
		w.trimSpace()
	case *p.Node_DropStmt:
		w.drop(n.DropStmt)
	case *p.Node_DropSubscriptionStmt:
		w.text("DROP SUBSCRIPTION ")
		if n.DropSubscriptionStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.ident(n.DropSubscriptionStmt.Subname)
	case *p.Node_DropTableSpaceStmt:
		w.text("DROP TABLESPACE ")
		if n.DropTableSpaceStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.ident(n.DropTableSpaceStmt.Tablespacename)
	case *p.Node_DropRoleStmt:
		w.text("DROP ROLE ")
		if n.DropRoleStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.roles(n.DropRoleStmt.Roles)
	case *p.Node_DropUserMappingStmt:
		w.text("DROP USER MAPPING ")
		if n.DropUserMappingStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.text("FOR ")
		w.role(n.DropUserMappingStmt.User)
		w.text(" SERVER ")
		w.ident(n.DropUserMappingStmt.Servername)
	case *p.Node_DropdbStmt:
		w.text("DROP DATABASE ")
		if n.DropdbStmt.MissingOk {
			w.text("IF EXISTS ")
		}
		w.ident(n.DropdbStmt.Dbname)
		w.text(" ")
		if len(n.DropdbStmt.Options) > 0 {
			w.text("(")
			for i, node := range n.DropdbStmt.Options {
				if i > 0 {
					w.text(", ")
				}
				d := node.GetDefElem()
				if d == nil || d.Defname != "force" {
					w.invalid("invalid DROP DATABASE option")
					return true
				}
				w.text("FORCE")
			}
			w.text(")")
		}
		w.trimSpace()
	case *p.Node_ExecuteStmt:
		w.text("EXECUTE ")
		w.ident(n.ExecuteStmt.Name)
		if len(n.ExecuteStmt.Params) > 0 {
			w.text("(")
			w.exprs(n.ExecuteStmt.Params)
			w.text(")")
		}
	case *p.Node_ExplainStmt:
		w.text("EXPLAIN ")
		w.utilityOpts(n.ExplainStmt.Options)
		w.explainable(n.ExplainStmt.Query)
	case *p.Node_FetchStmt:
		w.fetch(n.FetchStmt)
	case *p.Node_GrantStmt:
		w.grant(n.GrantStmt)
	case *p.Node_GrantRoleStmt:
		w.grantRole(n.GrantRoleStmt)
	case *p.Node_ImportForeignSchemaStmt:
		w.text("IMPORT FOREIGN SCHEMA ")
		w.text(n.ImportForeignSchemaStmt.RemoteSchema)
		w.text(" ")
		switch n.ImportForeignSchemaStmt.ListType.String() {
		case "FDW_IMPORT_SCHEMA_LIMIT_TO":
			w.text("LIMIT TO (")
			w.relationExprList(n.ImportForeignSchemaStmt.TableList)
			w.text(") ")
		case "FDW_IMPORT_SCHEMA_EXCEPT":
			w.text("EXCEPT (")
			w.relationExprList(n.ImportForeignSchemaStmt.TableList)
			w.text(") ")
		}
		w.text("FROM SERVER ")
		w.ident(n.ImportForeignSchemaStmt.ServerName)
		w.text(" INTO ")
		w.ident(n.ImportForeignSchemaStmt.LocalSchema)
		w.text(" ")
		w.genericOptions(n.ImportForeignSchemaStmt.Options, false)
		w.trimSpace()
	case *p.Node_IndexStmt:
		w.index(n.IndexStmt)
	case *p.Node_ListenStmt:
		w.text("LISTEN ")
		w.ident(n.ListenStmt.Conditionname)
	case *p.Node_RefreshMatViewStmt:
		w.text("REFRESH MATERIALIZED VIEW ")
		if n.RefreshMatViewStmt.Concurrent {
			w.text("CONCURRENTLY ")
		}
		w.relation(n.RefreshMatViewStmt.Relation, deparseRelationNormal)
		w.text(" ")
		if n.RefreshMatViewStmt.SkipData {
			w.text("WITH NO DATA")
		}
		w.trimSpace()
	case *p.Node_LoadStmt:
		w.text("LOAD ")
		w.literal(n.LoadStmt.Filename)
	case *p.Node_LockStmt:
		w.text("LOCK TABLE ")
		w.relationExprList(n.LockStmt.Relations)
		w.text(" ")
		switch n.LockStmt.Mode {
		case 8:
		case 1:
			w.text("IN ACCESS SHARE MODE ")
		case 2:
			w.text("IN ROW SHARE MODE ")
		case 3:
			w.text("IN ROW EXCLUSIVE MODE ")
		case 4:
			w.text("IN SHARE UPDATE EXCLUSIVE MODE ")
		case 5:
			w.text("IN SHARE MODE ")
		case 6:
			w.text("IN SHARE ROW EXCLUSIVE MODE ")
		case 7:
			w.text("IN EXCLUSIVE MODE ")
		default:
			w.invalid("invalid lock mode %d", n.LockStmt.Mode)
			return true
		}
		if n.LockStmt.Nowait {
			w.text("NOWAIT ")
		}
		w.trimSpace()
	case *p.Node_NotifyStmt:
		w.text("NOTIFY ")
		w.ident(n.NotifyStmt.Conditionname)
		if n.NotifyStmt.Payload != "" {
			w.text(", ")
			w.literal(n.NotifyStmt.Payload)
		}
	case *p.Node_PrepareStmt:
		w.text("PREPARE ")
		w.ident(n.PrepareStmt.Name)
		if len(n.PrepareStmt.Argtypes) > 0 {
			w.text("(")
			w.typeList(n.PrepareStmt.Argtypes)
			w.text(")")
		}
		w.text(" AS ")
		w.preparable(n.PrepareStmt.Query)
	case *p.Node_ReassignOwnedStmt:
		w.text("REASSIGN OWNED BY ")
		w.roles(n.ReassignOwnedStmt.Roles)
		w.text(" TO ")
		w.role(n.ReassignOwnedStmt.Newrole)
	case *p.Node_ReindexStmt:
		w.text("REINDEX ")
		w.utilityOpts(n.ReindexStmt.Params)
		switch n.ReindexStmt.Kind.String() {
		case "REINDEX_OBJECT_INDEX":
			w.text("INDEX ")
		case "REINDEX_OBJECT_TABLE":
			w.text("TABLE ")
		case "REINDEX_OBJECT_SCHEMA":
			w.text("SCHEMA ")
		case "REINDEX_OBJECT_SYSTEM":
			w.text("SYSTEM ")
		case "REINDEX_OBJECT_DATABASE":
			w.text("DATABASE ")
		default:
			w.invalid("invalid REINDEX object type %v", n.ReindexStmt.Kind)
			return true
		}
		if n.ReindexStmt.Relation != nil {
			w.relation(n.ReindexStmt.Relation, deparseRelationNormal)
		} else if n.ReindexStmt.Name != "" {
			w.ident(n.ReindexStmt.Name)
		}
		w.trimSpace()
	case *p.Node_RenameStmt:
		w.rename(n.RenameStmt)
	case *p.Node_RuleStmt:
		w.rule(n.RuleStmt)
	case *p.Node_SecLabelStmt:
		w.secLabel(n.SecLabelStmt)
	case *p.Node_TransactionStmt:
		w.transaction(n.TransactionStmt)
	case *p.Node_TruncateStmt:
		w.text("TRUNCATE ")
		w.relationExprList(n.TruncateStmt.Relations)
		w.text(" ")
		if n.TruncateStmt.RestartSeqs {
			w.text("RESTART IDENTITY ")
		}
		w.dropBehavior(n.TruncateStmt.Behavior)
		w.trimSpace()
	case *p.Node_UnlistenStmt:
		w.text("UNLISTEN ")
		if n.UnlistenStmt.Conditionname == "" {
			w.text("*")
		} else {
			w.ident(n.UnlistenStmt.Conditionname)
		}
	case *p.Node_VacuumStmt:
		w.vacuum(n.VacuumStmt)
	case *p.Node_VariableSetStmt:
		w.variableSet(n.VariableSetStmt)
	case *p.Node_VariableShowStmt:
		w.text("SHOW ")
		switch n.VariableShowStmt.Name {
		case "timezone":
			w.text("TIME ZONE")
		case "transaction_isolation":
			w.text("TRANSACTION ISOLATION LEVEL")
		case "session_authorization":
			w.text("SESSION AUTHORIZATION")
		case "all":
			w.text("ALL")
		default:
			w.ident(n.VariableShowStmt.Name)
		}
	case *p.Node_ViewStmt:
		w.text("CREATE ")
		if n.ViewStmt.Replace {
			w.text("OR REPLACE ")
		}
		vrp := ""
		if n.ViewStmt.View != nil {
			vrp = n.ViewStmt.View.Relpersistence
		}
		w.persistence(vrp)
		w.text("VIEW ")
		w.relation(n.ViewStmt.View, deparseRelationNormal)
		w.text(" ")
		if len(n.ViewStmt.Aliases) > 0 {
			w.text("(")
			w.columnList(n.ViewStmt.Aliases)
			w.text(") ")
		}
		w.withOptions(n.ViewStmt.Options)
		if len(n.ViewStmt.Options) > 0 {
			w.text(" ")
		}
		w.text("AS ")
		w.selectStmt(n.ViewStmt.Query.GetSelectStmt(), false)
		switch n.ViewStmt.WithCheckOption.String() {
		case "NO_CHECK_OPTION":
		case "LOCAL_CHECK_OPTION":
			w.text(" WITH LOCAL CHECK OPTION")
		case "CASCADED_CHECK_OPTION":
			w.text(" WITH CHECK OPTION")
		default:
			w.invalid("invalid check option %v", n.ViewStmt.WithCheckOption)
			return true
		}
		w.text(" ")
	case *p.Node_CompositeTypeStmt:
		w.text("CREATE TYPE ")
		w.relation(n.CompositeTypeStmt.Typevar, deparseRelationCreate)
		w.text(" AS (")
		w.columnDefs(n.CompositeTypeStmt.Coldeflist)
		w.text(")")
	case *p.Node_CreateEnumStmt:
		w.text("CREATE TYPE ")
		w.name(n.CreateEnumStmt.TypeName)
		w.text(" AS ENUM (")
		for i, node := range n.CreateEnumStmt.Vals {
			if i > 0 {
				w.text(", ")
			}
			w.literal(deparseString(node))
		}
		w.text(")")
	case *p.Node_CreateRangeStmt:
		w.text("CREATE TYPE ")
		w.name(n.CreateRangeStmt.TypeName)
		w.text(" AS RANGE ")
		w.definition(n.CreateRangeStmt.Params)
	default:
		return false
	}
	return true
}
