%{
package neoparse

import (
 "strings"
 "fmt"
 pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

func grammarType(name any, mods any) *TypeName {
	n := toTypeNameNode(name)
	n.Typmods = toNodeSlice(mods)
	return n
}

func grammarSystemType(name string, mods any) *TypeName {
	return &TypeName{Names: toPgStringNodes([]string{"pg_catalog", name}), Typemod: -1, Typmods: toNodeSlice(mods), Location: -1}
}

func grammarArrayType(name any, bounds any, setof bool) *TypeName {
	n := toTypeNameNode(name)
	for _, bound := range grammarBoundValues(bounds) {
		n.ArrayBounds = append(n.ArrayBounds, toNode(&IntegerVal{Ival: bound}))
	}
	n.Setof = setof
	return n
}

func grammarArrayBounds(bounds any, bound int32) []int32 {
	list, _ := bounds.([]int32)
	return append(list, bound)
}

func grammarNegate(value any, loc int) Node {
	n := toNode(value)
	if c := n.GetAConst(); c != nil {
		grammarSetLocation(n, loc)
		switch v := c.Val.(type) {
		case *pg_query.A_Const_Ival:
			v.Ival.Ival = -v.Ival.Ival
			return n
		case *pg_query.A_Const_Fval:
			s := strings.TrimPrefix(v.Fval.Fval, "+")
			if strings.HasPrefix(s, "-") { s = s[1:] } else { s = "-" + s }
			v.Fval.Fval = s
			return n
		}
	}
	out := &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"-"}), Rexpr: n, Location: -1}
	grammarSetLocation(out, loc)
	return toNode(out)
}

func grammarIndirection(arg any, value any) Node {
	items := toNodeSlice(value)
	if len(items) == 0 { return toNode(arg) }
	return toNode(&A_Indirection{Arg: toNode(arg), Indirection: items})
}

func grammarConst(value any, location int) *pg_query.A_Const {
	n := makeAConst(value)
	n.Location = int32(location)
	return n
}

func grammarColumnRef(name any, value any, location int) Node {
	ref := &ColumnRef{Fields: []Node{toNode(&String{Sval: toStr(name)})}, Location: int32(location)}
	items := toNodeSlice(value)
	for i, item := range items {
		if item.GetAIndices() != nil {
			return toNode(&A_Indirection{Arg: toNode(ref), Indirection: items[i:]})
		}
		ref.Fields = append(ref.Fields, item)
	}
	return toNode(ref)
}

func grammarInfer(value any) *InferClause {
	n, _ := value.(*InferClause)
	return n
}

func grammarCreateTable(rel, elements any, missingOK bool) *CreateStmt {
	return &CreateStmt{Relation: toRangeVar(rel), TableElts: toNodeSlice(elements), IfNotExists: missingOK, Oncommit: pg_query.OnCommitAction(1)}
}

type grammarPeriodColumns struct {
	cols []string
	period bool
}

func grammarConstraintAttrError(kind ConstraintType, value any) string {
	a := toConstrAttr(value)
	check := kind == ConstrCheck || kind == ConstrNotNull
	if check && a.deferrable { return "constraint cannot be marked DEFERRABLE" }
	if !check && kind != ConstrForeignKey && a.skipValidation { return "constraint cannot be marked NOT VALID" }
	if !check && a.noInherit { return "constraint cannot be marked NO INHERIT" }
	if kind != ConstrCheck && kind != ConstrForeignKey && (a.seen["ENFORCED"] || a.seen["NOT ENFORCED"]) { return "constraint cannot be marked ENFORCED or NOT ENFORCED" }
	return ""
}

func grammarConstraintAttrs(n *Constraint, value any) *Constraint {
	a := toConstrAttr(value)
	n.Deferrable, n.Initdeferred = a.deferrable, a.initdeferred
	n.SkipValidation, n.IsNoInherit = a.skipValidation, a.noInherit
	n.IsEnforced = !a.notEnforced && (n.Contype == ConstrCheck || n.Contype == ConstrForeignKey)
	if n.Contype == ConstrCheck || n.Contype == ConstrNotNull || n.Contype == ConstrForeignKey {
		n.InitiallyValid = !a.skipValidation && !a.notEnforced
	}
	return n
}

type grammarKeyAction struct {
	action string
	cols []string
}

type grammarKeyActions struct {
	update, delete grammarKeyAction
}

func grammarForeignKey(rel, cols, match, actions any) *Constraint {
	n := &Constraint{Contype: ConstrForeignKey, Pktable: toRangeVar(rel), PkAttrs: toPgStringNodes(toStrSlice(cols)), FkMatchtype: toStr(match), FkUpdAction: "a", FkDelAction: "a", InitiallyValid: true, Location: -1}
	if c, ok := cols.(grammarPeriodColumns); ok { n.PkAttrs, n.PkWithPeriod = toPgStringNodes(c.cols), c.period }
	if a, ok := actions.(grammarKeyActions); ok {
		if a.update.action != "" { n.FkUpdAction = toPgReferentialAction(a.update.action) }
		if a.delete.action != "" { n.FkDelAction = toPgReferentialAction(a.delete.action) }
		n.FkDelSetCols = toPgStringNodes(a.delete.cols)
	}
	n.IsEnforced = true
	return n
}

func grammarTableOptions(n *CreateStmt, temp, inherit, part, method, options, commit, space any) *CreateStmt {
	n.InhRelations = toRangeVarSlice(inherit)
	n.Partspec, _ = part.(*PartitionSpec)
	n.AccessMethod, n.Options, n.Oncommit, n.Tablespacename = toStr(method), toDefElemSlice(options), enumOnCommitAction(toStr(commit)), toStr(space)
	if n.Relation != nil {
		n.Relation = grammarUtilityRelation(n.Relation, temp)
	}
	return n
}

func grammarColumn(name, typ, storage, compression, options, quals any) *ColumnDef {
	n := &ColumnDef{Colname: toStr(name), IsLocal: true, StorageName: toStr(storage), Compression: toStr(compression), Fdwoptions: toNodeSlice(toDefElemSlice(options)), Location: -1}
	if typ != nil { n.TypeName = toTypeNameNode(typ) }
	for _, q := range toNodeSlice(quals) {
		if q.GetConstraint() != nil { n.Constraints = append(n.Constraints, q) }
		if coll := q.GetCollateClause(); coll != nil { n.CollClause = coll }
	}
	return n
}
%}

%loctype int

%union {
	node any
	str string
	int int64
	float float64
}

/* Tokens matching pg_query_go exactly */
%token <str> IDENT 258
%token <str> UIDENT 259
%token <str> FCONST 260
%token <str> SCONST 261
%token <str> USCONST 262
%token <str> BCONST 263
%token <str> XCONST 264
%token <str> Op 265
%token <int> ICONST 266
%token <int> PARAM 267
%token <node> TYPECAST 268
%token <node> DOT_DOT 269
%token <node> COLON_EQUALS 270
%token <node> EQUALS_GREATER 271
%token <node> LESS_EQUALS 272
%token <node> GREATER_EQUALS 273
%token <node> NOT_EQUALS 274
%token <node> SQL_COMMENT 275
%token <node> C_COMMENT 276
%token <str> ABORT_P 277
%token <str> ABSENT 278
%token <str> ABSOLUTE_P 279
%token <str> ACCESS 280
%token <str> ACTION 281
%token <str> ADD_P 282
%token <str> ADMIN 283
%token <str> AFTER 284
%token <str> AGGREGATE 285
%token <str> ALL 286
%token <str> ALSO 287
%token <str> ALTER 288
%token <str> ALWAYS 289
%token <str> ANALYSE 290
%token <str> ANALYZE 291
%token <str> AND 292
%token <str> ANY 293
%token <str> ARRAY 294
%token <str> AS 295
%token <str> ASC 296
%token <str> ASENSITIVE 297
%token <str> ASSERTION 298
%token <str> ASSIGNMENT 299
%token <str> ASYMMETRIC 300
%token <str> ATOMIC 301
%token <str> AT 302
%token <str> ATTACH 303
%token <str> ATTRIBUTE 304
%token <str> AUTHORIZATION 305
%token <str> BACKWARD 306
%token <str> BEFORE 307
%token <str> BEGIN_P 308
%token <str> BETWEEN 309
%token <str> BIGINT 310
%token <str> BINARY 311
%token <str> BIT 312
%token <str> BOOLEAN_P 313
%token <str> BOTH 314
%token <str> BREADTH 315
%token <str> BY 316
%token <str> CACHE 317
%token <str> CALL 318
%token <str> CALLED 319
%token <str> CASCADE 320
%token <str> CASCADED 321
%token <str> CASE 322
%token <str> CAST 323
%token <str> CATALOG_P 324
%token <str> CHAIN 325
%token <str> CHAR_P 326
%token <str> CHARACTER 327
%token <str> CHARACTERISTICS 328
%token <str> CHECK 329
%token <str> CHECKPOINT 330
%token <str> CLASS 331
%token <str> CLOSE 332
%token <str> CLUSTER 333
%token <str> COALESCE 334
%token <str> COLLATE 335
%token <str> COLLATION 336
%token <str> COLUMN 337
%token <str> COLUMNS 338
%token <str> COMMENT 339
%token <str> COMMENTS 340
%token <str> COMMIT 341
%token <str> COMMITTED 342
%token <str> COMPRESSION 343
%token <str> CONCURRENTLY 344
%token <str> CONDITIONAL 345
%token <str> CONFIGURATION 346
%token <str> CONFLICT 347
%token <str> CONNECTION 348
%token <str> CONSTRAINT 349
%token <str> CONSTRAINTS 350
%token <str> CONTENT_P 351
%token <str> CONTINUE_P 352
%token <str> CONVERSION_P 353
%token <str> COPY 354
%token <str> COST 355
%token <str> CREATE 356
%token <str> CROSS 357
%token <str> CSV 358
%token <str> CUBE 359
%token <str> CURRENT_P 360
%token <str> CURRENT_CATALOG 361
%token <str> CURRENT_DATE 362
%token <str> CURRENT_ROLE 363
%token <str> CURRENT_SCHEMA 364
%token <str> CURRENT_TIME 365
%token <str> CURRENT_TIMESTAMP 366
%token <str> CURRENT_USER 367
%token <str> CURSOR 368
%token <str> CYCLE 369
%token <str> DATA_P 370
%token <str> DATABASE 371
%token <str> DAY_P 372
%token <str> DEALLOCATE 373
%token <str> DEC 374
%token <str> DECIMAL_P 375
%token <str> DECLARE 376
%token <str> DEFAULT 377
%token <str> DEFAULTS 378
%token <str> DEFERRABLE 379
%token <str> DEFERRED 380
%token <str> DEFINER 381
%token <str> DELETE_P 382
%token <str> DELIMITER 383
%token <str> DELIMITERS 384
%token <str> DEPENDS 385
%token <str> DEPTH 386
%token <str> DESC 387
%token <str> DETACH 388
%token <str> DICTIONARY 389
%token <str> DISABLE_P 390
%token <str> DISCARD 391
%token <str> DISTINCT 392
%token <str> DO 393
%token <str> DOCUMENT_P 394
%token <str> DOMAIN_P 395
%token <str> DOUBLE_P 396
%token <str> DROP 397
%token <str> EACH 398
%token <str> ELSE 399
%token <str> EMPTY_P 400
%token <str> ENABLE_P 401
%token <str> ENCODING 402
%token <str> ENCRYPTED 403
%token <str> END_P 404
%token <str> ENUM_P 406
%token <str> ERROR_P 407
%token <str> ESCAPE 408
%token <str> EVENT 409
%token <str> EXCEPT 410
%token <str> EXCLUDE 411
%token <str> EXCLUDING 412
%token <str> EXCLUSIVE 413
%token <str> EXECUTE 414
%token <str> EXISTS 415
%token <str> EXPLAIN 416
%token <str> EXPRESSION 417
%token <str> EXTENSION 418
%token <str> EXTERNAL 419
%token <str> EXTRACT 420
%token <str> FALSE_P 421
%token <str> FAMILY 422
%token <str> FETCH 423
%token <str> FILTER 424
%token <str> FINALIZE 425
%token <str> FIRST_P 426
%token <str> FLOAT_P 427
%token <str> FOLLOWING 428
%token <str> FOR 429
%token <str> FORCE 430
%token <str> FOREIGN 431
%token <str> FORMAT 432
%token <str> FORWARD 433
%token <str> FREEZE 434
%token <str> FROM 435
%token <str> FULL 436
%token <str> FUNCTION 437
%token <str> FUNCTIONS 438
%token <str> GENERATED 439
%token <str> GLOBAL 440
%token <str> GRANT 441
%token <str> GRANTED 442
%token <str> GREATEST 443
%token <str> GROUP_P 444
%token <str> GROUPING 445
%token <str> GROUPS 446
%token <str> HANDLER 447
%token <str> HAVING 448
%token <str> HEADER_P 449
%token <str> HOLD 450
%token <str> HOUR_P 451
%token <str> IDENTITY_P 452
%token <str> IF_P 453
%token <str> ILIKE 454
%token <str> IMMEDIATE 455
%token <str> IMMUTABLE 456
%token <str> IMPLICIT_P 457
%token <str> IMPORT_P 458
%token <str> IN_P 459
%token <str> INCLUDE 460
%token <str> INCLUDING 461
%token <str> INCREMENT 462
%token <str> INDENT 463
%token <str> INDEX 464
%token <str> INDEXES 465
%token <str> INHERIT 466
%token <str> INHERITS 467
%token <str> INITIALLY 468
%token <str> INLINE_P 469
%token <str> INNER_P 470
%token <str> INOUT 471
%token <str> INPUT_P 472
%token <str> INSENSITIVE 473
%token <str> INSERT 474
%token <str> INSTEAD 475
%token <str> INT_P 476
%token <str> INTEGER 477
%token <str> INTERSECT 478
%token <str> INTERVAL 479
%token <str> INTO 480
%token <str> INVOKER 481
%token <str> IS 482
%token <str> ISNULL 483
%token <str> ISOLATION 484
%token <str> JOIN 485
%token <str> JSON 486
%token <str> JSON_ARRAY 487
%token <str> JSON_ARRAYAGG 488
%token <str> JSON_EXISTS 489
%token <str> JSON_OBJECT 490
%token <str> JSON_OBJECTAGG 491
%token <str> JSON_QUERY 492
%token <str> JSON_SCALAR 493
%token <str> JSON_SERIALIZE 494
%token <str> JSON_TABLE 495
%token <str> JSON_VALUE 496
%token <str> KEEP 497
%token <str> KEY 498
%token <str> KEYS 499
%token <str> LABEL 500
%token <str> LANGUAGE 501
%token <str> LARGE_P 502
%token <str> LAST_P 503
%token <str> LATERAL_P 504
%token <str> LEADING 505
%token <str> LEAKPROOF 506
%token <str> LEAST 507
%token <str> LEFT 508
%token <str> LEVEL 509
%token <str> LIKE 510
%token <str> LIMIT 511
%token <str> LISTEN 512
%token <str> LOAD 513
%token <str> LOCAL 514
%token <str> LOCALTIME 515
%token <str> LOCALTIMESTAMP 516
%token <str> LOCATION 517
%token <str> LOCK_P 518
%token <str> LOCKED 519
%token <str> LOGGED 520
%token <str> MAPPING 521
%token <str> MATCH 522
%token <str> MATCHED 523
%token <str> MATERIALIZED 524
%token <str> MAXVALUE 525
%token <str> MERGE 526
%token <str> MERGE_ACTION 527
%token <str> METHOD 528
%token <str> MINUTE_P 529
%token <str> MINVALUE 530
%token <str> MODE 531
%token <str> MONTH_P 532
%token <str> MOVE 533
%token <str> NAME_P 534
%token <str> NAMES 535
%token <str> NATIONAL 536
%token <str> NATURAL 537
%token <str> NCHAR 538
%token <str> NESTED 539
%token <str> NEW 540
%token <str> NEXT 541
%token <str> NFC 542
%token <str> NFD 543
%token <str> NFKC 544
%token <str> NFKD 545
%token <str> NO 546
%token <str> NONE 547
%token <str> NORMALIZE 548
%token <str> NORMALIZED 549
%token <str> NOT 550
%token <str> NOTHING 551
%token <str> NOTIFY 552
%token <str> NOTNULL 553
%token <str> NOWAIT 554
%token <str> NULL_P 555
%token <str> NULLIF 556
%token <str> NULLS_P 557
%token <str> NUMERIC 558
%token <str> OBJECT_P 559
%token <str> OF 561
%token <str> OFF 562
%token <str> OFFSET 563
%token <str> OIDS 564
%token <str> OLD 565
%token <str> OMIT 566
%token <str> ON 567
%token <str> ONLY 568
%token <str> OPERATOR 569
%token <str> OPTION 570
%token <str> OPTIONS 571
%token <str> OR 572
%token <str> ORDER 573
%token <str> ORDINALITY 574
%token <str> OTHERS 575
%token <str> OUT_P 576
%token <str> OUTER_P 577
%token <str> OVER 578
%token <str> OVERLAPS 579
%token <str> OVERLAY 580
%token <str> OVERRIDING 581
%token <str> OWNED 582
%token <str> OWNER 583
%token <str> PARALLEL 584
%token <str> PARAMETER 585
%token <str> PARSER 586
%token <str> PARTIAL 587
%token <str> PARTITION 588
%token <str> PASSING 589
%token <str> PASSWORD 590
%token <str> PATH 591
%token <str> PLACING 593
%token <str> PLAN 594
%token <str> PLANS 595
%token <str> POLICY 596
%token <str> POSITION 597
%token <str> PRECEDING 598
%token <str> PRECISION 599
%token <str> PRESERVE 600
%token <str> PREPARE 601
%token <str> PREPARED 602
%token <str> PRIMARY 603
%token <str> PRIOR 604
%token <str> PRIVILEGES 605
%token <str> PROCEDURAL 606
%token <str> PROCEDURE 607
%token <str> PROCEDURES 608
%token <str> PROGRAM 609
%token <str> PUBLICATION 610
%token <str> QUOTE 611
%token <str> QUOTES 612
%token <str> RANGE 613
%token <str> READ 614
%token <str> REAL 615
%token <str> REASSIGN 616
%token <str> RECURSIVE 617
%token <str> REF_P 618
%token <str> REFERENCES 619
%token <str> REFERENCING 620
%token <str> REFRESH 621
%token <str> REINDEX 622
%token <str> RELATIVE_P 623
%token <str> RELEASE 624
%token <str> RENAME 625
%token <str> REPEATABLE 626
%token <str> REPLACE 627
%token <str> REPLICA 628
%token <str> RESET 629
%token <str> RESTART 630
%token <str> RESTRICT 631
%token <str> RETURN 632
%token <str> RETURNING 633
%token <str> RETURNS 634
%token <str> REVOKE 635
%token <str> RIGHT 636
%token <str> ROLE 637
%token <str> ROLLBACK 638
%token <str> ROLLUP 639
%token <str> ROUTINE 640
%token <str> ROUTINES 641
%token <str> ROW 642
%token <str> ROWS 643
%token <str> RULE 644
%token <str> SAVEPOINT 645
%token <str> SCALAR 646
%token <str> SCHEMA 647
%token <str> SCHEMAS 648
%token <str> SCROLL 649
%token <str> SEARCH 650
%token <str> SECOND_P 651
%token <str> SECURITY 652
%token <str> SELECT 653
%token <str> SEQUENCE 654
%token <str> SEQUENCES 655
%token <str> SERIALIZABLE 656
%token <str> SERVER 657
%token <str> SESSION 658
%token <str> SESSION_USER 659
%token <str> SET 660
%token <str> SETS 661
%token <str> SETOF 662
%token <str> SHARE 663
%token <str> SHOW 664
%token <str> SIMILAR 665
%token <str> SIMPLE 666
%token <str> SKIP 667
%token <str> SMALLINT 668
%token <str> SNAPSHOT 669
%token <str> SOME 670
%token <str> SOURCE 671
%token <str> SQL_P 672
%token <str> STABLE 673
%token <str> STANDALONE_P 674
%token <str> START 675
%token <str> STATEMENT 676
%token <str> STATISTICS 677
%token <str> STDIN 678
%token <str> STDOUT 679
%token <str> STORAGE 680
%token <str> STORED 681
%token <str> STRICT_P 682
%token <str> STRING_P 683
%token <str> STRIP_P 684
%token <str> SUBSCRIPTION 685
%token <str> SUBSTRING 686
%token <str> SUPPORT 687
%token <str> SYMMETRIC 688
%token <str> SYSID 689
%token <str> SYSTEM_P 690
%token <str> SYSTEM_USER 691
%token <str> TABLE 692
%token <str> TABLES 693
%token <str> TABLESAMPLE 694
%token <str> TABLESPACE 695
%token <str> TARGET 696
%token <str> TEMP 697
%token <str> TEMPLATE 698
%token <str> TEMPORARY 699
%token <str> TEXT_P 700
%token <str> THEN 701
%token <str> TIES 702
%token <str> TIME 703
%token <str> TIMESTAMP 704
%token <str> TO 705
%token <str> TRAILING 706
%token <str> TRANSACTION 707
%token <str> TRANSFORM 708
%token <str> TREAT 709
%token <str> TRIGGER 710
%token <str> TRIM 711
%token <str> TRUE_P 712
%token <str> TRUNCATE 713
%token <str> TRUSTED 714
%token <str> TYPE_P 715
%token <str> TYPES_P 716
%token <str> UESCAPE 717
%token <str> UNBOUNDED 718
%token <str> UNCONDITIONAL 719
%token <str> UNCOMMITTED 720
%token <str> UNENCRYPTED 721
%token <str> UNION 722
%token <str> UNIQUE 723
%token <str> UNKNOWN 724
%token <str> UNLISTEN 725
%token <str> UNLOGGED 726
%token <str> UNTIL 727
%token <str> UPDATE 728
%token <str> USER 729
%token <str> USING 730
%token <str> VACUUM 731
%token <str> VALID 732
%token <str> VALIDATE 733
%token <str> VALIDATOR 734
%token <str> VALUE_P 735
%token <str> VALUES 736
%token <str> VARCHAR 737
%token <str> VARIADIC 738
%token <str> VARYING 739
%token <str> VERBOSE 740
%token <str> VERSION_P 741
%token <str> VIEW 742
%token <str> VIEWS 743
%token <str> VOLATILE 745
%token <str> WHEN 746
%token <str> WHERE 747
%token <str> WHITESPACE_P 748
%token <str> WINDOW 749
%token <str> WITH 750
%token <str> WITHIN 751
%token <str> WITHOUT 752
%token <str> WORK 753
%token <str> WRAPPER 754
%token <str> WRITE 755
%token <str> XML_P 756
%token <str> XMLATTRIBUTES 757
%token <str> XMLCONCAT 758
%token <str> XMLELEMENT 759
%token <str> XMLEXISTS 760
%token <str> XMLFOREST 761
%token <str> XMLNAMESPACES 762
%token <str> XMLPARSE 763
%token <str> XMLPI 764
%token <str> XMLROOT 765
%token <str> XMLSERIALIZE 766
%token <str> XMLTABLE 767
%token <str> YEAR_P 768
%token <str> YES_P 769
%token <str> ZONE 770
%token <node> FORMAT_LA 771
%token <node> NOT_LA 772
%token <node> NULLS_LA 773
%token <node> WITH_LA 774
%token <node> WITHOUT_LA 775
%token <node> MODE_TYPE_NAME 776
%token <node> MODE_PLPGSQL_EXPR 777
%token <node> MODE_PLPGSQL_ASSIGN1 778
%token <node> MODE_PLPGSQL_ASSIGN2 779
%token <node> MODE_PLPGSQL_ASSIGN3 780
%token <node> UMINUS 781
%token <str> ENFORCED 405
%token <str> OBJECTS_P 560
%token <str> PERIOD 592
%token <str> VIRTUAL 744

/* Precedence declarations directly from pg18/gram.y */
%left		UNION EXCEPT
%left		INTERSECT
%left		OR
%left		AND
%right		NOT
%nonassoc	IS ISNULL NOTNULL
%nonassoc	'<' '>' '=' LESS_EQUALS GREATER_EQUALS NOT_EQUALS
%nonassoc	BETWEEN IN_P LIKE ILIKE SIMILAR NOT_LA
%nonassoc	ESCAPE
%nonassoc	UNBOUNDED NESTED
%nonassoc	IDENT PARTITION RANGE ROWS GROUPS PRECEDING FOLLOWING CUBE ROLLUP SET KEYS OBJECT_P SCALAR VALUE_P WITH WITHOUT PATH NULLS_LA WITH_LA WITHOUT_LA FORMAT_LA
%left		Op OPERATOR
%left		'+' '-'
%left		'*' '/' '%'
%left		'^'
%left		AT
%left		COLLATE
%right		UMINUS
%left		'[' ']'
%left		'(' ')'
%left		TYPECAST
%left		'.'
%left		JOIN CROSS LEFT FULL RIGHT INNER_P NATURAL

/* All 733 Nonterminal Type Declarations */
%type <node> parse_toplevel stmtmulti toplevel_stmt stmt opt_single_name opt_qualified_name opt_concurrently opt_drop_behavior CallStmt CreateRoleStmt
%type <node> opt_with OptRoleList AlterOptRoleList AlterOptRoleElem CreateOptRoleElem CreateUserStmt AlterRoleStmt opt_in_database AlterRoleSetStmt DropRoleStmt
%type <node> CreateGroupStmt AlterGroupStmt add_drop CreateSchemaStmt OptSchemaEltList schema_stmt VariableSetStmt set_rest generic_set set_rest_more
%type <node> var_name var_list var_value iso_level opt_boolean_or_string zone_value opt_encoding NonReservedWord_or_Sconst VariableResetStmt reset_rest
%type <node> generic_reset SetResetClause FunctionSetResetClause VariableShowStmt ConstraintsSetStmt constraints_set_list constraints_set_mode CheckPointStmt DiscardStmt AlterTableStmt
%type <node> alter_table_cmds partition_cmd index_partition_cmd alter_table_cmd alter_column_default opt_collate_clause alter_using replica_identity reloptions opt_reloptions
%type <node> reloption_list reloption_elem alter_identity_column_option_list alter_identity_column_option set_statistics_value set_access_method_name PartitionBoundSpec hash_partbound_elem hash_partbound AlterCompositeTypeStmt
%type <node> alter_type_cmds alter_type_cmd ClosePortalStmt CopyStmt copy_from opt_program copy_file_name copy_options copy_opt_list copy_opt_item
%type <node> opt_binary copy_delimiter opt_using copy_generic_opt_list copy_generic_opt_elem copy_generic_opt_arg copy_generic_opt_arg_list copy_generic_opt_arg_list_item CreateStmt OptTemp
%type <node> OptTableElementList OptTypedTableElementList TableElementList TypedTableElementList TableElement TypedTableElement columnDef columnOptions column_compression opt_column_compression
%type <node> column_storage opt_column_storage ColQualList ColConstraint ColConstraintElem opt_unique_null_treatment generated_when opt_virtual_or_stored ConstraintAttr TableLikeClause
%type <node> TableLikeOptionList TableLikeOption TableConstraint ConstraintElem DomainConstraint DomainConstraintElem opt_no_inherit opt_without_overlaps opt_column_list columnList
%type <node> optionalPeriodName opt_column_and_period_list columnElem opt_c_include key_match ExclusionConstraintList ExclusionConstraintElem OptWhereClause key_actions key_update
%type <node> key_delete key_action OptInherit OptPartitionSpec PartitionSpec part_params part_elem table_access_method_clause OptWith OnCommitOption
%type <node> OptTableSpace OptConsTableSpace ExistingIndex CreateStatsStmt stats_params stats_param AlterStatsStmt CreateAsStmt create_as_target opt_with_data
%type <node> CreateMatViewStmt create_mv_target OptNoLog RefreshMatViewStmt CreateSeqStmt AlterSeqStmt OptSeqOptList OptParenthesizedSeqOptList SeqOptList SeqOptElem
%type <node> opt_by NumericOnly NumericOnly_list CreatePLangStmt opt_trusted handler_name opt_inline_handler validator_clause opt_validator opt_procedural
%type <node> CreateTableSpaceStmt OptTableSpaceOwner DropTableSpaceStmt CreateExtensionStmt create_extension_opt_list create_extension_opt_item AlterExtensionStmt alter_extension_opt_list alter_extension_opt_item AlterExtensionContentsStmt
%type <node> CreateFdwStmt fdw_option fdw_options opt_fdw_options AlterFdwStmt create_generic_options generic_option_list alter_generic_options alter_generic_option_list alter_generic_option_elem
%type <node> generic_option_elem generic_option_name generic_option_arg CreateForeignServerStmt opt_type foreign_server_version opt_foreign_server_version AlterForeignServerStmt CreateForeignTableStmt ImportForeignSchemaStmt
%type <node> import_qualification_type import_qualification CreateUserMappingStmt auth_ident DropUserMappingStmt AlterUserMappingStmt CreatePolicyStmt AlterPolicyStmt RowSecurityOptionalExpr RowSecurityOptionalWithCheck
%type <node> RowSecurityDefaultToRole RowSecurityOptionalToRole RowSecurityDefaultPermissive RowSecurityDefaultForCmd row_security_cmd CreateAmStmt am_type CreateTrigStmt TriggerActionTime TriggerEvents
%type <node> TriggerOneEvent TriggerReferencing TriggerTransitions TriggerTransition TransitionOldOrNew TransitionRowOrTable TransitionRelName TriggerForSpec TriggerForOptEach TriggerForType
%type <node> TriggerWhen FUNCTION_or_PROCEDURE TriggerFuncArgs TriggerFuncArg OptConstrFromTable ConstraintAttributeSpec ConstraintAttributeElem CreateEventTrigStmt event_trigger_when_list event_trigger_when_item
%type <node> event_trigger_value_list AlterEventTrigStmt enable_trigger CreateAssertionStmt DefineStmt definition def_list def_elem def_arg old_aggr_definition
%type <node> old_aggr_list old_aggr_elem opt_enum_val_list enum_val_list AlterEnumStmt opt_if_not_exists CreateOpClassStmt opclass_item_list opclass_item opt_default
%type <node> opt_opfamily opclass_purpose CreateOpFamilyStmt AlterOpFamilyStmt opclass_drop_list opclass_drop DropOpClassStmt DropOpFamilyStmt DropOwnedStmt ReassignOwnedStmt
%type <node> DropStmt object_type_any_name object_type_name drop_type_name object_type_name_on_any_name any_name_list any_name attrs type_name_list TruncateStmt
%type <node> opt_restart_seqs CommentStmt comment_text SecLabelStmt opt_provider security_label FetchStmt fetch_args from_in opt_from_in
%type <node> GrantStmt RevokeStmt privileges privilege_list privilege parameter_name_list parameter_name privilege_target grantee_list grantee
%type <node> opt_grant_grant_option GrantRoleStmt RevokeRoleStmt grant_role_opt_list grant_role_opt grant_role_opt_value opt_granted_by AlterDefaultPrivilegesStmt DefACLOptionList DefACLOption
%type <node> DefACLAction defacl_privilege_target IndexStmt opt_unique access_method_clause index_params index_elem_options index_elem opt_include index_including_params
%type <node> opt_collate opt_asc_desc opt_nulls_order CreateFunctionStmt opt_or_replace func_args func_args_list function_with_argtypes_list function_with_argtypes func_args_with_defaults
%type <node> func_args_with_defaults_list func_arg arg_class param_name func_return func_type func_arg_with_default aggr_arg aggr_args aggr_args_list
%type <node> aggregate_with_argtypes aggregate_with_argtypes_list opt_createfunc_opt_list createfunc_opt_list common_func_opt_item createfunc_opt_item func_as ReturnStmt opt_routine_body routine_body_stmt_list
%type <node> routine_body_stmt transform_type_list opt_definition table_func_column table_func_column_list AlterFunctionStmt alterfunc_opt_list opt_restrict RemoveFuncStmt RemoveAggrStmt
%type <node> RemoveOperStmt oper_argtypes any_operator operator_with_argtypes_list operator_with_argtypes DoStmt dostmt_opt_list dostmt_opt_item CreateCastStmt cast_context
%type <node> DropCastStmt opt_if_exists CreateTransformStmt transform_element_list DropTransformStmt ReindexStmt reindex_target_relation reindex_target_all opt_reindex_option_list AlterTblSpcStmt
%type <node> RenameStmt opt_column opt_set_data AlterObjectDependsStmt opt_no AlterObjectSchemaStmt AlterOperatorStmt operator_def_list operator_def_elem operator_def_arg
%type <node> AlterTypeStmt AlterOwnerStmt CreatePublicationStmt PublicationObjSpec pub_obj_list AlterPublicationStmt CreateSubscriptionStmt AlterSubscriptionStmt DropSubscriptionStmt RuleStmt
%type <node> RuleActionList RuleActionMulti RuleActionStmt RuleActionStmtOrEmpty event opt_instead NotifyStmt notify_payload ListenStmt UnlistenStmt
%type <node> TransactionStmt TransactionStmtLegacy opt_transaction transaction_mode_item transaction_mode_list transaction_mode_list_or_empty opt_transaction_chain ViewStmt opt_check_option LoadStmt
%type <node> CreatedbStmt createdb_opt_list createdb_opt_items createdb_opt_item createdb_opt_name opt_equal AlterDatabaseStmt AlterDatabaseSetStmt DropdbStmt drop_option_list
%type <node> drop_option AlterCollationStmt AlterSystemStmt CreateDomainStmt AlterDomainStmt opt_as AlterTSDictionaryStmt AlterTSConfigurationStmt any_with CreateConversionStmt
%type <node> ClusterStmt cluster_index_specification VacuumStmt AnalyzeStmt utility_option_list analyze_keyword utility_option_elem utility_option_name utility_option_arg opt_analyze
%type <node> opt_verbose opt_full opt_freeze opt_name_list vacuum_relation vacuum_relation_list opt_vacuum_relation_list ExplainStmt ExplainableStmt PrepareStmt
%type <node> prep_type_clause PreparableStmt ExecuteStmt execute_param_clause DeallocateStmt InsertStmt insert_target insert_rest override_kind insert_column_list
%type <node> insert_column_item opt_on_conflict opt_conf_expr returning_clause returning_with_clause returning_options returning_option returning_option_kind DeleteStmt using_clause
%type <node> LockStmt opt_lock lock_type opt_nowait opt_nowait_or_skip UpdateStmt set_clause_list set_clause set_target set_target_list
%type <node> MergeStmt merge_when_list merge_when_clause merge_when_tgt_matched merge_when_tgt_not_matched opt_merge_when_condition merge_update merge_delete merge_insert merge_values_clause
%type <node> DeclareCursorStmt cursor_name cursor_options opt_hold SelectStmt select_with_parens select_no_parens select_clause simple_select with_clause
%type <node> cte_list common_table_expr opt_materialized opt_search_clause opt_cycle_clause opt_with_clause into_clause OptTempTableName opt_table set_quantifier
%type <node> distinct_clause opt_all_clause opt_distinct_clause opt_sort_clause sort_clause sortby_list sortby select_limit opt_select_limit limit_clause
%type <node> offset_clause select_limit_value select_offset_value select_fetch_first_value I_or_F_const row_or_rows first_or_next group_clause group_by_list group_by_item
%type <node> empty_grouping_set rollup_clause cube_clause grouping_sets_clause having_clause for_locking_clause opt_for_locking_clause for_locking_items for_locking_item for_locking_strength
%type <node> locked_rels_list values_clause from_clause from_list table_ref joined_table alias_clause opt_alias_clause opt_alias_clause_for_join_using func_alias_clause
%type <node> join_type opt_outer join_qual relation_expr extended_relation_expr relation_expr_list relation_expr_opt_alias tablesample_clause opt_repeatable_clause func_table
%type <node> rowsfrom_item rowsfrom_list opt_col_def_list opt_ordinality where_clause where_or_current_clause OptTableFuncElementList TableFuncElementList TableFuncElement xmltable
%type <node> xmltable_column_list xmltable_column_el xmltable_column_option_list xmltable_column_option_el xml_namespace_list xml_namespace_el json_table json_table_path_name_opt json_table_column_definition_list json_table_column_definition
%type <node> path_opt json_table_column_path_clause_opt Typename opt_array_bounds SimpleTypename ConstTypename GenericType opt_type_modifiers Numeric opt_float
%type <node> Bit ConstBit BitWithLength BitWithoutLength Character ConstCharacter CharacterWithLength CharacterWithoutLength character opt_varying
%type <node> ConstDatetime ConstInterval opt_timezone opt_interval interval_second JsonType a_expr b_expr c_expr func_application
%type <node> func_expr func_expr_windowless func_expr_common_subexpr xml_root_version opt_xml_root_standalone xml_attributes xml_attribute_list xml_attribute_el document_or_content xml_indent_option
%type <node> xml_whitespace_option xmlexists_argument xml_passing_mech within_group_clause filter_clause window_clause window_definition_list window_definition over_clause window_specification
%type <node> opt_existing_window_name opt_partition_clause opt_frame_clause frame_extent frame_bound opt_window_exclusion_clause row explicit_row implicit_row sub_type
%type <node> all_Op MathOp qual_Op qual_all_Op subquery_Op expr_list func_arg_list func_arg_expr func_arg_list_opt type_list
%type <node> array_expr array_expr_list extract_list extract_arg unicode_normal_form overlay_list position_list substr_list trim_list case_expr
%type <node> when_clause_list when_clause case_default case_arg columnref indirection_el opt_slice_bound indirection opt_indirection opt_asymmetric
%type <node> json_passing_clause_opt json_arguments json_argument json_wrapper_behavior json_behavior json_behavior_type json_behavior_clause_opt json_on_error_clause_opt json_value_expr json_format_clause
%type <node> json_format_clause_opt json_quotes_clause_opt json_returning_clause_opt json_predicate_type_constraint json_key_uniqueness_constraint_opt json_name_and_value_list json_name_and_value json_object_constructor_null_clause_opt json_array_constructor_null_clause_opt json_value_expr_list
%type <node> json_aggregate_func json_array_aggregate_order_by_clause_opt opt_target_list target_list target_el qualified_name_list qualified_name name_list name attr_name
%type <node> file_name func_name AexprConst Iconst Sconst SignedIconst RoleId RoleSpec role_list PLpgSQL_Expr
%type <node> PLAssignStmt plassign_target plassign_equals ColId type_function_name NonReservedWord ColLabel BareColLabel unreserved_keyword col_name_keyword
%type <node> type_func_name_keyword reserved_keyword bare_label_keyword

%%

parse_toplevel:
	  stmtmulti
	{
		yylex.(*Lexer).statements = toStmtSlice($1)
		$$ = $1
	}
	| MODE_TYPE_NAME Typename
	{
		yylex.(*Lexer).statements = toStmtSlice($2)
		yylex.(*Lexer).startStatement(toStatement($2), @2)
		$$ = $2
	}
	| MODE_PLPGSQL_EXPR PLpgSQL_Expr
	{
		yylex.(*Lexer).statements = toStmtSlice($2)
		yylex.(*Lexer).startStatement(toStatement($2), @2)
		$$ = $2
	}
	| MODE_PLPGSQL_ASSIGN1 PLAssignStmt
	{
		$2.(*pg_query.PLAssignStmt).Nnames = 1
		yylex.(*Lexer).statements = toStmtSlice($2)
		yylex.(*Lexer).startStatement(toStatement($2), @2)
		$$ = $2
	}
	| MODE_PLPGSQL_ASSIGN2 PLAssignStmt
	{
		$2.(*pg_query.PLAssignStmt).Nnames = 2
		yylex.(*Lexer).statements = toStmtSlice($2)
		yylex.(*Lexer).startStatement(toStatement($2), @2)
		$$ = $2
	}
	| MODE_PLPGSQL_ASSIGN3 PLAssignStmt
	{
		$2.(*pg_query.PLAssignStmt).Nnames = 3
		yylex.(*Lexer).statements = toStmtSlice($2)
		yylex.(*Lexer).startStatement(toStatement($2), @2)
		$$ = $2
	}
	;

stmtmulti:
	  stmtmulti ';' toplevel_stmt
	{
		$$ = appendToStmtSlice($1, $3)
		yylex.(*Lexer).finishStatement(@2, toStatement($3), @3)
	}
	| toplevel_stmt
	{
		$$ = singleStmtSlice($1)
		yylex.(*Lexer).startStatement(toStatement($1), @1)
	}
	;

toplevel_stmt:
	  stmt
	{
		$$ = $1
	}
	| TransactionStmtLegacy
	{
		$$ = $1
	}
	;

stmt:
	  AlterEventTrigStmt
	{
		$$ = $1
	}
	| AlterCollationStmt
	{
		$$ = $1
	}
	| AlterDatabaseStmt
	{
		$$ = $1
	}
	| AlterDatabaseSetStmt
	{
		$$ = $1
	}
	| AlterDefaultPrivilegesStmt
	{
		$$ = $1
	}
	| AlterDomainStmt
	{
		$$ = $1
	}
	| AlterEnumStmt
	{
		$$ = $1
	}
	| AlterExtensionStmt
	{
		$$ = $1
	}
	| AlterExtensionContentsStmt
	{
		$$ = $1
	}
	| AlterFdwStmt
	{
		$$ = $1
	}
	| AlterForeignServerStmt
	{
		$$ = $1
	}
	| AlterFunctionStmt
	{
		$$ = $1
	}
	| AlterGroupStmt
	{
		$$ = $1
	}
	| AlterObjectDependsStmt
	{
		$$ = $1
	}
	| AlterObjectSchemaStmt
	{
		$$ = $1
	}
	| AlterOwnerStmt
	{
		$$ = $1
	}
	| AlterOperatorStmt
	{
		$$ = $1
	}
	| AlterTypeStmt
	{
		$$ = $1
	}
	| AlterPolicyStmt
	{
		$$ = $1
	}
	| AlterSeqStmt
	{
		$$ = $1
	}
	| AlterSystemStmt
	{
		$$ = $1
	}
	| AlterTableStmt
	{
		$$ = $1
	}
	| AlterTblSpcStmt
	{
		$$ = $1
	}
	| AlterCompositeTypeStmt
	{
		$$ = $1
	}
	| AlterPublicationStmt
	{
		$$ = $1
	}
	| AlterRoleSetStmt
	{
		$$ = $1
	}
	| AlterRoleStmt
	{
		$$ = $1
	}
	| AlterSubscriptionStmt
	{
		$$ = $1
	}
	| AlterStatsStmt
	{
		$$ = $1
	}
	| AlterTSConfigurationStmt
	{
		$$ = $1
	}
	| AlterTSDictionaryStmt
	{
		$$ = $1
	}
	| AlterUserMappingStmt
	{
		$$ = $1
	}
	| AnalyzeStmt
	{
		$$ = $1
	}
	| CallStmt
	{
		$$ = $1
	}
	| CheckPointStmt
	{
		$$ = $1
	}
	| ClosePortalStmt
	{
		$$ = $1
	}
	| ClusterStmt
	{
		$$ = $1
	}
	| CommentStmt
	{
		$$ = $1
	}
	| ConstraintsSetStmt
	{
		$$ = $1
	}
	| CopyStmt
	{
		$$ = $1
	}
	| CreateAmStmt
	{
		$$ = $1
	}
	| CreateAsStmt
	{
		$$ = $1
	}
	| CreateAssertionStmt
	{
		$$ = $1
	}
	| CreateCastStmt
	{
		$$ = $1
	}
	| CreateConversionStmt
	{
		$$ = $1
	}
	| CreateDomainStmt
	{
		$$ = $1
	}
	| CreateExtensionStmt
	{
		$$ = $1
	}
	| CreateFdwStmt
	{
		$$ = $1
	}
	| CreateForeignServerStmt
	{
		$$ = $1
	}
	| CreateForeignTableStmt
	{
		$$ = $1
	}
	| CreateFunctionStmt
	{
		$$ = $1
	}
	| CreateGroupStmt
	{
		$$ = $1
	}
	| CreateMatViewStmt
	{
		$$ = $1
	}
	| CreateOpClassStmt
	{
		$$ = $1
	}
	| CreateOpFamilyStmt
	{
		$$ = $1
	}
	| CreatePublicationStmt
	{
		$$ = $1
	}
	| AlterOpFamilyStmt
	{
		$$ = $1
	}
	| CreatePolicyStmt
	{
		$$ = $1
	}
	| CreatePLangStmt
	{
		$$ = $1
	}
	| CreateSchemaStmt
	{
		$$ = $1
	}
	| CreateSeqStmt
	{
		$$ = $1
	}
	| CreateStmt
	{
		$$ = $1
	}
	| CreateSubscriptionStmt
	{
		$$ = $1
	}
	| CreateStatsStmt
	{
		$$ = $1
	}
	| CreateTableSpaceStmt
	{
		$$ = $1
	}
	| CreateTransformStmt
	{
		$$ = $1
	}
	| CreateTrigStmt
	{
		$$ = $1
	}
	| CreateEventTrigStmt
	{
		$$ = $1
	}
	| CreateRoleStmt
	{
		$$ = $1
	}
	| CreateUserStmt
	{
		$$ = $1
	}
	| CreateUserMappingStmt
	{
		$$ = $1
	}
	| CreatedbStmt
	{
		$$ = $1
	}
	| DeallocateStmt
	{
		$$ = $1
	}
	| DeclareCursorStmt
	{
		$$ = $1
	}
	| DefineStmt
	{
		$$ = $1
	}
	| DeleteStmt
	{
		$$ = $1
	}
	| DiscardStmt
	{
		$$ = $1
	}
	| DoStmt
	{
		$$ = $1
	}
	| DropCastStmt
	{
		$$ = $1
	}
	| DropOpClassStmt
	{
		$$ = $1
	}
	| DropOpFamilyStmt
	{
		$$ = $1
	}
	| DropOwnedStmt
	{
		$$ = $1
	}
	| DropStmt
	{
		$$ = $1
	}
	| DropSubscriptionStmt
	{
		$$ = $1
	}
	| DropTableSpaceStmt
	{
		$$ = $1
	}
	| DropTransformStmt
	{
		$$ = $1
	}
	| DropRoleStmt
	{
		$$ = $1
	}
	| DropUserMappingStmt
	{
		$$ = $1
	}
	| DropdbStmt
	{
		$$ = $1
	}
	| ExecuteStmt
	{
		$$ = $1
	}
	| ExplainStmt
	{
		$$ = $1
	}
	| FetchStmt
	{
		$$ = $1
	}
	| GrantStmt
	{
		$$ = $1
	}
	| GrantRoleStmt
	{
		$$ = $1
	}
	| ImportForeignSchemaStmt
	{
		$$ = $1
	}
	| IndexStmt
	{
		$$ = $1
	}
	| InsertStmt
	{
		$$ = $1
	}
	| ListenStmt
	{
		$$ = $1
	}
	| RefreshMatViewStmt
	{
		$$ = $1
	}
	| LoadStmt
	{
		$$ = $1
	}
	| LockStmt
	{
		$$ = $1
	}
	| MergeStmt
	{
		$$ = $1
	}
	| NotifyStmt
	{
		$$ = $1
	}
	| PrepareStmt
	{
		$$ = $1
	}
	| ReassignOwnedStmt
	{
		$$ = $1
	}
	| ReindexStmt
	{
		$$ = $1
	}
	| RemoveAggrStmt
	{
		$$ = $1
	}
	| RemoveFuncStmt
	{
		$$ = $1
	}
	| RemoveOperStmt
	{
		$$ = $1
	}
	| RenameStmt
	{
		$$ = $1
	}
	| RevokeStmt
	{
		$$ = $1
	}
	| RevokeRoleStmt
	{
		$$ = $1
	}
	| RuleStmt
	{
		$$ = $1
	}
	| SecLabelStmt
	{
		$$ = $1
	}
	| SelectStmt
	{
		$$ = $1
	}
	| TransactionStmt
	{
		$$ = $1
	}
	| TruncateStmt
	{
		$$ = $1
	}
	| UnlistenStmt
	{
		$$ = $1
	}
	| UpdateStmt
	{
		$$ = $1
	}
	| VacuumStmt
	{
		$$ = $1
	}
	| VariableResetStmt
	{
		$$ = $1
	}
	| VariableSetStmt
	{
		$$ = $1
	}
	| VariableShowStmt
	{
		$$ = $1
	}
	| ViewStmt
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_single_name:
	  ColId
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_qualified_name:
	  any_name
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_concurrently:
	  CONCURRENTLY
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_drop_behavior:
	  CASCADE
	{
		$$ = $1
	}
	| RESTRICT
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

CallStmt:
	  CALL func_application
	{
		$$ = makeCallStmt($2)
	}
	;

CreateRoleStmt:
	  CREATE ROLE RoleId opt_with OptRoleList
	{
		$$ = makeCreateRoleStmt("ROLE", $3, $5)
	}
	;

opt_with:
	  WITH
	{
		$$ = $1
	}
	| WITH_LA
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptRoleList:
	  OptRoleList CreateOptRoleElem
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterOptRoleList:
	  AlterOptRoleList AlterOptRoleElem
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterOptRoleElem:
	  PASSWORD Sconst
	{
		$$ = makeDefElem("password", &String{Sval: toStr($2)})
		grammarSetLocation($$, @1)
	}
	| PASSWORD NULL_P
	{
		$$ = makeDefElem("password", nil)
		grammarSetLocation($$, @1)
	}
	| ENCRYPTED PASSWORD Sconst
	{
		$$ = makeDefElem("password", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| UNENCRYPTED PASSWORD Sconst
	{
		yylex.(*Lexer).errorAt("UNENCRYPTED PASSWORD is no longer supported", @1); return 1
	}
	| INHERIT
	{
		$$ = makeDefElem("inherit", &Boolean{Boolval: true})
		grammarSetLocation($$, @1)
	}
	| CONNECTION LIMIT SignedIconst
	{
		$$ = makeDefElem("connectionlimit", &IntegerVal{Ival: int32(int64(toIntDefault($3, 0)))})
		grammarSetLocation($$, @1)
	}
	| VALID UNTIL Sconst
	{
		$$ = makeDefElem("validUntil", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| USER role_list
	{
		$$ = makeDefElem("rolemembers", &List{Items: toNodeSlice($2)})
		grammarSetLocation($$, @1)
	}
	| IDENT
	{
		name := toStr($1)
		enabled := true
		if strings.HasPrefix(name, "no") { name = strings.TrimPrefix(name, "no"); enabled = false }
		switch name {
		case "superuser", "createrole", "createdb", "bypassrls", "inherit":
		case "replication": name = "isreplication"
		case "login": name = "canlogin"
		default: yylex.(*Lexer).errorAt("unrecognized role option", @1); return 1
		}
		$$ = makeDefElem(name, &Boolean{Boolval: enabled})
		grammarSetLocation($$, @1)
	}
	;

CreateOptRoleElem:
	  AlterOptRoleElem
	{
		$$ = $1
	}
	| SYSID Iconst
	{
		$$ = makeDefElem("sysid", &IntegerVal{Ival: int32(int64(toIntDefault($2, 0)))})
		grammarSetLocation($$, @1)
	}
	| ADMIN role_list
	{
		$$ = makeDefElem("adminmembers", &List{Items: toNodeSlice($2)})
		grammarSetLocation($$, @1)
	}
	| ROLE role_list
	{
		$$ = makeDefElem("rolemembers", &List{Items: toNodeSlice($2)})
		grammarSetLocation($$, @1)
	}
	| IN_P ROLE role_list
	{
		$$ = makeDefElem("addroleto", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	| IN_P GROUP_P role_list
	{
		$$ = makeDefElem("addroleto", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	;

CreateUserStmt:
	  CREATE USER RoleId opt_with OptRoleList
	{
		$$ = makeCreateRoleStmt("USER", $3, $5)
	}
	;

AlterRoleStmt:
	  ALTER ROLE RoleSpec opt_with AlterOptRoleList
	{
		$$ = makeAlterRoleStmt($3, $5, 1)
	}
	| ALTER USER RoleSpec opt_with AlterOptRoleList
	{
		$$ = makeAlterRoleStmt($3, $5, 1)
	}
	;

opt_in_database:
	  /* EMPTY */
	{
		$$ = nil
	}
	| IN_P DATABASE name
	{
		$$ = $3
	}
	;

AlterRoleSetStmt:
	  ALTER ROLE RoleSpec opt_in_database SetResetClause
	{
		$$ = &AlterRoleSetStmt{Role: toRoleSpec($3), Database: toStr($4), Setstmt: toVariableSetStmt($5)}
	}
	| ALTER ROLE ALL opt_in_database SetResetClause
	{
		$$ = &AlterRoleSetStmt{Role: nil, Database: toStr($4), Setstmt: toVariableSetStmt($5)}
	}
	| ALTER USER RoleSpec opt_in_database SetResetClause
	{
		$$ = &AlterRoleSetStmt{Role: toRoleSpec($3), Database: toStr($4), Setstmt: toVariableSetStmt($5)}
	}
	| ALTER USER ALL opt_in_database SetResetClause
	{
		$$ = &AlterRoleSetStmt{Role: nil, Database: toStr($4), Setstmt: toVariableSetStmt($5)}
	}
	;

DropRoleStmt:
	  DROP ROLE role_list
	{
		$$ = makeDropRoleStmt($3, false)
	}
	| DROP ROLE IF_P EXISTS role_list
	{
		$$ = makeDropRoleStmt($5, true)
	}
	| DROP USER role_list
	{
		$$ = makeDropRoleStmt($3, false)
	}
	| DROP USER IF_P EXISTS role_list
	{
		$$ = makeDropRoleStmt($5, true)
	}
	| DROP GROUP_P role_list
	{
		$$ = makeDropRoleStmt($3, false)
	}
	| DROP GROUP_P IF_P EXISTS role_list
	{
		$$ = makeDropRoleStmt($5, true)
	}
	;

CreateGroupStmt:
	  CREATE GROUP_P RoleId opt_with OptRoleList
	{
		$$ = makeCreateRoleStmt("GROUP", $3, $5)
	}
	;

AlterGroupStmt:
	  ALTER GROUP_P RoleSpec add_drop USER role_list
	{
		$$ = &AlterRoleStmt{Role: toRoleSpec($3), Action: int32(int(toInt32($4))), Options: toNodeSlice([]Node{toNode(&DefElem{Defname: "rolemembers", Arg: toNode(&List{Items: toNodeSlice($6)}), Location: int32(@6), Defaction: pg_query.DefElemAction(1)})})}
	}
	;

add_drop:
	  ADD_P
	{
		$$ = int32(1)
	}
	| DROP
	{
		$$ = int32(-1)
	}
	;

CreateSchemaStmt:
	  CREATE SCHEMA opt_single_name AUTHORIZATION RoleSpec OptSchemaEltList
	{
		$$ = &CreateSchemaStmt{Schemaname: toStr($3), Authrole: toRoleSpec($5), SchemaElts: toNodeSlice($6)}
	}
	| CREATE SCHEMA ColId OptSchemaEltList
	{
		$$ = &CreateSchemaStmt{Schemaname: toStr($3), SchemaElts: toNodeSlice($4)}
	}
	| CREATE SCHEMA IF_P NOT EXISTS opt_single_name AUTHORIZATION RoleSpec OptSchemaEltList
	{
		if len(toNodeSlice($9)) != 0 { yylex.(*Lexer).errorAt("CREATE SCHEMA IF NOT EXISTS cannot include schema elements", @9); return 1 }
		$$ = &CreateSchemaStmt{Schemaname: toStr($6), Authrole: toRoleSpec($8), IfNotExists: true}
	}
	| CREATE SCHEMA IF_P NOT EXISTS ColId OptSchemaEltList
	{
		if len(toNodeSlice($7)) != 0 { yylex.(*Lexer).errorAt("CREATE SCHEMA IF NOT EXISTS cannot include schema elements", @7); return 1 }
		$$ = &CreateSchemaStmt{Schemaname: toStr($6), IfNotExists: true}
	}
	;

OptSchemaEltList:
	  OptSchemaEltList schema_stmt
	{
		$$ = appendToNodeSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

schema_stmt:
	  CreateStmt
	{
		$$ = $1
	}
	| IndexStmt
	{
		$$ = $1
	}
	| CreateSeqStmt
	{
		$$ = $1
	}
	| CreateTrigStmt
	{
		$$ = $1
	}
	| GrantStmt
	{
		$$ = $1
	}
	| ViewStmt
	{
		$$ = $1
	}
	;

VariableSetStmt:
	  SET set_rest
	{
		st := toVariableSetStmt($2)
		if st != nil { st.IsLocal = false}
		$$ = st
	}
	| SET LOCAL set_rest
	{
		st := toVariableSetStmt($3)
		if st != nil { st.IsLocal = true}
		$$ = st
	}
	| SET SESSION set_rest
	{
		st := toVariableSetStmt($3)
		if st != nil { st.IsLocal = false}
		$$ = st
	}
	;

set_rest:
	  TRANSACTION transaction_mode_list
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("MULTI"), Name: "TRANSACTION", Args: toNodeSlice($2), JumbleArgs: true, Location: -1}
	}
	| SESSION CHARACTERISTICS AS TRANSACTION transaction_mode_list
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("MULTI"), Name: "SESSION CHARACTERISTICS", Args: toNodeSlice($5), JumbleArgs: true, Location: -1}
	}
	| set_rest_more
	{
		$$ = $1
	}
	;

generic_set:
	  var_name TO var_list
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: toStr($1), Args: toNodeSlice($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| var_name '=' var_list
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: toStr($1), Args: toNodeSlice($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| var_name TO DEFAULT
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("DEFAULT"), Name: toStr($1), Location: -1}
	}
	| var_name '=' DEFAULT
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("DEFAULT"), Name: toStr($1), Location: -1}
	}
	;

set_rest_more:
	  generic_set
	{
		$$ = $1
	}
	| var_name FROM CURRENT_P
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("CURRENT"), Name: toStr($1), Location: -1}
	}
	| TIME ZONE zone_value
	{
		st := &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: "timezone", Args: toNodeSlice($3), JumbleArgs: true, Location: -1}
		if $3 == nil { st.Kind = enumVariableSetKind("DEFAULT")}
		$$ = st
	}
	| CATALOG_P Sconst
	{
		yylex.(*Lexer).errorAt("current database cannot be changed", @2); return 1
	}
	| SCHEMA Sconst
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: "search_path", Args: []Node{toNode(grammarConst(toStr($2), @2))}, Location: -1}
		grammarSetLocation($$, @2)
	}
	| NAMES opt_encoding
	{
		st := &VariableSetStmt{Kind: enumVariableSetKind("DEFAULT"), Name: "client_encoding", Location: -1}
		if $2 != nil { st.Kind = enumVariableSetKind("SET"); st.Args = []Node{toNode(grammarConst(toStr($2), @2))}}
		$$ = st
		grammarSetLocation($$, @2)
	}
	| ROLE NonReservedWord_or_Sconst
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: "role", Args: []Node{toNode(grammarConst(toStr($2), @2))}, Location: -1}
		grammarSetLocation($$, @2)
	}
	| SESSION AUTHORIZATION NonReservedWord_or_Sconst
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: "session_authorization", Args: []Node{toNode(grammarConst(toStr($3), @3))}, Location: -1}
		grammarSetLocation($$, @3)
	}
	| SESSION AUTHORIZATION DEFAULT
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("DEFAULT"), Name: "session_authorization", Location: -1}
	}
	| XML_P OPTION document_or_content
	{
		option := "CONTENT"
		if toIntDefault($3, 0) == 1 { option = "DOCUMENT" }
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("SET"), Name: "xmloption", Args: []Node{toNode(grammarConst(option, @3))}, JumbleArgs: true, Location: -1}
		grammarSetLocation($$, -1)
	}
	| TRANSACTION SNAPSHOT Sconst
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("MULTI"), Name: "TRANSACTION SNAPSHOT", Args: []Node{toNode(grammarConst(toStr($3), @3))}, Location: -1}
		grammarSetLocation($$, @3)
	}
	;

var_name:
	  ColId
	{
		$$ = toStr($1)
	}
	| var_name '.' ColId
	{
		$$ = toStr($1) + "." + toStr($3)
	}
	;

var_list:
	  var_value
	{
		$$ = []Node{toNode($1)}
	}
	| var_list ',' var_value
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

var_value:
	  opt_boolean_or_string
	{
		$$ = makeAConst(toStr($1))
		grammarSetLocation($$, @1)
	}
	| NumericOnly
	{
		switch value := $1.(type) {
		case *IntegerVal: $$ = makeAConst(value.Ival)
		case *FloatVal: $$ = makeAConst(*value)
		default: $$ = makeAConst(value)
		}
		grammarSetLocation($$, @1)
	}
	;

iso_level:
	  READ UNCOMMITTED
	{
		$$ = "read uncommitted"
	}
	| READ COMMITTED
	{
		$$ = "read committed"
	}
	| REPEATABLE READ
	{
		$$ = "repeatable read"
	}
	| SERIALIZABLE
	{
		$$ = $1
	}
	;

opt_boolean_or_string:
	  TRUE_P
	{
		$$ = "true"
	}
	| FALSE_P
	{
		$$ = "false"
	}
	| ON
	{
		$$ = "on"
	}
	| NonReservedWord_or_Sconst
	{
		$$ = toStr($1)
	}
	;

zone_value:
	  Sconst
	{
		$$ = makeAConst(toStr($1))
		grammarSetLocation($$, @1)
	}
	| IDENT
	{
		$$ = makeAConst(toStr($1))
		grammarSetLocation($$, @1)
	}
	| ConstInterval Sconst opt_interval
	{
		t := toTypeNameNode($1)
		t.Typmods = toNodeSlice($3)
		if len(t.Typmods) > 0 {
			mask := toInt64(aConstValue(t.Typmods[0].GetAConst()))
			if mask & ^int64((1 << 10) | (1 << 11)) != 0 { yylex.(*Lexer).errorAt("time zone interval must be HOUR or HOUR TO MINUTE", @3); return 1 }
		}
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($2), @2)), TypeName: t, Location: -1}
	}
	| ConstInterval '(' Iconst ')' Sconst
	{
		t := toTypeNameNode($1)
		t.Typmods = []Node{toNode(makeAConst(int64(32767))), toNode(grammarConst(toInt64($3), @3))}
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($5), @5)), TypeName: t, Location: -1}
	}
	| NumericOnly
	{
		switch value := $1.(type) {
		case *IntegerVal: $$ = makeAConst(value.Ival)
		case *FloatVal: $$ = makeAConst(*value)
		}
		grammarSetLocation($$, @1)
	}
	| DEFAULT
	{
		$$ = nil
	}
	| LOCAL
	{
		$$ = nil
	}
	;

opt_encoding:
	  Sconst
	{
		$$ = $1
	}
	| DEFAULT
	{
		$$ = nil
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

NonReservedWord_or_Sconst:
	  NonReservedWord
	{
		$$ = $1
	}
	| Sconst
	{
		$$ = $1
	}
	;

VariableResetStmt:
	  RESET reset_rest
	{
		$$ = $2
	}
	;

reset_rest:
	  generic_reset
	{
		$$ = $1
	}
	| TIME ZONE
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("RESET"), Name: "timezone", Location: -1}
	}
	| TRANSACTION ISOLATION LEVEL
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("RESET"), Name: "transaction_isolation", Location: -1}
	}
	| SESSION AUTHORIZATION
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("RESET"), Name: "session_authorization", Location: -1}
	}
	;

generic_reset:
	  var_name
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("RESET"), Name: toStr($1), Location: -1}
	}
	| ALL
	{
		$$ = &VariableSetStmt{Kind: enumVariableSetKind("RESET_ALL"), Location: -1}
	}
	;

SetResetClause:
	  SET set_rest
	{
		$$ = $2
	}
	| VariableResetStmt
	{
		$$ = $1
	}
	;

FunctionSetResetClause:
	  SET set_rest_more
	{
		$$ = $2
	}
	| VariableResetStmt
	{
		$$ = $1
	}
	;

VariableShowStmt:
	  SHOW var_name
	{
		$$ = &VariableShowStmt{Name: toStr($2)}
	}
	| SHOW TIME ZONE
	{
		$$ = &VariableShowStmt{Name: "timezone"}
	}
	| SHOW TRANSACTION ISOLATION LEVEL
	{
		$$ = &VariableShowStmt{Name: "transaction_isolation"}
	}
	| SHOW SESSION AUTHORIZATION
	{
		$$ = &VariableShowStmt{Name: "session_authorization"}
	}
	| SHOW ALL
	{
		$$ = &VariableShowStmt{Name: "all"}
	}
	;

ConstraintsSetStmt:
	  SET CONSTRAINTS constraints_set_list constraints_set_mode
	{
		$$ = makeConstraintsSetStmt($3, toBool($4))
	}
	;

constraints_set_list:
	  ALL
	{
		$$ = nil
	}
	| qualified_name_list
	{
		$$ = $1
	}
	;

constraints_set_mode:
	  DEFERRED
	{
		$$ = true
	}
	| IMMEDIATE
	{
		$$ = false
	}
	;

CheckPointStmt:
	  CHECKPOINT
	{
		$$ = &CheckPointStmt{}
	}
	;

DiscardStmt:
	  DISCARD ALL
	{
		$$ = makeDiscardStmt("ALL")
	}
	| DISCARD TEMP
	{
		$$ = makeDiscardStmt("TEMP")
	}
	| DISCARD TEMPORARY
	{
		$$ = makeDiscardStmt("TEMP")
	}
	| DISCARD PLANS
	{
		$$ = makeDiscardStmt("PLANS")
	}
	| DISCARD SEQUENCES
	{
		$$ = makeDiscardStmt("SEQUENCES")
	}
	;

AlterTableStmt:
	  ALTER TABLE relation_expr alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("TABLE"), MissingOk: false}
	}
	| ALTER TABLE IF_P EXISTS relation_expr alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($5), Cmds: toNodeSlice(toAlterCmdSlice($6)), Objtype: enumObjectType("TABLE"), MissingOk: true}
	}
	| ALTER TABLE relation_expr partition_cmd
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("TABLE"), MissingOk: false}
	}
	| ALTER TABLE IF_P EXISTS relation_expr partition_cmd
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($5), Cmds: toNodeSlice(toAlterCmdSlice($6)), Objtype: enumObjectType("TABLE"), MissingOk: true}
	}
	| ALTER TABLE ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("TABLE"), OrigTablespacename: toStr($6), NewTablespacename: toStr($9), Roles: nil, Nowait: toBool($10)}
	}
	| ALTER TABLE ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("TABLE"), OrigTablespacename: toStr($6), NewTablespacename: toStr($12), Roles: toNodeSlice($9), Nowait: toBool($13)}
	}
	| ALTER INDEX qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("INDEX"), MissingOk: false}
	}
	| ALTER INDEX IF_P EXISTS qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($5), Cmds: toNodeSlice(toAlterCmdSlice($6)), Objtype: enumObjectType("INDEX"), MissingOk: true}
	}
	| ALTER INDEX qualified_name index_partition_cmd
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("INDEX"), MissingOk: false}
	}
	| ALTER INDEX ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("INDEX"), OrigTablespacename: toStr($6), NewTablespacename: toStr($9), Roles: nil, Nowait: toBool($10)}
	}
	| ALTER INDEX ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("INDEX"), OrigTablespacename: toStr($6), NewTablespacename: toStr($12), Roles: toNodeSlice($9), Nowait: toBool($13)}
	}
	| ALTER SEQUENCE qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("SEQUENCE"), MissingOk: false}
	}
	| ALTER SEQUENCE IF_P EXISTS qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($5), Cmds: toNodeSlice(toAlterCmdSlice($6)), Objtype: enumObjectType("SEQUENCE"), MissingOk: true}
	}
	| ALTER VIEW qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($3), Cmds: toNodeSlice(toAlterCmdSlice($4)), Objtype: enumObjectType("VIEW"), MissingOk: false}
	}
	| ALTER VIEW IF_P EXISTS qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($5), Cmds: toNodeSlice(toAlterCmdSlice($6)), Objtype: enumObjectType("VIEW"), MissingOk: true}
	}
	| ALTER MATERIALIZED VIEW qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($4), Cmds: toNodeSlice(toAlterCmdSlice($5)), Objtype: enumObjectType("MATVIEW"), MissingOk: false}
	}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($6), Cmds: toNodeSlice(toAlterCmdSlice($7)), Objtype: enumObjectType("MATVIEW"), MissingOk: true}
	}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE name SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("MATVIEW"), OrigTablespacename: toStr($7), NewTablespacename: toStr($10), Roles: nil, Nowait: toBool($11)}
	}
	| ALTER MATERIALIZED VIEW ALL IN_P TABLESPACE name OWNED BY role_list SET TABLESPACE name opt_nowait
	{
		$$ = &AlterTableMoveAllStmt{Objtype: enumObjectType("MATVIEW"), OrigTablespacename: toStr($7), NewTablespacename: toStr($13), Roles: toNodeSlice($10), Nowait: toBool($14)}
	}
	| ALTER FOREIGN TABLE relation_expr alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($4), Cmds: toNodeSlice(toAlterCmdSlice($5)), Objtype: enumObjectType("FOREIGN_TABLE"), MissingOk: false}
	}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr alter_table_cmds
	{
		$$ = &AlterTableStmt{Relation: toRangeVar($6), Cmds: toNodeSlice(toAlterCmdSlice($7)), Objtype: enumObjectType("FOREIGN_TABLE"), MissingOk: true}
	}
	;

alter_table_cmds:
	  alter_table_cmd
	{
		$$ = []Node{toNode(toAlterCmd($1))}
	}
	| alter_table_cmds ',' alter_table_cmd
	{
		$$ = appendToAlterCmdSlice($1, $3)
	}
	;

partition_cmd:
	  ATTACH PARTITION qualified_name PartitionBoundSpec
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AttachPartition"), Def: toNode(&PartitionCmd{Name: toRangeVar($3), Bound: $4.(*PartitionBoundSpec)}), Behavior: pg_query.DropBehavior(1)}
	}
	| DETACH PARTITION qualified_name opt_concurrently
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DetachPartition"), Def: toNode(&PartitionCmd{Name: toRangeVar($3), Concurrent: $4 != nil}), Behavior: pg_query.DropBehavior(1)}
	}
	| DETACH PARTITION qualified_name FINALIZE
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DetachPartitionFinalize"), Def: toNode(&PartitionCmd{Name: toRangeVar($3)}), Behavior: pg_query.DropBehavior(1)}
	}
	;

index_partition_cmd:
	  ATTACH PARTITION qualified_name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AttachPartition"), Def: toNode(&PartitionCmd{Name: toRangeVar($3)}), Behavior: pg_query.DropBehavior(1)}
	}
	;

alter_table_cmd:
	  ADD_P columnDef
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddColumn"), Def: toNode($2), MissingOk: false, Behavior: pg_query.DropBehavior(1)}
	}
	| ADD_P IF_P NOT EXISTS columnDef
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddColumn"), Def: toNode($5), MissingOk: true, Behavior: pg_query.DropBehavior(1)}
	}
	| ADD_P COLUMN columnDef
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddColumn"), Def: toNode($3), MissingOk: false, Behavior: pg_query.DropBehavior(1)}
	}
	| ADD_P COLUMN IF_P NOT EXISTS columnDef
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddColumn"), Def: toNode($6), MissingOk: true, Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId alter_column_default
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ColumnDefault"), Name: toStr($3), Def: toNode($4), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId DROP NOT NULL_P
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropNotNull"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET NOT NULL_P
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetNotNull"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET EXPRESSION AS '(' a_expr ')'
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetExpression"), Name: toStr($3), Def: toNode($8), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId DROP EXPRESSION
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropExpression"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId DROP EXPRESSION IF_P EXISTS
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropExpression"), Name: toStr($3), MissingOk: true, Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET STATISTICS set_statistics_value
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetStatistics"), Name: toStr($3), Def: toNode($6), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column Iconst SET STATISTICS set_statistics_value
	{
		if toInt($3) <= 0 || toInt($3) > 32767 { yylex.(*Lexer).errorAt("column number must be in range from 1 to 32767", @3); return 1 }
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetStatistics"), Num: int32(toInt($3)), Def: toNode($6), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET reloptions
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetOptions"), Name: toStr($3), Def: toNode(&List{Items: toNodeSlice($5)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId RESET reloptions
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ResetOptions"), Name: toStr($3), Def: toNode(&List{Items: toNodeSlice($5)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET column_storage
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetStorage"), Name: toStr($3), Def: toNode(&String{Sval: toStr($5)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId SET column_compression
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetCompression"), Name: toStr($3), Def: toNode(&String{Sval: toStr($5)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId ADD_P GENERATED generated_when AS IDENTITY_P OptParenthesizedSeqOptList
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddIdentity"), Name: toStr($3), Def: toNode(&Constraint{Contype: pg_query.ConstrType(ConstrIdentity), GeneratedWhen: toStr($6), Options: toNodeSlice(toDefElemSlice($9)), Location: int32(@5)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId alter_identity_column_option_list
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetIdentity"), Name: toStr($3), Def: toNode(&List{Items: toNodeSlice($4)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId DROP IDENTITY_P
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropIdentity"), Name: toStr($3), MissingOk: false, Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER opt_column ColId DROP IDENTITY_P IF_P EXISTS
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropIdentity"), Name: toStr($3), MissingOk: true, Behavior: pg_query.DropBehavior(1)}
	}
	| DROP opt_column IF_P EXISTS ColId opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropColumn"), Name: toStr($5), Behavior: toPgDropBehavior(toBool($6)), MissingOk: true}
	}
	| DROP opt_column ColId opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropColumn"), Name: toStr($3), Behavior: toPgDropBehavior(toBool($4)), MissingOk: false}
	}
	| ALTER opt_column ColId opt_set_data TYPE_P Typename opt_collate_clause alter_using
	{
		coll, _ := $7.(*CollateClause)
		def := &ColumnDef{TypeName: toTypeNameNode($6), CollClause: coll, RawDefault: toNode($8), Location: int32(@3)}
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AlterColumnType"), Name: toStr($3), Def: toNode(def), Behavior: pg_query.DropBehavior(1)}
		grammarSetLocation($$, @3)
	}
	| ALTER opt_column ColId alter_generic_options
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AlterColumnGenericOptions"), Name: toStr($3), Def: toNode(&List{Items: toNodeSlice($4)}), Behavior: pg_query.DropBehavior(1)}
	}
	| ADD_P TableConstraint
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddConstraint"), Def: toNode($2), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER CONSTRAINT name ConstraintAttributeSpec
	{
		a := toConstrAttr($4)
		if a.seen["NOT VALID"] { yylex.(*Lexer).errorAt("constraints cannot be altered to be NOT VALID", @4); return 1 }
		c := &ATAlterConstraint{Conname: toStr($3), AlterEnforceability: a.seen["ENFORCED"] || a.seen["NOT ENFORCED"], IsEnforced: !a.notEnforced, AlterDeferrability: a.seen["DEFERRABLE"] || a.seen["NOT DEFERRABLE"] || a.seen["INITIALLY DEFERRED"] || a.seen["INITIALLY IMMEDIATE"], Deferrable: a.deferrable, Initdeferred: a.initdeferred, AlterInheritability: a.noInherit, Noinherit: a.noInherit}
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AlterConstraint"), Def: toNode(c), Behavior: pg_query.DropBehavior(1)}
	}
	| ALTER CONSTRAINT name INHERIT
	{
		c := &ATAlterConstraint{Conname: toStr($3), AlterInheritability: true}
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AlterConstraint"), Def: toNode(c), Behavior: pg_query.DropBehavior(1)}
	}
	| VALIDATE CONSTRAINT name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ValidateConstraint"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| DROP CONSTRAINT IF_P EXISTS name opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropConstraint"), Name: toStr($5), Behavior: toPgDropBehavior(toBool($6)), MissingOk: true}
	}
	| DROP CONSTRAINT name opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropConstraint"), Name: toStr($3), Behavior: toPgDropBehavior(toBool($4)), MissingOk: false}
	}
	| SET WITHOUT OIDS
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropOids"), Behavior: pg_query.DropBehavior(1)}
	}
	| CLUSTER ON name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ClusterOn"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| SET WITHOUT CLUSTER
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropCluster"), Behavior: pg_query.DropBehavior(1)}
	}
	| SET LOGGED
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetLogged"), Behavior: pg_query.DropBehavior(1)}
	}
	| SET UNLOGGED
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetUnLogged"), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P TRIGGER name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableTrig"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P ALWAYS TRIGGER name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableAlwaysTrig"), Name: toStr($4), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P REPLICA TRIGGER name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableReplicaTrig"), Name: toStr($4), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P TRIGGER ALL
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableTrigAll"), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P TRIGGER USER
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableTrigUser"), Behavior: pg_query.DropBehavior(1)}
	}
	| DISABLE_P TRIGGER name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DisableTrig"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| DISABLE_P TRIGGER ALL
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DisableTrigAll"), Behavior: pg_query.DropBehavior(1)}
	}
	| DISABLE_P TRIGGER USER
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DisableTrigUser"), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P RULE name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableRule"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P ALWAYS RULE name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableAlwaysRule"), Name: toStr($4), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P REPLICA RULE name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableReplicaRule"), Name: toStr($4), Behavior: pg_query.DropBehavior(1)}
	}
	| DISABLE_P RULE name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DisableRule"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| INHERIT qualified_name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddInherit"), Def: toNode($2), Behavior: pg_query.DropBehavior(1)}
	}
	| NO INHERIT qualified_name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropInherit"), Def: toNode($3), Behavior: pg_query.DropBehavior(1)}
	}
	| OF any_name
	{
		typ := toTypeNameNode($2)
		typ.Location = int32(@2)
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddOf"), Def: toNode(typ), Behavior: pg_query.DropBehavior(1)}
	}
	| NOT OF
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropOf"), Behavior: pg_query.DropBehavior(1)}
	}
	| OWNER TO RoleSpec
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ChangeOwner"), Newowner: toRoleSpec($3), Behavior: pg_query.DropBehavior(1)}
	}
	| SET ACCESS METHOD set_access_method_name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetAccessMethod"), Name: toStr($4), Behavior: pg_query.DropBehavior(1)}
	}
	| SET TABLESPACE name
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetTableSpace"), Name: toStr($3), Behavior: pg_query.DropBehavior(1)}
	}
	| SET reloptions
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_SetRelOptions"), Def: toNode(&List{Items: toNodeSlice($2)}), Behavior: pg_query.DropBehavior(1)}
	}
	| RESET reloptions
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ResetRelOptions"), Def: toNode(&List{Items: toNodeSlice($2)}), Behavior: pg_query.DropBehavior(1)}
	}
	| REPLICA IDENTITY_P replica_identity
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ReplicaIdentity"), Def: toNode($3), Behavior: pg_query.DropBehavior(1)}
	}
	| ENABLE_P ROW LEVEL SECURITY
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_EnableRowSecurity"), Behavior: pg_query.DropBehavior(1)}
	}
	| DISABLE_P ROW LEVEL SECURITY
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DisableRowSecurity"), Behavior: pg_query.DropBehavior(1)}
	}
	| FORCE ROW LEVEL SECURITY
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_ForceRowSecurity"), Behavior: pg_query.DropBehavior(1)}
	}
	| NO FORCE ROW LEVEL SECURITY
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_NoForceRowSecurity"), Behavior: pg_query.DropBehavior(1)}
	}
	| alter_generic_options
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_GenericOptions"), Def: toNode(&List{Items: toNodeSlice($1)}), Behavior: pg_query.DropBehavior(1)}
	}
	;

alter_column_default:
	  SET DEFAULT a_expr
	{
		$$ = $3
	}
	| DROP DEFAULT
	{
		$$ = nil
	}
	;

opt_collate_clause:
	  COLLATE any_name
	{
		$$ = &CollateClause{Collname: toPgStringNodes(toStrSlice($2)), Location: -1}
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

alter_using:
	  USING a_expr
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

replica_identity:
	  NOTHING
	{
		$$ = &ReplicaIdentityStmt{IdentityType: "n"}
	}
	| FULL
	{
		$$ = &ReplicaIdentityStmt{IdentityType: "f"}
	}
	| DEFAULT
	{
		$$ = &ReplicaIdentityStmt{IdentityType: "d"}
	}
	| USING INDEX name
	{
		$$ = &ReplicaIdentityStmt{IdentityType: "i", Name: toStr($3)}
	}
	;

reloptions:
	  '(' reloption_list ')'
	{
		$$ = $2
	}
	;

opt_reloptions:
	  WITH reloptions
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

reloption_list:
	  reloption_elem
	{
		$$ = []Node{toNode($1.(*DefElem))}
	}
	| reloption_list ',' reloption_elem
	{
		$$ = append(toDefElemSlice($1), toNode($3.(*DefElem)))
	}
	;

reloption_elem:
	  ColLabel '=' def_arg
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode($3), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel
	{
		$$ = &DefElem{Defname: toStr($1), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel '.' ColLabel '=' def_arg
	{
		$$ = &DefElem{Defname: toStr($3), Defnamespace: toStr($1), Arg: toNode($5), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel '.' ColLabel
	{
		$$ = &DefElem{Defname: toStr($3), Defnamespace: toStr($1), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

alter_identity_column_option_list:
	  alter_identity_column_option
	{
		$$ = []Node{toNode($1)}
	}
	| alter_identity_column_option_list alter_identity_column_option
	{
		$$ = appendToNodeSlice($1, $2)
	}
	;

alter_identity_column_option:
	  RESTART
	{
		$$ = &DefElem{Defname: "restart", Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| RESTART opt_with NumericOnly
	{
		$$ = &DefElem{Defname: "restart", Arg: toNode($3), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| SET SeqOptElem
	{
		$$ = $2
	}
	| SET GENERATED generated_when
	{
		$$ = &DefElem{Defname: "generated", Arg: toNode(&IntegerVal{Ival: int32(int64(toStr($3)[0]))}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

set_statistics_value:
	  SignedIconst
	{
		$$ = &IntegerVal{Ival: int32(toInt64($1))}
	}
	| DEFAULT
	{
		$$ = nil
	}
	;

set_access_method_name:
	  ColId
	{
		$$ = $1
	}
	| DEFAULT
	{
		$$ = nil
	}
	;

PartitionBoundSpec:
	  FOR VALUES WITH '(' hash_partbound ')'
	{
		n := &PartitionBoundSpec{Strategy: "h", Modulus: -1, Remainder: -1, Location: -1}
		for _, opt := range toDefElemSlice($5) {
		 value := int32(opt.GetDefElem().Arg.GetInteger().Ival)
		 switch opt.GetDefElem().Defname {
		 case "modulus": if n.Modulus != -1 { yylex.(*Lexer).errorAt("modulus specified more than once", @3); return 1 }; n.Modulus = value
		 case "remainder": if n.Remainder != -1 { yylex.(*Lexer).errorAt("remainder specified more than once", @3); return 1 }; n.Remainder = value
		 default: yylex.(*Lexer).errorAt("unrecognized hash partition bound specification", @3); return 1
		 }
		}
		if n.Modulus == -1 || n.Remainder == -1 { yylex.(*Lexer).errorAt("modulus and remainder must be specified", @3); return 1 }
		$$ = n
		grammarSetLocation($$, @3)
	}
	| FOR VALUES IN_P '(' expr_list ')'
	{
		$$ = &PartitionBoundSpec{Strategy: "l", Listdatums: toNodeSlice($5), Location: -1}
		grammarSetLocation($$, @3)
	}
	| FOR VALUES FROM '(' expr_list ')' TO '(' expr_list ')'
	{
		$$ = &PartitionBoundSpec{Strategy: "r", Lowerdatums: toNodeSlice($5), Upperdatums: toNodeSlice($9), Location: -1}
		grammarSetLocation($$, @3)
	}
	| DEFAULT
	{
		$$ = &PartitionBoundSpec{IsDefault: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

hash_partbound_elem:
	  NonReservedWord Iconst
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode(&IntegerVal{Ival: int32(toInt64($2))}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

hash_partbound:
	  hash_partbound_elem
	{
		$$ = []Node{toNode($1.(*DefElem))}
	}
	| hash_partbound ',' hash_partbound_elem
	{
		$$ = append(toDefElemSlice($1), toNode($3.(*DefElem)))
	}
	;

AlterCompositeTypeStmt:
	  ALTER TYPE_P any_name alter_type_cmds
	{
		relation := toRangeVar($3)
		relation.Inh = false
		relation.Location = int32(@3)
		$$ = &AlterTableStmt{Relation: relation, Cmds: toNodeSlice(toAlterTableCmdSlice($4)), Objtype: enumObjectType("OBJECT_TYPE")}
	}
	;

alter_type_cmds:
	  alter_type_cmd
	{
		$$ = toAlterTableCmdSlice($1)
	}
	| alter_type_cmds ',' alter_type_cmd
	{
		$$ = append(toAlterTableCmdSlice($1), toAlterTableCmdSlice($3)...)
	}
	;

alter_type_cmd:
	  ADD_P ATTRIBUTE TableFuncElement opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AddColumn"), Def: toNode(toNode($3)), Behavior: enumDropBehavior(toStr($4))}
	}
	| DROP ATTRIBUTE IF_P EXISTS ColId opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropColumn"), Name: toStr($5), Behavior: enumDropBehavior(toStr($6)), MissingOk: true}
	}
	| DROP ATTRIBUTE ColId opt_drop_behavior
	{
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_DropColumn"), Name: toStr($3), Behavior: enumDropBehavior(toStr($4)), MissingOk: false}
	}
	| ALTER ATTRIBUTE ColId opt_set_data TYPE_P Typename opt_collate_clause opt_drop_behavior
	{
		cd := &ColumnDef{TypeName: toTypeNameNode($6), CollClause: toCollateClause($7), Location: int32(@3)}
		$$ = &AlterTableCmd{Subtype: enumAlterTableType("AT_AlterColumnType"), Name: toStr($3), Def: toNode(cd), Behavior: enumDropBehavior(toStr($8))}
		grammarSetLocation($$, @3)
	}
	;

ClosePortalStmt:
	  CLOSE cursor_name
	{
		$$ = makeClosePortalStmt($2)
	}
	| CLOSE ALL
	{
		$$ = makeClosePortalStmt(nil)
	}
	;

CopyStmt:
	  COPY opt_binary qualified_name opt_column_list
			copy_from opt_program copy_file_name copy_delimiter opt_with
			copy_options where_clause
	{
		if !toBool($5) && $11 != nil { yylex.Error("WHERE clause not allowed with COPY TO"); return 1 }
		if toBool($6) && $7 == nil { yylex.Error("COPY PROGRAM requires a filename"); return 1 }
		opts := append(toDefElemSlice($2), toDefElemSlice($8)...)
		opts = append(opts, toDefElemSlice($10)...)
		$$ = makeCopyStmt($3, nil, $4, toBool($5), toBool($6), $7, opts, $11)
	}
	| COPY '(' PreparableStmt ')' TO opt_program copy_file_name opt_with copy_options
	{
		$$ = makeCopyStmt(nil, $3, nil, false, toBool($6), $7, $9, nil)
	}
	;

copy_from:
	  FROM
	{
		$$ = true
	}
	| TO
	{
		$$ = false
	}
	;

opt_program:
	  PROGRAM
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

copy_file_name:
	  Sconst
	{
		$$ = toStr($1)
	}
	| STDIN
	{
		$$ = nil
	}
	| STDOUT
	{
		$$ = nil
	}
	;

copy_options:
	  copy_opt_list
	{
		$$ = $1
	}
	| '(' copy_generic_opt_list ')'
	{
		$$ = $2
	}
	;

copy_opt_list:
	  copy_opt_list copy_opt_item
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

copy_opt_item:
	  BINARY
	{
		$$ = makeDefElem("format", &String{Sval: "binary"})
		grammarSetLocation($$, @1)
	}
	| FREEZE
	{
		$$ = makeDefElem("freeze", &Boolean{Boolval: true})
		grammarSetLocation($$, @1)
	}
	| DELIMITER opt_as Sconst
	{
		$$ = makeDefElem("delimiter", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| NULL_P opt_as Sconst
	{
		$$ = makeDefElem("null", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| CSV
	{
		$$ = makeDefElem("format", &String{Sval: "csv"})
		grammarSetLocation($$, @1)
	}
	| HEADER_P
	{
		$$ = makeDefElem("header", &Boolean{Boolval: true})
		grammarSetLocation($$, @1)
	}
	| QUOTE opt_as Sconst
	{
		$$ = makeDefElem("quote", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| ESCAPE opt_as Sconst
	{
		$$ = makeDefElem("escape", &String{Sval: toStr($3)})
		grammarSetLocation($$, @1)
	}
	| FORCE QUOTE columnList
	{
		$$ = makeDefElem("force_quote", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	| FORCE QUOTE '*'
	{
		$$ = makeDefElem("force_quote", &A_Star{})
		grammarSetLocation($$, @1)
	}
	| FORCE NOT NULL_P columnList
	{
		$$ = makeDefElem("force_not_null", &List{Items: toNodeSlice($4)})
		grammarSetLocation($$, @1)
	}
	| FORCE NOT NULL_P '*'
	{
		$$ = makeDefElem("force_not_null", &A_Star{})
		grammarSetLocation($$, @1)
	}
	| FORCE NULL_P columnList
	{
		$$ = makeDefElem("force_null", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	| FORCE NULL_P '*'
	{
		$$ = makeDefElem("force_null", &A_Star{})
		grammarSetLocation($$, @1)
	}
	| ENCODING Sconst
	{
		$$ = makeDefElem("encoding", &String{Sval: toStr($2)})
		grammarSetLocation($$, @1)
	}
	;

opt_binary:
	  BINARY
	{
		$$ = makeDefElem("format", &String{Sval: "binary"})
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

copy_delimiter:
	  opt_using DELIMITERS Sconst
	{
		$$ = makeDefElem("delimiter", &String{Sval: toStr($3)})
		grammarSetLocation($$, @2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_using:
	  USING
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

copy_generic_opt_list:
	  copy_generic_opt_elem
	{
		$$ = $1
	}
	| copy_generic_opt_list ',' copy_generic_opt_elem
	{
		$$ = appendToDefElemSlice($1, $3)
	}
	;

copy_generic_opt_elem:
	  ColLabel copy_generic_opt_arg
	{
		$$ = makeDefElem($1, $2)
		grammarSetLocation($$, @1)
	}
	;

copy_generic_opt_arg:
	  opt_boolean_or_string
	{
		$$ = &String{Sval: toStr($1)}
	}
	| NumericOnly
	{
		$$ = $1
	}
	| '*'
	{
		$$ = &A_Star{}
	}
	| DEFAULT
	{
		$$ = &String{Sval: "default"}
	}
	| '(' copy_generic_opt_arg_list ')'
	{
		$$ = &List{Items: toNodeSlice($2)}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

copy_generic_opt_arg_list:
	  copy_generic_opt_arg_list_item
	{
		$$ = $1
	}
	| copy_generic_opt_arg_list ',' copy_generic_opt_arg_list_item
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

copy_generic_opt_arg_list_item:
	  opt_boolean_or_string
	{
		$$ = &String{Sval: toStr($1)}
	}
	;

CreateStmt:
	  CREATE OptTemp TABLE qualified_name '(' OptTableElementList ')' 			OptInherit OptPartitionSpec table_access_method_clause OptWith 			OnCommitOption OptTableSpace
	{
		$$ = grammarTableOptions(grammarCreateTable($4, $6, false), $2, $8, $9, $10, $11, $12, $13)
	}
	| CREATE OptTemp TABLE IF_P NOT EXISTS qualified_name '(' 			OptTableElementList ')' OptInherit OptPartitionSpec table_access_method_clause 			OptWith OnCommitOption OptTableSpace
	{
		$$ = grammarTableOptions(grammarCreateTable($7, $9, true), $2, $11, $12, $13, $14, $15, $16)
	}
	| CREATE OptTemp TABLE qualified_name OF any_name 			OptTypedTableElementList OptPartitionSpec table_access_method_clause 			OptWith OnCommitOption OptTableSpace
	{
		n := grammarTableOptions(grammarCreateTable($4, $7, false), $2, nil, $8, $9, $10, $11, $12)
		n.OfTypename = toTypeNameNode($6)
		n.OfTypename.Location = int32(@6)
		$$ = n
		grammarSetLocation($$, @6)
	}
	| CREATE OptTemp TABLE IF_P NOT EXISTS qualified_name OF any_name 			OptTypedTableElementList OptPartitionSpec table_access_method_clause 			OptWith OnCommitOption OptTableSpace
	{
		n := grammarTableOptions(grammarCreateTable($7, $10, true), $2, nil, $11, $12, $13, $14, $15)
		n.OfTypename = toTypeNameNode($9)
		n.OfTypename.Location = int32(@9)
		$$ = n
		grammarSetLocation($$, @9)
	}
	| CREATE OptTemp TABLE qualified_name PARTITION OF qualified_name 			OptTypedTableElementList PartitionBoundSpec OptPartitionSpec 			table_access_method_clause OptWith OnCommitOption OptTableSpace
	{
		n := grammarTableOptions(grammarCreateTable($4, $8, false), $2, []Node{toNode(toRangeVar($7))}, $10, $11, $12, $13, $14)
		n.Partbound, _ = $9.(*PartitionBoundSpec)
		$$ = n
	}
	| CREATE OptTemp TABLE IF_P NOT EXISTS qualified_name PARTITION OF 			qualified_name OptTypedTableElementList PartitionBoundSpec OptPartitionSpec 			table_access_method_clause OptWith OnCommitOption OptTableSpace
	{
		n := grammarTableOptions(grammarCreateTable($7, $11, true), $2, []Node{toNode(toRangeVar($10))}, $13, $14, $15, $16, $17)
		n.Partbound, _ = $12.(*PartitionBoundSpec)
		$$ = n
	}
	;

OptTemp:
	  TEMPORARY
	{
		$$ = $1
	}
	| TEMP
	{
		$$ = $1
	}
| LOCAL TEMPORARY
{
	$$ = $2
}
| LOCAL TEMP
{
	$$ = $2
}
| GLOBAL TEMPORARY
{
	$$ = $2
}
| GLOBAL TEMP
{
	$$ = $2
}
	| UNLOGGED
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptTableElementList:
	  TableElementList
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptTypedTableElementList:
	  '(' TypedTableElementList ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

TableElementList:
	  TableElement
	{
		$$ = []Node{toNode($1)}
	}
	| TableElementList ',' TableElement
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

TypedTableElementList:
	  TypedTableElement
	{
		$$ = []Node{toNode($1)}
	}
	| TypedTableElementList ',' TypedTableElement
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

TableElement:
	  columnDef
	{
		$$ = $1
	}
	| TableLikeClause
	{
		$$ = $1
	}
	| TableConstraint
	{
		$$ = $1
	}
	;

TypedTableElement:
	  columnOptions
	{
		$$ = $1
	}
	| TableConstraint
	{
		$$ = $1
	}
	;

columnDef:
	  ColId Typename opt_column_storage opt_column_compression create_generic_options ColQualList
	{
		$$ = grammarColumn($1, $2, $3, $4, $5, $6)
		grammarSetLocation($$, @1)
	}
	;

columnOptions:
	  ColId ColQualList
	{
		$$ = grammarColumn($1, nil, nil, nil, nil, $2)
		grammarSetLocation($$, @1)
	}
	| ColId WITH OPTIONS ColQualList
	{
		$$ = grammarColumn($1, nil, nil, nil, nil, $4)
		grammarSetLocation($$, @1)
	}
	;

column_compression:
	  COMPRESSION ColId
	{
		$$ = $2
	}
	| COMPRESSION DEFAULT
	{
		$$ = "default"
	}
	;

opt_column_compression:
	  column_compression
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

column_storage:
	  STORAGE ColId
	{
		$$ = $2
	}
	| STORAGE DEFAULT
	{
		$$ = "default"
	}
	;

opt_column_storage:
	  column_storage
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ColQualList:
	  ColQualList ColConstraint
	{
		if _, collate := $2.(*CollateClause); collate {
		for _, q := range toNodeSlice($1) { if q.GetCollateClause() != nil { yylex.Error("multiple COLLATE clauses not allowed"); return 1 } }
		}
		$$ = appendToNodeSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ColConstraint:
	  CONSTRAINT name ColConstraintElem
	{
		n := toConstraint($3)
		n.Conname = toStr($2)
		$$ = n
		grammarSetLocation($$, @1)
	}
	| ColConstraintElem
	{
		$$ = $1
	}
	| ConstraintAttr
	{
		$$ = $1
	}
	| COLLATE any_name
	{
		$$ = &CollateClause{Collname: toPgStringNodes(toStrSlice($2)), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

ColConstraintElem:
	  NOT NULL_P opt_no_inherit
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrNotNull), IsNoInherit: toBool($3), InitiallyValid: true, IsEnforced: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| NULL_P
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrNull), Location: -1}
		grammarSetLocation($$, @1)
	}
	| UNIQUE opt_unique_null_treatment opt_definition OptConsTableSpace
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrUnique), NullsNotDistinct: toBool($2), Options: toNodeSlice(toDefElemSlice($3)), Indexspace: toStr($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	| PRIMARY KEY opt_definition OptConsTableSpace
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrPrimaryKey), Options: toNodeSlice(toDefElemSlice($3)), Indexspace: toStr($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	| CHECK '(' a_expr ')' opt_no_inherit
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrCheck), RawExpr: toNode($3), IsNoInherit: toBool($5), InitiallyValid: true, IsEnforced: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| DEFAULT b_expr
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrDefault), RawExpr: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| GENERATED generated_when AS IDENTITY_P OptParenthesizedSeqOptList
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrIdentity), GeneratedWhen: toStr($2), Options: toNodeSlice(toDefElemSlice($5)), Location: -1}
		grammarSetLocation($$, @1)
	}
	| GENERATED generated_when AS '(' a_expr ')' opt_virtual_or_stored
	{
		if toStr($2) != "a" { yylex.(*Lexer).errorAt("for a generated column, GENERATED ALWAYS must be specified", @2); return 1 }
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrGenerated), GeneratedWhen: toStr($2), RawExpr: toNode($5), GeneratedKind: toStr($7), Location: -1}
		grammarSetLocation($$, @1)
	}
	| REFERENCES qualified_name opt_column_list key_match key_actions
	{
		$$ = grammarForeignKey($2, $3, $4, $5)
		grammarSetLocation($$, @1)
	}
	;

opt_unique_null_treatment:
	  NULLS_P DISTINCT
	{
		$$ = false
	}
	| NULLS_P NOT DISTINCT
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

generated_when:
	  ALWAYS
	{
		$$ = "a"
	}
	| BY DEFAULT
	{
		$$ = "d"
	}
	;

opt_virtual_or_stored:
	  STORED
	{
		$$ = "s"
	}
	| VIRTUAL
	{
		$$ = "v"
	}
	| /* EMPTY */
	{
		$$ = "v"
	}
	;

ConstraintAttr:
	  DEFERRABLE
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrDeferrable), Location: -1}
		grammarSetLocation($$, @1)
	}
	| NOT DEFERRABLE
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrNotDeferrable), Location: -1}
		grammarSetLocation($$, @1)
	}
	| INITIALLY DEFERRED
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrDeferred), Location: -1}
		grammarSetLocation($$, @1)
	}
	| INITIALLY IMMEDIATE
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrImmediate), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ENFORCED
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrEnforced), Location: -1}
		grammarSetLocation($$, @1)
	}
	| NOT ENFORCED
	{
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrAttrNotEnforced), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

TableLikeClause:
	  LIKE qualified_name TableLikeOptionList
	{
		$$ = &TableLikeClause{Relation: toRangeVar($2), Options: uint32(toInt64($3))}
	}
	;

TableLikeOptionList:
	  TableLikeOptionList INCLUDING TableLikeOption
	{
		$$ = toInt64($1) | toInt64($3)
	}
	| TableLikeOptionList EXCLUDING TableLikeOption
	{
		$$ = toInt64($1) &^ toInt64($3)
	}
	| /* EMPTY */
	{
		$$ = int64(0)
	}
	;

TableLikeOption:
	  COMMENTS
	{
		$$ = int64(1)
	}
	| COMPRESSION
	{
		$$ = int64(2)
	}
	| CONSTRAINTS
	{
		$$ = int64(4)
	}
	| DEFAULTS
	{
		$$ = int64(8)
	}
	| IDENTITY_P
	{
		$$ = int64(32)
	}
	| GENERATED
	{
		$$ = int64(16)
	}
	| INDEXES
	{
		$$ = int64(64)
	}
	| STATISTICS
	{
		$$ = int64(128)
	}
	| STORAGE
	{
		$$ = int64(256)
	}
	| ALL
	{
		$$ = int64(2147483647)
	}
	;

TableConstraint:
	  CONSTRAINT name ConstraintElem
	{
		n := toConstraint($3)
		n.Conname = toStr($2)
		$$ = n
		grammarSetLocation($$, @1)
	}
	| ConstraintElem
	{
		$$ = $1
	}
	;

ConstraintElem:
	  CHECK '(' a_expr ')' ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrCheck, $5); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrCheck), RawExpr: toNode($3), Location: -1}, $5)
		grammarSetLocation($$, @1)
	}
	| NOT NULL_P ColId ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrNotNull, $4); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrNotNull), Keys: toPgStringNodes([]string{toStr($3)}), Location: -1}, $4)
		grammarSetLocation($$, @1)
	}
	| UNIQUE opt_unique_null_treatment '(' columnList opt_without_overlaps ')' opt_c_include opt_definition OptConsTableSpace 				ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrUnique, $10); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrUnique), NullsNotDistinct: toBool($2), Keys: toPgStringNodes(toStrSlice($4)), WithoutOverlaps: toBool($5), Including: toPgStringNodes(toStrSlice($7)), Options: toNodeSlice(toDefElemSlice($8)), Indexspace: toStr($9), Location: -1}, $10)
		grammarSetLocation($$, @1)
	}
	| UNIQUE ExistingIndex ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrUnique, $3); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrUnique), Indexname: toStr($2), Location: -1}, $3)
		grammarSetLocation($$, @1)
	}
	| PRIMARY KEY '(' columnList opt_without_overlaps ')' opt_c_include opt_definition OptConsTableSpace 				ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrPrimaryKey, $10); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrPrimaryKey), Keys: toPgStringNodes(toStrSlice($4)), WithoutOverlaps: toBool($5), Including: toPgStringNodes(toStrSlice($7)), Options: toNodeSlice(toDefElemSlice($8)), Indexspace: toStr($9), Location: -1}, $10)
		grammarSetLocation($$, @1)
	}
	| PRIMARY KEY ExistingIndex ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrPrimaryKey, $4); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrPrimaryKey), Indexname: toStr($3), Location: -1}, $4)
		grammarSetLocation($$, @1)
	}
	| EXCLUDE access_method_clause '(' ExclusionConstraintList ')' 				opt_c_include opt_definition OptConsTableSpace OptWhereClause 				ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrExclusion, $10); msg != "" { yylex.Error(msg); return 1 }
		$$ = grammarConstraintAttrs(&Constraint{Contype: pg_query.ConstrType(ConstrExclusion), AccessMethod: toStr($2), Exclusions: toNodeSlice(toNodeSlice($4)), Including: toPgStringNodes(toStrSlice($6)), Options: toNodeSlice(toDefElemSlice($7)), Indexspace: toStr($8), WhereClause: toNode($9), Location: -1}, $10)
		grammarSetLocation($$, @1)
	}
	| FOREIGN KEY '(' columnList optionalPeriodName ')' REFERENCES qualified_name 				opt_column_and_period_list key_match key_actions ConstraintAttributeSpec
	{
		if msg := grammarConstraintAttrError(ConstrForeignKey, $12); msg != "" { yylex.Error(msg); return 1 }
		n := grammarForeignKey($8, $9, $10, $11)
		n.FkAttrs = toPgStringNodes(toStrSlice($4))
		if toStr($5) != "" { n.FkAttrs = append(n.FkAttrs, toNode(&String{Sval: toStr($5)})); n.FkWithPeriod = true}
		$$ = grammarConstraintAttrs(n, $12)
		grammarSetLocation($$, @1)
	}
	;

DomainConstraint:
	  CONSTRAINT name DomainConstraintElem
	{
		n := toConstraint($3)
		n.Conname = toStr($2)
		$$ = n
		grammarSetLocation($$, @1)
	}
	| DomainConstraintElem
	{
		$$ = $1
	}
	;

DomainConstraintElem:
	  CHECK '(' a_expr ')' ConstraintAttributeSpec
	{
		a := toConstrAttr($5)
		if a.deferrable || a.seen["ENFORCED"] || a.seen["NOT ENFORCED"] {
			yylex.Error("unsupported domain CHECK constraint attribute")
			return 1
		}
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrCheck), RawExpr: toNode($3), SkipValidation: a.skipValidation, IsNoInherit: a.noInherit, IsEnforced: true, InitiallyValid: !a.skipValidation, Location: -1}
		grammarSetLocation($$, @1)
	}
	| NOT NULL_P ConstraintAttributeSpec
	{
		a := toConstrAttr($3)
		if a.deferrable || a.skipValidation || a.noInherit || a.seen["ENFORCED"] || a.seen["NOT ENFORCED"] {
			yylex.Error("unsupported domain NOT NULL constraint attribute")
			return 1
		}
		$$ = &Constraint{Contype: pg_query.ConstrType(ConstrNotNull), Keys: toPgStringNodes([]string{"value"}), InitiallyValid: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

opt_no_inherit:
	  NO INHERIT
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_without_overlaps:
	  WITHOUT OVERLAPS
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_column_list:
	  '(' columnList ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

columnList:
	  columnElem
	{
		$$ = []string{toStr($1)}
	}
	| columnList ',' columnElem
	{
		$$ = appendToStrSlice($1, $3)
	}
	;

optionalPeriodName:
	  ',' PERIOD columnElem
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_column_and_period_list:
	  '(' columnList optionalPeriodName ')'
	{
		cols := toStrSlice($2)
		if toStr($3) != "" { cols = append(cols, toStr($3)) }
		$$ = grammarPeriodColumns{cols: cols, period: toStr($3) != ""}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

columnElem:
	  ColId
	{
		$$ = $1
	}
	;

opt_c_include:
	  INCLUDE '(' columnList ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

key_match:
	  MATCH FULL
	{
		$$ = "f"
	}
	| MATCH PARTIAL
	{
		yylex.(*Lexer).errorAt("MATCH PARTIAL not yet implemented", @1)
		return 1
	}
	| MATCH SIMPLE
	{
		$$ = "s"
	}
	| /* EMPTY */
	{
		$$ = "s"
	}
	;

ExclusionConstraintList:
	  ExclusionConstraintElem
	{
		$$ = []Node{toNode($1.(*List))}
	}
	| ExclusionConstraintList ',' ExclusionConstraintElem
	{
		$$ = append(toNodeSlice($1), toNode($3.(*List)))
	}
	;

ExclusionConstraintElem:
	  index_elem WITH any_operator
	{
		names := make([]Node, 0, len(toStrSlice($3)))
		for _, name := range toStrSlice($3) { names = append(names, toNode(&String{Sval: name})) }
		$$ = &List{Items: []Node{toNode($1),toNode(&List{Items: toNodeSlice(names)})}}
	}
	| index_elem WITH OPERATOR '(' any_operator ')'
	{
		names := make([]Node, 0, len(toStrSlice($5)))
		for _, name := range toStrSlice($5) { names = append(names, toNode(&String{Sval: name})) }
		$$ = &List{Items: []Node{toNode($1),toNode(&List{Items: toNodeSlice(names)})}}
	}
	;

OptWhereClause:
	  WHERE '(' a_expr ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

key_actions:
	  key_update
	{
		$$ = $1
	}
	| key_delete
	{
		$$ = $1
	}
	| key_update key_delete
	{
		a, b := $1.(grammarKeyActions), $2.(grammarKeyActions)
		$$ = grammarKeyActions{update: a.update, delete: b.delete}
	}
	| key_delete key_update
	{
		a, b := $1.(grammarKeyActions), $2.(grammarKeyActions)
		$$ = grammarKeyActions{update: b.update, delete: a.delete}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

key_update:
	  ON UPDATE key_action
	{
		a := $3.(grammarKeyAction)
		if len(a.cols) != 0 { yylex.(*Lexer).errorAt("column lists are only supported for ON DELETE actions", @1); return 1 }
		$$ = grammarKeyActions{update: a}
	}
	;

key_delete:
	  ON DELETE_P key_action
	{
		$$ = grammarKeyActions{delete: $3.(grammarKeyAction)}
	}
	;

key_action:
	  NO ACTION
	{
		$$ = grammarKeyAction{action: "NO ACTION"}
	}
	| RESTRICT
	{
		$$ = grammarKeyAction{action: "RESTRICT"}
	}
	| CASCADE
	{
		$$ = grammarKeyAction{action: "CASCADE"}
	}
	| SET NULL_P opt_column_list
	{
		$$ = grammarKeyAction{action: "SET NULL", cols: toStrSlice($3)}
	}
	| SET DEFAULT opt_column_list
	{
		$$ = grammarKeyAction{action: "SET DEFAULT", cols: toStrSlice($3)}
	}
	;

OptInherit:
	  INHERITS '(' qualified_name_list ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptPartitionSpec:
	  PartitionSpec
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

PartitionSpec:
	  PARTITION BY ColId '(' part_params ')'
	{
		strategy := strings.ToLower(toStr($3))
		if strategy != "hash" && strategy != "list" && strategy != "range" { yylex.Error("unrecognized partitioning strategy"); return 1 }
		$$ = &PartitionSpec{Strategy: enumPartitionStrategy(strategy), PartParams: toNodeSlice($5), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

part_params:
	  part_elem
	{
		$$ = []Node{toNode($1)}
	}
	| part_params ',' part_elem
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

part_elem:
	  ColId opt_collate opt_qualified_name
	{
		$$ = &PartitionElem{Name: toStr($1), Collation: toPgStringNodes(toStrSlice($2)), Opclass: toPgStringNodes(toStrSlice($3)), Location: -1}
		grammarSetLocation($$, @1)
	}
	| func_expr_windowless opt_collate opt_qualified_name
	{
		$$ = &PartitionElem{Expr: toNode($1), Collation: toPgStringNodes(toStrSlice($2)), Opclass: toPgStringNodes(toStrSlice($3)), Location: -1}
		grammarSetLocation($$, @1)
	}
	| '(' a_expr ')' opt_collate opt_qualified_name
	{
		$$ = &PartitionElem{Expr: toNode($2), Collation: toPgStringNodes(toStrSlice($4)), Opclass: toPgStringNodes(toStrSlice($5)), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

table_access_method_clause:
	  USING name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptWith:
	  WITH reloptions
	{
		$$ = $2
	}
	| WITHOUT OIDS
	{
		$$ = nil
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OnCommitOption:
	  ON COMMIT DROP
	{
		$$ = "DROP"
	}
	| ON COMMIT DELETE_P ROWS
	{
		$$ = "DELETE_ROWS"
	}
	| ON COMMIT PRESERVE ROWS
	{
		$$ = "PRESERVE_ROWS"
	}
	| /* EMPTY */
	{
		$$ = "NOOP"
	}
	;

OptTableSpace:
	  TABLESPACE name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptConsTableSpace:
	  USING INDEX TABLESPACE name
	{
		$$ = $4
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ExistingIndex:
	  USING INDEX name
	{
		$$ = $3
	}
	;

CreateStatsStmt:
	  CREATE STATISTICS opt_qualified_name opt_name_list ON stats_params FROM from_list
	{
		$$ = makeCreateStatsStmt($3, $4, $6, $8, false)
	}
	| CREATE STATISTICS IF_P NOT EXISTS any_name opt_name_list ON stats_params FROM from_list
	{
		$$ = makeCreateStatsStmt($6, $7, $9, $11, true)
	}
	;

stats_params:
	  stats_param
	{
		$$ = $1
	}
	| stats_params ',' stats_param
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

stats_param:
	  ColId
	{
		$$ = &StatsElem{Name: toStr($1)}
	}
	| func_expr_windowless
	{
		$$ = &StatsElem{Expr: toNode($1)}
	}
	| '(' a_expr ')'
	{
		$$ = &StatsElem{Expr: toNode($2)}
	}
	;

AlterStatsStmt:
	  ALTER STATISTICS any_name SET STATISTICS set_statistics_value
	{
		$$ = &AlterStatsStmt{Defnames: toPgStringNodes(toStrSlice($3)), Stxstattarget: toNode($6), MissingOk: false}
	}
	| ALTER STATISTICS IF_P EXISTS any_name SET STATISTICS set_statistics_value
	{
		$$ = &AlterStatsStmt{Defnames: toPgStringNodes(toStrSlice($5)), Stxstattarget: toNode($8), MissingOk: true}
	}
	;

CreateAsStmt:
	  CREATE OptTemp TABLE create_as_target AS SelectStmt opt_with_data
	{
		$$ = makeCreateTableAsStmt($2, $4, $6, $7, false, "table")
	}
	| CREATE OptTemp TABLE IF_P NOT EXISTS create_as_target AS SelectStmt opt_with_data
	{
		$$ = makeCreateTableAsStmt($2, $7, $9, $10, true, "table")
	}
	;

create_as_target:
	  qualified_name opt_column_list table_access_method_clause 			OptWith OnCommitOption OptTableSpace
	{
		$$ = &IntoClause{Rel: toRangeVar($1), ColNames: toPgStringNodes(toStrSlice($2)), AccessMethod: toStr($3), Options: toNodeSlice(toDefElemSlice($4)), OnCommit: enumOnCommitAction(toStr($5)), TableSpaceName: toStr($6)}
	}
	;

opt_with_data:
	  WITH DATA_P
	{
		$$ = true
	}
	| WITH NO DATA_P
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = true
	}
	;

CreateMatViewStmt:
	  CREATE OptNoLog MATERIALIZED VIEW create_mv_target AS SelectStmt opt_with_data
	{
		$$ = makeCreateTableAsStmt($2, $5, $7, $8, false, "materialized view")
	}
	| CREATE OptNoLog MATERIALIZED VIEW IF_P NOT EXISTS create_mv_target AS SelectStmt opt_with_data
	{
		$$ = makeCreateTableAsStmt($2, $8, $10, $11, true, "materialized view")
	}
	;

create_mv_target:
	  qualified_name opt_column_list table_access_method_clause opt_reloptions OptTableSpace
	{
		$$ = &IntoClause{Rel: toRangeVar($1), ColNames: toPgStringNodes(toStrSlice($2)), AccessMethod: toStr($3), Options: toNodeSlice(toDefElemSlice($4)), OnCommit: enumOnCommitAction("NOOP"), TableSpaceName: toStr($5)}
	}
	;

OptNoLog:
	  UNLOGGED
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

RefreshMatViewStmt:
	  REFRESH MATERIALIZED VIEW opt_concurrently qualified_name opt_with_data
	{
		$$ = makeRefreshMatViewStmt(toBool($4), !toBool($6), $5)
	}
	;

CreateSeqStmt:
	  CREATE OptTemp SEQUENCE qualified_name OptSeqOptList
	{
		rel := grammarUtilityRelation($4, $2)
		$$ = makeCreateSeqStmt(rel, $5, false)
	}
	| CREATE OptTemp SEQUENCE IF_P NOT EXISTS qualified_name OptSeqOptList
	{
		rel := grammarUtilityRelation($7, $2)
		$$ = makeCreateSeqStmt(rel, $8, true)
	}
	;

AlterSeqStmt:
	  ALTER SEQUENCE qualified_name SeqOptList
	{
		$$ = makeAlterSeqStmt($3, $4, false)
	}
	| ALTER SEQUENCE IF_P EXISTS qualified_name SeqOptList
	{
		$$ = makeAlterSeqStmt($5, $6, true)
	}
	;

OptSeqOptList:
	  SeqOptList
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptParenthesizedSeqOptList:
	  '(' SeqOptList ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

SeqOptList:
	  SeqOptElem
	{
		$$ = $1
	}
	| SeqOptList SeqOptElem
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	;

SeqOptElem:
	  AS SimpleTypename
	{
		$$ = makeDefElem("as", $2)
		grammarSetLocation($$, @1)
	}
	| CACHE NumericOnly
	{
		$$ = makeDefElem("cache", $2)
		grammarSetLocation($$, @1)
	}
	| CYCLE
	{
		$$ = makeDefElem("cycle", &Boolean{Boolval: true})
		grammarSetLocation($$, @1)
	}
	| NO CYCLE
	{
		$$ = makeDefElem("cycle", &Boolean{Boolval: false})
		grammarSetLocation($$, @1)
	}
	| INCREMENT opt_by NumericOnly
	{
		$$ = makeDefElem("increment", $3)
		grammarSetLocation($$, @1)
	}
	| LOGGED
	{
		$$ = makeDefElem("logged", nil)
		grammarSetLocation($$, @1)
	}
	| MAXVALUE NumericOnly
	{
		$$ = makeDefElem("maxvalue", $2)
		grammarSetLocation($$, @1)
	}
	| MINVALUE NumericOnly
	{
		$$ = makeDefElem("minvalue", $2)
		grammarSetLocation($$, @1)
	}
	| NO MAXVALUE
	{
		$$ = makeDefElem("maxvalue", nil)
		grammarSetLocation($$, @1)
	}
	| NO MINVALUE
	{
		$$ = makeDefElem("minvalue", nil)
		grammarSetLocation($$, @1)
	}
	| OWNED BY any_name
	{
		$$ = makeDefElem("owned_by", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	| SEQUENCE NAME_P any_name
	{
		$$ = makeDefElem("sequence_name", &List{Items: toNodeSlice($3)})
		grammarSetLocation($$, @1)
	}
	| START opt_with NumericOnly
	{
		$$ = makeDefElem("start", $3)
		grammarSetLocation($$, @1)
	}
	| RESTART
	{
		$$ = makeDefElem("restart", nil)
		grammarSetLocation($$, @1)
	}
	| RESTART opt_with NumericOnly
	{
		$$ = makeDefElem("restart", $3)
		grammarSetLocation($$, @1)
	}
	| UNLOGGED
	{
		$$ = makeDefElem("unlogged", nil)
		grammarSetLocation($$, @1)
	}
	;

opt_by:
	  BY
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

NumericOnly:
	  FCONST
	{
		$$ = &FloatVal{Str: toStr($1)}
	}
	| '+' FCONST
	{
		$$ = &FloatVal{Str: toStr($2)}
	}
	| '-' FCONST
	{
		$$ = &FloatVal{Str: "-" + toStr($2)}
	}
	| SignedIconst
	{
		$$ = &IntegerVal{Ival: int32(toInt64($1))}
	}
	;

NumericOnly_list:
	  NumericOnly
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| NumericOnly_list ',' NumericOnly
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

CreatePLangStmt:
	  CREATE opt_or_replace opt_trusted opt_procedural LANGUAGE name
	{
		$$ = &CreateExtensionStmt{IfNotExists: toBool($2), Extname: toStr($6)}
	}
	| CREATE opt_or_replace opt_trusted opt_procedural LANGUAGE name 			  HANDLER handler_name opt_inline_handler opt_validator
	{
		$$ = &CreatePLangStmt{Replace: toBool($2), Pltrusted: toBool($3), Plname: toStr($6), Plhandler: toPgStringNodes(toStrSlice($8)), Plinline: toPgStringNodes(toStrSlice($9)), Plvalidator: toPgStringNodes(toStrSlice($10))}
	}
	;

opt_trusted:
	  TRUSTED
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

handler_name:
	  name
	{
		$$ = []string{toStr($1)}
	}
	| name attrs
	{
		$$ = append([]string{toStr($1)}, toStrSlice($2)...)
	}
	;

opt_inline_handler:
	  INLINE_P handler_name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

validator_clause:
	  VALIDATOR handler_name
	{
		$$ = $2
	}
	| NO VALIDATOR
	{
		$$ = nil
	}
	;

opt_validator:
	  validator_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_procedural:
	  PROCEDURAL
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

CreateTableSpaceStmt:
	  CREATE TABLESPACE name OptTableSpaceOwner LOCATION Sconst opt_reloptions
	{
		$$ = &CreateTableSpaceStmt{Tablespacename: toStr($3), Owner: toRoleSpec($4), Location: toStr($6), Options: toNodeSlice(toDefElemSlice($7))}
	}
	;

OptTableSpaceOwner:
	  OWNER RoleSpec
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

DropTableSpaceStmt:
	  DROP TABLESPACE name
	{
		$$ = &DropTableSpaceStmt{Tablespacename: toStr($3), MissingOk: false}
	}
	| DROP TABLESPACE IF_P EXISTS name
	{
		$$ = &DropTableSpaceStmt{Tablespacename: toStr($5), MissingOk: true}
	}
	;

CreateExtensionStmt:
  CREATE EXTENSION name opt_with create_extension_opt_list
{
	$$ = &CreateExtensionStmt{Extname: toStr($3), Options: toNodeSlice($5)}
	grammarSetLocation($$, @3)
}
| CREATE EXTENSION IF_P NOT EXISTS name opt_with create_extension_opt_list
{
	$$ = &CreateExtensionStmt{Extname: toStr($6), IfNotExists: true, Options: toNodeSlice($8)}
	grammarSetLocation($$, @6)
}
;

create_extension_opt_list:
  create_extension_opt_list create_extension_opt_item
{
	$$ = appendToDefElemSlice($1, $2)
}
| /* EMPTY */
{
	$$ = nil
}
;

create_extension_opt_item:
  SCHEMA name
{
	$$ = makeDefElem("schema", &String{Sval: toStr($2)})
	grammarSetLocation($$, @1)
}
| VERSION_P NonReservedWord_or_Sconst
{
	$$ = makeDefElem("new_version", &String{Sval: toStr($2)})
	grammarSetLocation($$, @1)
}
| FROM NonReservedWord_or_Sconst
{
	yylex.(*Lexer).errorAt("CREATE EXTENSION ... FROM is no longer supported", @1)
	return 1
}
| CASCADE
{
	$$ = makeDefElem("cascade", &Boolean{Boolval: true})
	grammarSetLocation($$, @1)
}
;

AlterExtensionStmt:
  ALTER EXTENSION name UPDATE alter_extension_opt_list
{
	$$ = &AlterExtensionStmt{Extname: toStr($3), Options: toNodeSlice($5)}
	grammarSetLocation($$, @3)
}
;

alter_extension_opt_list:
  alter_extension_opt_list alter_extension_opt_item
{
	$$ = appendToDefElemSlice($1, $2)
}
| /* EMPTY */
{
	$$ = nil
}
;

alter_extension_opt_item:
  TO NonReservedWord_or_Sconst
{
	$$ = makeDefElem("new_version", &String{Sval: toStr($2)})
	grammarSetLocation($$, @1)
}
;

AlterExtensionContentsStmt:
  ALTER EXTENSION name add_drop object_type_name name
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType(toStr($5)), Object: toNode(&String{Sval: toStr($6)})}
}
| ALTER EXTENSION name add_drop object_type_any_name any_name
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType(toStr($5)), Object: toNode(toCommentTarget($6))}
}
| ALTER EXTENSION name add_drop AGGREGATE aggregate_with_argtypes
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("aggregate"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop CAST '(' Typename AS Typename ')'
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("cast"), Object: toNode(&List{Items: []Node{toNode(toTypeNameNode($7)), toNode(toTypeNameNode($9))}})}
}
| ALTER EXTENSION name add_drop DOMAIN_P Typename
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("domain"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop FUNCTION function_with_argtypes
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("function"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop OPERATOR operator_with_argtypes
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("operator"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop OPERATOR CLASS any_name USING name
{
	items := append([]Node{toNode(&String{Sval: toStr($9)})}, toNodeSlice(toStrSlice($7))...)
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("opclass"), Object: toNode(&List{Items: items})}
}
| ALTER EXTENSION name add_drop OPERATOR FAMILY any_name USING name
{
	items := append([]Node{toNode(&String{Sval: toStr($9)})}, toNodeSlice(toStrSlice($7))...)
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("opfamily"), Object: toNode(&List{Items: items})}
}
| ALTER EXTENSION name add_drop PROCEDURE function_with_argtypes
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("procedure"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop ROUTINE function_with_argtypes
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("routine"), Object: toNode($6)}
}
| ALTER EXTENSION name add_drop TRANSFORM FOR Typename LANGUAGE name
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("transform"), Object: toNode(&List{Items: []Node{toNode(toTypeNameNode($7)), toNode(&String{Sval: toStr($9)})}})}
}
| ALTER EXTENSION name add_drop TYPE_P Typename
{
	$$ = &AlterExtensionContentsStmt{Extname: toStr($3), Action: toInt32($4), Objtype: enumObjectType("type"), Object: toNode($6)}
}
;

CreateFdwStmt:
	  CREATE FOREIGN DATA_P WRAPPER name opt_fdw_options create_generic_options
	{
		$$ = &CreateFdwStmt{Fdwname: toStr($5), FuncOptions: toNodeSlice($6), Options: toNodeSlice($7)}
	}
	;

fdw_option:
	  HANDLER handler_name
	{
		names := toStrSlice($2)
		items := make([]Node, 0, len(names))
		for _, n := range names {
			items = append(items, toNode(&String{Sval: n}))
		}
		$$ = &DefElem{Defname: "handler", Arg: toNode(&List{Items: toNodeSlice(items)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| NO HANDLER
	{
		$$ = &DefElem{Defname: "handler", Arg: nil, Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| VALIDATOR handler_name
	{
		names := toStrSlice($2)
		items := make([]Node, 0, len(names))
		for _, n := range names {
			items = append(items, toNode(&String{Sval: n}))
		}
		$$ = &DefElem{Defname: "validator", Arg: toNode(&List{Items: toNodeSlice(items)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| NO VALIDATOR
	{
		$$ = &DefElem{Defname: "validator", Arg: nil, Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

fdw_options:
	  fdw_option
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| fdw_options fdw_option
	{
		$$ = append(toNodeSlice($1), toNode($2))
	}
	;

opt_fdw_options:
	  fdw_options
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterFdwStmt:
	  ALTER FOREIGN DATA_P WRAPPER name opt_fdw_options alter_generic_options
	{
		$$ = &AlterFdwStmt{Fdwname: toStr($5), FuncOptions: toNodeSlice($6), Options: toNodeSlice($7)}
	}
	| ALTER FOREIGN DATA_P WRAPPER name fdw_options
	{
		$$ = &AlterFdwStmt{Fdwname: toStr($5), FuncOptions: toNodeSlice($6)}
	}
	;

create_generic_options:
	  OPTIONS '(' generic_option_list ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

generic_option_list:
	  generic_option_elem
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| generic_option_list ',' generic_option_elem
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

alter_generic_options:
	  OPTIONS	'(' alter_generic_option_list ')'
	{
		$$ = $3
	}
	;

alter_generic_option_list:
	  alter_generic_option_elem
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| alter_generic_option_list ',' alter_generic_option_elem
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

alter_generic_option_elem:
	  generic_option_elem
	{
		$$ = $1
	}
	| SET generic_option_elem
	{
		elem := $2.(*DefElem)
		elem.Defaction = enumDefElemAction("SET")
		$$ = elem
	}
	| ADD_P generic_option_elem
	{
		elem := $2.(*DefElem)
		elem.Defaction = enumDefElemAction("ADD")
		$$ = elem
	}
	| DROP generic_option_name
	{
		$$ = &DefElem{Defname: toStr($2), Defaction: enumDefElemAction("DROP"), Location: -1}
		grammarSetLocation($$, @2)
	}
	;

generic_option_elem:
	  generic_option_name generic_option_arg
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode(&String{Sval: toStr($2)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

generic_option_name:
	  ColLabel
	{
		$$ = $1
	}
	;

generic_option_arg:
	  Sconst
	{
		$$ = $1
	}
	;

CreateForeignServerStmt:
	  CREATE SERVER name opt_type opt_foreign_server_version FOREIGN DATA_P WRAPPER name create_generic_options
	{
		$$ = &CreateForeignServerStmt{Servername: toStr($3), Servertype: toStr($4), Version: toStr($5), Fdwname: toStr($9), IfNotExists: false, Options: toNodeSlice($10)}
	}
	| CREATE SERVER IF_P NOT EXISTS name opt_type opt_foreign_server_version FOREIGN DATA_P WRAPPER name create_generic_options
	{
		$$ = &CreateForeignServerStmt{Servername: toStr($6), Servertype: toStr($7), Version: toStr($8), Fdwname: toStr($12), IfNotExists: true, Options: toNodeSlice($13)}
	}
	;

opt_type:
	  TYPE_P Sconst
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

foreign_server_version:
	  VERSION_P Sconst
	{
		$$ = $2
	}
	| VERSION_P NULL_P
	{
		$$ = ""
	}
	;

opt_foreign_server_version:
	  foreign_server_version
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterForeignServerStmt:
	  ALTER SERVER name foreign_server_version alter_generic_options
	{
		$$ = &AlterForeignServerStmt{Servername: toStr($3), Version: toStr($4), HasVersion: true, Options: toNodeSlice($5)}
	}
	| ALTER SERVER name foreign_server_version
	{
		$$ = &AlterForeignServerStmt{Servername: toStr($3), Version: toStr($4), HasVersion: true}
	}
	| ALTER SERVER name alter_generic_options
	{
		$$ = &AlterForeignServerStmt{Servername: toStr($3), Options: toNodeSlice($4)}
	}
	;

CreateForeignTableStmt:
	  CREATE FOREIGN TABLE qualified_name '(' OptTableElementList ')' OptInherit SERVER name create_generic_options
	{
		$$ = &CreateForeignTableStmt{BaseStmt: grammarTableOptions(grammarCreateTable($4, $6, false), nil, $8, nil, nil, nil, "NOOP", nil), Servername: toStr($10), Options: toNodeSlice($11)}
	}
	| CREATE FOREIGN TABLE IF_P NOT EXISTS qualified_name '(' OptTableElementList ')' OptInherit SERVER name create_generic_options
	{
		$$ = &CreateForeignTableStmt{BaseStmt: grammarTableOptions(grammarCreateTable($7, $9, true), nil, $11, nil, nil, nil, "NOOP", nil), Servername: toStr($13), Options: toNodeSlice($14)}
	}
	| CREATE FOREIGN TABLE qualified_name PARTITION OF qualified_name OptTypedTableElementList PartitionBoundSpec SERVER name create_generic_options
	{
		n := grammarTableOptions(grammarCreateTable($4, $8, false), nil, []Node{toNode(toRangeVar($7))}, nil, nil, nil, "NOOP", nil)
		n.Partbound, _ = $9.(*PartitionBoundSpec)
		$$ = &CreateForeignTableStmt{BaseStmt: n, Servername: toStr($11), Options: toNodeSlice($12)}
	}
	| CREATE FOREIGN TABLE IF_P NOT EXISTS qualified_name PARTITION OF qualified_name OptTypedTableElementList PartitionBoundSpec SERVER name create_generic_options
	{
		n := grammarTableOptions(grammarCreateTable($7, $11, true), nil, []Node{toNode(toRangeVar($10))}, nil, nil, nil, "NOOP", nil)
		n.Partbound, _ = $12.(*PartitionBoundSpec)
		$$ = &CreateForeignTableStmt{BaseStmt: n, Servername: toStr($14), Options: toNodeSlice($15)}
	}
	;

ImportForeignSchemaStmt:
	  IMPORT_P FOREIGN SCHEMA name import_qualification 		  FROM SERVER name INTO name create_generic_options
	{
		qual := toImportQual($5)
		$$ = &ImportForeignSchemaStmt{ServerName: toStr($8), RemoteSchema: toStr($4), LocalSchema: toStr($10), ListType: pg_query.ImportForeignSchemaType(qual.listType), TableList: toNodeSlice(qual.tableList), Options: toNodeSlice(toDefElemSlice($11))}
	}
	;

import_qualification_type:
	  LIMIT TO
	{
		$$ = int32(2)
	}
	| EXCEPT
	{
		$$ = int32(3)
	}
	;

import_qualification:
	  import_qualification_type '(' relation_expr_list ')'
	{
		$$ = &importQual{listType: $1.(int32), tableList: toRangeVarSlice($3)}
	}
	| /* EMPTY */
	{
		$$ = &importQual{listType: int32(1)}
	}
	;

CreateUserMappingStmt:
	  CREATE USER MAPPING FOR auth_ident SERVER name create_generic_options
	{
		$$ = &CreateUserMappingStmt{User: toRoleSpec($5), Servername: toStr($7), IfNotExists: false, Options: toNodeSlice($8)}
	}
	| CREATE USER MAPPING IF_P NOT EXISTS FOR auth_ident SERVER name create_generic_options
	{
		$$ = &CreateUserMappingStmt{User: toRoleSpec($8), Servername: toStr($10), IfNotExists: true, Options: toNodeSlice($11)}
	}
	;

auth_ident:
	  RoleSpec
	{
		$$ = $1
	}
	| USER
	{
		$$ = &RoleSpec{Roletype: enumRoleSpecType("ROLESPEC_CURRENT_USER"), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

DropUserMappingStmt:
	  DROP USER MAPPING FOR auth_ident SERVER name
	{
		$$ = &DropUserMappingStmt{User: toRoleSpec($5), Servername: toStr($7), MissingOk: false}
	}
	| DROP USER MAPPING IF_P EXISTS FOR auth_ident SERVER name
	{
		$$ = &DropUserMappingStmt{User: toRoleSpec($7), Servername: toStr($9), MissingOk: true}
	}
	;

AlterUserMappingStmt:
	  ALTER USER MAPPING FOR auth_ident SERVER name alter_generic_options
	{
		$$ = &AlterUserMappingStmt{User: toRoleSpec($5), Servername: toStr($7), Options: toNodeSlice($8)}
	}
	;

CreatePolicyStmt:
	  CREATE POLICY name ON qualified_name RowSecurityDefaultPermissive
				RowSecurityDefaultForCmd RowSecurityDefaultToRole
				RowSecurityOptionalExpr RowSecurityOptionalWithCheck
	{
		$$ = makeCreatePolicyStmt($3, $5, $7, toBool($6), $8, $9, $10)
	}
	;

AlterPolicyStmt:
	  ALTER POLICY name ON qualified_name RowSecurityOptionalToRole 				RowSecurityOptionalExpr RowSecurityOptionalWithCheck
	{
		$$ = &AlterPolicyStmt{PolicyName: toStr($3), Table: toRangeVar($5), Roles: toNodeSlice($6), Qual: toNode($7), WithCheck: toNode($8)}
	}
	;

RowSecurityOptionalExpr:
	  USING '(' a_expr ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

RowSecurityOptionalWithCheck:
	  WITH CHECK '(' a_expr ')'
	{
		$$ = $4
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

RowSecurityDefaultToRole:
	  TO role_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = []Node{toNode(&RoleSpec{Roletype: enumRoleSpecType("ROLESPEC_PUBLIC"), Location: -1})}
	}
	;

RowSecurityOptionalToRole:
	  TO role_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

RowSecurityDefaultPermissive:
	  AS IDENT
	{
		switch toStr($2) {
		case "permissive": $$ = true
		case "restrictive": $$ = false
		default: yylex.(*Lexer).errorAt("unrecognized row security option", @2); return 1
		}
	}
	| /* EMPTY */
	{
		$$ = true
	}
	;

RowSecurityDefaultForCmd:
	  FOR row_security_cmd
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = "all"
	}
	;

row_security_cmd:
	  ALL
	{
		$$ = $1
	}
	| SELECT
	{
		$$ = $1
	}
	| INSERT
	{
		$$ = $1
	}
	| UPDATE
	{
		$$ = $1
	}
	| DELETE_P
	{
		$$ = $1
	}
	;

CreateAmStmt:
	  CREATE ACCESS METHOD name TYPE_P am_type HANDLER handler_name
	{
		$$ = &CreateAmStmt{Amname: toStr($4), Amtype: toStr($6), HandlerName: toPgStringNodes(toStrSlice($8))}
	}
	;

am_type:
	  INDEX
	{
		$$ = "i"
	}
	| TABLE
	{
		$$ = "t"
	}
	;

CreateTrigStmt:
	  CREATE opt_or_replace TRIGGER name TriggerActionTime TriggerEvents ON
	  qualified_name TriggerReferencing TriggerForSpec TriggerWhen
	  EXECUTE FUNCTION_or_PROCEDURE func_name '(' TriggerFuncArgs ')'
	{
		ev := $6.(*triggerEventsInfo)
		$$ = &CreateTrigStmt{Replace: toBool($2), Trigname: toStr($4), Timing: int32(toInt($5)), Events: ev.events, Columns: toNodeSlice(ev.columns), Relation: toRangeVar($8), TransitionRels: toNodeSlice($9), Row: toBool($10), WhenClause: toNode($11), Funcname: toPgStringNodes(toStrSlice($14)), Args: toNodeSlice($16)}
	}
	| CREATE opt_or_replace CONSTRAINT TRIGGER name AFTER TriggerEvents ON
	  qualified_name OptConstrFromTable ConstraintAttributeSpec
	  FOR EACH ROW TriggerWhen
	  EXECUTE FUNCTION_or_PROCEDURE func_name '(' TriggerFuncArgs ')'
	{
		ev := $7.(*triggerEventsInfo)
		ca := toConstrAttr($11)
		$$ = &CreateTrigStmt{Replace: toBool($2), Isconstraint: true, Trigname: toStr($5), Timing: 0, Events: ev.events, Columns: toNodeSlice(ev.columns), Relation: toRangeVar($9), Constrrel: toRangeVar($10), Deferrable: ca.deferrable, Initdeferred: ca.initdeferred, Row: true, WhenClause: toNode($15), Funcname: toPgStringNodes(toStrSlice($18)), Args: toNodeSlice($20)}
	}
	;

TriggerActionTime:
	  BEFORE
	{
		$$ = 2
	}
	| AFTER
	{
		$$ = 0
	}
	| INSTEAD OF
	{
		$$ = 64
	}
	;

TriggerEvents:
	  TriggerOneEvent
	{
		$$ = $1
	}
	| TriggerEvents OR TriggerOneEvent
	{
		ev1 := $1.(*triggerEventsInfo)
		ev2 := $3.(*triggerEventsInfo)
		if ev1.events & ev2.events != 0 { yylex.Error("duplicate trigger events specified"); return 1 }
		$$ = &triggerEventsInfo{
			events:  ev1.events | ev2.events,
			columns: append(ev1.columns, ev2.columns...),
		}
	}
	;

TriggerOneEvent:
	  INSERT
	{
		$$ = &triggerEventsInfo{events: 4}
	}
	| DELETE_P
	{
		$$ = &triggerEventsInfo{events: 8}
	}
	| UPDATE
	{
		$$ = &triggerEventsInfo{events: 16}
	}
	| UPDATE OF columnList
	{
		var cols []Node
		for _, col := range toStrSlice($3) {
			cols = append(cols, toNode(&String{Sval: col}))
		}
		$$ = &triggerEventsInfo{events: 16, columns: cols}
	}
	| TRUNCATE
	{
		$$ = &triggerEventsInfo{events: 32}
	}
	;

TriggerReferencing:
	  REFERENCING TriggerTransitions
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

TriggerTransitions:
	  TriggerTransition
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| TriggerTransitions TriggerTransition
	{
		list := toNodeSlice($1)
		if $2 != nil {
			list = append(list, toNode($2))
		}
		$$ = list
	}
	;

TriggerTransition:
	  TransitionOldOrNew TransitionRowOrTable opt_as TransitionRelName
	{
		$$ = &TriggerTransition{Name: toStr($4), IsNew: toStr($1) == "NEW", IsTable: toStr($2) == "TABLE"}
	}
	;

TransitionOldOrNew:
	  NEW
	{
		$$ = "NEW"
	}
	| OLD
	{
		$$ = "OLD"
	}
	;

TransitionRowOrTable:
	  TABLE
	{
		$$ = "TABLE"
	}
	| ROW
	{
		$$ = "ROW"
	}
	;

TransitionRelName:
	  ColId
	{
		$$ = $1
	}
	;

TriggerForSpec:
	  FOR TriggerForOptEach TriggerForType
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

TriggerForOptEach:
	  EACH
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

TriggerForType:
	  ROW
	{
		$$ = true
	}
	| STATEMENT
	{
		$$ = false
	}
	;

TriggerWhen:
	  WHEN '(' a_expr ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

FUNCTION_or_PROCEDURE:
	  FUNCTION
	{
		$$ = $1
	}
	| PROCEDURE
	{
		$$ = $1
	}
	;

TriggerFuncArgs:
	  TriggerFuncArg
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| TriggerFuncArgs ',' TriggerFuncArg
	{
		list := toNodeSlice($1)
		if $3 != nil {
			list = append(list, toNode($3))
		}
		$$ = list
	}
	| /* EMPTY */
	{
		$$ = []Node(nil)
	}
	;

TriggerFuncArg:
	  Iconst
	{
		$$ = makeTriggerFuncArg($1)
	}
	| FCONST
	{
		$$ = &String{Sval: toStr($1)}
	}
	| Sconst
	{
		$$ = &String{Sval: toStr($1)}
	}
	| ColLabel
	{
		$$ = &String{Sval: toStr($1)}
	}
	;

OptConstrFromTable:
	  FROM qualified_name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ConstraintAttributeSpec:
	  /* EMPTY */
	{
		$$ = constrAttr{}
	}
	| ConstraintAttributeSpec ConstraintAttributeElem
	{
		ca := toConstrAttr($1)
		elem := toStr($2)
		if ca.seen == nil { ca.seen = make(map[string]bool) }
		ca.seen[elem] = true
		if ca.seen["NOT DEFERRABLE"] && (ca.seen["DEFERRABLE"] || ca.seen["INITIALLY DEFERRED"]) || ca.seen["INITIALLY IMMEDIATE"] && ca.seen["INITIALLY DEFERRED"] || ca.seen["ENFORCED"] && ca.seen["NOT ENFORCED"] { yylex.(*Lexer).errorAt("conflicting constraint properties", @2); return 1 }
		ca.deferrable = ca.seen["DEFERRABLE"] || ca.seen["INITIALLY DEFERRED"]
		ca.initdeferred = ca.seen["INITIALLY DEFERRED"]
		ca.notEnforced = ca.seen["NOT ENFORCED"]
		ca.skipValidation = ca.seen["NOT VALID"] || ca.notEnforced
		ca.noInherit = ca.seen["NO INHERIT"]
		$$ = ca
	}
	;

ConstraintAttributeElem:
	  NOT DEFERRABLE
	{
		$$ = "NOT DEFERRABLE"
	}
	| DEFERRABLE
	{
		$$ = "DEFERRABLE"
	}
	| INITIALLY IMMEDIATE
	{
		$$ = "INITIALLY IMMEDIATE"
	}
	| INITIALLY DEFERRED
	{
		$$ = "INITIALLY DEFERRED"
	}
	| NOT VALID
	{
		$$ = "NOT VALID"
	}
	| NO INHERIT
	{
		$$ = "NO INHERIT"
	}
	| NOT ENFORCED
	{
		$$ = "NOT ENFORCED"
	}
	| ENFORCED
	{
		$$ = "ENFORCED"
	}

	;

CreateEventTrigStmt:
	  CREATE EVENT TRIGGER name ON ColLabel EXECUTE FUNCTION_or_PROCEDURE func_name '(' ')'
	{
		$$ = &CreateEventTrigStmt{Trigname: toStr($4), Eventname: toStr($6), Funcname: toPgStringNodes(toStrSlice($9))}
	}
	| CREATE EVENT TRIGGER name ON ColLabel WHEN event_trigger_when_list EXECUTE FUNCTION_or_PROCEDURE func_name '(' ')'
	{
		$$ = &CreateEventTrigStmt{Trigname: toStr($4), Eventname: toStr($6), Whenclause: toNodeSlice($8), Funcname: toPgStringNodes(toStrSlice($11))}
	}
	;

event_trigger_when_list:
	  event_trigger_when_item
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| event_trigger_when_list AND event_trigger_when_item
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

event_trigger_when_item:
	  ColId IN_P '(' event_trigger_value_list ')'
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode(&List{Items: toNodeSlice($4)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

event_trigger_value_list:
	  SCONST
	{
		$$ = []Node{toNode(&String{Sval: toStr($1)})}
	}
	| event_trigger_value_list ',' SCONST
	{
		$$ = append(toNodeSlice($1), toNode(&String{Sval: toStr($3)}))
	}
	;

AlterEventTrigStmt:
	  ALTER EVENT TRIGGER name enable_trigger
	{
		$$ = &AlterEventTrigStmt{Trigname: toStr($4), Tgenabled: toStr($5)}
	}
	;

enable_trigger:
	  ENABLE_P
	{
		$$ = "O"
	}
	| ENABLE_P REPLICA
	{
		$$ = "R"
	}
	| ENABLE_P ALWAYS
	{
		$$ = "A"
	}
	| DISABLE_P
	{
		$$ = "D"
	}
	;

CreateAssertionStmt:
  CREATE ASSERTION any_name CHECK '(' a_expr ')' ConstraintAttributeSpec
{
	yylex.(*Lexer).errorAt("CREATE ASSERTION is not yet implemented", @1)
	return 1
}
;

DefineStmt:
	  CREATE opt_or_replace AGGREGATE func_name aggr_args definition
	{
		var args []Node
		if aai, ok := $5.(*aggrArgsInfo); ok {
			var parameters Node
			if len(aai.args) != 0 { parameters = toNode(&List{Items: toNodeSlice(aai.args)}) }
			args = []Node{toNode(parameters),toNode(&IntegerVal{Ival: int32(int64(aai.numDirectArgs))}),
			}
		}
		$$ = &DefineStmt{Kind: enumObjectType("aggregate"), Replace: toBool($2), Defnames: toPgStringNodes(toStrSlice($4)), Args: toNodeSlice(args), Definition: toNodeSlice($6)}
	}
	| CREATE opt_or_replace AGGREGATE func_name old_aggr_definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("aggregate"), Replace: toBool($2), Oldstyle: true, Defnames: toPgStringNodes(toStrSlice($4)), Definition: toNodeSlice($5)}
	}
	| CREATE OPERATOR any_operator definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("operator"), Defnames: toPgStringNodes(toStrSlice($3)), Definition: toNodeSlice($4)}
	}
	| CREATE TYPE_P any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("type"), Defnames: toPgStringNodes(toStrSlice($3)), Definition: toNodeSlice($4)}
	}
	| CREATE TYPE_P any_name
	{
		$$ = &DefineStmt{Kind: enumObjectType("type"), Defnames: toPgStringNodes(toStrSlice($3))}
	}
	| CREATE TYPE_P any_name AS '(' OptTableFuncElementList ')'
	{
		stmt := makeCompositeTypeStmt($3, $6)
		stmt.Typevar.Location = int32(@3)
		$$ = stmt
	}
	| CREATE TYPE_P any_name AS ENUM_P '(' opt_enum_val_list ')'
	{
		$$ = &CreateEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), Vals: toPgStringNodes(toStrSlice($7))}
	}
	| CREATE TYPE_P any_name AS RANGE definition
	{
		$$ = &CreateRangeStmt{TypeName: toPgStringNodes(toStrSlice($3)), Params: toNodeSlice($6)}
	}
	| CREATE TEXT_P SEARCH PARSER any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("tsparser"), Defnames: toPgStringNodes(toStrSlice($5)), Definition: toNodeSlice($6)}
	}
	| CREATE TEXT_P SEARCH DICTIONARY any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("tsdictionary"), Defnames: toPgStringNodes(toStrSlice($5)), Definition: toNodeSlice($6)}
	}
	| CREATE TEXT_P SEARCH TEMPLATE any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("tstemplate"), Defnames: toPgStringNodes(toStrSlice($5)), Definition: toNodeSlice($6)}
	}
	| CREATE TEXT_P SEARCH CONFIGURATION any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("tsconfiguration"), Defnames: toPgStringNodes(toStrSlice($5)), Definition: toNodeSlice($6)}
	}
	| CREATE COLLATION any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("collation"), Defnames: toPgStringNodes(toStrSlice($3)), Definition: toNodeSlice($4)}
	}
	| CREATE COLLATION IF_P NOT EXISTS any_name definition
	{
		$$ = &DefineStmt{Kind: enumObjectType("collation"), IfNotExists: true, Defnames: toPgStringNodes(toStrSlice($6)), Definition: toNodeSlice($7)}
	}
	| CREATE COLLATION any_name FROM any_name
	{
		$$ = &DefineStmt{Kind: enumObjectType("collation"), Defnames: toPgStringNodes(toStrSlice($3)), Definition: []Node{toNode(&DefElem{Defname: "from", Arg: toNode(makeDefElemOperatorArg($5)), Location: int32(@5), Defaction: pg_query.DefElemAction(1)})}}
	}
	| CREATE COLLATION IF_P NOT EXISTS any_name FROM any_name
	{
		$$ = &DefineStmt{Kind: enumObjectType("collation"), IfNotExists: true, Defnames: toPgStringNodes(toStrSlice($6)), Definition: []Node{toNode(&DefElem{Defname: "from", Arg: toNode(makeDefElemOperatorArg($8)), Location: int32(@8), Defaction: pg_query.DefElemAction(1)})}}
	}
	;

definition:
	  '(' def_list ')'
	{
		$$ = $2
	}
	;

def_list:
	  def_elem
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| def_list ',' def_elem
	{
		list := toNodeSlice($1)
		if $3 != nil {
			list = append(list, toNode($3))
		}
		$$ = list
	}
	;

def_elem:
	  ColLabel '=' def_arg
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode($3), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel
	{
		$$ = &DefElem{Defname: toStr($1), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

def_arg:
	  func_type
	{
		$$ = toTypeNameNode($1)
	}
	| reserved_keyword
	{
		$$ = &String{Sval: toStr($1)}
	}
	| qual_all_Op
	{
		$$ = makeDefElemOperatorArg($1)
	}
	| NumericOnly
	{
		$$ = toNode($1)
	}
	| Sconst
	{
		$$ = &String{Sval: toStr($1)}
	}
	| NONE
	{
		$$ = &String{Sval: toStr($1)}
	}
	;

old_aggr_definition:
	  '(' old_aggr_list ')'
	{
		$$ = $2
	}
	;

old_aggr_list:
	  old_aggr_elem
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| old_aggr_list ',' old_aggr_elem
	{
		list := toNodeSlice($1)
		if $3 != nil {
			list = append(list, toNode($3))
		}
		$$ = list
	}
	;

old_aggr_elem:
	  IDENT '=' def_arg
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode($3), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

opt_enum_val_list:
	  enum_val_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = []string(nil)
	}
	;

enum_val_list:
	  Sconst
	{
		$$ = []string{toStr($1)}
	}
	| enum_val_list ',' Sconst
	{
		$$ = append(toStrSlice($1), toStr($3))
	}

	;

AlterEnumStmt:
	  ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst
	{
		$$ = &AlterEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), NewVal: toStr($7), NewValIsAfter: true, SkipIfNewValExists: toBool($6)}
	}
	| ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst BEFORE Sconst
	{
		$$ = &AlterEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), NewVal: toStr($7), NewValNeighbor: toStr($9), NewValIsAfter: false, SkipIfNewValExists: toBool($6)}
	}
	| ALTER TYPE_P any_name ADD_P VALUE_P opt_if_not_exists Sconst AFTER Sconst
	{
		$$ = &AlterEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), NewVal: toStr($7), NewValNeighbor: toStr($9), NewValIsAfter: true, SkipIfNewValExists: toBool($6)}
	}
	| ALTER TYPE_P any_name RENAME VALUE_P Sconst TO Sconst
	{
		$$ = &AlterEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), OldVal: toStr($6), NewVal: toStr($8)}
	}
	| ALTER TYPE_P any_name DROP VALUE_P Sconst
	{
		$$ = &AlterEnumStmt{TypeName: toPgStringNodes(toStrSlice($3)), OldVal: toStr($6)}
	}
	;

opt_if_not_exists:
	  IF_P NOT EXISTS
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

CreateOpClassStmt:
	  CREATE OPERATOR CLASS any_name opt_default FOR TYPE_P Typename USING name opt_opfamily AS opclass_item_list
	{
		$$ = &CreateOpClassStmt{Opclassname: toPgStringNodes(toStrSlice($4)), IsDefault: toBool($5), Datatype: toTypeNameNode($8), Amname: toStr($10), Opfamilyname: toPgStringNodes(toStrSlice($11)), Items: toNodeSlice($13)}
	}
	;

opclass_item_list:
	  opclass_item
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| opclass_item_list ',' opclass_item
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

opclass_item:
	  OPERATOR Iconst any_operator opclass_purpose
	{
		$$ = &CreateOpClassItem{Itemtype: 1, Number: int32(toInt($2)), Name: &ObjectWithArgs{Objname: toPgStringNodes(toStrSlice($3))}, OrderFamily: toPgStringNodes(toStrSlice($4))}
	}
	| OPERATOR Iconst operator_with_argtypes opclass_purpose
	{
		$$ = &CreateOpClassItem{Itemtype: 1, Number: int32(toInt($2)), Name: toObjectWithArgs($3), OrderFamily: toPgStringNodes(toStrSlice($4))}
	}
	| FUNCTION Iconst function_with_argtypes
	{
		$$ = &CreateOpClassItem{Itemtype: 2, Number: int32(toInt($2)), Name: toObjectWithArgs($3)}
	}
	| FUNCTION Iconst '(' type_list ')' function_with_argtypes
	{
		$$ = &CreateOpClassItem{Itemtype: 2, Number: int32(toInt($2)), ClassArgs: toNodeSlice(makeTypeNameNodes($4)), Name: toObjectWithArgs($6)}
	}
	| STORAGE Typename
	{
		$$ = &CreateOpClassItem{Itemtype: 3, Storedtype: toTypeNameNode($2)}
	}
	;

opt_default:
	  DEFAULT
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

opt_opfamily:
	  FAMILY any_name
	{
		$$ = toStrSlice($2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opclass_purpose:
	  FOR SEARCH
	{
		$$ = nil
	}
	| FOR ORDER BY any_name
	{
		$$ = toStrSlice($4)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

CreateOpFamilyStmt:
	  CREATE OPERATOR FAMILY any_name USING name
	{
		$$ = &CreateOpFamilyStmt{Opfamilyname: toPgStringNodes(toStrSlice($4)), Amname: toStr($6)}
	}
	;

AlterOpFamilyStmt:
	  ALTER OPERATOR FAMILY any_name USING name ADD_P opclass_item_list
	{
		$$ = &AlterOpFamilyStmt{Opfamilyname: toPgStringNodes(toStrSlice($4)), Amname: toStr($6), IsDrop: false, Items: toNodeSlice($8)}
	}
	| ALTER OPERATOR FAMILY any_name USING name DROP opclass_drop_list
	{
		$$ = &AlterOpFamilyStmt{Opfamilyname: toPgStringNodes(toStrSlice($4)), Amname: toStr($6), IsDrop: true, Items: toNodeSlice($8)}
	}
	;

opclass_drop_list:
	  opclass_drop
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| opclass_drop_list ',' opclass_drop
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

opclass_drop:
	  OPERATOR Iconst '(' type_list ')'
	{
		$$ = &CreateOpClassItem{Itemtype: 1, Number: int32(toInt($2)), ClassArgs: toNodeSlice(makeTypeNameNodes($4))}
	}
	| FUNCTION Iconst '(' type_list ')'
	{
		$$ = &CreateOpClassItem{Itemtype: 2, Number: int32(toInt($2)), ClassArgs: toNodeSlice(makeTypeNameNodes($4))}
	}
	;

DropOpClassStmt:
	  DROP OPERATOR CLASS any_name USING name opt_drop_behavior
	{
		$$ = makeDropOpStmt("operator class", $4, $6, false, $7)
	}
	| DROP OPERATOR CLASS IF_P EXISTS any_name USING name opt_drop_behavior
	{
		$$ = makeDropOpStmt("operator class", $6, $8, true, $9)
	}
	;

DropOpFamilyStmt:
	  DROP OPERATOR FAMILY any_name USING name opt_drop_behavior
	{
		$$ = makeDropOpStmt("operator family", $4, $6, false, $7)
	}
	| DROP OPERATOR FAMILY IF_P EXISTS any_name USING name opt_drop_behavior
	{
		$$ = makeDropOpStmt("operator family", $6, $8, true, $9)
	}
	;

DropOwnedStmt:
	  DROP OWNED BY role_list opt_drop_behavior
	{
		$$ = makeDropOwnedStmt($4, $5)
	}
	;

ReassignOwnedStmt:
	  REASSIGN OWNED BY role_list TO RoleSpec
	{
		$$ = makeReassignOwnedStmt($4, $6)
	}
	;

DropStmt:
	  DROP object_type_any_name IF_P EXISTS any_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	| DROP object_type_any_name any_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: false, Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP drop_type_name IF_P EXISTS name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	| DROP drop_type_name name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: false, Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP object_type_name_on_any_name name ON any_name opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: false, Behavior: toPgDropBehavior(toBool($6)), Objects: []Node{toNode(makeDropOnObject($3, $5))}}
	}
	| DROP object_type_name_on_any_name IF_P EXISTS name ON any_name opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType(toStr($2)), MissingOk: true, Behavior: toPgDropBehavior(toBool($8)), Objects: []Node{toNode(makeDropOnObject($5, $7))}}
	}
	| DROP TYPE_P type_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("type"), MissingOk: false, Behavior: toPgDropBehavior(toBool($4)), Objects: toNodeSlice($3)}
	}
	| DROP TYPE_P IF_P EXISTS type_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("type"), MissingOk: true, Behavior: toPgDropBehavior(toBool($6)), Objects: toNodeSlice($5)}
	}
	| DROP DOMAIN_P type_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("domain"), MissingOk: false, Behavior: toPgDropBehavior(toBool($4)), Objects: toNodeSlice($3)}
	}
	| DROP DOMAIN_P IF_P EXISTS type_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("domain"), MissingOk: true, Behavior: toPgDropBehavior(toBool($6)), Objects: toNodeSlice($5)}
	}
	| DROP INDEX CONCURRENTLY any_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("index"), Concurrent: true, MissingOk: false, Objects: toNodeSlice($4), Behavior: toPgDropBehavior(toBool($5))}
	}
	| DROP INDEX CONCURRENTLY IF_P EXISTS any_name_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("index"), Concurrent: true, MissingOk: true, Objects: toNodeSlice($6), Behavior: toPgDropBehavior(toBool($7))}
	}
	;

object_type_any_name:
	  TABLE
	{
		$$ = "table"
	}
	| SEQUENCE
	{
		$$ = "sequence"
	}
	| VIEW
	{
		$$ = "view"
	}
	| MATERIALIZED VIEW
	{
		$$ = "materialized view"
	}
	| INDEX
	{
		$$ = "index"
	}
	| FOREIGN TABLE
	{
		$$ = "foreign table"
	}
	| COLLATION
	{
		$$ = "collation"
	}
	| CONVERSION_P
	{
		$$ = "conversion"
	}
	| STATISTICS
	{
		$$ = "statistics"
	}
	| TEXT_P SEARCH PARSER
	{
		$$ = "text search parser"
	}
	| TEXT_P SEARCH DICTIONARY
	{
		$$ = "text search dictionary"
	}
	| TEXT_P SEARCH TEMPLATE
	{
		$$ = "text search template"
	}
	| TEXT_P SEARCH CONFIGURATION
	{
		$$ = "text search configuration"
	}
	;

object_type_name:
	  drop_type_name
	{
		$$ = $1
	}
	| DATABASE
	{
		$$ = "database"
	}
	| ROLE
	{
		$$ = "role"
	}
	| SUBSCRIPTION
	{
		$$ = "subscription"
	}
	| TABLESPACE
	{
		$$ = "tablespace"
	}
	;

drop_type_name:
	  ACCESS METHOD
	{
		$$ = "access method"
	}
	| EVENT TRIGGER
	{
		$$ = "event trigger"
	}
	| EXTENSION
	{
		$$ = "extension"
	}
	| FOREIGN DATA_P WRAPPER
	{
		$$ = "foreign data wrapper"
	}
	| opt_procedural LANGUAGE
	{
		$$ = "language"
	}
	| PUBLICATION
	{
		$$ = "publication"
	}
	| SCHEMA
	{
		$$ = "schema"
	}
	| SERVER
	{
		$$ = "server"
	}
	;

object_type_name_on_any_name:
	  POLICY
	{
		$$ = "policy"
	}
	| RULE
	{
		$$ = "rule"
	}
	| TRIGGER
	{
		$$ = "trigger"
	}
	;

any_name_list:
	  any_name
	{
		$$ = []Node{toNode(&List{Items: toPgStringNodes(toStrSlice($1))})}
	}
	| any_name_list ',' any_name
	{
		$$ = append(toNodeSlice($1), toNode(&List{Items: toPgStringNodes(toStrSlice($3))}))
	}
	;

any_name:
	  ColId
	{
		$$ = []string{toStr($1)}
	}
	| ColId attrs
	{
		$$ = append([]string{toStr($1)}, toStrSlice($2)...)
	}
	;

attrs:
	  '.' attr_name
	{
		$$ = []string{toStr($2)}
	}
	| attrs '.' attr_name
	{
		$$ = appendToStrSlice($1, $3)
	}
	;

type_name_list:
	  Typename
	{
		$$ = []Node{toNode(toTypeNameNode($1))}
	}
	| type_name_list ',' Typename
	{
		$$ = appendToNodeSlice($1, toTypeNameNode($3))
	}
	;

TruncateStmt:
	  TRUNCATE opt_table relation_expr_list opt_restart_seqs opt_drop_behavior
	{
		$$ = &TruncateStmt{Relations: toNodeSlice(toRangeVarSlice($3)), RestartSeqs: toBool($4), Behavior: toPgDropBehavior(toBool($5))}
	}
	;

opt_restart_seqs:
	  CONTINUE_P IDENTITY_P
	{
		$$ = false
	}
	| RESTART IDENTITY_P
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

CommentStmt:
	  COMMENT ON object_type_any_name any_name IS comment_text
	{
		$$ = makeCommentStmt($3, $4, $6)
	}
	| COMMENT ON COLUMN any_name IS comment_text
	{
		$$ = makeCommentStmt("column", $4, $6)
	}
	| COMMENT ON object_type_name name IS comment_text
	{
		$$ = makeCommentStmt($3, $4, $6)
	}
	| COMMENT ON TYPE_P Typename IS comment_text
	{
		$$ = makeCommentStmt("type", $4, $6)
	}
	| COMMENT ON DOMAIN_P Typename IS comment_text
	{
		$$ = makeCommentStmt("domain", $4, $6)
	}
	| COMMENT ON AGGREGATE aggregate_with_argtypes IS comment_text
	{
		$$ = makeCommentStmt("aggregate", $4, $6)
	}
	| COMMENT ON FUNCTION function_with_argtypes IS comment_text
	{
		$$ = makeCommentStmt("function", $4, $6)
	}
	| COMMENT ON OPERATOR operator_with_argtypes IS comment_text
	{
		$$ = makeCommentStmt("operator", $4, $6)
	}
	| COMMENT ON CONSTRAINT name ON any_name IS comment_text
	{
		$$ = makeCommentStmt("constraint", append(toStrSlice($6), toStr($4)), $8)
	}
	| COMMENT ON CONSTRAINT name ON DOMAIN_P any_name IS comment_text
	{
		$$ = makeCommentStmt("dom_constraint", &List{Items: []Node{toNode(&TypeName{Names: toPgStringNodes(toStrSlice($7)), Typemod: -1, Location: -1}),toNode(&String{Sval: toStr($4)})}}, $9)
	}
	| COMMENT ON object_type_name_on_any_name name ON any_name IS comment_text
	{
		$$ = makeCommentStmt($3, append(toStrSlice($6), toStr($4)), $8)
	}
	| COMMENT ON PROCEDURE function_with_argtypes IS comment_text
	{
		$$ = makeCommentStmt("procedure", $4, $6)
	}
	| COMMENT ON ROUTINE function_with_argtypes IS comment_text
	{
		$$ = makeCommentStmt("routine", $4, $6)
	}
	| COMMENT ON TRANSFORM FOR Typename LANGUAGE name IS comment_text
	{
		$$ = makeCommentStmt("transform", &List{Items: []Node{toNode($5), toNode(&String{Sval: toStr($7)})}}, $9)
	}
	| COMMENT ON OPERATOR CLASS any_name USING name IS comment_text
	{
		$$ = makeCommentStmt("operator class", append([]string{toStr($7)}, toStrSlice($5)...), $9)
	}
	| COMMENT ON OPERATOR FAMILY any_name USING name IS comment_text
	{
		$$ = makeCommentStmt("operator family", append([]string{toStr($7)}, toStrSlice($5)...), $9)
	}
	| COMMENT ON LARGE_P OBJECT_P NumericOnly IS comment_text
	{
		$$ = makeCommentStmt("large object", $5, $7)
	}
	| COMMENT ON CAST '(' Typename AS Typename ')' IS comment_text
	{
		$$ = makeCommentStmt("cast", []any{$5, $7}, $10)
	}
	;

comment_text:
	  Sconst
	{
		$$ = $1
	}
	| NULL_P
	{
		$$ = nil
	}
	;

SecLabelStmt:
	  SECURITY LABEL opt_provider ON object_type_any_name any_name 			IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType(toStr($5)), Object: toNode(toAnyNameNode($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON COLUMN any_name 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("COLUMN"), Object: toNode(toAnyNameNode($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON object_type_name name 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType(toStr($5)), Object: toNode(&String{Sval: toStr($6)}), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON TYPE_P Typename 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("TYPE"), Object: toNode(toTypeNameNode($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON DOMAIN_P Typename 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("DOMAIN"), Object: toNode(toTypeNameNode($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON AGGREGATE aggregate_with_argtypes 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("AGGREGATE"), Object: toNode(toObjectWithArgs($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON FUNCTION function_with_argtypes 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("FUNCTION"), Object: toNode(toObjectWithArgs($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON LARGE_P OBJECT_P NumericOnly 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("LARGEOBJECT"), Object: toNode(toNode($7)), Label: toStr($9)}
	}
	| SECURITY LABEL opt_provider ON PROCEDURE function_with_argtypes 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("PROCEDURE"), Object: toNode(toObjectWithArgs($6)), Label: toStr($8)}
	}
	| SECURITY LABEL opt_provider ON ROUTINE function_with_argtypes 			  IS security_label
	{
		$$ = &SecLabelStmt{Provider: toStr($3), Objtype: enumObjectType("ROUTINE"), Object: toNode(toObjectWithArgs($6)), Label: toStr($8)}
	}
	;

opt_provider:
	  FOR NonReservedWord_or_Sconst
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

security_label:
	  Sconst
	{
		$$ = toStr($1)
	}
	| NULL_P
	{
		$$ = nil
	}
	;

FetchStmt:
	  FETCH fetch_args
	{
		$$ = $2
	}
	| MOVE fetch_args
	{
		if f, ok := $2.(*FetchStmt); ok {
			f.Ismove = true
		}
		$$ = $2
	}
	;

fetch_args:
	  cursor_name
	{
		$$ = makeFetchStmt("FORWARD", int64(1), $1, false)
	}
	| from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", int64(1), $2, false)
	}
	| NEXT opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", int64(1), $3, false)
	}
	| PRIOR opt_from_in cursor_name
	{
		$$ = makeFetchStmt("BACKWARD", int64(1), $3, false)
	}
	| FIRST_P opt_from_in cursor_name
	{
		$$ = makeFetchStmt("ABSOLUTE", int64(1), $3, false)
	}
	| LAST_P opt_from_in cursor_name
	{
		$$ = makeFetchStmt("ABSOLUTE", int64(-1), $3, false)
	}
	| ABSOLUTE_P SignedIconst opt_from_in cursor_name
	{
		$$ = makeFetchStmt("ABSOLUTE", $2, $4, false)
	}
	| RELATIVE_P SignedIconst opt_from_in cursor_name
	{
		$$ = makeFetchStmt("RELATIVE", $2, $4, false)
	}
	| SignedIconst opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", $1, $3, false)
	}
	| ALL opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", "ALL", $3, false)
	}
	| FORWARD opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", int64(1), $3, false)
	}
	| FORWARD SignedIconst opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", $2, $4, false)
	}
	| FORWARD ALL opt_from_in cursor_name
	{
		$$ = makeFetchStmt("FORWARD", "ALL", $4, false)
	}
	| BACKWARD opt_from_in cursor_name
	{
		$$ = makeFetchStmt("BACKWARD", int64(1), $3, false)
	}
	| BACKWARD SignedIconst opt_from_in cursor_name
	{
		$$ = makeFetchStmt("BACKWARD", $2, $4, false)
	}
	| BACKWARD ALL opt_from_in cursor_name
	{
		$$ = makeFetchStmt("BACKWARD", "ALL", $4, false)
	}
	;

from_in:
	  FROM
	{
		$$ = $1
	}
	| IN_P
	{
		$$ = $1
	}
	;

opt_from_in:
	  from_in
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

GrantStmt:
	  GRANT privileges ON privilege_target TO grantee_list opt_grant_grant_option opt_granted_by
	{
		st := makeGrantStmt(true, "", "", $4, $2, $6, toBool($7), "")
		st.Grantor = toRoleSpec($8)
		$$ = st
	}
	;

RevokeStmt:
	  REVOKE privileges ON privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
	{
		st := makeGrantStmt(false, "", "", $4, $2, $6, false, toStr($8))
		st.Grantor = toRoleSpec($7)
		$$ = st
	}
	| REVOKE GRANT OPTION FOR privileges ON privilege_target FROM grantee_list opt_granted_by opt_drop_behavior
	{
		st := makeGrantStmt(false, "", "", $7, $5, $9, true, toStr($11))
		st.Grantor = toRoleSpec($10)
		$$ = st
	}
	;

privileges:
	  privilege_list
	{
		$$ = $1
	}
	| ALL
	{
		$$ = nil
	}
	| ALL PRIVILEGES
	{
		$$ = nil
	}
	| ALL '(' columnList ')'
	{
		$$ = &AccessPriv{Cols: toNodeSlice($3)}
	}
	| ALL PRIVILEGES '(' columnList ')'
	{
		$$ = &AccessPriv{Cols: toNodeSlice($4)}
	}
	;

privilege_list:
	  privilege
	{
		$$ = $1
	}
	| privilege_list ',' privilege
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

privilege:
	  SELECT opt_column_list
	{
		$$ = &AccessPriv{PrivName: "select", Cols: toNodeSlice($2)}
	}
	| REFERENCES opt_column_list
	{
		$$ = &AccessPriv{PrivName: "references", Cols: toNodeSlice($2)}
	}
	| CREATE opt_column_list
	{
		$$ = &AccessPriv{PrivName: "create", Cols: toNodeSlice($2)}
	}
	| ALTER SYSTEM_P
	{
		$$ = &AccessPriv{PrivName: "alter system"}
	}
	| ColId opt_column_list
	{
		$$ = &AccessPriv{PrivName: toStr($1), Cols: toNodeSlice($2)}
	}
	;

parameter_name_list:
	  parameter_name
	{
		$$ = $1
	}
	| parameter_name_list ',' parameter_name
	{
		$$ = appendToStrSlice($1, $3)
	}
	;

parameter_name:
	  ColId
	{
		$$ = $1
	}
	| parameter_name '.' ColId
	{
		$$ = toStr($1) + "." + toStr($3)
	}
	;

privilege_target:
	  qualified_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("TABLE"), Objects: toNodeSlice($1), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| TABLE qualified_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("TABLE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| SEQUENCE qualified_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("SEQUENCE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| FOREIGN DATA_P WRAPPER name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("FDW"), Objects: toNodeSlice($4), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| FOREIGN SERVER name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("FOREIGN_SERVER"), Objects: toNodeSlice($3), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| FUNCTION function_with_argtypes_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("FUNCTION"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| PROCEDURE function_with_argtypes_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("PROCEDURE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| ROUTINE function_with_argtypes_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("ROUTINE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| DATABASE name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("DATABASE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| DOMAIN_P any_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("DOMAIN"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| LANGUAGE name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("LANGUAGE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| LARGE_P OBJECT_P NumericOnly_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("LARGEOBJECT"), Objects: toNodeSlice($3), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| PARAMETER parameter_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("PARAMETER_ACL"), Objects: toNodeSlice(toStrSlice($2)), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| SCHEMA name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("SCHEMA"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| TABLESPACE name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("TABLESPACE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| TYPE_P any_name_list
	{
		$$ = &GrantStmt{Objtype: enumObjectType("TYPE"), Objects: toNodeSlice($2), Targtype: pg_query.GrantTargetType(1), Behavior: pg_query.DropBehavior(1)}
	}
	| ALL TABLES IN_P SCHEMA name_list
	{
		$$ = &GrantStmt{Targtype: enumGrantTargetType("SCHEMA"), Objtype: enumObjectType("TABLE"), Objects: toNodeSlice($5), Behavior: pg_query.DropBehavior(1)}
	}
	| ALL SEQUENCES IN_P SCHEMA name_list
	{
		$$ = &GrantStmt{Targtype: enumGrantTargetType("SCHEMA"), Objtype: enumObjectType("SEQUENCE"), Objects: toNodeSlice($5), Behavior: pg_query.DropBehavior(1)}
	}
	| ALL FUNCTIONS IN_P SCHEMA name_list
	{
		$$ = &GrantStmt{Targtype: enumGrantTargetType("SCHEMA"), Objtype: enumObjectType("FUNCTION"), Objects: toNodeSlice($5), Behavior: pg_query.DropBehavior(1)}
	}
	| ALL PROCEDURES IN_P SCHEMA name_list
	{
		$$ = &GrantStmt{Targtype: enumGrantTargetType("SCHEMA"), Objtype: enumObjectType("PROCEDURE"), Objects: toNodeSlice($5), Behavior: pg_query.DropBehavior(1)}
	}
	| ALL ROUTINES IN_P SCHEMA name_list
	{
		$$ = &GrantStmt{Targtype: enumGrantTargetType("SCHEMA"), Objtype: enumObjectType("ROUTINE"), Objects: toNodeSlice($5), Behavior: pg_query.DropBehavior(1)}
	}
	;

grantee_list:
	  grantee
	{
		$$ = []Node{toNode(toRoleSpec($1))}
	}
	| grantee_list ',' grantee
	{
		$$ = append(toRoleSpecSlice($1), toNode(toRoleSpec($3)))
	}
	;

grantee:
	  RoleSpec
	{
		$$ = toRoleSpec($1)
	}
	| GROUP_P RoleSpec
	{
		$$ = toRoleSpec($2)
	}
	;

opt_grant_grant_option:
	  WITH GRANT OPTION
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

GrantRoleStmt:
	  GRANT privilege_list TO role_list opt_granted_by
	{
		$$ = makeGrantRoleStmt($2, $4, true, nil, nil, $5)
	}
	| GRANT privilege_list TO role_list WITH grant_role_opt_list opt_granted_by
	{
		$$ = makeGrantRoleStmt($2, $4, true, $6, nil, $7)
	}
	;

RevokeRoleStmt:
	  REVOKE privilege_list FROM role_list opt_granted_by opt_drop_behavior
	{
		$$ = makeGrantRoleStmt($2, $4, false, nil, $6, $5)
	}
	| REVOKE ColId OPTION FOR privilege_list FROM role_list opt_granted_by opt_drop_behavior
	{
		option := makeDefElem($2, &Boolean{Boolval: false})
		option.Location = int32(@2)
		$$ = makeGrantRoleStmt($5, $7, false, option, $9, $8)
	}
	;

grant_role_opt_list:
	  grant_role_opt_list ',' grant_role_opt
	{
		$$ = appendToDefElemSlice($1, $3)
	}
	| grant_role_opt
	{
		$$ = $1
	}
	;

grant_role_opt:
	  ColLabel grant_role_opt_value
	{
		$$ = makeDefElem($1, &Boolean{Boolval: toBool($2)})
		grammarSetLocation($$, @1)
	}
	;

grant_role_opt_value:
	  OPTION
	{
		$$ = true
	}
	| TRUE_P
	{
		$$ = true
	}
	| FALSE_P
	{
		$$ = false
	}
	;

opt_granted_by:
	  GRANTED BY RoleSpec
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterDefaultPrivilegesStmt:
	  ALTER DEFAULT PRIVILEGES DefACLOptionList DefACLAction
	{
		$$ = &AlterDefaultPrivilegesStmt{Options: toNodeSlice(toDefElemSlice($4)), Action: $5.(*GrantStmt)}
	}
	;

DefACLOptionList:
	  DefACLOptionList DefACLOption
	{
		$$ = append(toDefElemSlice($1), toNode($2.(*DefElem)))
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

DefACLOption:
	  IN_P SCHEMA name_list
	{
		items := make([]Node, len(toStrSlice($3)))
		for i, s := range toStrSlice($3) {
			items[i] = toNode(&String{Sval: s})
		}
		$$ = &DefElem{Defname: "schemas", Arg: toNode(&List{Items: toNodeSlice(items)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| FOR ROLE role_list
	{
		$$ = &DefElem{Defname: "roles", Arg: toNode(&List{Items: toNodeSlice($3)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| FOR USER role_list
	{
		$$ = &DefElem{Defname: "roles", Arg: toNode(&List{Items: toNodeSlice($3)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

DefACLAction:
	  GRANT privileges ON defacl_privilege_target TO grantee_list 			opt_grant_grant_option
	{
		$$ = &GrantStmt{IsGrant: true, Privileges: toNodeSlice($2), Targtype: enumGrantTargetType("ACL_TARGET_DEFAULTS"), Objtype: enumObjectType(toStr($4)), Grantees: toNodeSlice(toRoleSpecSlice($6)), GrantOption: toBool($7), Behavior: pg_query.DropBehavior(1)}
	}
	| REVOKE privileges ON defacl_privilege_target 			FROM grantee_list opt_drop_behavior
	{
		$$ = &GrantStmt{IsGrant: false, GrantOption: false, Privileges: toNodeSlice($2), Targtype: enumGrantTargetType("ACL_TARGET_DEFAULTS"), Objtype: enumObjectType(toStr($4)), Grantees: toNodeSlice(toRoleSpecSlice($6)), Behavior: enumDropBehavior(toStr($7))}
	}
	| REVOKE GRANT OPTION FOR privileges ON defacl_privilege_target 			FROM grantee_list opt_drop_behavior
	{
		$$ = &GrantStmt{IsGrant: false, GrantOption: true, Privileges: toNodeSlice($5), Targtype: enumGrantTargetType("ACL_TARGET_DEFAULTS"), Objtype: enumObjectType(toStr($7)), Grantees: toNodeSlice(toRoleSpecSlice($9)), Behavior: enumDropBehavior(toStr($10))}
	}
	;

defacl_privilege_target:
	  TABLES
	{
		$$ = "OBJECT_TABLE"
	}
	| FUNCTIONS
	{
		$$ = "OBJECT_FUNCTION"
	}
	| ROUTINES
	{
		$$ = "OBJECT_FUNCTION"
	}
	| SEQUENCES
	{
		$$ = "OBJECT_SEQUENCE"
	}
	| TYPES_P
	{
		$$ = "OBJECT_TYPE"
	}
	| SCHEMAS
	{
		$$ = "OBJECT_SCHEMA"
	}
	| LARGE_P OBJECTS_P
	{
		$$ = "OBJECT_LARGEOBJECT"
	}
	;

IndexStmt:
	  CREATE opt_unique INDEX opt_concurrently opt_single_name 			ON relation_expr access_method_clause '(' index_params ')' 			opt_include opt_unique_null_treatment opt_reloptions OptTableSpace where_clause
	{
		$$ = &IndexStmt{Unique: $2 != nil, Concurrent: $4 != nil, Idxname: toStr($5), Relation: toRangeVar($7), AccessMethod: toStr($8), IndexParams: toNodeSlice(toIndexElemSlice($10)), IndexIncludingParams: toNodeSlice(toIndexElemSlice($12)), NullsNotDistinct: toBool($13), Options: toNodeSlice(toDefElemSlice($14)), TableSpace: toStr($15), WhereClause: toNode($16)}
	}
	| CREATE opt_unique INDEX opt_concurrently IF_P NOT EXISTS name 			ON relation_expr access_method_clause '(' index_params ')' 			opt_include opt_unique_null_treatment opt_reloptions OptTableSpace where_clause
	{
		$$ = &IndexStmt{Unique: $2 != nil, Concurrent: $4 != nil, IfNotExists: true, Idxname: toStr($8), Relation: toRangeVar($10), AccessMethod: toStr($11), IndexParams: toNodeSlice(toIndexElemSlice($13)), IndexIncludingParams: toNodeSlice(toIndexElemSlice($15)), NullsNotDistinct: toBool($16), Options: toNodeSlice(toDefElemSlice($17)), TableSpace: toStr($18), WhereClause: toNode($19)}
	}
	;

opt_unique:
	  UNIQUE
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

access_method_clause:
	  USING name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = "btree"
	}
	;

index_params:
	  index_elem
	{
		$$ = toIndexElemSlice($1)
	}
	| index_params ',' index_elem
	{
		$$ = appendToIndexElemSlice($1, $3)
	}
	;

index_elem_options:
	  opt_collate opt_qualified_name opt_asc_desc opt_nulls_order
	{
		dir := SortByDirDefault
		if $3 != nil { dir = $3.(SortByDir) }
		nulls := SortByNullsDefault
		if $4 != nil { nulls = $4.(SortByNulls) }
		$$ = &IndexElemOptions{Ordering: dir, NullsOrdering: nulls, Collation: toStrSlice($1), Opclass: toStrSlice($2)}
	}
	| opt_collate any_name reloptions opt_asc_desc opt_nulls_order
	{
		dir := SortByDirDefault
		if $4 != nil { dir = $4.(SortByDir) }
		nulls := SortByNullsDefault
		if $5 != nil { nulls = $5.(SortByNulls) }
		$$ = &IndexElemOptions{Ordering: dir, NullsOrdering: nulls, Collation: toStrSlice($1), Opclass: toStrSlice($2), OpclassOptions: toDefElemSlice($3)}
	}
	;

index_elem:
	  ColId index_elem_options
	{
		$$ = toIndexElem($1, nil, $2)
	}
	| func_expr_windowless index_elem_options
	{
		$$ = toIndexElem(nil, $1, $2)
	}
	| '(' a_expr ')' index_elem_options
	{
		$$ = toIndexElem(nil, $2, $4)
	}
	;

opt_include:
	  INCLUDE '(' index_including_params ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

index_including_params:
	  index_elem
	{
		$$ = toIndexElemSlice($1)
	}
	| index_including_params ',' index_elem
	{
		$$ = appendToIndexElemSlice($1, $3)
	}
	;

opt_collate:
	  COLLATE any_name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_asc_desc:
	  ASC
	{
		$$ = SortByAsc
	}
	| DESC
	{
		$$ = SortByDesc
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_nulls_order:
	  NULLS_LA FIRST_P
	{
		$$ = SortByNullsFirst
	}
	| NULLS_LA LAST_P
	{
		$$ = SortByNullsLast
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

CreateFunctionStmt:
	  CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	  RETURNS func_return opt_createfunc_opt_list opt_routine_body
	{
		$$ = &CreateFunctionStmt{IsProcedure: false, Replace: toBool($2), Funcname: toPgStringNodes(toStrSlice($4)), Parameters: toNodeSlice($5), ReturnType: toTypeNameNode($7), Options: toNodeSlice($8), SqlBody: toNode($9)}
	}
	| CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	  RETURNS TABLE '(' table_func_column_list ')' opt_createfunc_opt_list opt_routine_body
	{
		for _, argument := range toNodeSlice($5) {
			mode := argument.GetFunctionParameter().Mode
			if mode != enumFunctionParameterMode("") && mode != enumFunctionParameterMode("in") && mode != enumFunctionParameterMode("variadic") {
				yylex.Error("OUT and INOUT arguments aren't allowed in TABLE functions")
				return 1
			}
		}
		returnType := tableFunctionReturnType($9)
		returnType.Location = int32(@7)
		$$ = &CreateFunctionStmt{IsProcedure: false, Replace: toBool($2), Funcname: toPgStringNodes(toStrSlice($4)), Parameters: toNodeSlice(append(toNodeSlice($5), toNodeSlice($9)...)), ReturnType: returnType, Options: toNodeSlice($11), SqlBody: toNode($12)}
	}
	| CREATE opt_or_replace FUNCTION func_name func_args_with_defaults
	  opt_createfunc_opt_list opt_routine_body
	{
		$$ = &CreateFunctionStmt{IsProcedure: false, Replace: toBool($2), Funcname: toPgStringNodes(toStrSlice($4)), Parameters: toNodeSlice($5), Options: toNodeSlice($6), SqlBody: toNode($7)}
	}
	| CREATE opt_or_replace PROCEDURE func_name func_args_with_defaults
	  opt_createfunc_opt_list opt_routine_body
	{
		$$ = &CreateFunctionStmt{IsProcedure: true, Replace: toBool($2), Funcname: toPgStringNodes(toStrSlice($4)), Parameters: toNodeSlice($5), Options: toNodeSlice($6), SqlBody: toNode($7)}
	}
	;

opt_or_replace:
	  OR REPLACE
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

func_args:
	  '(' func_args_list ')'
	{
		$$ = $2
	}
	| '(' ')'
	{
		$$ = []Node{}
	}
	;

func_args_list:
	  func_arg
	{
		$$ = []Node{toNode($1)}
	}
	| func_args_list ',' func_arg
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

function_with_argtypes_list:
	  function_with_argtypes
	{
		$$ = []Node{toNode($1)}
	}
	| function_with_argtypes_list ',' function_with_argtypes
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

function_with_argtypes:
	  func_name func_args
	{
		$$ = makeFuncWithArgs($1, $2)
	}
	| type_func_name_keyword
	{
		$$ = makeFuncWithArgsUnspecified([]string{toStr($1)})
	}
	| ColId
	{
		$$ = makeFuncWithArgsUnspecified([]string{toStr($1)})
	}
	| ColId indirection
	{
		$$ = makeFuncWithArgsUnspecified(append([]string{toStr($1)}, toStrSlice($2)...))
	}
	;

func_args_with_defaults:
	  '(' func_args_with_defaults_list ')'
	{
		$$ = $2
	}
	| '(' ')'
	{
		$$ = []Node{}
	}
	;

func_args_with_defaults_list:
	  func_arg_with_default
	{
		$$ = []Node{toNode($1)}
	}
	| func_args_with_defaults_list ',' func_arg_with_default
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

func_arg:
	  arg_class param_name func_type
	{
		$$ = &FunctionParameter{Mode: enumFunctionParameterMode(toStr($1)), Name: toStr($2), ArgType: toTypeNameNode($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| param_name arg_class func_type
	{
		$$ = &FunctionParameter{Name: toStr($1), Mode: enumFunctionParameterMode(toStr($2)), ArgType: toTypeNameNode($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| param_name func_type
	{
		$$ = &FunctionParameter{Name: toStr($1), ArgType: toTypeNameNode($2), Location: -1, Mode: pg_query.FunctionParameterMode_FUNC_PARAM_DEFAULT}
		grammarSetLocation($$, @1)
	}
	| arg_class func_type
	{
		$$ = &FunctionParameter{Mode: enumFunctionParameterMode(toStr($1)), ArgType: toTypeNameNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| func_type
	{
		$$ = &FunctionParameter{ArgType: toTypeNameNode($1), Location: -1, Mode: pg_query.FunctionParameterMode_FUNC_PARAM_DEFAULT}
		grammarSetLocation($$, @1)
	}
	;

arg_class:
	  IN_P
	{
		$$ = "in"
	}
	| OUT_P
	{
		$$ = "out"
	}
	| INOUT
	{
		$$ = "inout"
	}
	| IN_P OUT_P
	{
		$$ = "inout"
	}
	| VARIADIC
	{
		$$ = "variadic"
	}
	;

param_name:
	  type_function_name
	{
		$$ = $1
	}
	;

func_return:
	  func_type
	{
		$$ = $1
	}
	;

func_type:
	  Typename
	{
		$$ = $1
	}
	| type_function_name attrs '%' TYPE_P
	{
		$$ = &TypeName{Names: toPgStringNodes(append([]string{toStr($1)}, toStrSlice($2)...)), Typemod: -1, PctType: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| SETOF type_function_name attrs '%' TYPE_P
	{
		$$ = &TypeName{Names: toPgStringNodes(append([]string{toStr($2)}, toStrSlice($3)...)), Typemod: -1, PctType: true, Setof: true, Location: -1}
		grammarSetLocation($$, @2)
	}
	;

func_arg_with_default:
	  func_arg
	{
		$$ = $1
	}
	| func_arg DEFAULT a_expr
	{
		fp := $1.(*FunctionParameter)
		fp.Defexpr = toNode($3)
		$$ = fp
	}
	| func_arg '=' a_expr
	{
		fp := $1.(*FunctionParameter)
		fp.Defexpr = toNode($3)
		$$ = fp
	}
	;

aggr_arg:
	  func_arg
	{
		fp := $1.(*FunctionParameter)
		if fp.Mode != enumFunctionParameterMode("") && fp.Mode != enumFunctionParameterMode("in") && fp.Mode != enumFunctionParameterMode("variadic") {
			yylex.(*Lexer).errorAt("aggregate arguments cannot be OUT or INOUT", @1)
			return 1
		}
		$$ = fp
	}
	;

aggr_args:
	  '(' '*' ')'
	{
		$$ = &aggrArgsInfo{args: nil, numDirectArgs: -1}
	}
	| '(' aggr_args_list ')'
	{
		$$ = &aggrArgsInfo{args: toNodeSlice($2), numDirectArgs: -1}
	}
	| '(' ORDER BY aggr_args_list ')'
	{
		$$ = &aggrArgsInfo{args: toNodeSlice($4), numDirectArgs: 0}
	}
	| '(' aggr_args_list ORDER BY aggr_args_list ')'
	{
		args, err := makeOrderedAggregateArgs(toNodeSlice($2), toNodeSlice($5))
		if err != nil {
			yylex.Error(err.Error())
			return 1
		}
		$$ = args
	}
	;

aggr_args_list:
	  aggr_arg
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| aggr_args_list ',' aggr_arg
	{
		list := toNodeSlice($1)
		if $3 != nil {
			list = append(list, toNode($3))
		}
		$$ = list
	}
	;

aggregate_with_argtypes:
	  func_name aggr_args
	{
		$$ = makeFuncWithArgs($1, $2)
	}
	;

aggregate_with_argtypes_list:
	  aggregate_with_argtypes
	{
		$$ = []Node{toNode($1)}
	}
	| aggregate_with_argtypes_list ',' aggregate_with_argtypes
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

opt_createfunc_opt_list:
	  createfunc_opt_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

createfunc_opt_list:
	  createfunc_opt_item
	{
		$$ = []Node{toNode($1)}
	}
	| createfunc_opt_list createfunc_opt_item
	{
		$$ = append(toNodeSlice($1), toNode($2))
	}
	;

common_func_opt_item:
	  CALLED ON NULL_P INPUT_P
	{
		$$ = &DefElem{Defname: "strict", Arg: toNode(&Boolean{Boolval: false}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| RETURNS NULL_P ON NULL_P INPUT_P
	{
		$$ = &DefElem{Defname: "strict", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| STRICT_P
	{
		$$ = &DefElem{Defname: "strict", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| IMMUTABLE
	{
		$$ = &DefElem{Defname: "volatility", Arg: toNode(&String{Sval: "immutable"}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| STABLE
	{
		$$ = &DefElem{Defname: "volatility", Arg: toNode(&String{Sval: "stable"}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| VOLATILE
	{
		$$ = &DefElem{Defname: "volatility", Arg: toNode(&String{Sval: "volatile"}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| EXTERNAL SECURITY DEFINER
	{
		$$ = &DefElem{Defname: "security", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| EXTERNAL SECURITY INVOKER
	{
		$$ = &DefElem{Defname: "security", Arg: toNode(&Boolean{Boolval: false}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| SECURITY DEFINER
	{
		$$ = &DefElem{Defname: "security", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| SECURITY INVOKER
	{
		$$ = &DefElem{Defname: "security", Arg: toNode(&Boolean{Boolval: false}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| LEAKPROOF
	{
		$$ = &DefElem{Defname: "leakproof", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| NOT LEAKPROOF
	{
		$$ = &DefElem{Defname: "leakproof", Arg: toNode(&Boolean{Boolval: false}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| COST NumericOnly
	{
		$$ = &DefElem{Defname: "cost", Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ROWS NumericOnly
	{
		$$ = &DefElem{Defname: "rows", Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| SUPPORT any_name
	{
		$$ = &DefElem{Defname: "support", Arg: toNode(&List{Items: toNodeSlice($2)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| FunctionSetResetClause
	{
		$$ = &DefElem{Defname: "set", Arg: toNode($1), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| PARALLEL ColId
	{
		$$ = &DefElem{Defname: "parallel", Arg: toNode(&String{Sval: toStr($2)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

createfunc_opt_item:
	  AS func_as
	{
		$$ = &DefElem{Defname: "as", Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| LANGUAGE NonReservedWord_or_Sconst
	{
		$$ = &DefElem{Defname: "language", Arg: toNode(&String{Sval: toStr($2)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| TRANSFORM transform_type_list
	{
		$$ = &DefElem{Defname: "transform", Arg: toNode(&List{Items: toNodeSlice($2)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| WINDOW
	{
		$$ = &DefElem{Defname: "window", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| common_func_opt_item
	{
		$$ = $1
	}
	;

func_as:
	  Sconst
	{
		$$ = &List{Items: []Node{toNode(&String{Sval: toStr($1)})}}
	}
	| Sconst ',' Sconst
	{
		$$ = &List{Items: []Node{toNode(&String{Sval: toStr($1)}),toNode(&String{Sval: toStr($3)})}}
	}
	;

ReturnStmt:
	  RETURN a_expr
	{
		$$ = &ReturnStmt{Returnval: toNode($2)}
	}
	;

opt_routine_body:
	  ReturnStmt
	{
		$$ = $1
	}
	| BEGIN_P ATOMIC routine_body_stmt_list END_P
	{
		var body Node
		if statements := toNodeSlice($3); len(statements) != 0 {
			body = toNode(&List{Items: statements})
		}
		$$ = &List{Items: []Node{toNode(body)}}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

routine_body_stmt_list:
	  routine_body_stmt_list routine_body_stmt ';'
	{
		$$ = appendToNodeSlice($1, $2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

routine_body_stmt:
	  stmt
	{
		$$ = $1
	}
	| ReturnStmt
	{
		$$ = $1
	}
	;

transform_type_list:
	  FOR TYPE_P Typename
	{
		$$ = []Node{toNode(toTypeNameNode($3))}
	}
	| transform_type_list ',' FOR TYPE_P Typename
	{
		$$ = append(toNodeSlice($1), toNode(toTypeNameNode($5)))
	}
	;

opt_definition:
	  WITH definition
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

table_func_column:
	  param_name func_type
	{
		$$ = &FunctionParameter{Name: toStr($1), ArgType: toTypeNameNode($2), Mode: enumFunctionParameterMode("table"), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

table_func_column_list:
	  table_func_column
	{
		$$ = []Node{toNode($1)}
	}
	| table_func_column_list ',' table_func_column
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

AlterFunctionStmt:
	  ALTER FUNCTION function_with_argtypes alterfunc_opt_list opt_restrict
	{
		$$ = &AlterFunctionStmt{Objtype: enumObjectType("OBJECT_FUNCTION"), Func: toObjectWithArgs($3), Actions: toNodeSlice($4)}
	}
	| ALTER PROCEDURE function_with_argtypes alterfunc_opt_list opt_restrict
	{
		$$ = &AlterFunctionStmt{Objtype: enumObjectType("OBJECT_PROCEDURE"), Func: toObjectWithArgs($3), Actions: toNodeSlice($4)}
	}
	| ALTER ROUTINE function_with_argtypes alterfunc_opt_list opt_restrict
	{
		$$ = &AlterFunctionStmt{Objtype: enumObjectType("OBJECT_ROUTINE"), Func: toObjectWithArgs($3), Actions: toNodeSlice($4)}
	}
	;

alterfunc_opt_list:
	  common_func_opt_item
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| alterfunc_opt_list common_func_opt_item
	{
		$$ = append(toNodeSlice($1), toNode($2))
	}
	;

opt_restrict:
	  RESTRICT
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

RemoveFuncStmt:
	  DROP FUNCTION function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("function"), Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP FUNCTION IF_P EXISTS function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("function"), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	| DROP PROCEDURE function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("procedure"), Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP PROCEDURE IF_P EXISTS function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("procedure"), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	| DROP ROUTINE function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("routine"), Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP ROUTINE IF_P EXISTS function_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("routine"), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	;

RemoveAggrStmt:
	  DROP AGGREGATE aggregate_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("aggregate"), Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP AGGREGATE IF_P EXISTS aggregate_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("aggregate"), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	;

RemoveOperStmt:
	  DROP OPERATOR operator_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("operator"), Objects: toNodeSlice($3), Behavior: toPgDropBehavior(toBool($4))}
	}
	| DROP OPERATOR IF_P EXISTS operator_with_argtypes_list opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("operator"), MissingOk: true, Objects: toNodeSlice($5), Behavior: toPgDropBehavior(toBool($6))}
	}
	;

oper_argtypes:
	  '(' Typename ')'
	{
		yylex.(*Lexer).errorAt("operator argument types must be specified with two arguments", @3)
		return 1
	}
	| '(' Typename ',' Typename ')'
	{
		$$ = []Node{toNode(toTypeNameNode($2)),toNode(toTypeNameNode($4))}
	}
	| '(' NONE ',' Typename ')'
	{
		$$ = []Node{nil,toNode(toTypeNameNode($4))}
	}
	| '(' Typename ',' NONE ')'
	{
		$$ = []Node{toNode(toTypeNameNode($2)), nil}
	}
	;

any_operator:
	  all_Op
	{
		$$ = []string{toStr($1)}
	}
	| ColId '.' any_operator
	{
		$$ = append([]string{toStr($1)}, toStrSlice($3)...)
	}
	;

operator_with_argtypes_list:
	  operator_with_argtypes
	{
		$$ = []Node{toNode($1)}
	}
	| operator_with_argtypes_list ',' operator_with_argtypes
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

operator_with_argtypes:
	  any_operator oper_argtypes
	{
		$$ = &ObjectWithArgs{Objname: toPgStringNodes(toStrSlice($1)), Objargs: toNodeSlice($2)}
	}
	;

DoStmt:
	  DO dostmt_opt_list
	{
		$$ = &DoStmt{Args: toNodeSlice(toDefElemSlice($2))}
	}
	;

dostmt_opt_list:
	  dostmt_opt_item
	{
		$$ = singleDefElemSlice($1)
	}
	| dostmt_opt_list dostmt_opt_item
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	;

dostmt_opt_item:
	  Sconst
	{
		$$ = makeDefElem("as", &String{Sval: toStr($1)})
		grammarSetLocation($$, @1)
	}
	| LANGUAGE NonReservedWord_or_Sconst
	{
		$$ = makeDefElem("language", &String{Sval: toStr($2)})
		grammarSetLocation($$, @1)
	}
	;

CreateCastStmt:
	  CREATE CAST '(' Typename AS Typename ')' 					WITH FUNCTION function_with_argtypes cast_context
	{
		$$ = &CreateCastStmt{Sourcetype: toTypeNameNode($4), Targettype: toTypeNameNode($6), Func: toObjectWithArgs($10), Context: pg_query.CoercionContext(toInt32($11)), Inout: false}
	}
	| CREATE CAST '(' Typename AS Typename ')' 					WITHOUT FUNCTION cast_context
	{
		$$ = &CreateCastStmt{Sourcetype: toTypeNameNode($4), Targettype: toTypeNameNode($6), Func: nil, Context: pg_query.CoercionContext(toInt32($10)), Inout: false}
	}
	| CREATE CAST '(' Typename AS Typename ')' 					WITH INOUT cast_context
	{
		$$ = &CreateCastStmt{Sourcetype: toTypeNameNode($4), Targettype: toTypeNameNode($6), Func: nil, Context: pg_query.CoercionContext(toInt32($10)), Inout: true}
	}
	;

cast_context:
	  AS IMPLICIT_P
	{
		$$ = int32(1)
	}
	| AS ASSIGNMENT
	{
		$$ = int32(2)
	}
	| /* EMPTY */
	{
		$$ = int32(4)
	}
	;

DropCastStmt:
	  DROP CAST opt_if_exists '(' Typename AS Typename ')' opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("OBJECT_CAST"), Objects: []Node{toNode(&List{Items: []Node{toNode(toTypeNameNode($5)),toNode(toTypeNameNode($7))}}),
			}, MissingOk: toBool($3), Behavior: toPgDropBehavior(strings.EqualFold(toStr($9), "CASCADE"))}
	}
	;

opt_if_exists:
	  IF_P EXISTS
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

CreateTransformStmt:
	  CREATE opt_or_replace TRANSFORM FOR Typename LANGUAGE name '(' transform_element_list ')'
	{
		elems := toTransformElements($9)
		$$ = &CreateTransformStmt{Replace: toBool($2), TypeName: toTypeNameNode($5), Lang: toStr($7), Fromsql: elems.fromsql, Tosql: elems.tosql}
	}
	;

transform_element_list:
	  FROM SQL_P WITH FUNCTION function_with_argtypes ',' TO SQL_P WITH FUNCTION function_with_argtypes
	{
		$$ = &transformElements{fromsql: toObjectWithArgs($5), tosql: toObjectWithArgs($11)}
	}
	| TO SQL_P WITH FUNCTION function_with_argtypes ',' FROM SQL_P WITH FUNCTION function_with_argtypes
	{
		$$ = &transformElements{fromsql: toObjectWithArgs($11), tosql: toObjectWithArgs($5)}
	}
	| FROM SQL_P WITH FUNCTION function_with_argtypes
	{
		$$ = &transformElements{fromsql: toObjectWithArgs($5), tosql: nil}
	}
	| TO SQL_P WITH FUNCTION function_with_argtypes
	{
		$$ = &transformElements{fromsql: nil, tosql: toObjectWithArgs($5)}
	}
	;

DropTransformStmt:
	  DROP TRANSFORM opt_if_exists FOR Typename LANGUAGE name opt_drop_behavior
	{
		$$ = &DropStmt{RemoveType: enumObjectType("OBJECT_TRANSFORM"), Objects: []Node{toNode(&List{Items: []Node{toNode(toTypeNameNode($5)),toNode(&String{Sval: toStr($7)})}})}, MissingOk: toBool($3), Behavior: toPgDropBehavior(strings.EqualFold(toStr($8), "CASCADE"))}
	}
	;

ReindexStmt:
	  REINDEX opt_reindex_option_list reindex_target_relation opt_concurrently qualified_name
	{
		n := makeReindexStmt(toStr($3), $5, nil, $2)
		if $4 != nil { n.Params = append(n.Params, toNode(&DefElem{Defname: "concurrently", Location: int32(@4), Defaction: pg_query.DefElemAction(1)}))}
		$$ = n
	}
	| REINDEX opt_reindex_option_list SCHEMA opt_concurrently name
	{
		n := makeReindexStmt("SCHEMA", nil, $5, $2)
		if $4 != nil { n.Params = append(n.Params, toNode(&DefElem{Defname: "concurrently", Location: int32(@4), Defaction: pg_query.DefElemAction(1)}))}
		$$ = n
	}
	| REINDEX opt_reindex_option_list reindex_target_all opt_concurrently opt_single_name
	{
		n := makeReindexStmt(toStr($3), nil, $5, $2)
		if $4 != nil { n.Params = append(n.Params, toNode(&DefElem{Defname: "concurrently", Location: int32(@4), Defaction: pg_query.DefElemAction(1)}))}
		$$ = n
	}
	;

reindex_target_relation:
	  INDEX
	{
		$$ = $1
	}
	| TABLE
	{
		$$ = $1
	}
	;

reindex_target_all:
	  SYSTEM_P
	{
		$$ = $1
	}
	| DATABASE
	{
		$$ = $1
	}
	;

opt_reindex_option_list:
	  '(' utility_option_list ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterTblSpcStmt:
	  ALTER TABLESPACE name SET reloptions
	{
		$$ = &AlterTableSpaceOptionsStmt{Tablespacename: toStr($3), Options: toNodeSlice(toDefElemSlice($5)), IsReset: false}
	}
	| ALTER TABLESPACE name RESET reloptions
	{
		$$ = &AlterTableSpaceOptionsStmt{Tablespacename: toStr($3), Options: toNodeSlice(toDefElemSlice($5)), IsReset: true}
	}
	;

RenameStmt:
	  ALTER AGGREGATE aggregate_with_argtypes RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_AGGREGATE", "", nil, $3, "", $6, nil, false)
	}
	| ALTER COLLATION any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_COLLATION", "", nil, $3, "", $6, nil, false)
	}
	| ALTER CONVERSION_P any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_CONVERSION", "", nil, $3, "", $6, nil, false)
	}
	| ALTER DATABASE name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_DATABASE", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER DOMAIN_P any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_DOMAIN", "", nil, $3, "", $6, nil, false)
	}
	| ALTER DOMAIN_P any_name RENAME CONSTRAINT name TO name
	{
		$$ = makeRenameStmt("OBJECT_DOMCONSTRAINT", "", nil, $3, $6, $8, nil, false)
	}
	| ALTER FOREIGN DATA_P WRAPPER name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_FDW", "", nil, &String{Sval: toStr($5)}, "", $8, nil, false)
	}
	| ALTER FUNCTION function_with_argtypes RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_FUNCTION", "", nil, $3, "", $6, nil, false)
	}
	| ALTER GROUP_P RoleId RENAME TO RoleId
	{
		$$ = makeRenameStmt("OBJECT_ROLE", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER opt_procedural LANGUAGE name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_LANGUAGE", "", nil, &String{Sval: toStr($4)}, "", $7, nil, false)
	}
	| ALTER OPERATOR CLASS any_name USING name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_OPCLASS", "", nil, &List{Items: toNodeSlice(append([]Node{toNode(&String{Sval: toStr($6)})}, toNodeSlice($4)...))}, "", $9, nil, false)
	}
	| ALTER OPERATOR FAMILY any_name USING name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_OPFAMILY", "", nil, &List{Items: toNodeSlice(append([]Node{toNode(&String{Sval: toStr($6)})}, toNodeSlice($4)...))}, "", $9, nil, false)
	}
	| ALTER POLICY name ON qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_POLICY", "", $5, nil, $3, $8, nil, false)
	}
	| ALTER POLICY IF_P EXISTS name ON qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_POLICY", "", $7, nil, $5, $10, nil, true)
	}
	| ALTER PROCEDURE function_with_argtypes RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_PROCEDURE", "", nil, $3, "", $6, nil, false)
	}
	| ALTER PUBLICATION name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_PUBLICATION", "", nil, &String{Sval: toStr($3)}, "", $6, nil, false)
	}
	| ALTER ROUTINE function_with_argtypes RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_ROUTINE", "", nil, $3, "", $6, nil, false)
	}
	| ALTER SCHEMA name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_SCHEMA", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER SERVER name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_FOREIGN_SERVER", "", nil, &String{Sval: toStr($3)}, "", $6, nil, false)
	}
	| ALTER SUBSCRIPTION name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_SUBSCRIPTION", "", nil, &String{Sval: toStr($3)}, "", $6, nil, false)
	}
	| ALTER TABLE relation_expr RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TABLE", "", $3, nil, "", $6, nil, false)
	}
	| ALTER TABLE IF_P EXISTS relation_expr RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TABLE", "", $5, nil, "", $8, nil, true)
	}
	| ALTER SEQUENCE qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_SEQUENCE", "", $3, nil, "", $6, nil, false)
	}
	| ALTER SEQUENCE IF_P EXISTS qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_SEQUENCE", "", $5, nil, "", $8, nil, true)
	}
	| ALTER VIEW qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_VIEW", "", $3, nil, "", $6, nil, false)
	}
	| ALTER VIEW IF_P EXISTS qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_VIEW", "", $5, nil, "", $8, nil, true)
	}
	| ALTER MATERIALIZED VIEW qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_MATVIEW", "", $4, nil, "", $7, nil, false)
	}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_MATVIEW", "", $6, nil, "", $9, nil, true)
	}
	| ALTER INDEX qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_INDEX", "", $3, nil, "", $6, nil, false)
	}
	| ALTER INDEX IF_P EXISTS qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_INDEX", "", $5, nil, "", $8, nil, true)
	}
	| ALTER FOREIGN TABLE relation_expr RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_FOREIGN_TABLE", "", $4, nil, "", $7, nil, false)
	}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_FOREIGN_TABLE", "", $6, nil, "", $9, nil, true)
	}
	| ALTER TABLE relation_expr RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_TABLE", $3, nil, $6, $8, nil, false)
	}
	| ALTER TABLE IF_P EXISTS relation_expr RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_TABLE", $5, nil, $8, $10, nil, true)
	}
	| ALTER VIEW qualified_name RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_VIEW", $3, nil, $6, $8, nil, false)
	}
	| ALTER VIEW IF_P EXISTS qualified_name RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_VIEW", $5, nil, $8, $10, nil, true)
	}
	| ALTER MATERIALIZED VIEW qualified_name RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_MATVIEW", $4, nil, $7, $9, nil, false)
	}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_MATVIEW", $6, nil, $9, $11, nil, true)
	}
	| ALTER TABLE relation_expr RENAME CONSTRAINT name TO name
	{
		$$ = makeRenameStmt("OBJECT_TABCONSTRAINT", "", $3, nil, $6, $8, nil, false)
	}
	| ALTER TABLE IF_P EXISTS relation_expr RENAME CONSTRAINT name TO name
	{
		$$ = makeRenameStmt("OBJECT_TABCONSTRAINT", "", $5, nil, $8, $10, nil, true)
	}
	| ALTER FOREIGN TABLE relation_expr RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_FOREIGN_TABLE", $4, nil, $7, $9, nil, false)
	}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr RENAME opt_column name TO name
	{
		$$ = makeRenameStmt("OBJECT_COLUMN", "OBJECT_FOREIGN_TABLE", $6, nil, $9, $11, nil, true)
	}
	| ALTER RULE name ON qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_RULE", "", $5, nil, $3, $8, nil, false)
	}
	| ALTER TRIGGER name ON qualified_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TRIGGER", "", $5, nil, $3, $8, nil, false)
	}
	| ALTER EVENT TRIGGER name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_EVENT_TRIGGER", "", nil, &String{Sval: toStr($4)}, "", $7, nil, false)
	}
	| ALTER ROLE RoleId RENAME TO RoleId
	{
		$$ = makeRenameStmt("OBJECT_ROLE", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER USER RoleId RENAME TO RoleId
	{
		$$ = makeRenameStmt("OBJECT_ROLE", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER TABLESPACE name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TABLESPACE", "", nil, nil, $3, $6, nil, false)
	}
	| ALTER STATISTICS any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_STATISTIC_EXT", "", nil, $3, "", $6, nil, false)
	}
	| ALTER TEXT_P SEARCH PARSER any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TSPARSER", "", nil, $5, "", $8, nil, false)
	}
	| ALTER TEXT_P SEARCH DICTIONARY any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TSDICTIONARY", "", nil, $5, "", $8, nil, false)
	}
	| ALTER TEXT_P SEARCH TEMPLATE any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TSTEMPLATE", "", nil, $5, "", $8, nil, false)
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TSCONFIGURATION", "", nil, $5, "", $8, nil, false)
	}
	| ALTER TYPE_P any_name RENAME TO name
	{
		$$ = makeRenameStmt("OBJECT_TYPE", "", nil, $3, "", $6, nil, false)
	}
	| ALTER TYPE_P any_name RENAME ATTRIBUTE name TO name opt_drop_behavior
	{
		relation := toRangeVar($3)
		relation.Inh = false
		relation.Location = int32(@3)
		$$ = makeRenameStmt("OBJECT_ATTRIBUTE", "OBJECT_TYPE", relation, nil, $6, $8, $9, false)
	}
	;

opt_column:
	  COLUMN
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_set_data:
	  SET DATA_P
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;
AlterObjectDependsStmt:
  ALTER FUNCTION function_with_argtypes opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("function"), Object: toNode($3), Extname: &String{Sval: toStr($8)}, Remove: toBool($4)}
}
| ALTER PROCEDURE function_with_argtypes opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("procedure"), Object: toNode($3), Extname: &String{Sval: toStr($8)}, Remove: toBool($4)}
}
| ALTER ROUTINE function_with_argtypes opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("routine"), Object: toNode($3), Extname: &String{Sval: toStr($8)}, Remove: toBool($4)}
}
| ALTER TRIGGER name ON qualified_name opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("trigger"), Relation: toRangeVar($5), Object: toNode(&List{Items: []Node{toNode(&String{Sval: toStr($3)})}}), Extname: &String{Sval: toStr($10)}, Remove: toBool($6)}
}
| ALTER MATERIALIZED VIEW qualified_name opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("matview"), Relation: toRangeVar($4), Extname: &String{Sval: toStr($9)}, Remove: toBool($5)}
}
| ALTER INDEX qualified_name opt_no DEPENDS ON EXTENSION name
{
	$$ = &AlterObjectDependsStmt{ObjectType: enumObjectType("index"), Relation: toRangeVar($3), Extname: &String{Sval: toStr($8)}, Remove: toBool($4)}
}
;

opt_no:
  NO
{
	$$ = true
}
| /* EMPTY */
{
	$$ = false
}
;

AlterObjectSchemaStmt:
	  ALTER AGGREGATE aggregate_with_argtypes SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("aggregate"), Object: toNode($3), Newschema: toStr($6)}
	}
	| ALTER COLLATION any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("collation"), Object: toNode(toCommentTarget($3)), Newschema: toStr($6)}
	}
	| ALTER CONVERSION_P any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("conversion"), Object: toNode(toCommentTarget($3)), Newschema: toStr($6)}
	}
	| ALTER DOMAIN_P any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("domain"), Object: toNode(toCommentTarget($3)), Newschema: toStr($6)}
	}
	| ALTER EXTENSION name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("extension"), Object: toNode(&String{Sval: toStr($3)}), Newschema: toStr($6)}
	}
	| ALTER FUNCTION function_with_argtypes SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("function"), Object: toNode($3), Newschema: toStr($6)}
	}
	| ALTER OPERATOR operator_with_argtypes SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("operator"), Object: toNode($3), Newschema: toStr($6)}
	}
	| ALTER OPERATOR CLASS any_name USING name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("operator class"), Object: toNode(toCommentTarget(append([]string{toStr($6)}, toStrSlice($4)...))), Newschema: toStr($9)}
	}
	| ALTER OPERATOR FAMILY any_name USING name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("operator family"), Object: toNode(toCommentTarget(append([]string{toStr($6)}, toStrSlice($4)...))), Newschema: toStr($9)}
	}
	| ALTER PROCEDURE function_with_argtypes SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("procedure"), Object: toNode($3), Newschema: toStr($6)}
	}
	| ALTER ROUTINE function_with_argtypes SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("routine"), Object: toNode($3), Newschema: toStr($6)}
	}
	| ALTER TABLE relation_expr SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("table"), Relation: toRangeVar($3), Newschema: toStr($6)}
	}
	| ALTER TABLE IF_P EXISTS relation_expr SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("table"), Relation: toRangeVar($5), MissingOk: true, Newschema: toStr($8)}
	}
	| ALTER STATISTICS any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("statistics"), Object: toNode(toCommentTarget($3)), Newschema: toStr($6)}
	}
	| ALTER TEXT_P SEARCH PARSER any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("tsparser"), Object: toNode(toCommentTarget($5)), Newschema: toStr($8)}
	}
	| ALTER TEXT_P SEARCH DICTIONARY any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("tsdictionary"), Object: toNode(toCommentTarget($5)), Newschema: toStr($8)}
	}
	| ALTER TEXT_P SEARCH TEMPLATE any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("tstemplate"), Object: toNode(toCommentTarget($5)), Newschema: toStr($8)}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("tsconfiguration"), Object: toNode(toCommentTarget($5)), Newschema: toStr($8)}
	}
	| ALTER SEQUENCE qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("sequence"), Relation: toRangeVar($3), Newschema: toStr($6)}
	}
	| ALTER SEQUENCE IF_P EXISTS qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("sequence"), Relation: toRangeVar($5), MissingOk: true, Newschema: toStr($8)}
	}
	| ALTER VIEW qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("view"), Relation: toRangeVar($3), Newschema: toStr($6)}
	}
	| ALTER VIEW IF_P EXISTS qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("view"), Relation: toRangeVar($5), MissingOk: true, Newschema: toStr($8)}
	}
	| ALTER MATERIALIZED VIEW qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("materialized view"), Relation: toRangeVar($4), Newschema: toStr($7)}
	}
	| ALTER MATERIALIZED VIEW IF_P EXISTS qualified_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("materialized view"), Relation: toRangeVar($6), MissingOk: true, Newschema: toStr($9)}
	}
	| ALTER FOREIGN TABLE relation_expr SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("foreign table"), Relation: toRangeVar($4), Newschema: toStr($7)}
	}
	| ALTER FOREIGN TABLE IF_P EXISTS relation_expr SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("foreign table"), Relation: toRangeVar($6), MissingOk: true, Newschema: toStr($9)}
	}
	| ALTER TYPE_P any_name SET SCHEMA name
	{
		$$ = &AlterObjectSchemaStmt{ObjectType: enumObjectType("type"), Object: toNode(toCommentTarget($3)), Newschema: toStr($6)}
	}
	;

AlterOperatorStmt:
	  ALTER OPERATOR operator_with_argtypes SET '(' operator_def_list ')'
	{
		$$ = &AlterOperatorStmt{Opername: toObjectWithArgs($3), Options: toNodeSlice($6)}
	}
	;

operator_def_list:
	  operator_def_elem
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| operator_def_list ',' operator_def_elem
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

operator_def_elem:
	  ColLabel '=' NONE
	{
		$$ = &DefElem{Defname: toStr($1), Arg: nil, Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel '=' operator_def_arg
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode($3), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| ColLabel
	{
		$$ = &DefElem{Defname: toStr($1), Arg: nil, Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

operator_def_arg:
	  func_type
	{
		$$ = toTypeNameNode($1)
	}
	| reserved_keyword
	{
		$$ = &String{Sval: toStr($1)}
	}
	| qual_all_Op
	{
		$$ = makeDefElemOperatorArg($1)
	}
	| NumericOnly
	{
		$$ = toNode($1)
	}
	| Sconst
	{
		$$ = &String{Sval: toStr($1)}
	}
	;

AlterTypeStmt:
	  ALTER TYPE_P any_name SET '(' operator_def_list ')'
	{
		$$ = &AlterTypeStmt{TypeName: toPgStringNodes(toStrSlice($3)), Options: toNodeSlice($6)}
	}
	;

AlterOwnerStmt:
	  ALTER AGGREGATE aggregate_with_argtypes OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_AGGREGATE", nil, $3, $6)
	}
	| ALTER COLLATION any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_COLLATION", nil, $3, $6)
	}
	| ALTER CONVERSION_P any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_CONVERSION", nil, $3, $6)
	}
	| ALTER DATABASE name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_DATABASE", nil, &String{Sval: toStr($3)}, $6)
	}
	| ALTER DOMAIN_P any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_DOMAIN", nil, $3, $6)
	}
	| ALTER FUNCTION function_with_argtypes OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_FUNCTION", nil, $3, $6)
	}
	| ALTER opt_procedural LANGUAGE name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_LANGUAGE", nil, &String{Sval: toStr($4)}, $7)
	}
	| ALTER LARGE_P OBJECT_P NumericOnly OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_LARGEOBJECT", nil, $4, $7)
	}
	| ALTER OPERATOR operator_with_argtypes OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_OPERATOR", nil, $3, $6)
	}
	| ALTER OPERATOR CLASS any_name USING name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_OPCLASS", nil, append([]string{toStr($6)}, toStrSlice($4)...), $9)
	}
	| ALTER OPERATOR FAMILY any_name USING name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_OPFAMILY", nil, append([]string{toStr($6)}, toStrSlice($4)...), $9)
	}
	| ALTER PROCEDURE function_with_argtypes OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_PROCEDURE", nil, $3, $6)
	}
	| ALTER ROUTINE function_with_argtypes OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_ROUTINE", nil, $3, $6)
	}
	| ALTER SCHEMA name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_SCHEMA", nil, &String{Sval: toStr($3)}, $6)
	}
	| ALTER TYPE_P any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_TYPE", nil, $3, $6)
	}
	| ALTER TABLESPACE name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_TABLESPACE", nil, &String{Sval: toStr($3)}, $6)
	}
	| ALTER STATISTICS any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_STATISTIC_EXT", nil, $3, $6)
	}
	| ALTER TEXT_P SEARCH DICTIONARY any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_TSDICTIONARY", nil, $5, $8)
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_TSCONFIGURATION", nil, $5, $8)
	}
	| ALTER FOREIGN DATA_P WRAPPER name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_FDW", nil, &String{Sval: toStr($5)}, $8)
	}
	| ALTER SERVER name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_FOREIGN_SERVER", nil, &String{Sval: toStr($3)}, $6)
	}
	| ALTER EVENT TRIGGER name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_EVENT_TRIGGER", nil, &String{Sval: toStr($4)}, $7)
	}
	| ALTER PUBLICATION name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_PUBLICATION", nil, &String{Sval: toStr($3)}, $6)
	}
	| ALTER SUBSCRIPTION name OWNER TO RoleSpec
	{
		$$ = makeAlterOwnerStmt("OBJECT_SUBSCRIPTION", nil, &String{Sval: toStr($3)}, $6)
	}
	;

CreatePublicationStmt:
	  CREATE PUBLICATION name opt_definition
	{
		$$ = makeCreatePublicationStmt($3, $4, nil, false)
	}
	| CREATE PUBLICATION name FOR ALL TABLES opt_definition
	{
		$$ = makeCreatePublicationStmt($3, $7, nil, true)
	}
	| CREATE PUBLICATION name FOR pub_obj_list opt_definition
	{
		if !preprocessPublicationObjects(yylex, toNodeSlice($5)) { return 1 }
		$$ = makeCreatePublicationStmt($3, $6, $5, false)
	}
	;

PublicationObjSpec:
	  TABLE relation_expr opt_column_list OptWhereClause
	{
		$$ = makePublicationObjSpec(1, "", makePublicationTable($2, $4, $3))
	}
	| TABLES IN_P SCHEMA ColId
	{
		$$ = makePublicationObjSpec(2, toStr($4), nil)
		grammarSetLocation($$, @4)
	}
	| TABLES IN_P SCHEMA CURRENT_SCHEMA
	{
		$$ = makePublicationObjSpec(3, "", nil)
		grammarSetLocation($$, @4)
	}
	| ColId opt_column_list OptWhereClause
	{
		if $2 != nil || $3 != nil {
			$$ = makePublicationObjSpec(4, "", makePublicationTable(&RangeVar{Relname: toStr($1), Inh: true, Location: int32(@1), Relpersistence: "p"}, $3, $2))
		} else {
			$$ = makePublicationObjSpec(4, toStr($1), nil)
		}
		grammarSetLocation($$, @1)
	}
	| ColId indirection opt_column_list OptWhereClause
	{
		parts := toNodeSlice($2)
		if len(parts) > 2 { yylex.Error("improper qualified name (too many dotted names)"); return 1 }
		for _, part := range parts { if part.GetString_() == nil { yylex.Error("invalid relation name"); return 1 } }
		relation := makeRangeVarWithIndirection($1, $2)
		relation.Location = int32(@1)
		$$ = makePublicationObjSpec(4, "", makePublicationTable(relation, $4, $3))
		grammarSetLocation($$, @1)
	}
	| extended_relation_expr opt_column_list OptWhereClause
	{
		$$ = makePublicationObjSpec(4, "", makePublicationTable(toRangeVar($1), $3, $2))
	}
	| CURRENT_SCHEMA
	{
		$$ = makePublicationObjSpec(4, "", nil)
		grammarSetLocation($$, @1)
	}
	;

pub_obj_list:
	  PublicationObjSpec
	{
		$$ = []Node{toNode($1)}
	}
	| pub_obj_list ',' PublicationObjSpec
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

AlterPublicationStmt:
	  ALTER PUBLICATION name SET definition
	{
		$$ = makeAlterPublicationStmt($3, $5, nil, false, "")
	}
	| ALTER PUBLICATION name ADD_P pub_obj_list
	{
		if !preprocessPublicationObjects(yylex, toNodeSlice($5)) { return 1 }
		$$ = makeAlterPublicationStmt($3, nil, $5, false, "ADD")
	}
	| ALTER PUBLICATION name SET pub_obj_list
	{
		if !preprocessPublicationObjects(yylex, toNodeSlice($5)) { return 1 }
		$$ = makeAlterPublicationStmt($3, nil, $5, false, "SET")
	}
	| ALTER PUBLICATION name DROP pub_obj_list
	{
		if !preprocessPublicationObjects(yylex, toNodeSlice($5)) { return 1 }
		$$ = makeAlterPublicationStmt($3, nil, $5, false, "DROP")
	}
	;

CreateSubscriptionStmt:
	  CREATE SUBSCRIPTION name CONNECTION Sconst PUBLICATION name_list opt_definition
	{
		$$ = &CreateSubscriptionStmt{Subname: toStr($3), Conninfo: toStr($5), Publication: toPgStringNodes(toStrSlice($7)), Options: toNodeSlice($8)}
	}
	;

AlterSubscriptionStmt:
	  ALTER SUBSCRIPTION name SET definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("OPTIONS"), Subname: toStr($3), Options: toNodeSlice($5)}
	}
	| ALTER SUBSCRIPTION name CONNECTION Sconst
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("CONNECTION"), Subname: toStr($3), Conninfo: toStr($5)}
	}
	| ALTER SUBSCRIPTION name REFRESH PUBLICATION opt_definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("REFRESH"), Subname: toStr($3), Options: toNodeSlice($6)}
	}
	| ALTER SUBSCRIPTION name ADD_P PUBLICATION name_list opt_definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("ADD_PUBLICATION"), Subname: toStr($3), Publication: toPgStringNodes(toStrSlice($6)), Options: toNodeSlice($7)}
	}
	| ALTER SUBSCRIPTION name DROP PUBLICATION name_list opt_definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("DROP_PUBLICATION"), Subname: toStr($3), Publication: toPgStringNodes(toStrSlice($6)), Options: toNodeSlice($7)}
	}
	| ALTER SUBSCRIPTION name SET PUBLICATION name_list opt_definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("SET_PUBLICATION"), Subname: toStr($3), Publication: toPgStringNodes(toStrSlice($6)), Options: toNodeSlice($7)}
	}
	| ALTER SUBSCRIPTION name ENABLE_P
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("ENABLED"), Subname: toStr($3), Options: []Node{toNode(&DefElem{Defname: "enabled", Arg: toNode(&Boolean{Boolval: true}), Location: int32(@1), Defaction: pg_query.DefElemAction(1)})}}
	}
	| ALTER SUBSCRIPTION name DISABLE_P
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("ENABLED"), Subname: toStr($3), Options: []Node{toNode(&DefElem{Defname: "enabled", Arg: toNode(&Boolean{Boolval: false}), Location: int32(@1), Defaction: pg_query.DefElemAction(1)})}}
	}
	| ALTER SUBSCRIPTION name SKIP definition
	{
		$$ = &AlterSubscriptionStmt{Kind: enumAlterSubscriptionType("SKIP"), Subname: toStr($3), Options: toNodeSlice($5)}
	}
	;

DropSubscriptionStmt:
	  DROP SUBSCRIPTION name opt_drop_behavior
	{
		$$ = makeDropSubscriptionStmt(toStr($3), false, $4)
	}
	| DROP SUBSCRIPTION IF_P EXISTS name opt_drop_behavior
	{
		$$ = makeDropSubscriptionStmt(toStr($5), true, $6)
	}
	;

RuleStmt:
	  CREATE opt_or_replace RULE name AS ON event TO qualified_name where_clause DO opt_instead RuleActionList
	{
		$$ = makeRuleStmt($9, $4, $10, toStr($7), toBool($12), $13, toBool($2))
	}
	;

RuleActionList:
	  NOTHING
	{
		$$ = nil
	}
	| RuleActionStmt
	{
		$$ = $1
	}
	| '(' RuleActionMulti ')'
	{
		$$ = $2
	}
	;

RuleActionMulti:
	  RuleActionMulti ';' RuleActionStmtOrEmpty
	{
		$$ = append(toNodeSlice($1), toNodeSlice($3)...)
	}
	| RuleActionStmtOrEmpty
	{
		$$ = $1
	}
	;

RuleActionStmt:
	  SelectStmt
	{
		$$ = $1
	}
	| InsertStmt
	{
		$$ = $1
	}
	| UpdateStmt
	{
		$$ = $1
	}
	| DeleteStmt
	{
		$$ = $1
	}
	| NotifyStmt
	{
		$$ = $1
	}
	;

RuleActionStmtOrEmpty:
	  RuleActionStmt
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

event:
	  SELECT
	{
		$$ = $1
	}
	| UPDATE
	{
		$$ = $1
	}
	| DELETE_P
	{
		$$ = $1
	}
	| INSERT
	{
		$$ = $1
	}
	;

opt_instead:
	  INSTEAD
	{
		$$ = true
	}
	| ALSO
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

NotifyStmt:
	  NOTIFY ColId notify_payload
	{
		$$ = &NotifyStmt{Conditionname: toStr($2), Payload: toStr($3)}
	}
	;

notify_payload:
	  ',' Sconst
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ListenStmt:
	  LISTEN ColId
	{
		$$ = &ListenStmt{Conditionname: toStr($2)}
	}
	;

UnlistenStmt:
	  UNLISTEN ColId
	{
		$$ = &UnlistenStmt{Conditionname: toStr($2)}
	}
	| UNLISTEN '*'
	{
		$$ = &UnlistenStmt{Conditionname: ""}
	}
	;

TransactionStmt:
	  ABORT_P opt_transaction opt_transaction_chain
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("ROLLBACK"), Chain: toBool($3), Location: -1}
	}
	| START TRANSACTION transaction_mode_list_or_empty
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("START"), Options: toNodeSlice(toDefElemSlice($3)), Location: -1}
	}
	| COMMIT opt_transaction opt_transaction_chain
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("COMMIT"), Chain: toBool($3), Location: -1}
	}
	| ROLLBACK opt_transaction opt_transaction_chain
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("ROLLBACK"), Chain: toBool($3), Location: -1}
	}
	| SAVEPOINT ColId
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("SAVEPOINT"), SavepointName: toStr($2), Location: -1}
		grammarSetLocation($$, @2)
	}
	| RELEASE SAVEPOINT ColId
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("RELEASE"), SavepointName: toStr($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| RELEASE ColId
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("RELEASE"), SavepointName: toStr($2), Location: -1}
		grammarSetLocation($$, @2)
	}
	| ROLLBACK opt_transaction TO SAVEPOINT ColId
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("ROLLBACK_TO"), SavepointName: toStr($5), Location: -1}
		grammarSetLocation($$, @5)
	}
	| ROLLBACK opt_transaction TO ColId
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("ROLLBACK_TO"), SavepointName: toStr($4), Location: -1}
		grammarSetLocation($$, @4)
	}
	| PREPARE TRANSACTION Sconst
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("PREPARE"), Gid: toStr($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| COMMIT PREPARED Sconst
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("COMMIT_PREPARED"), Gid: toStr($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| ROLLBACK PREPARED Sconst
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("ROLLBACK_PREPARED"), Gid: toStr($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	;

TransactionStmtLegacy:
	  BEGIN_P opt_transaction transaction_mode_list_or_empty
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("BEGIN"), Options: toNodeSlice(toDefElemSlice($3)), Location: -1}
	}
	| END_P opt_transaction opt_transaction_chain
	{
		$$ = &TransactionStmt{Kind: enumTransactionStmtKind("COMMIT"), Chain: toBool($3), Location: -1}
	}
	;

opt_transaction:
	  WORK
	{
		$$ = $1
	}
	| TRANSACTION
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

transaction_mode_item:
	  ISOLATION LEVEL iso_level
	{
		$$ = makeDefElem("transaction_isolation", grammarConst(toStr($3), @3))
		grammarSetLocation($$, @1)
	}
	| READ ONLY
	{
		$$ = makeDefElem("transaction_read_only", grammarConst(int64(1), @1))
		grammarSetLocation($$, @1)
	}
	| READ WRITE
	{
		$$ = makeDefElem("transaction_read_only", grammarConst(int64(0), @1))
		grammarSetLocation($$, @1)
	}
	| DEFERRABLE
	{
		$$ = makeDefElem("transaction_deferrable", grammarConst(int64(1), @1))
		grammarSetLocation($$, @1)
	}
	| NOT DEFERRABLE
	{
		$$ = makeDefElem("transaction_deferrable", grammarConst(int64(0), @1))
		grammarSetLocation($$, @1)
	}
	;

transaction_mode_list:
	  transaction_mode_item
	{
		$$ = singleDefElemSlice($1)
	}
	| transaction_mode_list ',' transaction_mode_item
	{
		$$ = appendToDefElemSlice($1, $3)
	}
	| transaction_mode_list transaction_mode_item
	{
		$$ = appendToDefElemSlice($1, $2)
	}
	;

transaction_mode_list_or_empty:
	  transaction_mode_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_transaction_chain:
	  AND CHAIN
	{
		$$ = true
	}
	| AND NO CHAIN
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

ViewStmt:
	  CREATE OptTemp VIEW qualified_name opt_column_list opt_reloptions 				AS SelectStmt opt_check_option
	{
		$$ = grammarView($4, $2, $5, $8, $6, $9, false, false)
	}
	| CREATE OR REPLACE OptTemp VIEW qualified_name opt_column_list opt_reloptions 				AS SelectStmt opt_check_option
	{
		$$ = grammarView($6, $4, $7, $10, $8, $11, true, false)
	}
	| CREATE OptTemp RECURSIVE VIEW qualified_name '(' columnList ')' opt_reloptions 				AS SelectStmt opt_check_option
	{
		$$ = grammarView($5, $2, $7, $11, $9, $12, false, true)
	}
	| CREATE OR REPLACE OptTemp RECURSIVE VIEW qualified_name '(' columnList ')' opt_reloptions 				AS SelectStmt opt_check_option
	{
		$$ = grammarView($7, $4, $9, $13, $11, $14, true, true)
	}
	;

opt_check_option:
	  WITH CHECK OPTION
	{
		$$ = "CASCADED"
	}
	| WITH CASCADED CHECK OPTION
	{
		$$ = "CASCADED"
	}
	| WITH LOCAL CHECK OPTION
	{
		$$ = "LOCAL"
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

LoadStmt:
	  LOAD file_name
	{
		$$ = &LoadStmt{Filename: toStr($2)}
	}
	;

CreatedbStmt:
	  CREATE DATABASE name opt_with createdb_opt_list
	{
		$$ = &CreatedbStmt{Dbname: toStr($3), Options: toNodeSlice(toDefElemSlice($5))}
	}
	;

createdb_opt_list:
	  createdb_opt_items
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

createdb_opt_items:
	  createdb_opt_item
	{
		$$ = []Node{toNode($1.(*DefElem))}
	}
	| createdb_opt_items createdb_opt_item
	{
		$$ = append(toDefElemSlice($1), toNode($2.(*DefElem)))
	}
	;

createdb_opt_item:
	  createdb_opt_name opt_equal NumericOnly
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode(toNode($3)), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| createdb_opt_name opt_equal opt_boolean_or_string
	{
		$$ = &DefElem{Defname: toStr($1), Arg: toNode(&String{Sval: toStr($3)}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| createdb_opt_name opt_equal DEFAULT
	{
		$$ = &DefElem{Defname: toStr($1), Arg: nil, Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

createdb_opt_name:
	  IDENT
	{
		$$ = toStr($1)
	}
	| CONNECTION LIMIT
	{
		$$ = "connection_limit"
	}
	| ENCODING
	{
		$$ = toStr($1)
	}
	| LOCATION
	{
		$$ = toStr($1)
	}
	| OWNER
	{
		$$ = toStr($1)
	}
	| TABLESPACE
	{
		$$ = toStr($1)
	}
	| TEMPLATE
	{
		$$ = toStr($1)
	}
	;

opt_equal:
	  '='
	{
		$$ = nil
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterDatabaseStmt:
	  ALTER DATABASE name WITH createdb_opt_list
	{
		$$ = &AlterDatabaseStmt{Dbname: toStr($3), Options: toNodeSlice(toDefElemSlice($5))}
	}
	| ALTER DATABASE name createdb_opt_list
	{
		$$ = &AlterDatabaseStmt{Dbname: toStr($3), Options: toNodeSlice(toDefElemSlice($4))}
	}
	| ALTER DATABASE name SET TABLESPACE name
	{
		$$ = &AlterDatabaseStmt{Dbname: toStr($3), Options: toNodeSlice([]Node{toNode(&DefElem{Defname: "tablespace", Arg: toNode(&String{Sval: toStr($6)}), Location: int32(@6), Defaction: pg_query.DefElemAction(1)})})}
	}
	| ALTER DATABASE name REFRESH COLLATION VERSION_P
	{
		$$ = &AlterDatabaseStmt{Dbname: toStr($3)}
	}
	;

AlterDatabaseSetStmt:
	  ALTER DATABASE name SetResetClause
	{
		$$ = &AlterDatabaseSetStmt{Dbname: toStr($3), Setstmt: toVariableSetStmt($4)}
	}
	;

DropdbStmt:
	  DROP DATABASE name
	{
		$$ = &DropdbStmt{Dbname: toStr($3), MissingOk: false, Options: nil}
	}
	| DROP DATABASE IF_P EXISTS name
	{
		$$ = &DropdbStmt{Dbname: toStr($5), MissingOk: true, Options: nil}
	}
	| DROP DATABASE name opt_with '(' drop_option_list ')'
	{
		$$ = &DropdbStmt{Dbname: toStr($3), MissingOk: false, Options: toNodeSlice(toDefElemSlice($6))}
	}
	| DROP DATABASE IF_P EXISTS name opt_with '(' drop_option_list ')'
	{
		$$ = &DropdbStmt{Dbname: toStr($5), MissingOk: true, Options: toNodeSlice(toDefElemSlice($8))}
	}
	;

drop_option_list:
	  drop_option
	{
		$$ = []Node{toNode($1.(*DefElem))}
	}
	| drop_option_list ',' drop_option
	{
		$$ = append(toDefElemSlice($1), toNode($3.(*DefElem)))
	}
	;

drop_option:
	  FORCE
	{
		$$ = &DefElem{Defname: "force", Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

AlterCollationStmt:
	  ALTER COLLATION any_name REFRESH VERSION_P
	{
		$$ = &AlterCollationStmt{Collname: toPgStringNodes(toStrSlice($3))}
	}
	;

AlterSystemStmt:
  ALTER SYSTEM_P SET generic_set
{
	$$ = &AlterSystemStmt{Setstmt: toVariableSetStmt($4)}
}
| ALTER SYSTEM_P RESET generic_reset
{
	$$ = &AlterSystemStmt{Setstmt: toVariableSetStmt($4)}
}
;

CreateDomainStmt:
	  CREATE DOMAIN_P any_name opt_as Typename ColQualList
	{
		$$ = makeCreateDomainStmt($3, $5, $6)
	}
	;

AlterDomainStmt:
	  ALTER DOMAIN_P any_name alter_column_default
	{
		$$ = makeAlterDomainStmt("T", $3, $4, nil, nil, false)
	}
	| ALTER DOMAIN_P any_name DROP NOT NULL_P
	{
		$$ = makeAlterDomainStmt("N", $3, nil, nil, nil, false)
	}
	| ALTER DOMAIN_P any_name SET NOT NULL_P
	{
		$$ = makeAlterDomainStmt("O", $3, nil, nil, nil, false)
	}
	| ALTER DOMAIN_P any_name ADD_P DomainConstraint
	{
		$$ = makeAlterDomainStmt("C", $3, $5, nil, nil, false)
	}
	| ALTER DOMAIN_P any_name DROP CONSTRAINT name opt_drop_behavior
	{
		$$ = makeAlterDomainStmt("X", $3, nil, $6, $7, false)
	}
	| ALTER DOMAIN_P any_name DROP CONSTRAINT IF_P EXISTS name opt_drop_behavior
	{
		$$ = makeAlterDomainStmt("X", $3, nil, $8, $9, true)
	}
	| ALTER DOMAIN_P any_name VALIDATE CONSTRAINT name
	{
		$$ = makeAlterDomainStmt("V", $3, nil, $6, nil, false)
	}
	;

opt_as:
	  AS
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

AlterTSDictionaryStmt:
	  ALTER TEXT_P SEARCH DICTIONARY any_name definition
	{
		$$ = &AlterTSDictionaryStmt{Dictname: toPgStringNodes(toStrSlice($5)), Options: toNodeSlice(toDefElemSlice($6))}
	}
	;

AlterTSConfigurationStmt:
	  ALTER TEXT_P SEARCH CONFIGURATION any_name ADD_P MAPPING FOR name_list any_with any_name_list
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(1), Cfgname: toPgStringNodes(toStrSlice($5)), Tokentype: toPgStringNodes(toStrSlice($9)), Dicts: toNodeSlice($11), Override: false, Replace: false}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING FOR name_list any_with any_name_list
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(2), Cfgname: toPgStringNodes(toStrSlice($5)), Tokentype: toPgStringNodes(toStrSlice($9)), Dicts: toNodeSlice($11), Override: true, Replace: false}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING REPLACE any_name any_with any_name
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(3), Cfgname: toPgStringNodes(toStrSlice($5)), Dicts: []Node{toNode(toAnyNameNode($9)),toNode(toAnyNameNode($11))}, Override: false, Replace: true}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name ALTER MAPPING FOR name_list REPLACE any_name any_with any_name
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(4), Cfgname: toPgStringNodes(toStrSlice($5)), Tokentype: toPgStringNodes(toStrSlice($9)), Dicts: []Node{toNode(toAnyNameNode($11)),toNode(toAnyNameNode($13))}, Override: false, Replace: true}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name DROP MAPPING FOR name_list
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(5), Cfgname: toPgStringNodes(toStrSlice($5)), Tokentype: toPgStringNodes(toStrSlice($9)), MissingOk: false}
	}
	| ALTER TEXT_P SEARCH CONFIGURATION any_name DROP MAPPING IF_P EXISTS FOR name_list
	{
		$$ = &AlterTSConfigurationStmt{Kind: pg_query.AlterTSConfigType(5), Cfgname: toPgStringNodes(toStrSlice($5)), Tokentype: toPgStringNodes(toStrSlice($11)), MissingOk: true}
	}
	;

any_with:
	  WITH
	{
		$$ = $1
	}
	| WITH_LA
	{
		$$ = $1
	}
	;

CreateConversionStmt:
	  CREATE opt_default CONVERSION_P any_name FOR Sconst 			TO Sconst FROM any_name
	{
		$$ = &CreateConversionStmt{ConversionName: toPgStringNodes(toStrSlice($4)), ForEncodingName: toStr($6), ToEncodingName: toStr($8), FuncName: toPgStringNodes(toStrSlice($10)), Def: toBool($2)}
	}
	;

ClusterStmt:
	  CLUSTER '(' utility_option_list ')' qualified_name cluster_index_specification
	{
		$$ = makeClusterStmt($5, $6, $3)
	}
	| CLUSTER '(' utility_option_list ')'
	{
		$$ = makeClusterStmt(nil, nil, $3)
	}
	| CLUSTER opt_verbose qualified_name cluster_index_specification
	{
		$$ = makeClusterStmt($3, $4, nil)
	}
	| CLUSTER opt_verbose
	{
		$$ = makeClusterStmt(nil, nil, nil)
	}
	| CLUSTER opt_verbose name ON qualified_name
	{
		$$ = makeClusterStmt($5, $3, nil)
	}
	;

cluster_index_specification:
	  USING name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

VacuumStmt:
	  VACUUM opt_full opt_freeze opt_verbose opt_analyze opt_vacuum_relation_list
	{
		$$ = makeVacuumStmt($2, $3, $4, $5, $6, true, [4]int{@2, @3, @4, @5})
	}
	| VACUUM '(' utility_option_list ')' opt_vacuum_relation_list
	{
		$$ = &VacuumStmt{Options: toNodeSlice(toDefElemSlice($3)), Rels: toNodeSlice(toVacuumRelationSlice($5)), IsVacuumcmd: true}
	}
	;

AnalyzeStmt:
	  analyze_keyword opt_verbose opt_vacuum_relation_list
	{
		$$ = makeVacuumStmt(nil, nil, $2, nil, $3, false, [4]int{-1, -1, @2, -1})
	}
	| analyze_keyword '(' utility_option_list ')' opt_vacuum_relation_list
	{
		$$ = &VacuumStmt{Options: toNodeSlice(toDefElemSlice($3)), Rels: toNodeSlice(toVacuumRelationSlice($5)), IsVacuumcmd: false}
	}
	;

utility_option_list:
	  utility_option_elem
	{
		$$ = singleDefElemSlice($1)
	}
	| utility_option_list ',' utility_option_elem
	{
		$$ = appendToDefElemSlice($1, $3)
	}
	;

analyze_keyword:
	  ANALYZE
	{
		$$ = $1
	}
	| ANALYSE
	{
		$$ = $1
	}
	;

utility_option_elem:
	  utility_option_name utility_option_arg
	{
		$$ = makeDefElem($1, $2)
		grammarSetLocation($$, @1)
	}
	;

utility_option_name:
	  NonReservedWord
	{
		$$ = toStr($1)
	}
	| analyze_keyword
	{
		$$ = "analyze"
	}
	| FORMAT_LA
	{
		$$ = "format"
	}
	;

utility_option_arg:
	  opt_boolean_or_string
	{
		$$ = &String{Sval: toStr($1)}
	}
	| NumericOnly
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_analyze:
	  analyze_keyword
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_verbose:
	  VERBOSE
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_full:
	  FULL
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_freeze:
	  FREEZE
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_name_list:
	  '(' name_list ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

vacuum_relation:
	  relation_expr opt_name_list
	{
		$$ = toVacuumRelation($1, $2)
	}
	;

vacuum_relation_list:
	  vacuum_relation
	{
		$$ = singleVacuumRelationSlice($1)
	}
	| vacuum_relation_list ',' vacuum_relation
	{
		$$ = appendToVacuumRelationSlice($1, $3)
	}
	;

opt_vacuum_relation_list:
	  vacuum_relation_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ExplainStmt:
	  EXPLAIN ExplainableStmt
	{
		$$ = &ExplainStmt{Query: toNode(toStatement($2))}
	}
	| EXPLAIN analyze_keyword opt_verbose ExplainableStmt
	{
		var opts []Node
		opts = append(opts, toNode(&DefElem{Defname: "analyze", Location: -1, Defaction: pg_query.DefElemAction(1)}))
		if toBool($3) {
			opts = append(opts, toNode(&DefElem{Defname: "verbose", Location: -1, Defaction: pg_query.DefElemAction(1)}))
		}
		$$ = &ExplainStmt{Query: toNode(toStatement($4)), Options: toNodeSlice(opts)}
	}
	| EXPLAIN VERBOSE ExplainableStmt
	{
		$$ = &ExplainStmt{Query: toNode(toStatement($3)), Options: toNodeSlice([]Node{toNode(&DefElem{Defname: "verbose", Location: -1, Defaction: pg_query.DefElemAction(1)})})}
	}
	| EXPLAIN '(' utility_option_list ')' ExplainableStmt
	{
		$$ = &ExplainStmt{Query: toNode(toStatement($5)), Options: toNodeSlice(toDefElemSlice($3))}
	}
	;

ExplainableStmt:
	  SelectStmt
	{
		$$ = $1
	}
	| InsertStmt
	{
		$$ = $1
	}
	| UpdateStmt
	{
		$$ = $1
	}
	| DeleteStmt
	{
		$$ = $1
	}
	| MergeStmt
	{
		$$ = $1
	}
	| DeclareCursorStmt
	{
		$$ = $1
	}
	| CreateAsStmt
	{
		$$ = $1
	}
	| CreateMatViewStmt
	{
		$$ = $1
	}
	| RefreshMatViewStmt
	{
		$$ = $1
	}
	| ExecuteStmt
	{
		$$ = $1
	}
	;

PrepareStmt:
	  PREPARE name prep_type_clause AS PreparableStmt
	{
		$$ = &PrepareStmt{Name: toStr($2), Argtypes: toNodeSlice($3), Query: toNode(toStatement($5))}
	}
	;

prep_type_clause:
	  '(' type_list ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

PreparableStmt:
	  SelectStmt
	{
		$$ = $1
	}
	| InsertStmt
	{
		$$ = $1
	}
	| UpdateStmt
	{
		$$ = $1
	}
	| DeleteStmt
	{
		$$ = $1
	}
	| MergeStmt
	{
		$$ = $1
	}
	;

ExecuteStmt:
	  EXECUTE name execute_param_clause
	{
		$$ = &ExecuteStmt{Name: toStr($2), Params: toNodeSlice($3)}
	}
	| CREATE OptTemp TABLE create_as_target AS 				EXECUTE name execute_param_clause opt_with_data
	{
		execStmt := &ExecuteStmt{Name: toStr($7), Params: toNodeSlice($8)}
		$$ = makeCreateTableAsStmt($2, $4, execStmt, $9, false, "table")
	}
	| CREATE OptTemp TABLE IF_P NOT EXISTS create_as_target AS 				EXECUTE name execute_param_clause opt_with_data
	{
		execStmt := &ExecuteStmt{Name: toStr($10), Params: toNodeSlice($11)}
		$$ = makeCreateTableAsStmt($2, $7, execStmt, $12, true, "table")
	}
	;

execute_param_clause:
	  '(' expr_list ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

DeallocateStmt:
	  DEALLOCATE name
	{
		$$ = &DeallocateStmt{Name: toStr($2), Location: -1}
		grammarSetLocation($$, @2)
	}
	| DEALLOCATE PREPARE name
	{
		$$ = &DeallocateStmt{Name: toStr($3), Location: -1}
		grammarSetLocation($$, @3)
	}
	| DEALLOCATE ALL
	{
		$$ = &DeallocateStmt{Isall: true, Location: -1}
	}
	| DEALLOCATE PREPARE ALL
	{
		$$ = &DeallocateStmt{Isall: true, Location: -1}
	}
	;

InsertStmt:
	  opt_with_clause INSERT INTO insert_target insert_rest 			opt_on_conflict returning_clause
	{
		ins := &InsertStmt{Relation: toRangeVar($4), ReturningClause: grammarReturning($7), Override: pg_query.OverridingKind(1)}
		if $1 != nil { ins.WithClause = toWithClause($1)}
		if oc, ok := $6.(*OnConflictClause); ok { ins.OnConflictClause = oc}
		if rest, ok := $5.(*InsertStmt); ok {
			ins.Cols = toNodeSlice(rest.Cols)
			ins.SelectStmt = toNode(rest.SelectStmt)
			ins.Override = rest.Override
		} else if sel, ok := $5.(*SelectStmt); ok {
			ins.SelectStmt = toNode(sel)
		}
		$$ = ins
	}
	;

insert_target:
	  qualified_name
	{
		$$ = makeRangeVar($1, true)
	}
	| qualified_name AS ColId
	{
		rv := makeRangeVar($1, true)
		rv.Alias = &pg_query.Alias{Aliasname: toStr($3)}
		$$ = rv
	}
	;

insert_rest:
	  SelectStmt
	{
		$$ = $1
	}
	| OVERRIDING override_kind VALUE_P SelectStmt
	{
		$$ = &InsertStmt{SelectStmt: toNode($4), Override: enumOverridingKind(toStr($2))}
	}
	| '(' insert_column_list ')' SelectStmt
	{
		$$ = &InsertStmt{Cols: toNodeSlice(toResTargetSlice($2)), SelectStmt: toNode(toSelectStmtPtr($4)), Override: pg_query.OverridingKind(1)}
	}
	| '(' insert_column_list ')' OVERRIDING override_kind VALUE_P SelectStmt
	{
		$$ = &InsertStmt{Cols: toNodeSlice(toResTargetSlice($2)), SelectStmt: toNode(toSelectStmtPtr($7)), Override: enumOverridingKind(toStr($5))}
	}
	| DEFAULT VALUES
	{
		$$ = &InsertStmt{Override: pg_query.OverridingKind(1)}
	}
	;

override_kind:
	  USER
	{
		$$ = "USER"
	}
	| SYSTEM_P
	{
		$$ = "SYSTEM"
	}
	;

insert_column_list:
	  insert_column_item
	{
		$$ = []Node{toNode(toResTarget($1))}
	}
	| insert_column_list ',' insert_column_item
	{
		$$ = append(toResTargetSlice($1), toNode(toResTarget($3)))
	}
	;

insert_column_item:
	  ColId opt_indirection
	{
		$$ = &ResTarget{Name: toStr($1), Indirection: toNodeSlice($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

opt_on_conflict:
	  ON CONFLICT opt_conf_expr DO UPDATE SET set_clause_list	where_clause
	{
		$$ = &OnConflictClause{Action: enumOnConflictAction("UPDATE"), Infer: grammarInfer($3), TargetList: toNodeSlice(toResTargetSlice($7)), WhereClause: toNode($8), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ON CONFLICT opt_conf_expr DO NOTHING
	{
		$$ = &OnConflictClause{Action: enumOnConflictAction("NOTHING"), Infer: grammarInfer($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_conf_expr:
	  '(' index_params ')' where_clause
	{
		$$ = &InferClause{IndexElems: toNodeSlice(toIndexElemSlice($2)), WhereClause: toNode($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ON CONSTRAINT name
	{
		$$ = &InferClause{Conname: toStr($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

returning_clause:
	  RETURNING returning_with_clause target_list
	{
		$$ = &ReturningClause{Options: toNodeSlice($2), Exprs: toNodeSlice(toResTargetSlice($3))}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

returning_with_clause:
	  WITH '(' returning_options ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

returning_options:
	  returning_option
	{
		$$ = []Node{toNode($1)}
	}
	| returning_options ',' returning_option
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

returning_option:
	  returning_option_kind AS ColId
	{
		$$ = &ReturningOption{Option: enumReturningOptionKind(toStr($1)), Value: toStr($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

returning_option_kind:
	  OLD
	{
		$$ = "OLD"
	}
	| NEW
	{
		$$ = "NEW"
	}
	;

DeleteStmt:
	  opt_with_clause DELETE_P FROM relation_expr_opt_alias 			using_clause where_or_current_clause returning_clause
	{
		del := &DeleteStmt{Relation: toRangeVar($4), UsingClause: toNodeSlice($5), WhereClause: toNode($6), ReturningClause: grammarReturning($7)}
		if $1 != nil { del.WithClause = toWithClause($1)}
		$$ = del
	}
	;

using_clause:
	  USING from_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

LockStmt:
	  LOCK_P opt_table relation_expr_list opt_lock opt_nowait
	{
		$$ = &LockStmt{Relations: toNodeSlice(toRangeVarSlice($3)), Mode: int32(toIntDefault($4, 8)), Nowait: toBool($5)}
	}
	;

opt_lock:
	  IN_P lock_type MODE
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = 8
	}
	;

lock_type:
	  ACCESS SHARE
	{
		$$ = 1
	}
	| ROW SHARE
	{
		$$ = 2
	}
	| ROW EXCLUSIVE
	{
		$$ = 3
	}
	| SHARE UPDATE EXCLUSIVE
	{
		$$ = 4
	}
	| SHARE
	{
		$$ = 5
	}
	| SHARE ROW EXCLUSIVE
	{
		$$ = 6
	}
	| EXCLUSIVE
	{
		$$ = 7
	}
	| ACCESS EXCLUSIVE
	{
		$$ = 8
	}
	;

opt_nowait:
	  NOWAIT
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

opt_nowait_or_skip:
	  NOWAIT
	{
		$$ = $1
	}
	| SKIP LOCKED
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

UpdateStmt:
	  opt_with_clause UPDATE relation_expr_opt_alias 			SET set_clause_list 			from_clause 			where_or_current_clause 			returning_clause
	{
		upd := &UpdateStmt{Relation: toRangeVar($3), TargetList: toNodeSlice(toResTargetSlice($5)), FromClause: toNodeSlice($6), WhereClause: toNode($7), ReturningClause: grammarReturning($8)}
		if $1 != nil { upd.WithClause = toWithClause($1)}
		$$ = upd
	}
	;

set_clause_list:
	  set_clause
	{
		$$ = $1
	}
	| set_clause_list ',' set_clause
	{
		$$ = append(toResTargetSlice($1), toResTargetSlice($3)...)
	}
	;

set_clause:
	  set_target '=' a_expr
	{
		target := toResTarget($1)
		target.Val = toNode($3)
		$$ = []Node{toNode(target)}
	}
	| '(' set_target_list ')' '=' a_expr
	{
		targets := toResTargetSlice($2)
		for i, target := range targets {
			target.GetResTarget().Val = toNode(&MultiAssignRef{Source: toNode($5), Colno: int32(i+1), Ncolumns: int32(len(targets))})
		}
		$$ = targets
	}
	;

set_target:
	  ColId opt_indirection
	{
		$$ = &ResTarget{Name: toStr($1), Indirection: toNodeSlice($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

set_target_list:
	  set_target
	{
		$$ = []Node{toNode(toResTarget($1))}
	}
	| set_target_list ',' set_target
	{
		$$ = append(toResTargetSlice($1), toNode(toResTarget($3)))
	}
	;

MergeStmt:
	  opt_with_clause MERGE INTO relation_expr_opt_alias 			USING table_ref 			ON a_expr 			merge_when_list 			returning_clause
	{
		m := &MergeStmt{Relation: toRangeVar($4), SourceRelation: toNode($6), JoinCondition: toNode($8), MergeWhenClauses: toNodeSlice($9), ReturningClause: grammarReturning($10)}
		if $1 != nil {
			m.WithClause = toWithClause($1)
		}
		$$ = m
	}
	;

merge_when_list:
	  merge_when_clause
	{
		if $1 != nil {
			$$ = []Node{toNode(toNode($1))}
		} else {
			$$ = []Node(nil)
		}
	}
	| merge_when_list merge_when_clause
	{
		list := toNodeSlice($1)
		if $2 != nil {
			list = append(list, toNode($2))
		}
		$$ = list
	}
	;

merge_when_clause:
	  merge_when_tgt_matched opt_merge_when_condition THEN merge_update
	{
		mwc := $4.(*MergeWhenClause)
		mwc.MatchKind = enumMergeMatchKind(toStr($1))
		mwc.Condition = toNode($2)
		$$ = mwc
	}
	| merge_when_tgt_matched opt_merge_when_condition THEN merge_delete
	{
		mwc := $4.(*MergeWhenClause)
		mwc.MatchKind = enumMergeMatchKind(toStr($1))
		mwc.Condition = toNode($2)
		$$ = mwc
	}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN merge_insert
	{
		mwc := $4.(*MergeWhenClause)
		mwc.MatchKind = enumMergeMatchKind(toStr($1))
		mwc.Condition = toNode($2)
		$$ = mwc
	}
	| merge_when_tgt_matched opt_merge_when_condition THEN DO NOTHING
	{
		$$ = &MergeWhenClause{MatchKind: enumMergeMatchKind(toStr($1)), CommandType: enumCmdType("nothing"), Condition: toNode($2), Override: pg_query.OverridingKind(1)}
	}
	| merge_when_tgt_not_matched opt_merge_when_condition THEN DO NOTHING
	{
		$$ = &MergeWhenClause{MatchKind: enumMergeMatchKind(toStr($1)), CommandType: enumCmdType("nothing"), Condition: toNode($2), Override: pg_query.OverridingKind(1)}
	}
	;

merge_when_tgt_matched:
	  WHEN MATCHED
	{
		$$ = "matched"
	}
	| WHEN NOT MATCHED BY SOURCE
	{
		$$ = "not_matched_by_source"
	}
	;

merge_when_tgt_not_matched:
	  WHEN NOT MATCHED
	{
		$$ = "not_matched_by_target"
	}
	| WHEN NOT MATCHED BY TARGET
	{
		$$ = "not_matched_by_target"
	}
	;

opt_merge_when_condition:
	  AND a_expr
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

merge_update:
	  UPDATE SET set_clause_list
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("update"), TargetList: toNodeSlice(toResTargetSlice($3)), MatchKind: pg_query.MergeMatchKind(1), Override: pg_query.OverridingKind(1)}
	}
	;

merge_delete:
	  DELETE_P
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("delete"), MatchKind: pg_query.MergeMatchKind(1), Override: pg_query.OverridingKind(1)}
	}
	;

merge_insert:
	  INSERT merge_values_clause
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("insert"), Values: toNodeSlice($2), MatchKind: pg_query.MergeMatchKind(1), Override: pg_query.OverridingKind(1)}
	}
	| INSERT OVERRIDING override_kind VALUE_P merge_values_clause
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("insert"), Override: enumOverridingKind(toStr($3)), Values: toNodeSlice($5), MatchKind: pg_query.MergeMatchKind(1)}
	}
	| INSERT '(' insert_column_list ')' merge_values_clause
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("insert"), TargetList: toNodeSlice(toResTargetSlice($3)), Values: toNodeSlice($5), MatchKind: pg_query.MergeMatchKind(1), Override: pg_query.OverridingKind(1)}
	}
	| INSERT '(' insert_column_list ')' OVERRIDING override_kind VALUE_P merge_values_clause
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("insert"), TargetList: toNodeSlice(toResTargetSlice($3)), Override: enumOverridingKind(toStr($6)), Values: toNodeSlice($8), MatchKind: pg_query.MergeMatchKind(1)}
	}
	| INSERT DEFAULT VALUES
	{
		$$ = &MergeWhenClause{CommandType: enumCmdType("insert"), MatchKind: pg_query.MergeMatchKind(1), Override: pg_query.OverridingKind(1)}
	}
	;

merge_values_clause:
	  VALUES '(' expr_list ')'
	{
		$$ = $3
	}
	;

DeclareCursorStmt:
	  DECLARE cursor_name cursor_options CURSOR opt_hold FOR SelectStmt
	{
		$$ = makeDeclareCursorStmt($2, int32(toIntDefault($3, 0) | toIntDefault($5, 0) | 256), $7)
	}
	;

cursor_name:
	  name
	{
		$$ = $1
	}
	;

cursor_options:
	  /* EMPTY */
	{
		$$ = nil
	}
	| cursor_options NO SCROLL
	{
		$$ = toIntDefault($1, 0) | 4
	}
	| cursor_options SCROLL
	{
		$$ = toIntDefault($1, 0) | 2
	}
	| cursor_options BINARY
	{
		$$ = toIntDefault($1, 0) | 1
	}
	| cursor_options ASENSITIVE
	{
		$$ = $1
	}
	| cursor_options INSENSITIVE
	{
		$$ = toIntDefault($1, 0) | 8
	}
	;

opt_hold:
	  /* EMPTY */
	{
		$$ = nil
	}
	| WITH HOLD
	{
		$$ = 32
	}
	| WITHOUT HOLD
	{
		$$ = 0
	}
	;

SelectStmt:
	  select_no_parens			%prec UMINUS
	{
		$$ = $1
	}
	| select_with_parens		%prec UMINUS
	{
		$$ = $1
	}
	;

select_with_parens:
	  '(' select_no_parens ')'
	{
		$$ = $2
	}
	| '(' select_with_parens ')'
	{
		$$ = $2
	}
	;

select_no_parens:
	  simple_select
	{
		$$ = $1
	}
	| select_clause sort_clause
	{
		sel := toSelectStmtPtr($1)
		if message := applySelectOptions(sel, $2, nil, nil, nil); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| select_clause opt_sort_clause for_locking_clause opt_select_limit
	{
		sel := toSelectStmtPtr($1)
		if message := applySelectOptions(sel, $2, $3, $4, nil); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| select_clause opt_sort_clause select_limit opt_for_locking_clause
	{
		sel := toSelectStmtPtr($1)
		if message := applySelectOptions(sel, $2, $4, $3, nil); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| with_clause select_clause
	{
		sel := toSelectStmtPtr($2)
		if message := applySelectOptions(sel, nil, nil, nil, $1); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| with_clause select_clause sort_clause
	{
		sel := toSelectStmtPtr($2)
		if message := applySelectOptions(sel, $3, nil, nil, $1); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| with_clause select_clause opt_sort_clause for_locking_clause opt_select_limit
	{
		sel := toSelectStmtPtr($2)
		if message := applySelectOptions(sel, $3, $4, $5, $1); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	| with_clause select_clause opt_sort_clause select_limit opt_for_locking_clause
	{
		sel := toSelectStmtPtr($2)
		if message := applySelectOptions(sel, $3, $5, $4, $1); message != "" { yylex.Error(message); return 1 }
		$$ = sel
	}
	;

select_clause:
	  simple_select
	{
		$$ = $1
	}
	| select_with_parens
	{
		$$ = $1
	}
	;

simple_select:
	  SELECT opt_all_clause opt_target_list 			into_clause from_clause where_clause 			group_clause having_clause window_clause
	{
		sel := toSelectStmt($3, $5, $6, $7, $8, $9)
		if $4 != nil { sel.IntoClause = $4.(*IntoClause)}
		$$ = sel
	}
	| SELECT distinct_clause target_list 			into_clause from_clause where_clause 			group_clause having_clause window_clause
	{
		sel := toSelectStmt($3, $5, $6, $7, $8, $9)
		sel.DistinctClause = toNodeSlice($2)
		if $4 != nil { sel.IntoClause = $4.(*IntoClause)}
		$$ = sel
	}
	| values_clause
	{
		$$ = &SelectStmt{ValuesLists: toNodeSlice(toValuesList($1)), LimitOption: pg_query.LimitOption(1), Op: pg_query.SetOperation(1)}
	}
	| TABLE relation_expr
	{
		$$ = &SelectStmt{TargetList: toNodeSlice([]Node{toNode(&ResTarget{Val: toNode(&ColumnRef{Fields: []Node{toNode(&A_Star{})}, Location: -1}), Location: -1})}), FromClause: []Node{toNode($2)}, LimitOption: pg_query.LimitOption(1), Op: pg_query.SetOperation(1)}
	}
	| select_clause UNION set_quantifier select_clause
	{
		$$ = makeSetOp(pg_query.SetOperation_SETOP_UNION, $3, $1, $4)
	}
	| select_clause INTERSECT set_quantifier select_clause
	{
		$$ = makeSetOp(pg_query.SetOperation_SETOP_INTERSECT, $3, $1, $4)
	}
	| select_clause EXCEPT set_quantifier select_clause
	{
		$$ = makeSetOp(pg_query.SetOperation_SETOP_EXCEPT, $3, $1, $4)
	}
	;

with_clause:
	  WITH cte_list
	{
		$$ = &WithClause{Ctes: toNodeSlice(toCommonTableExprSlice($2)), Recursive: false, Location: -1}
		grammarSetLocation($$, @1)
	}
	| WITH_LA cte_list
	{
		$$ = &WithClause{Ctes: toNodeSlice(toCommonTableExprSlice($2)), Recursive: false, Location: -1}
		grammarSetLocation($$, @1)
	}
	| WITH RECURSIVE cte_list
	{
		$$ = &WithClause{Ctes: toNodeSlice(toCommonTableExprSlice($3)), Recursive: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

cte_list:
	  common_table_expr
	{
		$$ = []Node{toNode(toCommonTableExpr($1))}
	}
	| cte_list ',' common_table_expr
	{
		$$ = appendToCommonTableExprSlice($1, $3)
	}
	;

common_table_expr:
	  name opt_name_list AS opt_materialized '(' PreparableStmt ')' opt_search_clause opt_cycle_clause
	{
		cte := &CommonTableExpr{Ctename: toStr($1), Aliascolnames: toPgStringNodes(toStrSlice($2)), Ctematerialized: pg_query.CTEMaterialize(toInt32($4) + 1), Ctequery: toNode($6), Location: -1}
		if $8 != nil { cte.SearchClause = $8.(*CTESearchClause)}
		if $9 != nil { cte.CycleClause = $9.(*CTECycleClause)}
		$$ = cte
		grammarSetLocation($$, @1)
	}
	;

opt_materialized:
	  MATERIALIZED
	{
		$$ = int64(1)
	}
	| NOT MATERIALIZED
	{
		$$ = int64(2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_search_clause:
	  SEARCH DEPTH FIRST_P BY columnList SET ColId
	{
		$$ = &CTESearchClause{SearchColList: toPgStringNodes(toStrSlice($5)), SearchSeqColumn: toStr($7), Location: -1}
		grammarSetLocation($$, @1)
	}
	| SEARCH BREADTH FIRST_P BY columnList SET ColId
	{
		$$ = &CTESearchClause{SearchColList: toPgStringNodes(toStrSlice($5)), SearchSeqColumn: toStr($7), SearchBreadthFirst: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_cycle_clause:
	  CYCLE columnList SET ColId TO AexprConst DEFAULT AexprConst USING ColId
	{
		$$ = &CTECycleClause{CycleColList: toPgStringNodes(toStrSlice($2)), CycleMarkColumn: toStr($4), CycleMarkValue: toNode($6), CycleMarkDefault: toNode($8), CyclePathColumn: toStr($10), Location: -1}
		grammarSetLocation($$, @1)
	}
	| CYCLE columnList SET ColId USING ColId
	{
		$$ = &CTECycleClause{CycleColList: toPgStringNodes(toStrSlice($2)), CycleMarkColumn: toStr($4), CycleMarkValue: toNode(makeAConst(true)), CycleMarkDefault: toNode(makeAConst(false)), CyclePathColumn: toStr($6), Location: -1}
		grammarSetLocation($$, @1)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_with_clause:
	  with_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

into_clause:
	  INTO OptTempTableName
	{
		$$ = &IntoClause{Rel: toRangeVar($2), OnCommit: pg_query.OnCommitAction(1)}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptTempTableName:
	  TEMPORARY opt_table qualified_name
	{
		rv := toRangeVar($3)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| TEMP opt_table qualified_name
	{
		rv := toRangeVar($3)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| LOCAL TEMPORARY opt_table qualified_name
	{
		rv := toRangeVar($4)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| LOCAL TEMP opt_table qualified_name
	{
		rv := toRangeVar($4)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| GLOBAL TEMPORARY opt_table qualified_name
	{
		rv := toRangeVar($4)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| GLOBAL TEMP opt_table qualified_name
	{
		rv := toRangeVar($4)
		rv.Relpersistence = "t"
		$$ = rv
	}
	| UNLOGGED opt_table qualified_name
	{
		rv := toRangeVar($3)
		rv.Relpersistence = "u"
		$$ = rv
	}
	| TABLE qualified_name
	{
		$$ = $2
	}
	| qualified_name
	{
		$$ = $1
	}
	;

opt_table:
	  TABLE
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

set_quantifier:
	  ALL
	{
		$$ = $1
	}
	| DISTINCT
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

distinct_clause:
	  DISTINCT
	{
		$$ = []Node{nil}
	}
	| DISTINCT ON '(' expr_list ')'
	{
		$$ = $4
	}
	;

opt_all_clause:
	  ALL
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_distinct_clause:
	  distinct_clause
	{
		$$ = $1
	}
	| opt_all_clause
	{
		$$ = $1
	}
	;

opt_sort_clause:
	  sort_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

sort_clause:
	  ORDER BY sortby_list
	{
		$$ = $3
	}
	;

sortby_list:
	  sortby
	{
		$$ = []Node{toNode(toSortBy($1))}
	}
	| sortby_list ',' sortby
	{
		$$ = appendToSortBySlice($1, $3)
	}
	;

sortby:
	  a_expr USING qual_all_Op opt_nulls_order
	{
		$$ = &SortBy{Node: toNode($1), SortbyDir: pg_query.SortByDir_SORTBY_USING, SortbyNulls: toSortByNulls($4), UseOp: toPgStringNodes(toStrSlice($3)), Location: -1}
		grammarSetLocation($$, @3)
	}
	| a_expr opt_asc_desc opt_nulls_order
	{
		$$ = &SortBy{Node: toNode($1), SortbyDir: toSortByDir($2), SortbyNulls: toSortByNulls($3), Location: -1}
	}
	;

select_limit:
	  limit_clause offset_clause
	{
		$$ = combineLimitOffset($1, $2)
	}
	| offset_clause limit_clause
	{
		$$ = combineLimitOffset($2, $1)
	}
	| limit_clause
	{
		$$ = $1
	}
	| offset_clause
	{
		$$ = $1
	}
	;

opt_select_limit:
	  select_limit
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

limit_clause:
	  LIMIT select_limit_value
	{
		$$ = &LimitInfo{Count: toNode($2), Option: 1}
	}
	| LIMIT select_limit_value ',' select_offset_value
	{
		yylex.(*Lexer).errorAt("LIMIT #,# syntax is not supported", @1)
		return 1
	}
	| FETCH first_or_next select_fetch_first_value row_or_rows ONLY
	{
		$$ = &LimitInfo{Count: toNode($3), Option: 1}
	}
	| FETCH first_or_next select_fetch_first_value row_or_rows WITH TIES
	{
		$$ = &LimitInfo{Count: toNode($3), Option: 2}
	}
	| FETCH first_or_next row_or_rows ONLY
	{
		$$ = &LimitInfo{Count: toNode(makeAConst(int64(1))), Option: 1}
	}
	| FETCH first_or_next row_or_rows WITH TIES
	{
		$$ = &LimitInfo{Count: toNode(makeAConst(int64(1))), Option: 2}
	}
	;

offset_clause:
	  OFFSET select_offset_value
	{
		$$ = &LimitInfo{Offset: toNode($2)}
	}
	| OFFSET select_fetch_first_value row_or_rows
	{
		$$ = &LimitInfo{Offset: toNode($2)}
	}
	;

select_limit_value:
	  a_expr
	{
		$$ = $1
	}
	| ALL
	{
		$$ = makeAConst(nil)
		grammarSetLocation($$, @1)
	}
	;

select_offset_value:
	  a_expr
	{
		$$ = $1
	}
	;

select_fetch_first_value:
	  c_expr
	{
		$$ = $1
	}
	| '+' I_or_F_const
	{
		$$ = $2
	}
	| '-' I_or_F_const
	{
		$$ = grammarNegate($2, @1)
	}
	;

I_or_F_const:
	  Iconst
	{
		$$ = makeAConst(toInt64($1))
		grammarSetLocation($$, @1)
	}
	| FCONST
	{
		$$ = makeAConst(FloatVal{Str: toStr($1)})
		grammarSetLocation($$, @1)
	}
	;

row_or_rows:
	  ROW
	{
		$$ = $1
	}
	| ROWS
	{
		$$ = $1
	}
	;

first_or_next:
	  FIRST_P
	{
		$$ = $1
	}
	| NEXT
	{
		$$ = $1
	}
	;

group_clause:
	  GROUP_P BY set_quantifier group_by_list
	{
		$$ = &GroupClause{Items: toNodeSlice($4), Distinct: strings.EqualFold(toStr($3), "DISTINCT")}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

group_by_list:
	  group_by_item
	{
		$$ = []Node{toNode($1)}
	}
	| group_by_list ',' group_by_item
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

group_by_item:
	  a_expr
	{
		$$ = $1
	}
	| empty_grouping_set
	{
		$$ = $1
	}
	| cube_clause
	{
		$$ = $1
	}
	| rollup_clause
	{
		$$ = $1
	}
	| grouping_sets_clause
	{
		$$ = $1
	}
	;

empty_grouping_set:
	  '(' ')'
	{
		$$ = &GroupingSet{Kind: pg_query.GroupingSetKind(1), Location: int32(@1)}
	}
	;

rollup_clause:
	  ROLLUP '(' expr_list ')'
	{
		$$ = &GroupingSet{Kind: pg_query.GroupingSetKind(3), Content: toNodeSlice($3), Location: int32(@1)}
	}
	;

cube_clause:
	  CUBE '(' expr_list ')'
	{
		$$ = &GroupingSet{Kind: pg_query.GroupingSetKind(4), Content: toNodeSlice($3), Location: int32(@1)}
	}
	;

grouping_sets_clause:
	  GROUPING SETS '(' group_by_list ')'
	{
		$$ = &GroupingSet{Kind: pg_query.GroupingSetKind(5), Content: toNodeSlice($4), Location: int32(@1)}
	}
	;

having_clause:
	  HAVING a_expr
	{
		$$ = toNode($2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

for_locking_clause:
	  for_locking_items
	{
		$$ = $1
	}
	| FOR READ ONLY
	{
		$$ = nil
	}
	;

opt_for_locking_clause:
	  for_locking_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

for_locking_items:
	  for_locking_item
	{
		$$ = []Node{toNode($1)}
	}
	| for_locking_items for_locking_item
	{
		$$ = appendToNodeSlice($1, $2)
	}
	;

for_locking_item:
	  for_locking_strength locked_rels_list opt_nowait_or_skip
	{
		wait := int32(1)
		switch strings.ToUpper(toStr($3)) { case "NOWAIT": wait = 3; case "SKIP": wait = 2 }
		$$ = &LockingClause{Strength: pg_query.LockClauseStrength(toInt32($1)), LockedRels: toNodeSlice($2), WaitPolicy: pg_query.LockWaitPolicy(wait)}
	}
	;

for_locking_strength:
	  FOR UPDATE
	{
		$$ = int64(5)
	}
	| FOR NO KEY UPDATE
	{
		$$ = int64(4)
	}
	| FOR SHARE
	{
		$$ = int64(3)
	}
	| FOR KEY SHARE
	{
		$$ = int64(2)
	}
	;

locked_rels_list:
	  OF qualified_name_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

values_clause:
	  VALUES '(' expr_list ')'
	{
		$$ = []Node{toNode(&List{Items: toNodeSlice($3)})}
	}
	| values_clause ',' '(' expr_list ')'
	{
		$$ = append(toValuesList($1), toNode(&List{Items: toNodeSlice($4)}))
	}
	;

from_clause:
	  FROM from_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

from_list:
	  table_ref
	{
		$$ = []Node{toNode($1)}
	}
	| from_list ',' table_ref
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

table_ref:
	  relation_expr opt_alias_clause
	{
		rv := toRangeVar($1)
		applyAliasToRangeVar(rv, $2)
		$$ = rv
	}
	| relation_expr opt_alias_clause tablesample_clause
	{
		rv := toRangeVar($1)
		applyAliasToRangeVar(rv, $2)
		sample := $3.(*RangeTableSample)
		sample.Relation = toNode(rv)
		$$ = sample
	}
	| func_table func_alias_clause
	{
		rf := $1.(*RangeFunction)
		applyRangeFunctionAlias(rf, $2)
		$$ = rf
	}
	| LATERAL_P func_table func_alias_clause
	{
		rf := $2.(*RangeFunction)
		rf.Lateral = true
		applyRangeFunctionAlias(rf, $3)
		$$ = rf
	}
	| xmltable opt_alias_clause
	{
		n := $1.(*RangeTableFunc)
		n.Alias = grammarAlias($2)
		$$ = n
	}
	| LATERAL_P xmltable opt_alias_clause
	{
		n := $2.(*RangeTableFunc)
		n.Lateral = true
		n.Alias = grammarAlias($3)
		$$ = n
	}
	| select_with_parens opt_alias_clause
	{
		$$ = makeRangeSubselect($1, $2, false)
	}
	| LATERAL_P select_with_parens opt_alias_clause
	{
		$$ = makeRangeSubselect($2, $3, true)
	}
	| joined_table
	{
		$$ = $1
	}
	| '(' joined_table ')' alias_clause
	{
		jt := toJoinExpr($2)
		if jt != nil {
			name, cols := toAliasParts($4)
			jt.Alias = &pg_query.Alias{Aliasname: name, Colnames: toPgStringNodes(cols)}
		}
		$$ = jt
	}
	| json_table opt_alias_clause
	{
		n := $1.(*JsonTable)
		n.Alias = grammarAlias($2)
		$$ = n
	}
	| LATERAL_P json_table opt_alias_clause
	{
		n := $2.(*JsonTable)
		n.Lateral = true
		n.Alias = grammarAlias($3)
		$$ = n
	}
	;

joined_table:
	  '(' joined_table ')'
	{
		$$ = $2
	}
	| table_ref CROSS JOIN table_ref
	{
		$$ = makeJoinExpr(JoinInner, $1, $4, nil)
	}
	| table_ref join_type JOIN table_ref join_qual
	{
		$$ = makeJoinExpr(toJoinType($2), $1, $4, $5)
	}
	| table_ref JOIN table_ref join_qual
	{
		$$ = makeJoinExpr(JoinInner, $1, $3, $4)
	}
	| table_ref NATURAL join_type JOIN table_ref
	{
		$$ = makeNaturalJoinExpr(toJoinType($3), $1, $5)
	}
	| table_ref NATURAL JOIN table_ref
	{
		$$ = makeNaturalJoinExpr(JoinInner, $1, $4)
	}
	;

alias_clause:
	  AS ColId '(' name_list ')'
	{
		$$ = makeAliasClause($2, $4)
	}
	| AS ColId
	{
		$$ = makeAliasClause($2, nil)
	}
	| ColId '(' name_list ')'
	{
		$$ = makeAliasClause($1, $3)
	}
	| ColId
	{
		$$ = makeAliasClause($1, nil)
	}
	;

opt_alias_clause:
	  alias_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_alias_clause_for_join_using:
	  AS ColId
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

func_alias_clause:
	  alias_clause
	{
		$$ = $1
	}
	| AS '(' TableFuncElementList ')'
	{
		$$ = &RangeFunction{Coldeflist: toNodeSlice($3)}
	}
	| AS ColId '(' TableFuncElementList ')'
	{
		$$ = &RangeFunction{Alias: &pg_query.Alias{Aliasname: toStr($2)}, Coldeflist: toNodeSlice($4)}
	}
	| ColId '(' TableFuncElementList ')'
	{
		$$ = &RangeFunction{Alias: &pg_query.Alias{Aliasname: toStr($1)}, Coldeflist: toNodeSlice($3)}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

join_type:
	  FULL opt_outer
	{
		$$ = JoinFull
	}
	| LEFT opt_outer
	{
		$$ = JoinLeft
	}
	| RIGHT opt_outer
	{
		$$ = JoinRight
	}
	| INNER_P
	{
		$$ = JoinInner
	}
	;

opt_outer:
	  OUTER_P
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

join_qual:
	  USING '(' name_list ')' opt_alias_clause_for_join_using
	{
		$$ = &JoinQual{Using: toStrSlice($3), Alias: toStr($5)}
	}
	| ON a_expr
	{
		$$ = &JoinQual{Quals: toNode($2)}
	}
	;

relation_expr:
	  qualified_name
	{
		$$ = makeRangeVar($1, true)
	}
	| extended_relation_expr
	{
		$$ = $1
	}
	;

extended_relation_expr:
	  qualified_name '*'
	{
		$$ = makeRangeVar($1, true)
	}
	| ONLY qualified_name
	{
		$$ = makeRangeVar($2, false)
	}
	| ONLY '(' qualified_name ')'
	{
		$$ = makeRangeVar($3, false)
	}
	;

relation_expr_list:
	  relation_expr
	{
		$$ = singleRangeVarSlice($1)
	}
	| relation_expr_list ',' relation_expr
	{
		$$ = appendToRangeVarSlice($1, $3)
	}
	;

relation_expr_opt_alias:
	  relation_expr					%prec UMINUS
	{
		$$ = $1
	}
	| relation_expr ColId
	{
		rv := toRangeVar($1)
		rv.Alias = &pg_query.Alias{Aliasname: toStr($2)}
		$$ = rv
	}
	| relation_expr AS ColId
	{
		rv := toRangeVar($1)
		rv.Alias = &pg_query.Alias{Aliasname: toStr($3)}
		$$ = rv
	}
	;

tablesample_clause:
	  TABLESAMPLE func_name '(' expr_list ')' opt_repeatable_clause
	{
		$$ = &RangeTableSample{Method: toPgStringNodes(toStrSlice($2)), Args: toNodeSlice($4), Repeatable: toNode($6), Location: -1}
		grammarSetLocation($$, @2)
	}
	;

opt_repeatable_clause:
	  REPEATABLE '(' a_expr ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

func_table:
	  func_expr_windowless opt_ordinality
	{
		$$ = &RangeFunction{Functions: []Node{toNode(&List{Items: []Node{toNode($1), &pg_query.Node{}}})}, Ordinality: $2 != nil}
	}
	| ROWS FROM '(' rowsfrom_list ')' opt_ordinality
	{
		$$ = &RangeFunction{Functions: toNodeSlice($4), IsRowsfrom: true, Ordinality: $6 != nil}
	}
	;

rowsfrom_item:
	  func_expr_windowless opt_col_def_list
	{
		var cols Node
		if $2 != nil { cols = toNode(&List{Items: toNodeSlice($2)}) } else { cols = &pg_query.Node{} }
		$$ = &List{Items: []Node{toNode($1),toNode(cols)}}
	}
	;

rowsfrom_list:
	  rowsfrom_item
	{
		$$ = []Node{toNode($1)}
	}
	| rowsfrom_list ',' rowsfrom_item
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

opt_col_def_list:
	  AS '(' TableFuncElementList ')'
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_ordinality:
	  WITH_LA ORDINALITY
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

where_clause:
	  WHERE a_expr
	{
		$$ = toNode($2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

where_or_current_clause:
	  WHERE a_expr
	{
		$$ = toNode($2)
	}
	| WHERE CURRENT_P OF cursor_name
	{
		$$ = &CurrentOfExpr{CursorName: toStr($4)}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

OptTableFuncElementList:
	  TableFuncElementList
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

TableFuncElementList:
	  TableFuncElement
	{
		$$ = []Node{toNode(toNode($1))}
	}
	| TableFuncElementList ',' TableFuncElement
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

TableFuncElement:
	  ColId Typename opt_collate_clause
	{
		coll, _ := $3.(*CollateClause)
		$$ = &ColumnDef{Colname: toStr($1), TypeName: toTypeNameNode($2), CollClause: coll, IsLocal: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

xmltable:
	  XMLTABLE '(' c_expr xmlexists_argument COLUMNS xmltable_column_list ')'
	{
		$$ = &RangeTableFunc{Rowexpr: toNode($3), Docexpr: toNode($4), Columns: toNodeSlice($6), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLTABLE '(' XMLNAMESPACES '(' xml_namespace_list ')' ',' 				c_expr xmlexists_argument COLUMNS xmltable_column_list ')'
	{
		$$ = &RangeTableFunc{Namespaces: toNodeSlice($5), Rowexpr: toNode($8), Docexpr: toNode($9), Columns: toNodeSlice($11), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

xmltable_column_list:
	  xmltable_column_el
	{
		$$ = []Node{toNode($1)}
	}
	| xmltable_column_list ',' xmltable_column_el
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

xmltable_column_el:
	  ColId Typename
	{
		$$ = &RangeTableFuncCol{Colname: toStr($1), TypeName: toTypeNameNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ColId Typename xmltable_column_option_list
	{
		col := &RangeTableFuncCol{Colname: toStr($1), TypeName: toTypeNameNode($2), Location: -1}
		seenNull := false
		for _, option := range toNodeSlice($3) {
		 def := option.GetDefElem()
		 switch def.Defname {
		 case "default": if col.Coldefexpr != nil { yylex.Error("only one DEFAULT value is allowed"); return 1 }; col.Coldefexpr = toNode(def.Arg)
		 case "path": if col.Colexpr != nil { yylex.Error("only one PATH value is allowed"); return 1 }; col.Colexpr = toNode(def.Arg)
		 case "__pg__is_not_null": if seenNull { yylex.Error("conflicting NULL declarations"); return 1 }; seenNull = true; col.IsNotNull = def.Arg.GetBoolean().Boolval
		 default: yylex.Error("unrecognized XMLTABLE column option"); return 1
		 }
		}
		$$ = col
		grammarSetLocation($$, @1)
	}
	| ColId FOR ORDINALITY
	{
		$$ = &RangeTableFuncCol{Colname: toStr($1), ForOrdinality: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

xmltable_column_option_list:
	  xmltable_column_option_el
	{
		$$ = []Node{toNode($1)}
	}
	| xmltable_column_option_list xmltable_column_option_el
	{
		$$ = appendToNodeSlice($1, $2)
	}
	;

xmltable_column_option_el:
	  IDENT b_expr
	{
		if toStr($1) == "__pg__is_not_null" { yylex.(*Lexer).errorAt("reserved XMLTABLE option name", @1); return 1 }
		$$ = &DefElem{Defname: toStr($1), Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| DEFAULT b_expr
	{
		$$ = &DefElem{Defname: "default", Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| NOT NULL_P
	{
		$$ = &DefElem{Defname: "__pg__is_not_null", Arg: toNode(&Boolean{Boolval: true}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| NULL_P
	{
		$$ = &DefElem{Defname: "__pg__is_not_null", Arg: toNode(&Boolean{Boolval: false}), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	| PATH b_expr
	{
		$$ = &DefElem{Defname: "path", Arg: toNode($2), Location: -1, Defaction: pg_query.DefElemAction(1)}
		grammarSetLocation($$, @1)
	}
	;

xml_namespace_list:
	  xml_namespace_el
	{
		$$ = []Node{toNode($1)}
	}
	| xml_namespace_list ',' xml_namespace_el
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

xml_namespace_el:
	  b_expr AS ColLabel
	{
		$$ = &ResTarget{Name: toStr($3), Val: toNode($1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| DEFAULT b_expr
	{
		$$ = &ResTarget{Val: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

json_table:
	  JSON_TABLE '(' 				json_value_expr ',' a_expr json_table_path_name_opt 				json_passing_clause_opt 				COLUMNS '(' json_table_column_definition_list ')' 				json_on_error_clause_opt 			')'
	{
		path, ok := $5.(*A_Const)
		if !ok { yylex.(*Lexer).errorAt("only string constants are supported in JSON_TABLE path specification", @5); return 1 }
		if _, ok := aConstValue(path).(string); !ok { yylex.(*Lexer).errorAt("only string constants are supported in JSON_TABLE path specification", @5); return 1 }
		nameLocation := int32(-1)
		if $6 != nil { nameLocation = int32(@6) }
		path.Location = int32(@5)
		$$ = &JsonTable{ContextItem: toNode($3).GetJsonValueExpr(), Pathspec: &JsonTablePathSpec{String_: toNode(path), Name: toStr($6), Location: int32(@5), NameLocation: nameLocation}, Passing: toNodeSlice($7), Columns: toNodeSlice($10), OnError: toNode($12).GetJsonBehavior(), Location: int32(@1)}
		grammarSetLocation($$, @1)
	}
	;

json_table_path_name_opt:
	  AS name
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_table_column_definition_list:
	  json_table_column_definition
	{
		$$ = []Node{toNode($1)}
	}
	| json_table_column_definition_list ',' json_table_column_definition
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

json_table_column_definition:
	  ColId FOR ORDINALITY
	{
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(1), Name: toStr($1), Wrapper: pg_query.JsonWrapper(1), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ColId Typename 				json_table_column_path_clause_opt 				json_wrapper_behavior 				json_quotes_clause_opt 				json_behavior_clause_opt
	{
		behavior := toNodeSlice($6)
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(2), Name: toStr($1), TypeName: toTypeNameNode($2), Format: toNode(&JsonFormat{FormatType: pg_query.JsonFormatType(1), Encoding: pg_query.JsonEncoding(1), Location: -1}).GetJsonFormat(), Pathspec: toNode(toNode($3)).GetJsonTablePathSpec(), Wrapper: pg_query.JsonWrapper(toInt32($4)), Quotes: pg_query.JsonQuotes(toInt32($5)), OnEmpty: toNode(behavior[0]).GetJsonBehavior(), OnError: toNode(behavior[1]).GetJsonBehavior(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ColId Typename json_format_clause 				json_table_column_path_clause_opt 				json_wrapper_behavior 				json_quotes_clause_opt 				json_behavior_clause_opt
	{
		behavior := toNodeSlice($7)
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(4), Name: toStr($1), TypeName: toTypeNameNode($2), Format: toNode(toNode($3)).GetJsonFormat(), Pathspec: toNode(toNode($4)).GetJsonTablePathSpec(), Wrapper: pg_query.JsonWrapper(toInt32($5)), Quotes: pg_query.JsonQuotes(toInt32($6)), OnEmpty: toNode(behavior[0]).GetJsonBehavior(), OnError: toNode(behavior[1]).GetJsonBehavior(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ColId Typename 				EXISTS json_table_column_path_clause_opt 				json_on_error_clause_opt
	{
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(3), Name: toStr($1), TypeName: toTypeNameNode($2), Format: toNode(&JsonFormat{FormatType: pg_query.JsonFormatType(1), Encoding: pg_query.JsonEncoding(1), Location: -1}).GetJsonFormat(), Pathspec: toNode(toNode($4)).GetJsonTablePathSpec(), OnError: toNode(toNode($5)).GetJsonBehavior(), Wrapper: pg_query.JsonWrapper(2), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| NESTED path_opt Sconst 				COLUMNS '(' json_table_column_definition_list ')'
	{
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(5), Pathspec: &JsonTablePathSpec{String_: toNode(grammarConst(toStr($3), @3)), Location: int32(@3), NameLocation: -1}, Columns: toNodeSlice($6), Wrapper: pg_query.JsonWrapper(1), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| NESTED path_opt Sconst AS name 				COLUMNS '(' json_table_column_definition_list ')'
	{
		$$ = &JsonTableColumn{Coltype: pg_query.JsonTableColumnType(5), Pathspec: &JsonTablePathSpec{String_: toNode(grammarConst(toStr($3), @3)), Name: toStr($5), Location: int32(@3), NameLocation: int32(@5)}, Columns: toNodeSlice($8), Wrapper: pg_query.JsonWrapper(1), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

path_opt:
	  PATH
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_table_column_path_clause_opt:
	  PATH Sconst
	{
		$$ = &JsonTablePathSpec{String_: toNode(grammarConst(toStr($2), @2)), Location: int32(@2), NameLocation: -1}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

Typename:
	  SimpleTypename opt_array_bounds
	{
		$$ = grammarArrayType($1, $2, false)
	}
	| SETOF SimpleTypename opt_array_bounds
	{
		$$ = grammarArrayType($2, $3, true)
	}
	| SimpleTypename ARRAY '[' Iconst ']'
	{
		$$ = grammarArrayType($1, []int32{toInt32($4)}, false)
	}
	| SETOF SimpleTypename ARRAY '[' Iconst ']'
	{
		$$ = grammarArrayType($2, []int32{toInt32($5)}, true)
	}
	| SimpleTypename ARRAY
	{
		$$ = grammarArrayType($1, []int32{-1}, false)
	}
	| SETOF SimpleTypename ARRAY
	{
		$$ = grammarArrayType($2, []int32{-1}, true)
	}
	;

opt_array_bounds:
	  opt_array_bounds '[' ']'
	{
		$$ = grammarArrayBounds($1, -1)
	}
	| opt_array_bounds '[' Iconst ']'
	{
		$$ = grammarArrayBounds($1, toInt32($3))
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

SimpleTypename:
	  GenericType
	{
		$$ = $1
	}
	| Numeric
	{
		$$ = $1
	}
	| Bit
	{
		$$ = $1
	}
	| Character
	{
		$$ = $1
	}
	| ConstDatetime
	{
		$$ = $1
	}
	| ConstInterval opt_interval
	{
		$$ = grammarType($1, $2)
	}
	| ConstInterval '(' Iconst ')'
	{
		$$ = grammarType($1, []Node{toNode(makeAConst(int64(32767))),toNode(makeAConst(toInt64($3)))})
	}
	| JsonType
	{
		$$ = $1
	}
	;

ConstTypename:
	  Numeric
	{
		$$ = $1
	}
	| ConstBit
	{
		$$ = $1
	}
	| ConstCharacter
	{
		$$ = $1
	}
	| ConstDatetime
	{
		$$ = $1
	}
	| JsonType
	{
		$$ = $1
	}
	;

GenericType:
	  type_function_name opt_type_modifiers
	{
		$$ = &TypeName{Names: toPgStringNodes([]string{toStr($1)}), Typemod: -1, Typmods: toNodeSlice($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| type_function_name attrs opt_type_modifiers
	{
		$$ = &TypeName{Names: toPgStringNodes(append([]string{toStr($1)}, toStrSlice($2)...)), Typemod: -1, Typmods: toNodeSlice($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

opt_type_modifiers:
	  '(' expr_list ')'
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

Numeric:
	  INT_P
	{
		$$ = grammarSystemType("int4", nil)
		grammarSetLocation($$, @1)
	}
	| INTEGER
	{
		$$ = grammarSystemType("int4", nil)
		grammarSetLocation($$, @1)
	}
	| SMALLINT
	{
		$$ = grammarSystemType("int2", nil)
		grammarSetLocation($$, @1)
	}
	| BIGINT
	{
		$$ = grammarSystemType("int8", nil)
		grammarSetLocation($$, @1)
	}
	| REAL
	{
		$$ = grammarSystemType("float4", nil)
		grammarSetLocation($$, @1)
	}
	| FLOAT_P opt_float
	{
		$$ = $2
		grammarSetLocation($$, @1)
	}
	| DOUBLE_P PRECISION
	{
		$$ = grammarSystemType("float8", nil)
		grammarSetLocation($$, @1)
	}
	| DECIMAL_P opt_type_modifiers
	{
		$$ = grammarSystemType("numeric", $2)
		grammarSetLocation($$, @1)
	}
	| DEC opt_type_modifiers
	{
		$$ = grammarSystemType("numeric", $2)
		grammarSetLocation($$, @1)
	}
	| NUMERIC opt_type_modifiers
	{
		$$ = grammarSystemType("numeric", $2)
		grammarSetLocation($$, @1)
	}
	| BOOLEAN_P
	{
		$$ = grammarSystemType("bool", nil)
		grammarSetLocation($$, @1)
	}
	;

opt_float:
	  '(' Iconst ')'
	{
		p := toInt64($2)
		if p < 1 || p > 53 { yylex.(*Lexer).errorAt("precision for type float must be between 1 and 53 bits", @2); return 1 }
		if p <= 24 { $$ = grammarSystemType("float4", nil) } else { $$ = grammarSystemType("float8", nil) }
	}
	| /* EMPTY */
	{
		$$ = grammarSystemType("float8", nil)
	}
	;

Bit:
	  BitWithLength
	{
		$$ = $1
	}
	| BitWithoutLength
	{
		$$ = $1
	}
	;

ConstBit:
	  BitWithLength
	{
		$$ = $1
	}
	| BitWithoutLength
	{
		$$ = grammarType($1, nil)
	}
	;

BitWithLength:
	  BIT opt_varying '(' expr_list ')'
	{
		name := "bit"
		if toBool($2) { name = "varbit" }
		$$ = grammarSystemType(name, $4)
		grammarSetLocation($$, @1)
	}
	;

BitWithoutLength:
	  BIT opt_varying
	{
		if toBool($2) { $$ = grammarSystemType("varbit", nil) } else { $$ = grammarSystemType("bit", []Node{toNode(makeAConst(int64(1)))}) }
		grammarSetLocation($$, @1)
	}
	;

Character:
	  CharacterWithLength
	{
		$$ = $1
	}
	| CharacterWithoutLength
	{
		$$ = $1
	}
	;

ConstCharacter:
	  CharacterWithLength
	{
		$$ = $1
	}
	| CharacterWithoutLength
	{
		$$ = grammarType($1, nil)
	}
	;

CharacterWithLength:
	  character '(' Iconst ')'
	{
		length := makeAConst(toInt64($3))
		length.Location = int32(@3)
		$$ = grammarSystemType(toStr($1), []Node{toNode(length)})
		grammarSetLocation($$, @1)
	}
	;

CharacterWithoutLength:
	  character
	{
		n := grammarSystemType(toStr($1), nil)
		if toStr($1) == "bpchar" { n.Typmods = []Node{toNode(makeAConst(int64(1)))}}
		$$ = n
		grammarSetLocation($$, @1)
	}
	;

character:
	  CHARACTER opt_varying
	{
		if toBool($2) { $$ = "varchar" } else { $$ = "bpchar" }
	}
	| CHAR_P opt_varying
	{
		if toBool($2) { $$ = "varchar" } else { $$ = "bpchar" }
	}
	| VARCHAR
	{
		$$ = "varchar"
	}
	| NATIONAL CHARACTER opt_varying
	{
		if toBool($3) { $$ = "varchar" } else { $$ = "bpchar" }
	}
	| NATIONAL CHAR_P opt_varying
	{
		if toBool($3) { $$ = "varchar" } else { $$ = "bpchar" }
	}
	| NCHAR opt_varying
	{
		if toBool($2) { $$ = "varchar" } else { $$ = "bpchar" }
	}
	;

opt_varying:
	  VARYING
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

ConstDatetime:
	  TIMESTAMP '(' Iconst ')' opt_timezone
	{
		name := "timestamp"
		if toBool($5) { name = "timestamptz" }
		$$ = grammarSystemType(name, []Node{toNode(grammarConst(toInt64($3), @3))})
		grammarSetLocation($$, @1)
	}
	| TIMESTAMP opt_timezone
	{
		name := "timestamp"
		if toBool($2) { name = "timestamptz" }
		$$ = grammarSystemType(name, nil)
		grammarSetLocation($$, @1)
	}
	| TIME '(' Iconst ')' opt_timezone
	{
		name := "time"
		if toBool($5) { name = "timetz" }
		$$ = grammarSystemType(name, []Node{toNode(grammarConst(toInt64($3), @3))})
		grammarSetLocation($$, @1)
	}
	| TIME opt_timezone
	{
		name := "time"
		if toBool($2) { name = "timetz" }
		$$ = grammarSystemType(name, nil)
		grammarSetLocation($$, @1)
	}
	;

ConstInterval:
	  INTERVAL
	{
		$$ = grammarSystemType("interval", nil)
		grammarSetLocation($$, @1)
	}
	;

opt_timezone:
	  WITH_LA TIME ZONE
	{
		$$ = true
	}
	| WITHOUT_LA TIME ZONE
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_interval:
	  YEAR_P
	{
		$$ = []Node{toNode(grammarConst(int64(4), @1))}
	}
	| MONTH_P
	{
		$$ = []Node{toNode(grammarConst(int64(2), @1))}
	}
	| DAY_P
	{
		$$ = []Node{toNode(grammarConst(int64(8), @1))}
	}
	| HOUR_P
	{
		$$ = []Node{toNode(grammarConst(int64(1024), @1))}
	}
	| MINUTE_P
	{
		$$ = []Node{toNode(grammarConst(int64(2048), @1))}
	}
	| interval_second
	{
		$$ = $1
	}
	| YEAR_P TO MONTH_P
	{
		$$ = []Node{toNode(grammarConst(int64(6), @1))}
	}
	| DAY_P TO HOUR_P
	{
		$$ = []Node{toNode(grammarConst(int64(1032), @1))}
	}
	| DAY_P TO MINUTE_P
	{
		$$ = []Node{toNode(grammarConst(int64(3080), @1))}
	}
	| DAY_P TO interval_second
	{
		mods := toNodeSlice($3)
		mods[0] = toNode(grammarConst(int64(7176), @1))
		$$ = mods
	}
	| HOUR_P TO MINUTE_P
	{
		$$ = []Node{toNode(grammarConst(int64(3072), @1))}
	}
	| HOUR_P TO interval_second
	{
		mods := toNodeSlice($3)
		mods[0] = toNode(grammarConst(int64(7168), @1))
		$$ = mods
	}
	| MINUTE_P TO interval_second
	{
		mods := toNodeSlice($3)
		mods[0] = toNode(grammarConst(int64(6144), @1))
		$$ = mods
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

interval_second:
	  SECOND_P
	{
		$$ = []Node{toNode(grammarConst(int64(4096), @1))}
	}
	| SECOND_P '(' Iconst ')'
	{
		$$ = []Node{toNode(grammarConst(int64(4096), @1)), toNode(grammarConst(toInt64($3), @3))}
	}
	;

JsonType:
	  JSON
	{
		$$ = grammarSystemType("json", nil)
		grammarSetLocation($$, @1)
	}
	;

a_expr:
	  c_expr
	{
		$$ = $1
	}
	| a_expr TYPECAST Typename
	{
		$$ = &TypeCast{Arg: toNode($1), TypeName: toTypeNameNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr COLLATE any_name
	{
		$$ = &CollateClause{Arg: toNode($1), Collname: toPgStringNodes(toStrSlice($3)), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr AT TIME ZONE a_expr			%prec AT
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "timezone"}), Args: []Node{toNode($5), toNode($1)}, Funcformat: pg_query.CoercionForm_COERCE_SQL_SYNTAX, Location: int32(@2)}
	}
	| a_expr AT LOCAL						%prec AT
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "timezone"}), Args: []Node{toNode($1)}, Funcformat: pg_query.CoercionForm_COERCE_SQL_SYNTAX, Location: -1}
	}
	| '+' a_expr					%prec UMINUS
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"+"}), Rexpr: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| '-' a_expr					%prec UMINUS
	{
		$$ = grammarNegate($2, @1)
	}
	| a_expr '+' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"+"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '-' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"-"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '*' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"*"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '/' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"/"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '%' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"%"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '^' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"^"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '<' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '>' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{">"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr '=' a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr LESS_EQUALS a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr GREATER_EQUALS a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{">="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_EQUALS a_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<>"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr qual_Op a_expr				%prec Op
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes(toStrSlice($2)), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| qual_Op a_expr					%prec Op
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes(toStrSlice($1)), Rexpr: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| a_expr AND a_expr
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_AND_EXPR, toNode(toNode($1)), toNode(toNode($3)))
		grammarSetLocation($$, @2)
	}
	| a_expr OR a_expr
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_OR_EXPR, toNode(toNode($1)), toNode(toNode($3)))
		grammarSetLocation($$, @2)
	}
	| NOT a_expr
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(toNode($2)))
		grammarSetLocation($$, @1)
	}
	| NOT_LA a_expr						%prec NOT
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(toNode($2)))
		grammarSetLocation($$, @1)
	}
	| a_expr LIKE a_expr
	{
		$$ = &A_Expr{Kind: AExprLike, Name: toPgStringNodes([]string{"~~"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr LIKE a_expr ESCAPE a_expr					%prec LIKE
	{
		$$ = &A_Expr{Kind: AExprLike, Name: toPgStringNodes([]string{"~~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "like_escape"}), Args: []Node{toNode($3), toNode($5)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA LIKE a_expr							%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprLike, Name: toPgStringNodes([]string{"!~~"}), Lexpr: toNode($1), Rexpr: toNode($4), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA LIKE a_expr ESCAPE a_expr			%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprLike, Name: toPgStringNodes([]string{"!~~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "like_escape"}), Args: []Node{toNode($4), toNode($6)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr ILIKE a_expr
	{
		$$ = &A_Expr{Kind: AExprILike, Name: toPgStringNodes([]string{"~~*"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr ILIKE a_expr ESCAPE a_expr					%prec ILIKE
	{
		$$ = &A_Expr{Kind: AExprILike, Name: toPgStringNodes([]string{"~~*"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "like_escape"}), Args: []Node{toNode($3), toNode($5)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA ILIKE a_expr						%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprILike, Name: toPgStringNodes([]string{"!~~*"}), Lexpr: toNode($1), Rexpr: toNode($4), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA ILIKE a_expr ESCAPE a_expr			%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprILike, Name: toPgStringNodes([]string{"!~~*"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "like_escape"}), Args: []Node{toNode($4), toNode($6)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr SIMILAR TO a_expr							%prec SIMILAR
	{
		$$ = &A_Expr{Kind: AExprSimilar, Name: toPgStringNodes([]string{"~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "similar_to_escape"}), Args: []Node{toNode($4)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr SIMILAR TO a_expr ESCAPE a_expr			%prec SIMILAR
	{
		$$ = &A_Expr{Kind: AExprSimilar, Name: toPgStringNodes([]string{"~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "similar_to_escape"}), Args: []Node{toNode($4), toNode($6)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA SIMILAR TO a_expr					%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprSimilar, Name: toPgStringNodes([]string{"!~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "similar_to_escape"}), Args: []Node{toNode($5)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA SIMILAR TO a_expr ESCAPE a_expr		%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprSimilar, Name: toPgStringNodes([]string{"!~"}), Lexpr: toNode($1), Rexpr: toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "similar_to_escape"}), Args: []Node{toNode($5), toNode($7)}, Location: int32(@2), Funcformat: pg_query.CoercionForm(1)}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NULL_P							%prec IS
	{
		$$ = &NullTest{Arg: toNode($1), Nulltesttype: grammarNullTestType(false), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr ISNULL
	{
		$$ = &NullTest{Arg: toNode($1), Nulltesttype: grammarNullTestType(false), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT NULL_P						%prec IS
	{
		$$ = &NullTest{Arg: toNode($1), Nulltesttype: grammarNullTestType(true), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOTNULL
	{
		$$ = &NullTest{Arg: toNode($1), Nulltesttype: grammarNullTestType(true), Location: -1}
		grammarSetLocation($$, @2)
	}
	| row OVERLAPS row
	{
		left, right := toNodeSlice($1), toNodeSlice($3)
		if len(left) != 2 || len(right) != 2 { yylex.Error("wrong number of parameters in OVERLAPS expression"); return 1 }
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "overlaps"}), Args: toNodeSlice(append(left, right...)), Funcformat: pg_query.CoercionForm(4), Location: int32(@2)}
	}
	| a_expr IS TRUE_P							%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsTrue, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT TRUE_P						%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsNotTrue, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS FALSE_P							%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsFalse, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT FALSE_P						%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsNotFalse, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS UNKNOWN							%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsUnknown, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT UNKNOWN						%prec IS
	{
		$$ = &BooleanTest{Arg: toNode($1), Booltesttype: BoolTestIsNotUnknown, Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS DISTINCT FROM a_expr			%prec IS
	{
		$$ = &A_Expr{Kind: AExprDistinct, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($5), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT DISTINCT FROM a_expr		%prec IS
	{
		$$ = &A_Expr{Kind: AExprNotDistinct, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($6), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr BETWEEN opt_asymmetric b_expr AND a_expr		%prec BETWEEN
	{
		$$ = &A_Expr{Kind: AExprBetween, Name: toPgStringNodes([]string{"BETWEEN"}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: []Node{toNode($4), toNode($6)}}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA BETWEEN opt_asymmetric b_expr AND a_expr %prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprNotBetween, Name: toPgStringNodes([]string{"NOT BETWEEN"}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: []Node{toNode($5), toNode($7)}}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr BETWEEN SYMMETRIC b_expr AND a_expr			%prec BETWEEN
	{
		$$ = &A_Expr{Kind: AExprBetweenSymmetric, Name: toPgStringNodes([]string{"BETWEEN SYMMETRIC"}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: []Node{toNode($4), toNode($6)}}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA BETWEEN SYMMETRIC b_expr AND a_expr		%prec NOT_LA
	{
		$$ = &A_Expr{Kind: AExprNotBetweenSymmetric, Name: toPgStringNodes([]string{"NOT BETWEEN SYMMETRIC"}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: []Node{toNode($5), toNode($7)}}), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IN_P select_with_parens
	{
		$$ = &SubLink{SubLinkType: SubLinkAny, Testexpr: toNode($1), Subselect: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IN_P '(' expr_list ')'
	{
		$$ = &A_Expr{Kind: AExprIn, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: toNodeSlice($4)}), RexprListStart: int32(@3), RexprListEnd: int32(@5), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA IN_P select_with_parens			%prec NOT_LA
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&SubLink{SubLinkType: SubLinkAny, Testexpr: toNode($1), Subselect: toNode($4), Location: int32(@2)}))
		grammarSetLocation($$, @2)
	}
	| a_expr NOT_LA IN_P '(' expr_list ')'
	{
		$$ = &A_Expr{Kind: AExprIn, Name: toPgStringNodes([]string{"<>"}), Lexpr: toNode($1), Rexpr: toNode(&List{Items: toNodeSlice($5)}), RexprListStart: int32(@4), RexprListEnd: int32(@6), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr subquery_Op sub_type select_with_parens	%prec Op
	{
		$$ = &SubLink{SubLinkType: $3.(SubLinkType), Testexpr: toNode($1), OperName: toPgStringNodes(toStrSlice($2)), Subselect: toNode($4), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr subquery_Op sub_type '(' a_expr ')'		%prec Op
	{
		kind := AExprOpAny
		if $3.(SubLinkType) == SubLinkAll { kind = AExprOpAll }
		$$ = &A_Expr{Kind: kind, Name: toPgStringNodes(toStrSlice($2)), Lexpr: toNode($1), Rexpr: toNode($5), Location: -1}
		grammarSetLocation($$, @2)
	}
	| UNIQUE opt_unique_null_treatment select_with_parens
	{
		yylex.(*Lexer).errorAt("UNIQUE predicate is not yet implemented", @1); return 1
	}
	| a_expr IS DOCUMENT_P					%prec IS
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(8), Args: []Node{toNode($1)}, Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT DOCUMENT_P				%prec IS
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&XmlExpr{Op: pg_query.XmlExprOp(8), Args: []Node{toNode($1)}, Xmloption: pg_query.XmlOptionType(1), Location: int32(@2)}))
		grammarSetLocation($$, @2)
	}
	| a_expr IS NORMALIZED								%prec IS
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "is_normalized"}), Args: []Node{toNode($1)}, Funcformat: pg_query.CoercionForm(4), Location: int32(@2)}
	}
	| a_expr IS unicode_normal_form NORMALIZED			%prec IS
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "is_normalized"}), Args: []Node{toNode($1), toNode(grammarConst(toStr($3), @3))}, Funcformat: pg_query.CoercionForm(4), Location: int32(@2)}
	}
	| a_expr IS NOT NORMALIZED							%prec IS
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "is_normalized"}), Args: []Node{toNode($1)}, Funcformat: pg_query.CoercionForm(4), Location: int32(@2)}))
		grammarSetLocation($$, @2)
	}
	| a_expr IS NOT unicode_normal_form NORMALIZED		%prec IS
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "is_normalized"}), Args: []Node{toNode($1), toNode(grammarConst(toStr($4), @4))}, Funcformat: pg_query.CoercionForm(4), Location: int32(@2)}))
		grammarSetLocation($$, @2)
	}
	| a_expr IS json_predicate_type_constraint 					json_key_uniqueness_constraint_opt		%prec IS
	{
		$$ = &JsonIsPredicate{Expr: toNode($1), Format: toNode(&JsonFormat{FormatType: pg_query.JsonFormatType(1), Encoding: pg_query.JsonEncoding(1), Location: -1}).GetJsonFormat(), ItemType: pg_query.JsonValueType(toInt32($3)), UniqueKeys: toBool($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	| a_expr IS NOT 					json_predicate_type_constraint 					json_key_uniqueness_constraint_opt		%prec IS
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&JsonIsPredicate{Expr: toNode($1), Format: &JsonFormat{FormatType: pg_query.JsonFormatType(1), Encoding: pg_query.JsonEncoding(1), Location: -1}, ItemType: pg_query.JsonValueType(toInt32($4)), UniqueKeys: toBool($5), Location: int32(@1)}))
		grammarSetLocation($$, @1)
	}
	| DEFAULT
	{
		$$ = &SetToDefault{}
		grammarSetLocation($$, @1)
	}
	;

b_expr:
	  c_expr
	{
		$$ = $1
	}
	| b_expr TYPECAST Typename
	{
		$$ = &TypeCast{Arg: toNode($1), TypeName: toTypeNameNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| '+' b_expr					%prec UMINUS
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"+"}), Rexpr: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| '-' b_expr					%prec UMINUS
	{
		$$ = grammarNegate($2, @1)
	}
	| b_expr '+' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"+"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '-' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"-"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '*' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"*"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '/' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"/"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '%' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"%"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '^' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"^"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '<' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '>' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{">"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr '=' b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr LESS_EQUALS b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr GREATER_EQUALS b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{">="}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr NOT_EQUALS b_expr
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes([]string{"<>"}), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr qual_Op b_expr				%prec Op
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes(toStrSlice($2)), Lexpr: toNode($1), Rexpr: toNode($3), Location: -1}
		grammarSetLocation($$, @2)
	}
	| qual_Op b_expr					%prec Op
	{
		$$ = &A_Expr{Kind: AExprOp, Name: toPgStringNodes(toStrSlice($1)), Rexpr: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| b_expr IS DISTINCT FROM b_expr		%prec IS
	{
		$$ = &A_Expr{Kind: AExprDistinct, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($5), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr IS NOT DISTINCT FROM b_expr	%prec IS
	{
		$$ = &A_Expr{Kind: AExprNotDistinct, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($1), Rexpr: toNode($6), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr IS DOCUMENT_P					%prec IS
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(8), Args: []Node{toNode($1)}, Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @2)
	}
	| b_expr IS NOT DOCUMENT_P				%prec IS
	{
		$$ = grammarBoolExpr(pg_query.BoolExprType_NOT_EXPR, toNode(nil), toNode(&XmlExpr{Op: pg_query.XmlExprOp(8), Args: []Node{toNode($1)}, Xmloption: pg_query.XmlOptionType(1), Location: int32(@2)}))
		grammarSetLocation($$, @2)
	}
	;

c_expr:
	  columnref
	{
		$$ = $1
	}
	| AexprConst
	{
		$$ = $1
	}
	| PARAM opt_indirection
	{
		$$ = grammarIndirection(toNode(&ParamRef{Number: int32(int(toInt64($1))), Location: -1}), $2)
		grammarSetLocation($$, @1)
	}
	| '(' a_expr ')' opt_indirection
	{
		$$ = grammarIndirection(toNode($2), $4)
	}
	| case_expr
	{
		$$ = $1
	}
	| func_expr
	{
		$$ = $1
	}
	| select_with_parens			%prec UMINUS
	{
		$$ = &SubLink{SubLinkType: SubLinkExpr, Subselect: toNode($1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| select_with_parens indirection
	{
		$$ = grammarIndirection(toNode(&SubLink{SubLinkType: SubLinkExpr, Subselect: toNode($1), Location: int32(@1)}), $2)
		grammarSetLocation($$, @1)
	}
	| EXISTS select_with_parens
	{
		$$ = &SubLink{SubLinkType: SubLinkExists, Subselect: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ARRAY select_with_parens
	{
		$$ = &SubLink{SubLinkType: SubLinkArray, Subselect: toNode($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| ARRAY array_expr
	{
		$$ = $2
		grammarSetLocation($$, @1)
	}
	| explicit_row
	{
		$$ = $1
		grammarSetLocation($$, @1)
	}
	| implicit_row
	{
		$$ = $1
		grammarSetLocation($$, @1)
	}
	| GROUPING '(' expr_list ')'
	{
		$$ = &GroupingFunc{Args: toNodeSlice($3)}
		grammarSetLocation($$, @1)
	}
	;

func_application:
	  func_name '(' ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' func_arg_list opt_sort_clause ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Args: toNodeSlice($3), AggOrder: toNodeSlice(toSortBySlice($4)), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' VARIADIC func_arg_expr opt_sort_clause ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Args: []Node{toNode($4)}, AggOrder: toNodeSlice(toSortBySlice($5)), FuncVariadic: true, Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' func_arg_list ',' VARIADIC func_arg_expr opt_sort_clause ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Args: toNodeSlice(appendToNodeSlice($3, $6)), AggOrder: toNodeSlice(toSortBySlice($7)), FuncVariadic: true, Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' ALL func_arg_list opt_sort_clause ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Args: toNodeSlice($4), AggOrder: toNodeSlice(toSortBySlice($5)), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' DISTINCT func_arg_list opt_sort_clause ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), Args: toNodeSlice($4), AggOrder: toNodeSlice(toSortBySlice($5)), AggDistinct: true, Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| func_name '(' '*' ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes(toStrSlice($1)), AggStar: true, Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	;

func_expr:
	  func_application within_group_clause filter_clause over_clause
	{
		fc := toFuncCall($1)
		if fc != nil {
			if $2 != nil {
				if len(fc.AggOrder) != 0 || fc.AggDistinct || fc.FuncVariadic {
					yylex.(*Lexer).errorAt("WITHIN GROUP cannot be combined with aggregate ORDER BY, DISTINCT, or VARIADIC", @2)
					return 1
				}
				fc.AggOrder = toNodeSlice(toSortBySlice($2))
				fc.AggWithinGroup = true
			}
			fc.AggFilter = toNode($3)
			fc.Over = toWindowDef($4)
		}
		$$ = fc
	}
	| json_aggregate_func filter_clause over_clause
	{
		var constructor *JsonAggConstructor
		switch n := $1.(type) { case *JsonObjectAgg: constructor = n.Constructor; case *JsonArrayAgg: constructor = n.Constructor }
		constructor.AggFilter = toNode($2)
		constructor.Over = toWindowDef($3)
		$$ = $1
	}
	| func_expr_common_subexpr
	{
		$$ = $1
		grammarSetLocation($$, @1)
	}
	;

func_expr_windowless:
	  func_application
	{
		$$ = $1
	}
	| func_expr_common_subexpr
	{
		$$ = $1
		grammarSetLocation($$, @1)
	}
	| json_aggregate_func
	{
		$$ = $1
	}
	;

func_expr_common_subexpr:
	  COLLATION FOR '(' a_expr ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "pg_collation_for"}), Args: []Node{toNode($4)}, Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| CURRENT_DATE
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(1), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_TIME
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(2), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_TIME '(' Iconst ')'
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(3), Typmod: toInt32($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_TIMESTAMP
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(4), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_TIMESTAMP '(' Iconst ')'
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(5), Typmod: toInt32($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| LOCALTIME
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(6), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| LOCALTIME '(' Iconst ')'
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(7), Typmod: toInt32($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| LOCALTIMESTAMP
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(8), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| LOCALTIMESTAMP '(' Iconst ')'
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(9), Typmod: toInt32($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_ROLE
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(10), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_USER
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(11), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| SESSION_USER
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(13), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| SYSTEM_USER
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "system_user"}), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| USER
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(12), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_CATALOG
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(14), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_SCHEMA
	{
		$$ = &SQLValueFunction{Op: pg_query.SQLValueFunctionOp(15), Typmod: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| CAST '(' a_expr AS Typename ')'
	{
		$$ = &TypeCast{Arg: toNode($3), TypeName: toTypeNameNode($5), Location: -1}
		grammarSetLocation($$, @1)
	}
	| EXTRACT '(' extract_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "extract"}), Args: toNodeSlice($3), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| NORMALIZE '(' a_expr ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "normalize"}), Args: []Node{toNode($3)}, Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| NORMALIZE '(' a_expr ',' unicode_normal_form ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "normalize"}), Args: []Node{toNode($3), toNode(grammarConst(toStr($5), @5))}, Funcformat: pg_query.CoercionForm(4), Location: int32(@1)}
	}
	| OVERLAY '(' overlay_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "overlay"}), Args: toNodeSlice($3), Funcformat: pg_query.CoercionForm_COERCE_SQL_SYNTAX, Location: int32(@1)}
	}
	| OVERLAY '(' func_arg_list_opt ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"overlay"}), Args: toNodeSlice($3), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| POSITION '(' position_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "position"}), Args: toNodeSlice($3), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| SUBSTRING '(' substr_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "substring"}), Args: toNodeSlice($3), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| SUBSTRING '(' func_arg_list_opt ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"substring"}), Args: toNodeSlice($3), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| TREAT '(' a_expr AS Typename ')'
	{
		typ := toTypeNameNode($5)
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", toStr(typ.Names[len(typ.Names)-1])}), Args: []Node{toNode($3)}, Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| TRIM '(' BOTH trim_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "btrim"}), Args: toNodeSlice($4), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| TRIM '(' LEADING trim_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "ltrim"}), Args: toNodeSlice($4), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| TRIM '(' TRAILING trim_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "rtrim"}), Args: toNodeSlice($4), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| TRIM '(' trim_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "btrim"}), Args: toNodeSlice($3), Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| NULLIF '(' a_expr ',' a_expr ')'
	{
		$$ = &A_Expr{Kind: AExprNullIf, Name: toPgStringNodes([]string{"="}), Lexpr: toNode($3), Rexpr: toNode($5), Location: -1}
		grammarSetLocation($$, @1)
	}
	| COALESCE '(' expr_list ')'
	{
		$$ = &CoalesceExpr{Args: toNodeSlice($3)}
		grammarSetLocation($$, @1)
	}
	| GREATEST '(' expr_list ')'
	{
		$$ = &MinMaxExpr{Args: toNodeSlice($3), Op: pg_query.MinMaxOp(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| LEAST '(' expr_list ')'
	{
		$$ = &MinMaxExpr{Args: toNodeSlice($3), Op: pg_query.MinMaxOp(2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLCONCAT '(' expr_list ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(1), Args: toNodeSlice($3), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLELEMENT '(' NAME_P ColLabel ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(2), Name: toStr($4), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLELEMENT '(' NAME_P ColLabel ',' xml_attributes ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(2), Name: toStr($4), NamedArgs: toNodeSlice($6), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLELEMENT '(' NAME_P ColLabel ',' expr_list ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(2), Name: toStr($4), Args: toNodeSlice($6), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLELEMENT '(' NAME_P ColLabel ',' xml_attributes ',' expr_list ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(2), Name: toStr($4), NamedArgs: toNodeSlice($6), Args: toNodeSlice($8), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLEXISTS '(' c_expr xmlexists_argument ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "xmlexists"}), Args: []Node{toNode($3), toNode($4)}, Funcformat: pg_query.CoercionForm(4), Location: -1}
	}
	| XMLFOREST '(' xml_attribute_list ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(3), NamedArgs: toNodeSlice($3), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLPARSE '(' document_or_content a_expr xml_whitespace_option ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(4), Args: []Node{toNode($4),toNode(makeAConst(toBool($5)))}, Xmloption: pg_query.XmlOptionType(toInt32($3)), Location: -1}
	}
	| XMLPI '(' NAME_P ColLabel ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(5), Name: toStr($4), Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLPI '(' NAME_P ColLabel ',' a_expr ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(5), Name: toStr($4), Args: []Node{toNode($6)}, Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLROOT '(' a_expr ',' xml_root_version opt_xml_root_standalone ')'
	{
		$$ = &XmlExpr{Op: pg_query.XmlExprOp(6), Args: []Node{toNode($3), toNode($5), toNode($6)}, Xmloption: pg_query.XmlOptionType(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| XMLSERIALIZE '(' document_or_content a_expr AS SimpleTypename xml_indent_option ')'
	{
		$$ = &XmlSerialize{Xmloption: pg_query.XmlOptionType(toInt32($3)), Expr: toNode($4), TypeName: toTypeNameNode($6), Indent: toBool($7), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_OBJECT '(' func_arg_list ')'
	{
		$$ = &FuncCall{Funcname: toPgStringNodes([]string{"pg_catalog", "json_object"}), Args: toNodeSlice($3), Location: int32(@1), Funcformat: pg_query.CoercionForm(1)}
	}
	| JSON_OBJECT '(' json_name_and_value_list 				json_object_constructor_null_clause_opt 				json_key_uniqueness_constraint_opt 				json_returning_clause_opt ')'
	{
		$$ = &JsonObjectConstructor{Exprs: toNodeSlice($3), AbsentOnNull: toBool($4), Unique: toBool($5), Output: toNode(toNode($6)).GetJsonOutput(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_OBJECT '(' json_returning_clause_opt ')'
	{
		$$ = &JsonObjectConstructor{Output: toNode(toNode($3)).GetJsonOutput(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_ARRAY '(' 				json_value_expr_list 				json_array_constructor_null_clause_opt 				json_returning_clause_opt 			')'
	{
		$$ = &JsonArrayConstructor{Exprs: toNodeSlice($3), AbsentOnNull: toBool($4), Output: toNode(toNode($5)).GetJsonOutput(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_ARRAY '(' 				select_no_parens 				json_format_clause_opt 				 				json_returning_clause_opt 			')'
	{
		$$ = &JsonArrayQueryConstructor{Query: toNode($3), Format: toNode(toNode($4)).GetJsonFormat(), Output: toNode(toNode($5)).GetJsonOutput(), AbsentOnNull: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_ARRAY '(' 				json_returning_clause_opt 			')'
	{
		$$ = &JsonArrayConstructor{Output: toNode(toNode($3)).GetJsonOutput(), AbsentOnNull: true, Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON '(' json_value_expr json_key_uniqueness_constraint_opt ')'
	{
		$$ = &JsonParseExpr{Expr: toNode(toNode($3)).GetJsonValueExpr(), UniqueKeys: toBool($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_SCALAR '(' a_expr ')'
	{
		$$ = &JsonScalarExpr{Expr: toNode($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_SERIALIZE '(' json_value_expr json_returning_clause_opt ')'
	{
		$$ = &JsonSerializeExpr{Expr: toNode(toNode($3)).GetJsonValueExpr(), Output: toNode(toNode($4)).GetJsonOutput(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| MERGE_ACTION '(' ')'
	{
		$$ = &MergeSupportFunc{Msftype: 25}
		grammarSetLocation($$, @1)
	}
	| JSON_QUERY '(' 				json_value_expr ',' a_expr json_passing_clause_opt 				json_returning_clause_opt 				json_wrapper_behavior 				json_quotes_clause_opt 				json_behavior_clause_opt 			')'
	{
		behavior := toNodeSlice($10)
		$$ = &JsonFuncExpr{Op: pg_query.JsonExprOp(2), ContextItem: toNode(toNode($3)).GetJsonValueExpr(), Pathspec: toNode($5), Passing: toNodeSlice($6), Output: toNode(toNode($7)).GetJsonOutput(), Wrapper: pg_query.JsonWrapper(toInt32($8)), Quotes: pg_query.JsonQuotes(toInt32($9)), OnEmpty: toNode(behavior[0]).GetJsonBehavior(), OnError: toNode(behavior[1]).GetJsonBehavior(), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_EXISTS '(' 				json_value_expr ',' a_expr json_passing_clause_opt 				json_on_error_clause_opt 			')'
	{
		$$ = &JsonFuncExpr{Op: pg_query.JsonExprOp(1), ContextItem: toNode(toNode($3)).GetJsonValueExpr(), Pathspec: toNode($5), Passing: toNodeSlice($6), OnError: toNode(toNode($7)).GetJsonBehavior(), Wrapper: pg_query.JsonWrapper(1), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| JSON_VALUE '(' 				json_value_expr ',' a_expr json_passing_clause_opt 				json_returning_clause_opt 				json_behavior_clause_opt 			')'
	{
		behavior := toNodeSlice($8)
		$$ = &JsonFuncExpr{Op: pg_query.JsonExprOp(3), ContextItem: toNode(toNode($3)).GetJsonValueExpr(), Pathspec: toNode($5), Passing: toNodeSlice($6), Output: toNode(toNode($7)).GetJsonOutput(), OnEmpty: toNode(behavior[0]).GetJsonBehavior(), OnError: toNode(behavior[1]).GetJsonBehavior(), Wrapper: pg_query.JsonWrapper(1), Quotes: pg_query.JsonQuotes(1), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

xml_root_version:
	  VERSION_P a_expr
	{
		$$ = $2
	}
	| VERSION_P NO VALUE_P
	{
		$$ = makeAConst(nil)
	}
	;

opt_xml_root_standalone:
	  ',' STANDALONE_P YES_P
	{
		$$ = makeAConst(int64(0))
	}
	| ',' STANDALONE_P NO
	{
		$$ = makeAConst(int64(1))
	}
	| ',' STANDALONE_P NO VALUE_P
	{
		$$ = makeAConst(int64(2))
	}
	| /* EMPTY */
	{
		$$ = makeAConst(int64(3))
	}
	;

xml_attributes:
	  XMLATTRIBUTES '(' xml_attribute_list ')'
	{
		$$ = $3
	}
	;

xml_attribute_list:
	  xml_attribute_el
	{
		$$ = []Node{toNode($1)}
	}
	| xml_attribute_list ',' xml_attribute_el
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

xml_attribute_el:
	  a_expr AS ColLabel
	{
		$$ = &ResTarget{Name: toStr($3), Val: toNode($1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| a_expr
	{
		$$ = &ResTarget{Val: toNode($1), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

document_or_content:
	  DOCUMENT_P
	{
		$$ = int64(1)
	}
	| CONTENT_P
	{
		$$ = int64(2)
	}
	;

xml_indent_option:
	  INDENT
	{
		$$ = true
	}
	| NO INDENT
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

xml_whitespace_option:
	  PRESERVE WHITESPACE_P
	{
		$$ = true
	}
	| STRIP_P WHITESPACE_P
	{
		$$ = false
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

xmlexists_argument:
	  PASSING c_expr
	{
		$$ = $2
	}
	| PASSING c_expr xml_passing_mech
	{
		$$ = $2
	}
	| PASSING xml_passing_mech c_expr
	{
		$$ = $3
	}
	| PASSING xml_passing_mech c_expr xml_passing_mech
	{
		$$ = $3
	}
	;

xml_passing_mech:
	  BY REF_P
	{
		$$ = $1
	}
	| BY VALUE_P
	{
		$$ = $1
	}
	;

within_group_clause:
	  WITHIN GROUP_P '(' sort_clause ')'
	{
		$$ = $4
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

filter_clause:
	  FILTER '(' WHERE a_expr ')'
	{
		$$ = $4
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

window_clause:
	  WINDOW window_definition_list
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

window_definition_list:
	  window_definition
	{
		$$ = []Node{toNode(toWindowDef($1))}
	}
	| window_definition_list ',' window_definition
	{
		$$ = appendToWindowDefSlice($1, $3)
	}
	;

window_definition:
	  ColId AS window_specification
	{
		w := toWindowDef($3)
		if w != nil { w.Name = toStr($1)}
		$$ = w
	}
	;

over_clause:
	  OVER window_specification
	{
		$$ = $2
	}
	| OVER ColId
	{
		$$ = &WindowDef{Name: toStr($2), FrameOptions: 1058, Location: -1}
		grammarSetLocation($$, @2)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

window_specification:
	  '(' opt_existing_window_name opt_partition_clause 						opt_sort_clause opt_frame_clause ')'
	{
		w := toWindowDef($5)
		w.Refname = toStr($2)
		w.PartitionClause = toNodeSlice($3)
		w.OrderClause = toNodeSlice(toSortBySlice($4))
		$$ = w
		grammarSetLocation($$, @1)
	}
	;

opt_existing_window_name:
	  ColId
	{
		$$ = $1
	}
	| %prec Op
	{
		$$ = nil
	}
	;

opt_partition_clause:
	  PARTITION BY expr_list
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_frame_clause:
	  RANGE frame_extent opt_window_exclusion_clause
	{
		w := toWindowDef($2)
		w.FrameOptions |= 1 | 2 | toInt32($3)
		$$ = w
	}
	| ROWS frame_extent opt_window_exclusion_clause
	{
		w := toWindowDef($2)
		w.FrameOptions |= 1 | 4 | toInt32($3)
		$$ = w
	}
	| GROUPS frame_extent opt_window_exclusion_clause
	{
		w := toWindowDef($2)
		w.FrameOptions |= 1 | 8 | toInt32($3)
		$$ = w
	}
	| /* EMPTY */
	{
		$$ = &WindowDef{FrameOptions: 1058, Location: -1}
	}
	;

frame_extent:
	  frame_bound
	{
		w := toWindowDef($1)
		if w.FrameOptions & (128 | 8192) != 0 { yylex.(*Lexer).errorAt("frame start cannot be FOLLOWING without a following frame end", @1); return 1 }
		w.FrameOptions |= 1024
		$$ = w
	}
	| BETWEEN frame_bound AND frame_bound
	{
		start, end := toWindowDef($2), toWindowDef($4)
		options := start.FrameOptions | (end.FrameOptions << 1) | 16
		if options & 128 != 0 { yylex.Error("frame start cannot be UNBOUNDED FOLLOWING"); return 1 }
		if options & 64 != 0 { yylex.Error("frame end cannot be UNBOUNDED PRECEDING"); return 1 }
		if options & 512 != 0 && options & 4096 != 0 { yylex.Error("frame starting from current row cannot have preceding rows"); return 1 }
		if options & 8192 != 0 && options & (4096 | 1024) != 0 { yylex.Error("frame starting from following row cannot have preceding rows"); return 1 }
		start.FrameOptions = options
		start.EndOffset = end.StartOffset
		$$ = start
	}
	;

frame_bound:
	  UNBOUNDED PRECEDING
	{
		$$ = &WindowDef{FrameOptions: 32, Location: -1}
	}
	| UNBOUNDED FOLLOWING
	{
		$$ = &WindowDef{FrameOptions: 128, Location: -1}
	}
	| CURRENT_P ROW
	{
		$$ = &WindowDef{FrameOptions: 512, Location: -1}
	}
	| a_expr PRECEDING
	{
		$$ = &WindowDef{FrameOptions: 2048, StartOffset: toNode($1), Location: -1}
	}
	| a_expr FOLLOWING
	{
		$$ = &WindowDef{FrameOptions: 8192, StartOffset: toNode($1), Location: -1}
	}
	;

opt_window_exclusion_clause:
	  EXCLUDE CURRENT_P ROW
	{
		$$ = int64(32768)
	}
	| EXCLUDE GROUP_P
	{
		$$ = int64(65536)
	}
	| EXCLUDE TIES
	{
		$$ = int64(131072)
	}
	| EXCLUDE NO OTHERS
	{
		$$ = int64(0)
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

row:
	  ROW '(' expr_list ')'
	{
		$$ = $3
	}
	| ROW '(' ')'
	{
		$$ = []Node(nil)
	}
	| '(' expr_list ',' a_expr ')'
	{
		$$ = appendToNodeSlice($2, $4)
	}
	;

explicit_row:
	  ROW '(' expr_list ')'
	{
		$$ = &RowExpr{Args: toNodeSlice($3), RowFormat: pg_query.CoercionForm(1), Location: -1}
	}
	| ROW '(' ')'
	{
		$$ = &RowExpr{RowFormat: pg_query.CoercionForm(1), Location: -1}
	}
	;

implicit_row:
	  '(' expr_list ',' a_expr ')'
	{
		$$ = &RowExpr{Args: toNodeSlice(appendToNodeSlice($2, $4)), RowFormat: pg_query.CoercionForm(3), Location: -1}
	}
	;

sub_type:
	  ANY
	{
		$$ = SubLinkAny
	}
	| SOME
	{
		$$ = SubLinkAny
	}
	| ALL
	{
		$$ = SubLinkAll
	}
	;

all_Op:
	  Op
	{
		$$ = $1
	}
	| MathOp
	{
		$$ = $1
	}
	;

MathOp:
	  '+'
	{
		$$ = "+"
	}
	| '-'
	{
		$$ = "-"
	}
	| '*'
	{
		$$ = "*"
	}
	| '/'
	{
		$$ = "/"
	}
	| '%'
	{
		$$ = "%"
	}
	| '^'
	{
		$$ = "^"
	}
	| '<'
	{
		$$ = "<"
	}
	| '>'
	{
		$$ = ">"
	}
	| '='
	{
		$$ = "="
	}
	| LESS_EQUALS
	{
		$$ = "<="
	}
	| GREATER_EQUALS
	{
		$$ = ">="
	}
	| NOT_EQUALS
	{
		$$ = "<>"
	}
	;

qual_Op:
	  Op
	{
		$$ = $1
	}
	| OPERATOR '(' any_operator ')'
	{
		$$ = $3
	}
	;

qual_all_Op:
	  all_Op
	{
		$$ = $1
	}
	| OPERATOR '(' any_operator ')'
	{
		$$ = $3
	}
	;

subquery_Op:
	  all_Op
	{
		$$ = $1
	}
	| OPERATOR '(' any_operator ')'
	{
		$$ = $3
	}
	| LIKE
	{
		$$ = "~~"
	}
	| NOT_LA LIKE
	{
		$$ = "!~~"
	}
	| ILIKE
	{
		$$ = "~~*"
	}
	| NOT_LA ILIKE
	{
		$$ = "!~~*"
	}
	;

expr_list:
	  a_expr
	{
		$$ = []Node{toNode($1)}
	}
	| expr_list ',' a_expr
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

func_arg_list:
	  func_arg_expr
	{
		$$ = []Node{toNode($1)}
	}
	| func_arg_list ',' func_arg_expr
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

func_arg_expr:
	  a_expr
	{
		$$ = $1
	}
	| param_name COLON_EQUALS a_expr
	{
		$$ = &NamedArgExpr{Name: toStr($1), Arg: toNode($3), Argnumber: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	| param_name EQUALS_GREATER a_expr
	{
		$$ = &NamedArgExpr{Name: toStr($1), Arg: toNode($3), Argnumber: -1, Location: -1}
		grammarSetLocation($$, @1)
	}
	;

func_arg_list_opt:
	  func_arg_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

type_list:
	  Typename
	{
		$$ = []Node{toNode(toTypeNameNode($1))}
	}
	| type_list ',' Typename
	{
		$$ = appendToNodeSlice($1, toTypeNameNode($3))
	}
	;

array_expr:
	  '[' expr_list ']'
	{
		$$ = &A_ArrayExpr{Elements: toNodeSlice($2), ListStart: int32(@1), ListEnd: int32(@3), Location: int32(@1)}
	}
	| '[' array_expr_list ']'
	{
		$$ = &A_ArrayExpr{Elements: toNodeSlice($2), ListStart: int32(@1), ListEnd: int32(@3), Location: int32(@1)}
	}
	| '[' ']'
	{
		$$ = &A_ArrayExpr{ListStart: int32(@1), ListEnd: int32(@2), Location: int32(@1)}
	}
	;

array_expr_list:
	  array_expr
	{
		$$ = []Node{toNode($1)}
	}
	| array_expr_list ',' array_expr
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

extract_list:
	  extract_arg FROM a_expr
	{
		$$ = []Node{toNode(grammarConst(toStr($1), @1)), toNode($3)}
	}
	;

extract_arg:
	  IDENT
	{
		$$ = $1
	}
	| YEAR_P
	{
		$$ = $1
	}
	| MONTH_P
	{
		$$ = $1
	}
	| DAY_P
	{
		$$ = $1
	}
	| HOUR_P
	{
		$$ = $1
	}
	| MINUTE_P
	{
		$$ = $1
	}
	| SECOND_P
	{
		$$ = $1
	}
	| Sconst
	{
		$$ = $1
	}
	;

unicode_normal_form:
	  NFC
	{
		$$ = "NFC"
	}
	| NFD
	{
		$$ = "NFD"
	}
	| NFKC
	{
		$$ = "NFKC"
	}
	| NFKD
	{
		$$ = "NFKD"
	}
	;

overlay_list:
	  a_expr PLACING a_expr FROM a_expr FOR a_expr
	{
		$$ = []Node{toNode($1), toNode($3), toNode($5), toNode($7)}
	}
	| a_expr PLACING a_expr FROM a_expr
	{
		$$ = []Node{toNode($1), toNode($3), toNode($5)}
	}
	;

position_list:
	  b_expr IN_P b_expr
	{
		$$ = []Node{toNode($3), toNode($1)}
	}
	;

substr_list:
	  a_expr FROM a_expr FOR a_expr
	{
		$$ = []Node{toNode($1), toNode($3), toNode($5)}
	}
	| a_expr FOR a_expr FROM a_expr
	{
		$$ = []Node{toNode($1), toNode($5), toNode($3)}
	}
	| a_expr FROM a_expr
	{
		$$ = []Node{toNode($1), toNode($3)}
	}
	| a_expr FOR a_expr
	{
		$$ = []Node{toNode($1),toNode(makeAConst(int64(1))),toNode(&TypeCast{Arg: toNode($3), TypeName: grammarSystemType("int4", nil), Location: -1})}
	}
	| a_expr SIMILAR a_expr ESCAPE a_expr
	{
		$$ = []Node{toNode($1), toNode($3), toNode($5)}
	}
	;

trim_list:
	  a_expr FROM expr_list
	{
		$$ = appendToNodeSlice($3, $1)
	}
	| FROM expr_list
	{
		$$ = $2
	}
	| expr_list
	{
		$$ = $1
	}
	;

case_expr:
	  CASE case_arg when_clause_list case_default END_P
	{
		$$ = &CaseExpr{Arg: toNode($2), Args: toNodeSlice(toCaseWhenSlice($3)), Defresult: toNode($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

when_clause_list:
	  when_clause
	{
		$$ = []Node{toNode(toCaseWhen($1))}
	}
	| when_clause_list when_clause
	{
		$$ = appendToCaseWhenSlice($1, $2)
	}
	;

when_clause:
	  WHEN a_expr THEN a_expr
	{
		$$ = &CaseWhen{Expr: toNode($2), Result: toNode($4), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

case_default:
	  ELSE a_expr
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

case_arg:
	  a_expr
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

columnref:
	  ColId
	{
		$$ = &ColumnRef{Fields: []Node{toNode(&String{Sval: toStr($1)})}, Location: -1}
		grammarSetLocation($$, @1)
	}
	| ColId indirection
	{
		$$ = grammarColumnRef($1, $2, @1)
		grammarSetLocation($$, @1)
	}
	;

indirection_el:
	  '.' attr_name
	{
		$$ = &String{Sval: toStr($2)}
	}
	| '.' '*'
	{
		$$ = &A_Star{}
	}
	| '[' a_expr ']'
	{
		$$ = &A_Indices{Uidx: toNode($2)}
	}
	| '[' opt_slice_bound ':' opt_slice_bound ']'
	{
		$$ = &A_Indices{IsSlice: true, Lidx: toNode($2), Uidx: toNode($4)}
	}
	;

opt_slice_bound:
	  a_expr
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

indirection:
	  indirection_el
	{
		$$ = []Node{toNode($1)}
	}
	| indirection indirection_el
	{
		items := toNodeSlice($1)
		if len(items) > 0 { if items[len(items)-1].GetAStar() != nil { yylex.Error("improper use of star"); return 1 } }
		$$ = append(items, toNode($2))
	}
	;

opt_indirection:
	  /* EMPTY */
	{
		$$ = nil
	}
	| opt_indirection indirection_el
	{
		items := toNodeSlice($1)
		if len(items) > 0 { if items[len(items)-1].GetAStar() != nil { yylex.Error("improper use of star"); return 1 } }
		$$ = append(items, toNode($2))
	}
	;

opt_asymmetric:
	  ASYMMETRIC
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_passing_clause_opt:
	  PASSING json_arguments
	{
		$$ = $2
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_arguments:
	  json_argument
	{
		$$ = []Node{toNode($1)}
	}
	| json_arguments ',' json_argument
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

json_argument:
	  json_value_expr AS ColLabel
	{
		$$ = &JsonArgument{Val: toNode(toNode($1)).GetJsonValueExpr(), Name: toStr($3)}
	}
	;

json_wrapper_behavior:
	  WITHOUT WRAPPER
	{
		$$ = int64(2)
	}
	| WITHOUT ARRAY	WRAPPER
	{
		$$ = int64(2)
	}
	| WITH WRAPPER
	{
		$$ = int64(4)
	}
	| WITH ARRAY WRAPPER
	{
		$$ = int64(4)
	}
	| WITH CONDITIONAL ARRAY WRAPPER
	{
		$$ = int64(3)
	}
	| WITH UNCONDITIONAL ARRAY WRAPPER
	{
		$$ = int64(4)
	}
	| WITH CONDITIONAL WRAPPER
	{
		$$ = int64(3)
	}
	| WITH UNCONDITIONAL WRAPPER
	{
		$$ = int64(4)
	}
	| /* EMPTY */
	{
		$$ = int64(1)
	}
	;

json_behavior:
	  DEFAULT a_expr
	{
		$$ = &JsonBehavior{Btype: pg_query.JsonBehaviorType(9), Expr: toNode($2), Location: int32(@1)}
	}
	| json_behavior_type
	{
		$$ = &JsonBehavior{Btype: pg_query.JsonBehaviorType(toInt32($1)), Location: int32(@1)}
	}
	;

json_behavior_type:
	  ERROR_P
	{
		$$ = int64(2)
	}
	| NULL_P
	{
		$$ = int64(1)
	}
	| TRUE_P
	{
		$$ = int64(4)
	}
	| FALSE_P
	{
		$$ = int64(5)
	}
	| UNKNOWN
	{
		$$ = int64(6)
	}
	| EMPTY_P ARRAY
	{
		$$ = int64(7)
	}
	| EMPTY_P OBJECT_P
	{
		$$ = int64(8)
	}
	| EMPTY_P
	{
		$$ = int64(7)
	}
	;

json_behavior_clause_opt:
	  json_behavior ON EMPTY_P
	{
		$$ = []Node{toNode($1), nil}
	}
	| json_behavior ON ERROR_P
	{
		$$ = []Node{nil, toNode($1)}
	}
	| json_behavior ON EMPTY_P json_behavior ON ERROR_P
	{
		$$ = []Node{toNode($1), toNode($4)}
	}
	| /* EMPTY */
	{
		$$ = []Node{nil, nil}
	}
	;

json_on_error_clause_opt:
	  json_behavior ON ERROR_P
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_value_expr:
	  a_expr json_format_clause_opt
	{
		$$ = &JsonValueExpr{RawExpr: toNode($1), Format: toNode(toNode($2)).GetJsonFormat()}
	}
	;

json_format_clause:
	  FORMAT_LA JSON ENCODING name
	{
		encoding := int32(0)
		switch strings.ToLower(toStr($4)) { case "utf8": encoding = 2; case "utf16": encoding = 3; case "utf32": encoding = 4; default: yylex.(*Lexer).errorAt("unrecognized JSON encoding", @4); return 1 }
		$$ = &JsonFormat{FormatType: pg_query.JsonFormatType(2), Encoding: pg_query.JsonEncoding(encoding), Location: int32(@1)}
	}
	| FORMAT_LA JSON
	{
		$$ = &JsonFormat{FormatType: pg_query.JsonFormatType(2), Encoding: pg_query.JsonEncoding(1), Location: int32(@1)}
	}
	;

json_format_clause_opt:
	  json_format_clause
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = &JsonFormat{FormatType: pg_query.JsonFormatType(1), Encoding: pg_query.JsonEncoding(1), Location: -1}
	}
	;

json_quotes_clause_opt:
	  KEEP QUOTES ON SCALAR STRING_P
	{
		$$ = int64(2)
	}
	| KEEP QUOTES
	{
		$$ = int64(2)
	}
	| OMIT QUOTES ON SCALAR STRING_P
	{
		$$ = int64(3)
	}
	| OMIT QUOTES
	{
		$$ = int64(3)
	}
	| /* EMPTY */
	{
		$$ = int64(1)
	}
	;

json_returning_clause_opt:
	  RETURNING Typename json_format_clause_opt
	{
		$$ = &JsonOutput{TypeName: toTypeNameNode($2), Returning: toNode(&JsonReturning{Format: toNode($3).GetJsonFormat()}).GetJsonReturning()}
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

json_predicate_type_constraint:
	  JSON					%prec UNBOUNDED
	{
		$$ = int64(1)
	}
	| JSON VALUE_P
	{
		$$ = int64(1)
	}
	| JSON ARRAY
	{
		$$ = int64(3)
	}
	| JSON OBJECT_P
	{
		$$ = int64(2)
	}
	| JSON SCALAR
	{
		$$ = int64(4)
	}
	;

json_key_uniqueness_constraint_opt:
	  WITH UNIQUE KEYS
	{
		$$ = true
	}
	| WITH UNIQUE				%prec UNBOUNDED
	{
		$$ = true
	}
	| WITHOUT UNIQUE KEYS
	{
		$$ = false
	}
	| WITHOUT UNIQUE			%prec UNBOUNDED
	{
		$$ = false
	}
	| %prec UNBOUNDED
	{
		$$ = false
	}
	;

json_name_and_value_list:
	  json_name_and_value
	{
		$$ = []Node{toNode($1)}
	}
	| json_name_and_value_list ',' json_name_and_value
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

json_name_and_value:
	  c_expr VALUE_P json_value_expr
	{
		$$ = &JsonKeyValue{Key: toNode($1), Value: toNode(toNode($3)).GetJsonValueExpr()}
	}
	| a_expr ':' json_value_expr
	{
		$$ = &JsonKeyValue{Key: toNode($1), Value: toNode(toNode($3)).GetJsonValueExpr()}
	}
	;

json_object_constructor_null_clause_opt:
	  NULL_P ON NULL_P
	{
		$$ = false
	}
	| ABSENT ON NULL_P
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = false
	}
	;

json_array_constructor_null_clause_opt:
	  NULL_P ON NULL_P
	{
		$$ = false
	}
	| ABSENT ON NULL_P
	{
		$$ = true
	}
	| /* EMPTY */
	{
		$$ = true
	}
	;

json_value_expr_list:
	  json_value_expr
	{
		$$ = []Node{toNode($1)}
	}
	| json_value_expr_list ',' json_value_expr
	{
		$$ = appendToNodeSlice($1, $3)
	}
	;

json_aggregate_func:
	  JSON_OBJECTAGG '(' 				json_name_and_value 				json_object_constructor_null_clause_opt 				json_key_uniqueness_constraint_opt 				json_returning_clause_opt 			')'
	{
		$$ = &JsonObjectAgg{Arg: toNode($3).GetJsonKeyValue(), AbsentOnNull: toBool($4), Unique: toBool($5), Constructor: &JsonAggConstructor{Output: toNode($6).GetJsonOutput(), Location: int32(@1)}}
		grammarSetLocation($$, @1)
	}
	| JSON_ARRAYAGG '(' 				json_value_expr 				json_array_aggregate_order_by_clause_opt 				json_array_constructor_null_clause_opt 				json_returning_clause_opt 			')'
	{
		$$ = &JsonArrayAgg{Arg: toNode($3).GetJsonValueExpr(), AbsentOnNull: toBool($5), Constructor: &JsonAggConstructor{Output: toNode($6).GetJsonOutput(), AggOrder: toNodeSlice(toSortBySlice($4)), Location: int32(@1)}}
		grammarSetLocation($$, @1)
	}
	;

json_array_aggregate_order_by_clause_opt:
	  ORDER BY sortby_list
	{
		$$ = $3
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

opt_target_list:
	  target_list
	{
		$$ = $1
	}
	| /* EMPTY */
	{
		$$ = nil
	}
	;

target_list:
	  target_el
	{
		$$ = []Node{toNode(toResTarget($1))}
	}
	| target_list ',' target_el
	{
		$$ = appendToResTargetSlice($1, $3)
	}
	;

target_el:
	  a_expr AS ColLabel
	{
		$$ = &ResTarget{Val: toNode($1), Name: toStr($3), Location: -1}
		grammarSetLocation($$, @1)
	}
	| a_expr BareColLabel
	{
		$$ = &ResTarget{Val: toNode($1), Name: toStr($2), Location: -1}
		grammarSetLocation($$, @1)
	}
	| a_expr
	{
		$$ = &ResTarget{Val: toNode($1), Location: -1}
		grammarSetLocation($$, @1)
	}
	| '*'
	{
		col := &ColumnRef{Fields: []Node{toNode(&A_Star{})}, Location: -1}
		grammarSetLocation(col, @1)
		rt := &ResTarget{Val: toNode(col), Location: -1}
		grammarSetLocation(rt, @1)
		$$ = rt
	}
	;

qualified_name_list:
	  qualified_name
	{
		$$ = []Node{toNode(toRangeVar($1))}
	}
	| qualified_name_list ',' qualified_name
	{
		$$ = appendToRangeVarSlice($1, $3)
	}
	;

qualified_name:
	  ColId
	{
		$$ = &RangeVar{Relname: toStr($1), Inh: true, Location: -1, Relpersistence: "p"}
		grammarSetLocation($$, @1)
	}
	| ColId indirection
	{
		parts := toNodeSlice($2)
		if len(parts) > 2 { yylex.Error("improper qualified name (too many dotted names)"); return 1 }
		for _, part := range parts { if part.GetString_() == nil { yylex.Error("invalid relation name"); return 1 } }
		$$ = makeRangeVarWithIndirection($1, $2)
		grammarSetLocation($$, @1)
	}
	;

name_list:
	  name
	{
		$$ = []string{toStr($1)}
	}
	| name_list ',' name
	{
		$$ = appendToStrSlice($1, $3)
	}
	;

name:
	  ColId
	{
		$$ = $1
	}
	;

attr_name:
	  ColLabel
	{
		$$ = toStr($1)
	}
	;

file_name:
	  Sconst
	{
		$$ = $1
	}
	;

func_name:
	  type_function_name
	{
		$$ = []string{toStr($1)}
	}
	| ColId indirection
	{
		for _, part := range toNodeSlice($2) { if part.GetString_() == nil { yylex.Error("invalid function name"); return 1 } }
		$$ = append([]string{toStr($1)}, toStrSlice($2)...)
	}
	;

AexprConst:
	  Iconst
	{
		$$ = makeAConst(toInt64($1))
		grammarSetLocation($$, @1)
	}
	| FCONST
	{
		$$ = makeAConst(FloatVal{Str: toStr($1)})
		grammarSetLocation($$, @1)
	}
	| Sconst
	{
		$$ = makeAConst(toStr($1))
		grammarSetLocation($$, @1)
	}
	| BCONST
	{
		$$ = makeAConst(BitStringVal(toStr($1)))
		grammarSetLocation($$, @1)
	}
	| XCONST
	{
		$$ = makeAConst(BitStringVal(toStr($1)))
		grammarSetLocation($$, @1)
	}
	| func_name Sconst
	{
		t := grammarType($1, nil)
		t.Location = int32(@1)
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($2), @2)), TypeName: t, Location: -1}
	}
	| func_name '(' func_arg_list opt_sort_clause ')' Sconst
	{
		if $4 != nil { yylex.(*Lexer).errorAt("type modifier cannot have ORDER BY", @4); return 1 }
		for _, argument := range toNodeSlice($3) {
			if named := argument.GetNamedArgExpr(); named != nil {
				yylex.(*Lexer).errorAt("type modifier cannot have parameter name", int(named.Location))
				return 1
			}
		}
		t := grammarType($1, $3)
		t.Location = int32(@1)
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($6), @6)), TypeName: t, Location: -1}
	}
	| ConstTypename Sconst
	{
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($2), @2)), TypeName: toTypeNameNode($1), Location: -1}
	}
	| ConstInterval Sconst opt_interval
	{
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($2), @2)), TypeName: grammarType($1, $3), Location: -1}
	}
	| ConstInterval '(' Iconst ')' Sconst
	{
		$$ = &TypeCast{Arg: toNode(grammarConst(toStr($5), @5)), TypeName: grammarType($1, []Node{toNode(makeAConst(int64(32767))), toNode(grammarConst(toInt64($3), @3))}), Location: -1}
	}
	| func_name PARAM
	{
		t := grammarType($1, nil)
		t.Location = int32(@1)
		$$ = &TypeCast{Arg: toNode(&ParamRef{Number: int32(toInt64($2)), Location: int32(@2)}), TypeName: t, Location: -1}
	}
	| func_name '(' func_arg_list opt_sort_clause ')' PARAM
	{
		for _, argument := range toNodeSlice($3) {
			if named := argument.GetNamedArgExpr(); named != nil {
				yylex.(*Lexer).errorAt("type modifier cannot have parameter name", int(named.Location))
				return 1
			}
		}
		if $4 != nil { yylex.(*Lexer).errorAt("type modifier cannot have ORDER BY", @4); return 1 }
		t := grammarType($1, $3)
		t.Location = int32(@1)
		$$ = &TypeCast{Arg: toNode(&ParamRef{Number: int32(toInt64($6)), Location: int32(@6)}), TypeName: t, Location: -1}
	}
	| ConstTypename PARAM
	{
		$$ = &TypeCast{Arg: toNode(&ParamRef{Number: int32(toInt64($2)), Location: int32(@2)}), TypeName: toTypeNameNode($1), Location: -1}
	}
	| ConstInterval PARAM opt_interval
	{
		$$ = &TypeCast{Arg: toNode(&ParamRef{Number: int32(toInt64($2)), Location: int32(@2)}), TypeName: grammarType($1, $3), Location: -1}
	}
	| ConstInterval '(' Iconst ')' PARAM
	{
		precision := makeAConst(toInt64($3))
		precision.Location = int32(@3)
		t := grammarType($1, []Node{toNode(makeAConst(int64(32767))), toNode(precision)})
		$$ = &TypeCast{Arg: toNode(&ParamRef{Number: int32(toInt64($5)), Location: int32(@5)}), TypeName: t, Location: -1}
	}
	| TRUE_P
	{
		$$ = makeAConst(true)
		grammarSetLocation($$, @1)
	}
	| FALSE_P
	{
		$$ = makeAConst(false)
		grammarSetLocation($$, @1)
	}
	| NULL_P
	{
		$$ = makeAConst(nil)
		grammarSetLocation($$, @1)
	}
	;

Iconst:
	  ICONST
	{
		$$ = $1
	}
	;

Sconst:
	  SCONST
	{
		$$ = $1
	}
	;

SignedIconst:
	  Iconst
	{
		$$ = $1
	}
	| '+' Iconst
	{
		$$ = $2
	}
	| '-' Iconst
	{
		$$ = -toInt64($2)
	}
	;

RoleId:
	  RoleSpec
	{
		$$ = $1
	}
	;

RoleSpec:
	  NonReservedWord
	{
		$$ = toRoleSpec($1)
		grammarSetLocation($$, @1)
	}
	| CURRENT_ROLE
	{
		$$ = &RoleSpec{Roletype: enumRoleSpecType("ROLESPEC_CURRENT_ROLE"), Location: -1}
		grammarSetLocation($$, @1)
	}
	| CURRENT_USER
	{
		$$ = &RoleSpec{Roletype: enumRoleSpecType("ROLESPEC_CURRENT_USER"), Location: -1}
		grammarSetLocation($$, @1)
	}
	| SESSION_USER
	{
		$$ = &RoleSpec{Roletype: enumRoleSpecType("ROLESPEC_SESSION_USER"), Location: -1}
		grammarSetLocation($$, @1)
	}
	;

role_list:
	  RoleSpec
	{
		$$ = []Node{toNode($1)}
	}
	| role_list ',' RoleSpec
	{
		$$ = append(toNodeSlice($1), toNode($3))
	}
	;

PLpgSQL_Expr:
	  opt_distinct_clause opt_target_list 			from_clause where_clause 			group_clause having_clause window_clause 			opt_sort_clause opt_select_limit opt_for_locking_clause
	{
		n := toSelectStmt($2, $3, $4, $5, $6, $7)
		n.DistinctClause = toNodeSlice($1)
		if message := applySelectOptions(n, $8, $10, $9, nil); message != "" {
			yylex.(*Lexer).errorAt(message, @9)
			return 1
		}
		$$ = n
	}
	;

PLAssignStmt:
	  plassign_target opt_indirection plassign_equals PLpgSQL_Expr
	{
		$$ = &pg_query.PLAssignStmt{Name: toStr($1), Indirection: toNodeSlice($2), Val: toSelectStmtPtr($4), Location: int32(@1)}
		grammarSetLocation($$, @1)
	}
	;

plassign_target:
	  ColId
	{
		$$ = $1
	}
	| PARAM
	{
		$$ = fmt.Sprintf("$%d", $1)
	}
	;

plassign_equals:
	  COLON_EQUALS
	{
		$$ = $1
	}
	| '='
	{
		$$ = nil
	}
	;

ColId:
	  IDENT
	{
		$$ = $1
	}
	| unreserved_keyword
	{
		$$ = $1
	}
	| col_name_keyword
	{
		$$ = $1
	}
	;

type_function_name:
	  IDENT
	{
		$$ = $1
	}
	| unreserved_keyword
	{
		$$ = $1
	}
	| type_func_name_keyword
	{
		$$ = $1
	}
	;

NonReservedWord:
	  IDENT
	{
		$$ = $1
	}
	| unreserved_keyword
	{
		$$ = $1
	}
	| col_name_keyword
	{
		$$ = $1
	}
	| type_func_name_keyword
	{
		$$ = $1
	}
	;

ColLabel:
	  IDENT
	{
		$$ = $1
	}
	| unreserved_keyword
	{
		$$ = $1
	}
	| col_name_keyword
	{
		$$ = $1
	}
	| type_func_name_keyword
	{
		$$ = $1
	}
	| reserved_keyword
	{
		$$ = $1
	}
	;

BareColLabel:
	  IDENT
	{
		$$ = $1
	}
	| bare_label_keyword
	{
		$$ = $1
	}
	;

unreserved_keyword:
	  ABORT_P
	{
		$$ = $1
	}
	| ABSENT
	{
		$$ = $1
	}
	| ABSOLUTE_P
	{
		$$ = $1
	}
	| ACCESS
	{
		$$ = $1
	}
	| ACTION
	{
		$$ = $1
	}
	| ADD_P
	{
		$$ = $1
	}
	| ADMIN
	{
		$$ = $1
	}
	| AFTER
	{
		$$ = $1
	}
	| AGGREGATE
	{
		$$ = $1
	}
	| ALSO
	{
		$$ = $1
	}
	| ALTER
	{
		$$ = $1
	}
	| ALWAYS
	{
		$$ = $1
	}
	| ASENSITIVE
	{
		$$ = $1
	}
	| ASSERTION
	{
		$$ = $1
	}
	| ASSIGNMENT
	{
		$$ = $1
	}
	| AT
	{
		$$ = $1
	}
	| ATOMIC
	{
		$$ = $1
	}
	| ATTACH
	{
		$$ = $1
	}
	| ATTRIBUTE
	{
		$$ = $1
	}
	| BACKWARD
	{
		$$ = $1
	}
	| BEFORE
	{
		$$ = $1
	}
	| BEGIN_P
	{
		$$ = $1
	}
	| BREADTH
	{
		$$ = $1
	}
	| BY
	{
		$$ = $1
	}
	| CACHE
	{
		$$ = $1
	}
	| CALL
	{
		$$ = $1
	}
	| CALLED
	{
		$$ = $1
	}
	| CASCADE
	{
		$$ = $1
	}
	| CASCADED
	{
		$$ = $1
	}
	| CATALOG_P
	{
		$$ = $1
	}
	| CHAIN
	{
		$$ = $1
	}
	| CHARACTERISTICS
	{
		$$ = $1
	}
	| CHECKPOINT
	{
		$$ = $1
	}
	| CLASS
	{
		$$ = $1
	}
	| CLOSE
	{
		$$ = $1
	}
	| CLUSTER
	{
		$$ = $1
	}
	| COLUMNS
	{
		$$ = $1
	}
	| COMMENT
	{
		$$ = $1
	}
	| COMMENTS
	{
		$$ = $1
	}
	| COMMIT
	{
		$$ = $1
	}
	| COMMITTED
	{
		$$ = $1
	}
	| COMPRESSION
	{
		$$ = $1
	}
	| CONDITIONAL
	{
		$$ = $1
	}
	| CONFIGURATION
	{
		$$ = $1
	}
	| CONFLICT
	{
		$$ = $1
	}
	| CONNECTION
	{
		$$ = $1
	}
	| CONSTRAINTS
	{
		$$ = $1
	}
	| CONTENT_P
	{
		$$ = $1
	}
	| CONTINUE_P
	{
		$$ = $1
	}
	| CONVERSION_P
	{
		$$ = $1
	}
	| COPY
	{
		$$ = $1
	}
	| COST
	{
		$$ = $1
	}
	| CSV
	{
		$$ = $1
	}
	| CUBE
	{
		$$ = $1
	}
	| CURRENT_P
	{
		$$ = $1
	}
	| CURSOR
	{
		$$ = $1
	}
	| CYCLE
	{
		$$ = $1
	}
	| DATA_P
	{
		$$ = $1
	}
	| DATABASE
	{
		$$ = $1
	}
	| DAY_P
	{
		$$ = $1
	}
	| DEALLOCATE
	{
		$$ = $1
	}
	| DECLARE
	{
		$$ = $1
	}
	| DEFAULTS
	{
		$$ = $1
	}
	| DEFERRED
	{
		$$ = $1
	}
	| DEFINER
	{
		$$ = $1
	}
	| DELETE_P
	{
		$$ = $1
	}
	| DELIMITER
	{
		$$ = $1
	}
	| DELIMITERS
	{
		$$ = $1
	}
	| DEPENDS
	{
		$$ = $1
	}
	| DEPTH
	{
		$$ = $1
	}
	| DETACH
	{
		$$ = $1
	}
	| DICTIONARY
	{
		$$ = $1
	}
	| DISABLE_P
	{
		$$ = $1
	}
	| DISCARD
	{
		$$ = $1
	}
	| DOCUMENT_P
	{
		$$ = $1
	}
	| DOMAIN_P
	{
		$$ = $1
	}
	| DOUBLE_P
	{
		$$ = $1
	}
	| DROP
	{
		$$ = $1
	}
	| EACH
	{
		$$ = $1
	}
	| EMPTY_P
	{
		$$ = $1
	}
	| ENABLE_P
	{
		$$ = $1
	}
	| ENCODING
	{
		$$ = $1
	}
	| ENCRYPTED
	{
		$$ = $1
	}
	| ENFORCED
	{
		$$ = $1
	}
	| ENUM_P
	{
		$$ = $1
	}
	| ERROR_P
	{
		$$ = $1
	}
	| ESCAPE
	{
		$$ = $1
	}
	| EVENT
	{
		$$ = $1
	}
	| EXCLUDE
	{
		$$ = $1
	}
	| EXCLUDING
	{
		$$ = $1
	}
	| EXCLUSIVE
	{
		$$ = $1
	}
	| EXECUTE
	{
		$$ = $1
	}
	| EXPLAIN
	{
		$$ = $1
	}
	| EXPRESSION
	{
		$$ = $1
	}
	| EXTENSION
	{
		$$ = $1
	}
	| EXTERNAL
	{
		$$ = $1
	}
	| FAMILY
	{
		$$ = $1
	}
	| FILTER
	{
		$$ = $1
	}
	| FINALIZE
	{
		$$ = $1
	}
	| FIRST_P
	{
		$$ = $1
	}
	| FOLLOWING
	{
		$$ = $1
	}
	| FORCE
	{
		$$ = $1
	}
	| FORMAT
	{
		$$ = $1
	}
	| FORWARD
	{
		$$ = $1
	}
	| FUNCTION
	{
		$$ = $1
	}
	| FUNCTIONS
	{
		$$ = $1
	}
	| GENERATED
	{
		$$ = $1
	}
	| GLOBAL
	{
		$$ = $1
	}
	| GRANTED
	{
		$$ = $1
	}
	| GROUPS
	{
		$$ = $1
	}
	| HANDLER
	{
		$$ = $1
	}
	| HEADER_P
	{
		$$ = $1
	}
	| HOLD
	{
		$$ = $1
	}
	| HOUR_P
	{
		$$ = $1
	}
	| IDENTITY_P
	{
		$$ = $1
	}
	| IF_P
	{
		$$ = $1
	}
	| IMMEDIATE
	{
		$$ = $1
	}
	| IMMUTABLE
	{
		$$ = $1
	}
	| IMPLICIT_P
	{
		$$ = $1
	}
	| IMPORT_P
	{
		$$ = $1
	}
	| INCLUDE
	{
		$$ = $1
	}
	| INCLUDING
	{
		$$ = $1
	}
	| INCREMENT
	{
		$$ = $1
	}
	| INDENT
	{
		$$ = $1
	}
	| INDEX
	{
		$$ = $1
	}
	| INDEXES
	{
		$$ = $1
	}
	| INHERIT
	{
		$$ = $1
	}
	| INHERITS
	{
		$$ = $1
	}
	| INLINE_P
	{
		$$ = $1
	}
	| INPUT_P
	{
		$$ = $1
	}
	| INSENSITIVE
	{
		$$ = $1
	}
	| INSERT
	{
		$$ = $1
	}
	| INSTEAD
	{
		$$ = $1
	}
	| INVOKER
	{
		$$ = $1
	}
	| ISOLATION
	{
		$$ = $1
	}
	| KEEP
	{
		$$ = $1
	}
	| KEY
	{
		$$ = $1
	}
	| KEYS
	{
		$$ = $1
	}
	| LABEL
	{
		$$ = $1
	}
	| LANGUAGE
	{
		$$ = $1
	}
	| LARGE_P
	{
		$$ = $1
	}
	| LAST_P
	{
		$$ = $1
	}
	| LEAKPROOF
	{
		$$ = $1
	}
	| LEVEL
	{
		$$ = $1
	}
	| LISTEN
	{
		$$ = $1
	}
	| LOAD
	{
		$$ = $1
	}
	| LOCAL
	{
		$$ = $1
	}
	| LOCATION
	{
		$$ = $1
	}
	| LOCK_P
	{
		$$ = $1
	}
	| LOCKED
	{
		$$ = $1
	}
	| LOGGED
	{
		$$ = $1
	}
	| MAPPING
	{
		$$ = $1
	}
	| MATCH
	{
		$$ = $1
	}
	| MATCHED
	{
		$$ = $1
	}
	| MATERIALIZED
	{
		$$ = $1
	}
	| MAXVALUE
	{
		$$ = $1
	}
	| MERGE
	{
		$$ = $1
	}
	| METHOD
	{
		$$ = $1
	}
	| MINUTE_P
	{
		$$ = $1
	}
	| MINVALUE
	{
		$$ = $1
	}
	| MODE
	{
		$$ = $1
	}
	| MONTH_P
	{
		$$ = $1
	}
	| MOVE
	{
		$$ = $1
	}
	| NAME_P
	{
		$$ = $1
	}
	| NAMES
	{
		$$ = $1
	}
	| NESTED
	{
		$$ = $1
	}
	| NEW
	{
		$$ = $1
	}
	| NEXT
	{
		$$ = $1
	}
	| NFC
	{
		$$ = $1
	}
	| NFD
	{
		$$ = $1
	}
	| NFKC
	{
		$$ = $1
	}
	| NFKD
	{
		$$ = $1
	}
	| NO
	{
		$$ = $1
	}
	| NORMALIZED
	{
		$$ = $1
	}
	| NOTHING
	{
		$$ = $1
	}
	| NOTIFY
	{
		$$ = $1
	}
	| NOWAIT
	{
		$$ = $1
	}
	| NULLS_P
	{
		$$ = $1
	}
	| OBJECT_P
	{
		$$ = $1
	}
	| OBJECTS_P
	{
		$$ = $1
	}
	| OF
	{
		$$ = $1
	}
	| OFF
	{
		$$ = $1
	}
	| OIDS
	{
		$$ = $1
	}
	| OLD
	{
		$$ = $1
	}
	| OMIT
	{
		$$ = $1
	}
	| OPERATOR
	{
		$$ = $1
	}
	| OPTION
	{
		$$ = $1
	}
	| OPTIONS
	{
		$$ = $1
	}
	| ORDINALITY
	{
		$$ = $1
	}
	| OTHERS
	{
		$$ = $1
	}
	| OVER
	{
		$$ = $1
	}
	| OVERRIDING
	{
		$$ = $1
	}
	| OWNED
	{
		$$ = $1
	}
	| OWNER
	{
		$$ = $1
	}
	| PARALLEL
	{
		$$ = $1
	}
	| PARAMETER
	{
		$$ = $1
	}
	| PARSER
	{
		$$ = $1
	}
	| PARTIAL
	{
		$$ = $1
	}
	| PARTITION
	{
		$$ = $1
	}
	| PASSING
	{
		$$ = $1
	}
	| PASSWORD
	{
		$$ = $1
	}
	| PATH
	{
		$$ = $1
	}
	| PERIOD
	{
		$$ = $1
	}
	| PLAN
	{
		$$ = $1
	}
	| PLANS
	{
		$$ = $1
	}
	| POLICY
	{
		$$ = $1
	}
	| PRECEDING
	{
		$$ = $1
	}
	| PREPARE
	{
		$$ = $1
	}
	| PREPARED
	{
		$$ = $1
	}
	| PRESERVE
	{
		$$ = $1
	}
	| PRIOR
	{
		$$ = $1
	}
	| PRIVILEGES
	{
		$$ = $1
	}
	| PROCEDURAL
	{
		$$ = $1
	}
	| PROCEDURE
	{
		$$ = $1
	}
	| PROCEDURES
	{
		$$ = $1
	}
	| PROGRAM
	{
		$$ = $1
	}
	| PUBLICATION
	{
		$$ = $1
	}
	| QUOTE
	{
		$$ = $1
	}
	| QUOTES
	{
		$$ = $1
	}
	| RANGE
	{
		$$ = $1
	}
	| READ
	{
		$$ = $1
	}
	| REASSIGN
	{
		$$ = $1
	}
	| RECURSIVE
	{
		$$ = $1
	}
	| REF_P
	{
		$$ = $1
	}
	| REFERENCING
	{
		$$ = $1
	}
	| REFRESH
	{
		$$ = $1
	}
	| REINDEX
	{
		$$ = $1
	}
	| RELATIVE_P
	{
		$$ = $1
	}
	| RELEASE
	{
		$$ = $1
	}
	| RENAME
	{
		$$ = $1
	}
	| REPEATABLE
	{
		$$ = $1
	}
	| REPLACE
	{
		$$ = $1
	}
	| REPLICA
	{
		$$ = $1
	}
	| RESET
	{
		$$ = $1
	}
	| RESTART
	{
		$$ = $1
	}
	| RESTRICT
	{
		$$ = $1
	}
	| RETURN
	{
		$$ = $1
	}
	| RETURNS
	{
		$$ = $1
	}
	| REVOKE
	{
		$$ = $1
	}
	| ROLE
	{
		$$ = $1
	}
	| ROLLBACK
	{
		$$ = $1
	}
	| ROLLUP
	{
		$$ = $1
	}
	| ROUTINE
	{
		$$ = $1
	}
	| ROUTINES
	{
		$$ = $1
	}
	| ROWS
	{
		$$ = $1
	}
	| RULE
	{
		$$ = $1
	}
	| SAVEPOINT
	{
		$$ = $1
	}
	| SCALAR
	{
		$$ = $1
	}
	| SCHEMA
	{
		$$ = $1
	}
	| SCHEMAS
	{
		$$ = $1
	}
	| SCROLL
	{
		$$ = $1
	}
	| SEARCH
	{
		$$ = $1
	}
	| SECOND_P
	{
		$$ = $1
	}
	| SECURITY
	{
		$$ = $1
	}
	| SEQUENCE
	{
		$$ = $1
	}
	| SEQUENCES
	{
		$$ = $1
	}
	| SERIALIZABLE
	{
		$$ = $1
	}
	| SERVER
	{
		$$ = $1
	}
	| SESSION
	{
		$$ = $1
	}
	| SET
	{
		$$ = $1
	}
	| SETS
	{
		$$ = $1
	}
	| SHARE
	{
		$$ = $1
	}
	| SHOW
	{
		$$ = $1
	}
	| SIMPLE
	{
		$$ = $1
	}
	| SKIP
	{
		$$ = $1
	}
	| SNAPSHOT
	{
		$$ = $1
	}
	| SOURCE
	{
		$$ = $1
	}
	| SQL_P
	{
		$$ = $1
	}
	| STABLE
	{
		$$ = $1
	}
	| STANDALONE_P
	{
		$$ = $1
	}
	| START
	{
		$$ = $1
	}
	| STATEMENT
	{
		$$ = $1
	}
	| STATISTICS
	{
		$$ = $1
	}
	| STDIN
	{
		$$ = $1
	}
	| STDOUT
	{
		$$ = $1
	}
	| STORAGE
	{
		$$ = $1
	}
	| STORED
	{
		$$ = $1
	}
	| STRICT_P
	{
		$$ = $1
	}
	| STRING_P
	{
		$$ = $1
	}
	| STRIP_P
	{
		$$ = $1
	}
	| SUBSCRIPTION
	{
		$$ = $1
	}
	| SUPPORT
	{
		$$ = $1
	}
	| SYSID
	{
		$$ = $1
	}
	| SYSTEM_P
	{
		$$ = $1
	}
	| TABLES
	{
		$$ = $1
	}
	| TABLESPACE
	{
		$$ = $1
	}
	| TARGET
	{
		$$ = $1
	}
	| TEMP
	{
		$$ = $1
	}
	| TEMPLATE
	{
		$$ = $1
	}
	| TEMPORARY
	{
		$$ = $1
	}
	| TEXT_P
	{
		$$ = $1
	}
	| TIES
	{
		$$ = $1
	}
	| TRANSACTION
	{
		$$ = $1
	}
	| TRANSFORM
	{
		$$ = $1
	}
	| TRIGGER
	{
		$$ = $1
	}
	| TRUNCATE
	{
		$$ = $1
	}
	| TRUSTED
	{
		$$ = $1
	}
	| TYPE_P
	{
		$$ = $1
	}
	| TYPES_P
	{
		$$ = $1
	}
	| UESCAPE
	{
		$$ = $1
	}
	| UNBOUNDED
	{
		$$ = $1
	}
	| UNCOMMITTED
	{
		$$ = $1
	}
	| UNCONDITIONAL
	{
		$$ = $1
	}
	| UNENCRYPTED
	{
		$$ = $1
	}
	| UNKNOWN
	{
		$$ = $1
	}
	| UNLISTEN
	{
		$$ = $1
	}
	| UNLOGGED
	{
		$$ = $1
	}
	| UNTIL
	{
		$$ = $1
	}
	| UPDATE
	{
		$$ = $1
	}
	| VACUUM
	{
		$$ = $1
	}
	| VALID
	{
		$$ = $1
	}
	| VALIDATE
	{
		$$ = $1
	}
	| VALIDATOR
	{
		$$ = $1
	}
	| VALUE_P
	{
		$$ = $1
	}
	| VARYING
	{
		$$ = $1
	}
	| VERSION_P
	{
		$$ = $1
	}
	| VIEW
	{
		$$ = $1
	}
	| VIEWS
	{
		$$ = $1
	}
	| VIRTUAL
	{
		$$ = $1
	}
	| VOLATILE
	{
		$$ = $1
	}
	| WHITESPACE_P
	{
		$$ = $1
	}
	| WITHIN
	{
		$$ = $1
	}
	| WITHOUT
	{
		$$ = $1
	}
	| WORK
	{
		$$ = $1
	}
	| WRAPPER
	{
		$$ = $1
	}
	| WRITE
	{
		$$ = $1
	}
	| XML_P
	{
		$$ = $1
	}
	| YEAR_P
	{
		$$ = $1
	}
	| YES_P
	{
		$$ = $1
	}
	| ZONE
	{
		$$ = $1
	}
	;

col_name_keyword:
	  BETWEEN
	{
		$$ = $1
	}
	| BIGINT
	{
		$$ = $1
	}
	| BIT
	{
		$$ = $1
	}
	| BOOLEAN_P
	{
		$$ = $1
	}
	| CHAR_P
	{
		$$ = $1
	}
	| CHARACTER
	{
		$$ = $1
	}
	| COALESCE
	{
		$$ = $1
	}
	| DEC
	{
		$$ = $1
	}
	| DECIMAL_P
	{
		$$ = $1
	}
	| EXISTS
	{
		$$ = $1
	}
	| EXTRACT
	{
		$$ = $1
	}
	| FLOAT_P
	{
		$$ = $1
	}
	| GREATEST
	{
		$$ = $1
	}
	| GROUPING
	{
		$$ = $1
	}
	| INOUT
	{
		$$ = $1
	}
	| INT_P
	{
		$$ = $1
	}
	| INTEGER
	{
		$$ = $1
	}
	| INTERVAL
	{
		$$ = $1
	}
	| JSON
	{
		$$ = $1
	}
	| JSON_ARRAY
	{
		$$ = $1
	}
	| JSON_ARRAYAGG
	{
		$$ = $1
	}
	| JSON_EXISTS
	{
		$$ = $1
	}
	| JSON_OBJECT
	{
		$$ = $1
	}
	| JSON_OBJECTAGG
	{
		$$ = $1
	}
	| JSON_QUERY
	{
		$$ = $1
	}
	| JSON_SCALAR
	{
		$$ = $1
	}
	| JSON_SERIALIZE
	{
		$$ = $1
	}
	| JSON_TABLE
	{
		$$ = $1
	}
	| JSON_VALUE
	{
		$$ = $1
	}
	| LEAST
	{
		$$ = $1
	}
	| MERGE_ACTION
	{
		$$ = $1
	}
	| NATIONAL
	{
		$$ = $1
	}
	| NCHAR
	{
		$$ = $1
	}
	| NONE
	{
		$$ = $1
	}
	| NORMALIZE
	{
		$$ = $1
	}
	| NULLIF
	{
		$$ = $1
	}
	| NUMERIC
	{
		$$ = $1
	}
	| OUT_P
	{
		$$ = $1
	}
	| OVERLAY
	{
		$$ = $1
	}
	| POSITION
	{
		$$ = $1
	}
	| PRECISION
	{
		$$ = $1
	}
	| REAL
	{
		$$ = $1
	}
	| ROW
	{
		$$ = $1
	}
	| SETOF
	{
		$$ = $1
	}
	| SMALLINT
	{
		$$ = $1
	}
	| SUBSTRING
	{
		$$ = $1
	}
	| TIME
	{
		$$ = $1
	}
	| TIMESTAMP
	{
		$$ = $1
	}
	| TREAT
	{
		$$ = $1
	}
	| TRIM
	{
		$$ = $1
	}
	| VALUES
	{
		$$ = $1
	}
	| VARCHAR
	{
		$$ = $1
	}
	| XMLATTRIBUTES
	{
		$$ = $1
	}
	| XMLCONCAT
	{
		$$ = $1
	}
	| XMLELEMENT
	{
		$$ = $1
	}
	| XMLEXISTS
	{
		$$ = $1
	}
	| XMLFOREST
	{
		$$ = $1
	}
	| XMLNAMESPACES
	{
		$$ = $1
	}
	| XMLPARSE
	{
		$$ = $1
	}
	| XMLPI
	{
		$$ = $1
	}
	| XMLROOT
	{
		$$ = $1
	}
	| XMLSERIALIZE
	{
		$$ = $1
	}
	| XMLTABLE
	{
		$$ = $1
	}
	;

type_func_name_keyword:
	  AUTHORIZATION
	{
		$$ = $1
	}
	| BINARY
	{
		$$ = $1
	}
	| COLLATION
	{
		$$ = $1
	}
	| CONCURRENTLY
	{
		$$ = $1
	}
	| CROSS
	{
		$$ = $1
	}
	| CURRENT_SCHEMA
	{
		$$ = $1
	}
	| FREEZE
	{
		$$ = $1
	}
	| FULL
	{
		$$ = $1
	}
	| ILIKE
	{
		$$ = $1
	}
	| INNER_P
	{
		$$ = $1
	}
	| IS
	{
		$$ = $1
	}
	| ISNULL
	{
		$$ = $1
	}
	| JOIN
	{
		$$ = $1
	}
	| LEFT
	{
		$$ = $1
	}
	| LIKE
	{
		$$ = $1
	}
	| NATURAL
	{
		$$ = $1
	}
	| NOTNULL
	{
		$$ = $1
	}
	| OUTER_P
	{
		$$ = $1
	}
	| OVERLAPS
	{
		$$ = $1
	}
	| RIGHT
	{
		$$ = $1
	}
	| SIMILAR
	{
		$$ = $1
	}
	| TABLESAMPLE
	{
		$$ = $1
	}
	| VERBOSE
	{
		$$ = $1
	}
	;

reserved_keyword:
	  ALL
	{
		$$ = $1
	}
	| ANALYSE
	{
		$$ = $1
	}
	| ANALYZE
	{
		$$ = $1
	}
	| AND
	{
		$$ = $1
	}
	| ANY
	{
		$$ = $1
	}
	| ARRAY
	{
		$$ = $1
	}
	| AS
	{
		$$ = $1
	}
	| ASC
	{
		$$ = $1
	}
	| ASYMMETRIC
	{
		$$ = $1
	}
	| BOTH
	{
		$$ = $1
	}
	| CASE
	{
		$$ = $1
	}
	| CAST
	{
		$$ = $1
	}
	| CHECK
	{
		$$ = $1
	}
	| COLLATE
	{
		$$ = $1
	}
	| COLUMN
	{
		$$ = $1
	}
	| CONSTRAINT
	{
		$$ = $1
	}
	| CREATE
	{
		$$ = $1
	}
	| CURRENT_CATALOG
	{
		$$ = $1
	}
	| CURRENT_DATE
	{
		$$ = $1
	}
	| CURRENT_ROLE
	{
		$$ = $1
	}
	| CURRENT_TIME
	{
		$$ = $1
	}
	| CURRENT_TIMESTAMP
	{
		$$ = $1
	}
	| CURRENT_USER
	{
		$$ = $1
	}
	| DEFAULT
	{
		$$ = $1
	}
	| DEFERRABLE
	{
		$$ = $1
	}
	| DESC
	{
		$$ = $1
	}
	| DISTINCT
	{
		$$ = $1
	}
	| DO
	{
		$$ = $1
	}
	| ELSE
	{
		$$ = $1
	}
	| END_P
	{
		$$ = $1
	}
	| EXCEPT
	{
		$$ = $1
	}
	| FALSE_P
	{
		$$ = $1
	}
	| FETCH
	{
		$$ = $1
	}
	| FOR
	{
		$$ = $1
	}
	| FOREIGN
	{
		$$ = $1
	}
	| FROM
	{
		$$ = $1
	}
	| GRANT
	{
		$$ = $1
	}
	| GROUP_P
	{
		$$ = $1
	}
	| HAVING
	{
		$$ = $1
	}
	| IN_P
	{
		$$ = $1
	}
	| INITIALLY
	{
		$$ = $1
	}
	| INTERSECT
	{
		$$ = $1
	}
	| INTO
	{
		$$ = $1
	}
	| LATERAL_P
	{
		$$ = $1
	}
	| LEADING
	{
		$$ = $1
	}
	| LIMIT
	{
		$$ = $1
	}
	| LOCALTIME
	{
		$$ = $1
	}
	| LOCALTIMESTAMP
	{
		$$ = $1
	}
	| NOT
	{
		$$ = $1
	}
	| NULL_P
	{
		$$ = $1
	}
	| OFFSET
	{
		$$ = $1
	}
	| ON
	{
		$$ = $1
	}
	| ONLY
	{
		$$ = $1
	}
	| OR
	{
		$$ = $1
	}
	| ORDER
	{
		$$ = $1
	}
	| PLACING
	{
		$$ = $1
	}
	| PRIMARY
	{
		$$ = $1
	}
	| REFERENCES
	{
		$$ = $1
	}
	| RETURNING
	{
		$$ = $1
	}
	| SELECT
	{
		$$ = $1
	}
	| SESSION_USER
	{
		$$ = $1
	}
	| SOME
	{
		$$ = $1
	}
	| SYMMETRIC
	{
		$$ = $1
	}
	| SYSTEM_USER
	{
		$$ = $1
	}
	| TABLE
	{
		$$ = $1
	}
	| THEN
	{
		$$ = $1
	}
	| TO
	{
		$$ = $1
	}
	| TRAILING
	{
		$$ = $1
	}
	| TRUE_P
	{
		$$ = $1
	}
	| UNION
	{
		$$ = $1
	}
	| UNIQUE
	{
		$$ = $1
	}
	| USER
	{
		$$ = $1
	}
	| USING
	{
		$$ = $1
	}
	| VARIADIC
	{
		$$ = $1
	}
	| WHEN
	{
		$$ = $1
	}
	| WHERE
	{
		$$ = $1
	}
	| WINDOW
	{
		$$ = $1
	}
	| WITH
	{
		$$ = $1
	}
	;

bare_label_keyword:
	  ABORT_P
	{
		$$ = $1
	}
	| ABSENT
	{
		$$ = $1
	}
	| ABSOLUTE_P
	{
		$$ = $1
	}
	| ACCESS
	{
		$$ = $1
	}
	| ACTION
	{
		$$ = $1
	}
	| ADD_P
	{
		$$ = $1
	}
	| ADMIN
	{
		$$ = $1
	}
	| AFTER
	{
		$$ = $1
	}
	| AGGREGATE
	{
		$$ = $1
	}
	| ALL
	{
		$$ = $1
	}
	| ALSO
	{
		$$ = $1
	}
	| ALTER
	{
		$$ = $1
	}
	| ALWAYS
	{
		$$ = $1
	}
	| ANALYSE
	{
		$$ = $1
	}
	| ANALYZE
	{
		$$ = $1
	}
	| AND
	{
		$$ = $1
	}
	| ANY
	{
		$$ = $1
	}
	| ASC
	{
		$$ = $1
	}
	| ASENSITIVE
	{
		$$ = $1
	}
	| ASSERTION
	{
		$$ = $1
	}
	| ASSIGNMENT
	{
		$$ = $1
	}
	| ASYMMETRIC
	{
		$$ = $1
	}
	| AT
	{
		$$ = $1
	}
	| ATOMIC
	{
		$$ = $1
	}
	| ATTACH
	{
		$$ = $1
	}
	| ATTRIBUTE
	{
		$$ = $1
	}
	| AUTHORIZATION
	{
		$$ = $1
	}
	| BACKWARD
	{
		$$ = $1
	}
	| BEFORE
	{
		$$ = $1
	}
	| BEGIN_P
	{
		$$ = $1
	}
	| BETWEEN
	{
		$$ = $1
	}
	| BIGINT
	{
		$$ = $1
	}
	| BINARY
	{
		$$ = $1
	}
	| BIT
	{
		$$ = $1
	}
	| BOOLEAN_P
	{
		$$ = $1
	}
	| BOTH
	{
		$$ = $1
	}
	| BREADTH
	{
		$$ = $1
	}
	| BY
	{
		$$ = $1
	}
	| CACHE
	{
		$$ = $1
	}
	| CALL
	{
		$$ = $1
	}
	| CALLED
	{
		$$ = $1
	}
	| CASCADE
	{
		$$ = $1
	}
	| CASCADED
	{
		$$ = $1
	}
	| CASE
	{
		$$ = $1
	}
	| CAST
	{
		$$ = $1
	}
	| CATALOG_P
	{
		$$ = $1
	}
	| CHAIN
	{
		$$ = $1
	}
	| CHARACTERISTICS
	{
		$$ = $1
	}
	| CHECK
	{
		$$ = $1
	}
	| CHECKPOINT
	{
		$$ = $1
	}
	| CLASS
	{
		$$ = $1
	}
	| CLOSE
	{
		$$ = $1
	}
	| CLUSTER
	{
		$$ = $1
	}
	| COALESCE
	{
		$$ = $1
	}
	| COLLATE
	{
		$$ = $1
	}
	| COLLATION
	{
		$$ = $1
	}
	| COLUMN
	{
		$$ = $1
	}
	| COLUMNS
	{
		$$ = $1
	}
	| COMMENT
	{
		$$ = $1
	}
	| COMMENTS
	{
		$$ = $1
	}
	| COMMIT
	{
		$$ = $1
	}
	| COMMITTED
	{
		$$ = $1
	}
	| COMPRESSION
	{
		$$ = $1
	}
	| CONCURRENTLY
	{
		$$ = $1
	}
	| CONDITIONAL
	{
		$$ = $1
	}
	| CONFIGURATION
	{
		$$ = $1
	}
	| CONFLICT
	{
		$$ = $1
	}
	| CONNECTION
	{
		$$ = $1
	}
	| CONSTRAINT
	{
		$$ = $1
	}
	| CONSTRAINTS
	{
		$$ = $1
	}
	| CONTENT_P
	{
		$$ = $1
	}
	| CONTINUE_P
	{
		$$ = $1
	}
	| CONVERSION_P
	{
		$$ = $1
	}
	| COPY
	{
		$$ = $1
	}
	| COST
	{
		$$ = $1
	}
	| CROSS
	{
		$$ = $1
	}
	| CSV
	{
		$$ = $1
	}
	| CUBE
	{
		$$ = $1
	}
	| CURRENT_P
	{
		$$ = $1
	}
	| CURRENT_CATALOG
	{
		$$ = $1
	}
	| CURRENT_DATE
	{
		$$ = $1
	}
	| CURRENT_ROLE
	{
		$$ = $1
	}
	| CURRENT_SCHEMA
	{
		$$ = $1
	}
	| CURRENT_TIME
	{
		$$ = $1
	}
	| CURRENT_TIMESTAMP
	{
		$$ = $1
	}
	| CURRENT_USER
	{
		$$ = $1
	}
	| CURSOR
	{
		$$ = $1
	}
	| CYCLE
	{
		$$ = $1
	}
	| DATA_P
	{
		$$ = $1
	}
	| DATABASE
	{
		$$ = $1
	}
	| DEALLOCATE
	{
		$$ = $1
	}
	| DEC
	{
		$$ = $1
	}
	| DECIMAL_P
	{
		$$ = $1
	}
	| DECLARE
	{
		$$ = $1
	}
	| DEFAULT
	{
		$$ = $1
	}
	| DEFAULTS
	{
		$$ = $1
	}
	| DEFERRABLE
	{
		$$ = $1
	}
	| DEFERRED
	{
		$$ = $1
	}
	| DEFINER
	{
		$$ = $1
	}
	| DELETE_P
	{
		$$ = $1
	}
	| DELIMITER
	{
		$$ = $1
	}
	| DELIMITERS
	{
		$$ = $1
	}
	| DEPENDS
	{
		$$ = $1
	}
	| DEPTH
	{
		$$ = $1
	}
	| DESC
	{
		$$ = $1
	}
	| DETACH
	{
		$$ = $1
	}
	| DICTIONARY
	{
		$$ = $1
	}
	| DISABLE_P
	{
		$$ = $1
	}
	| DISCARD
	{
		$$ = $1
	}
	| DISTINCT
	{
		$$ = $1
	}
	| DO
	{
		$$ = $1
	}
	| DOCUMENT_P
	{
		$$ = $1
	}
	| DOMAIN_P
	{
		$$ = $1
	}
	| DOUBLE_P
	{
		$$ = $1
	}
	| DROP
	{
		$$ = $1
	}
	| EACH
	{
		$$ = $1
	}
	| ELSE
	{
		$$ = $1
	}
	| EMPTY_P
	{
		$$ = $1
	}
	| ENABLE_P
	{
		$$ = $1
	}
	| ENCODING
	{
		$$ = $1
	}
	| ENCRYPTED
	{
		$$ = $1
	}
	| END_P
	{
		$$ = $1
	}
	| ENFORCED
	{
		$$ = $1
	}
	| ENUM_P
	{
		$$ = $1
	}
	| ERROR_P
	{
		$$ = $1
	}
	| ESCAPE
	{
		$$ = $1
	}
	| EVENT
	{
		$$ = $1
	}
	| EXCLUDE
	{
		$$ = $1
	}
	| EXCLUDING
	{
		$$ = $1
	}
	| EXCLUSIVE
	{
		$$ = $1
	}
	| EXECUTE
	{
		$$ = $1
	}
	| EXISTS
	{
		$$ = $1
	}
	| EXPLAIN
	{
		$$ = $1
	}
	| EXPRESSION
	{
		$$ = $1
	}
	| EXTENSION
	{
		$$ = $1
	}
	| EXTERNAL
	{
		$$ = $1
	}
	| EXTRACT
	{
		$$ = $1
	}
	| FALSE_P
	{
		$$ = $1
	}
	| FAMILY
	{
		$$ = $1
	}
	| FINALIZE
	{
		$$ = $1
	}
	| FIRST_P
	{
		$$ = $1
	}
	| FLOAT_P
	{
		$$ = $1
	}
	| FOLLOWING
	{
		$$ = $1
	}
	| FORCE
	{
		$$ = $1
	}
	| FOREIGN
	{
		$$ = $1
	}
	| FORMAT
	{
		$$ = $1
	}
	| FORWARD
	{
		$$ = $1
	}
	| FREEZE
	{
		$$ = $1
	}
	| FULL
	{
		$$ = $1
	}
	| FUNCTION
	{
		$$ = $1
	}
	| FUNCTIONS
	{
		$$ = $1
	}
	| GENERATED
	{
		$$ = $1
	}
	| GLOBAL
	{
		$$ = $1
	}
	| GRANTED
	{
		$$ = $1
	}
	| GREATEST
	{
		$$ = $1
	}
	| GROUPING
	{
		$$ = $1
	}
	| GROUPS
	{
		$$ = $1
	}
	| HANDLER
	{
		$$ = $1
	}
	| HEADER_P
	{
		$$ = $1
	}
	| HOLD
	{
		$$ = $1
	}
	| IDENTITY_P
	{
		$$ = $1
	}
	| IF_P
	{
		$$ = $1
	}
	| ILIKE
	{
		$$ = $1
	}
	| IMMEDIATE
	{
		$$ = $1
	}
	| IMMUTABLE
	{
		$$ = $1
	}
	| IMPLICIT_P
	{
		$$ = $1
	}
	| IMPORT_P
	{
		$$ = $1
	}
	| IN_P
	{
		$$ = $1
	}
	| INCLUDE
	{
		$$ = $1
	}
	| INCLUDING
	{
		$$ = $1
	}
	| INCREMENT
	{
		$$ = $1
	}
	| INDENT
	{
		$$ = $1
	}
	| INDEX
	{
		$$ = $1
	}
	| INDEXES
	{
		$$ = $1
	}
	| INHERIT
	{
		$$ = $1
	}
	| INHERITS
	{
		$$ = $1
	}
	| INITIALLY
	{
		$$ = $1
	}
	| INLINE_P
	{
		$$ = $1
	}
	| INNER_P
	{
		$$ = $1
	}
	| INOUT
	{
		$$ = $1
	}
	| INPUT_P
	{
		$$ = $1
	}
	| INSENSITIVE
	{
		$$ = $1
	}
	| INSERT
	{
		$$ = $1
	}
	| INSTEAD
	{
		$$ = $1
	}
	| INT_P
	{
		$$ = $1
	}
	| INTEGER
	{
		$$ = $1
	}
	| INTERVAL
	{
		$$ = $1
	}
	| INVOKER
	{
		$$ = $1
	}
	| IS
	{
		$$ = $1
	}
	| ISOLATION
	{
		$$ = $1
	}
	| JOIN
	{
		$$ = $1
	}
	| JSON
	{
		$$ = $1
	}
	| JSON_ARRAY
	{
		$$ = $1
	}
	| JSON_ARRAYAGG
	{
		$$ = $1
	}
	| JSON_EXISTS
	{
		$$ = $1
	}
	| JSON_OBJECT
	{
		$$ = $1
	}
	| JSON_OBJECTAGG
	{
		$$ = $1
	}
	| JSON_QUERY
	{
		$$ = $1
	}
	| JSON_SCALAR
	{
		$$ = $1
	}
	| JSON_SERIALIZE
	{
		$$ = $1
	}
	| JSON_TABLE
	{
		$$ = $1
	}
	| JSON_VALUE
	{
		$$ = $1
	}
	| KEEP
	{
		$$ = $1
	}
	| KEY
	{
		$$ = $1
	}
	| KEYS
	{
		$$ = $1
	}
	| LABEL
	{
		$$ = $1
	}
	| LANGUAGE
	{
		$$ = $1
	}
	| LARGE_P
	{
		$$ = $1
	}
	| LAST_P
	{
		$$ = $1
	}
	| LATERAL_P
	{
		$$ = $1
	}
	| LEADING
	{
		$$ = $1
	}
	| LEAKPROOF
	{
		$$ = $1
	}
	| LEAST
	{
		$$ = $1
	}
	| LEFT
	{
		$$ = $1
	}
	| LEVEL
	{
		$$ = $1
	}
	| LIKE
	{
		$$ = $1
	}
	| LISTEN
	{
		$$ = $1
	}
	| LOAD
	{
		$$ = $1
	}
	| LOCAL
	{
		$$ = $1
	}
	| LOCALTIME
	{
		$$ = $1
	}
	| LOCALTIMESTAMP
	{
		$$ = $1
	}
	| LOCATION
	{
		$$ = $1
	}
	| LOCK_P
	{
		$$ = $1
	}
	| LOCKED
	{
		$$ = $1
	}
	| LOGGED
	{
		$$ = $1
	}
	| MAPPING
	{
		$$ = $1
	}
	| MATCH
	{
		$$ = $1
	}
	| MATCHED
	{
		$$ = $1
	}
	| MATERIALIZED
	{
		$$ = $1
	}
	| MAXVALUE
	{
		$$ = $1
	}
	| MERGE
	{
		$$ = $1
	}
	| MERGE_ACTION
	{
		$$ = $1
	}
	| METHOD
	{
		$$ = $1
	}
	| MINVALUE
	{
		$$ = $1
	}
	| MODE
	{
		$$ = $1
	}
	| MOVE
	{
		$$ = $1
	}
	| NAME_P
	{
		$$ = $1
	}
	| NAMES
	{
		$$ = $1
	}
	| NATIONAL
	{
		$$ = $1
	}
	| NATURAL
	{
		$$ = $1
	}
	| NCHAR
	{
		$$ = $1
	}
	| NESTED
	{
		$$ = $1
	}
	| NEW
	{
		$$ = $1
	}
	| NEXT
	{
		$$ = $1
	}
	| NFC
	{
		$$ = $1
	}
	| NFD
	{
		$$ = $1
	}
	| NFKC
	{
		$$ = $1
	}
	| NFKD
	{
		$$ = $1
	}
	| NO
	{
		$$ = $1
	}
	| NONE
	{
		$$ = $1
	}
	| NORMALIZE
	{
		$$ = $1
	}
	| NORMALIZED
	{
		$$ = $1
	}
	| NOT
	{
		$$ = $1
	}
	| NOTHING
	{
		$$ = $1
	}
	| NOTIFY
	{
		$$ = $1
	}
	| NOWAIT
	{
		$$ = $1
	}
	| NULL_P
	{
		$$ = $1
	}
	| NULLIF
	{
		$$ = $1
	}
	| NULLS_P
	{
		$$ = $1
	}
	| NUMERIC
	{
		$$ = $1
	}
	| OBJECT_P
	{
		$$ = $1
	}
	| OBJECTS_P
	{
		$$ = $1
	}
	| OF
	{
		$$ = $1
	}
	| OFF
	{
		$$ = $1
	}
	| OIDS
	{
		$$ = $1
	}
	| OLD
	{
		$$ = $1
	}
	| OMIT
	{
		$$ = $1
	}
	| ONLY
	{
		$$ = $1
	}
	| OPERATOR
	{
		$$ = $1
	}
	| OPTION
	{
		$$ = $1
	}
	| OPTIONS
	{
		$$ = $1
	}
	| OR
	{
		$$ = $1
	}
	| ORDINALITY
	{
		$$ = $1
	}
	| OTHERS
	{
		$$ = $1
	}
	| OUT_P
	{
		$$ = $1
	}
	| OUTER_P
	{
		$$ = $1
	}
	| OVERLAY
	{
		$$ = $1
	}
	| OVERRIDING
	{
		$$ = $1
	}
	| OWNED
	{
		$$ = $1
	}
	| OWNER
	{
		$$ = $1
	}
	| PARALLEL
	{
		$$ = $1
	}
	| PARAMETER
	{
		$$ = $1
	}
	| PARSER
	{
		$$ = $1
	}
	| PARTIAL
	{
		$$ = $1
	}
	| PARTITION
	{
		$$ = $1
	}
	| PASSING
	{
		$$ = $1
	}
	| PASSWORD
	{
		$$ = $1
	}
	| PATH
	{
		$$ = $1
	}
	| PERIOD
	{
		$$ = $1
	}
	| PLACING
	{
		$$ = $1
	}
	| PLAN
	{
		$$ = $1
	}
	| PLANS
	{
		$$ = $1
	}
	| POLICY
	{
		$$ = $1
	}
	| POSITION
	{
		$$ = $1
	}
	| PRECEDING
	{
		$$ = $1
	}
	| PREPARE
	{
		$$ = $1
	}
	| PREPARED
	{
		$$ = $1
	}
	| PRESERVE
	{
		$$ = $1
	}
	| PRIMARY
	{
		$$ = $1
	}
	| PRIOR
	{
		$$ = $1
	}
	| PRIVILEGES
	{
		$$ = $1
	}
	| PROCEDURAL
	{
		$$ = $1
	}
	| PROCEDURE
	{
		$$ = $1
	}
	| PROCEDURES
	{
		$$ = $1
	}
	| PROGRAM
	{
		$$ = $1
	}
	| PUBLICATION
	{
		$$ = $1
	}
	| QUOTE
	{
		$$ = $1
	}
	| QUOTES
	{
		$$ = $1
	}
	| RANGE
	{
		$$ = $1
	}
	| READ
	{
		$$ = $1
	}
	| REAL
	{
		$$ = $1
	}
	| REASSIGN
	{
		$$ = $1
	}
	| RECURSIVE
	{
		$$ = $1
	}
	| REF_P
	{
		$$ = $1
	}
	| REFERENCES
	{
		$$ = $1
	}
	| REFERENCING
	{
		$$ = $1
	}
	| REFRESH
	{
		$$ = $1
	}
	| REINDEX
	{
		$$ = $1
	}
	| RELATIVE_P
	{
		$$ = $1
	}
	| RELEASE
	{
		$$ = $1
	}
	| RENAME
	{
		$$ = $1
	}
	| REPEATABLE
	{
		$$ = $1
	}
	| REPLACE
	{
		$$ = $1
	}
	| REPLICA
	{
		$$ = $1
	}
	| RESET
	{
		$$ = $1
	}
	| RESTART
	{
		$$ = $1
	}
	| RESTRICT
	{
		$$ = $1
	}
	| RETURN
	{
		$$ = $1
	}
	| RETURNS
	{
		$$ = $1
	}
	| REVOKE
	{
		$$ = $1
	}
	| RIGHT
	{
		$$ = $1
	}
	| ROLE
	{
		$$ = $1
	}
	| ROLLBACK
	{
		$$ = $1
	}
	| ROLLUP
	{
		$$ = $1
	}
	| ROUTINE
	{
		$$ = $1
	}
	| ROUTINES
	{
		$$ = $1
	}
	| ROW
	{
		$$ = $1
	}
	| ROWS
	{
		$$ = $1
	}
	| RULE
	{
		$$ = $1
	}
	| SAVEPOINT
	{
		$$ = $1
	}
	| SCALAR
	{
		$$ = $1
	}
	| SCHEMA
	{
		$$ = $1
	}
	| SCHEMAS
	{
		$$ = $1
	}
	| SCROLL
	{
		$$ = $1
	}
	| SEARCH
	{
		$$ = $1
	}
	| SECURITY
	{
		$$ = $1
	}
	| SELECT
	{
		$$ = $1
	}
	| SEQUENCE
	{
		$$ = $1
	}
	| SEQUENCES
	{
		$$ = $1
	}
	| SERIALIZABLE
	{
		$$ = $1
	}
	| SERVER
	{
		$$ = $1
	}
	| SESSION
	{
		$$ = $1
	}
	| SESSION_USER
	{
		$$ = $1
	}
	| SET
	{
		$$ = $1
	}
	| SETOF
	{
		$$ = $1
	}
	| SETS
	{
		$$ = $1
	}
	| SHARE
	{
		$$ = $1
	}
	| SHOW
	{
		$$ = $1
	}
	| SIMILAR
	{
		$$ = $1
	}
	| SIMPLE
	{
		$$ = $1
	}
	| SKIP
	{
		$$ = $1
	}
	| SMALLINT
	{
		$$ = $1
	}
	| SNAPSHOT
	{
		$$ = $1
	}
	| SOME
	{
		$$ = $1
	}
	| SOURCE
	{
		$$ = $1
	}
	| SQL_P
	{
		$$ = $1
	}
	| STABLE
	{
		$$ = $1
	}
	| STANDALONE_P
	{
		$$ = $1
	}
	| START
	{
		$$ = $1
	}
	| STATEMENT
	{
		$$ = $1
	}
	| STATISTICS
	{
		$$ = $1
	}
	| STDIN
	{
		$$ = $1
	}
	| STDOUT
	{
		$$ = $1
	}
	| STORAGE
	{
		$$ = $1
	}
	| STORED
	{
		$$ = $1
	}
	| STRICT_P
	{
		$$ = $1
	}
	| STRING_P
	{
		$$ = $1
	}
	| STRIP_P
	{
		$$ = $1
	}
	| SUBSCRIPTION
	{
		$$ = $1
	}
	| SUBSTRING
	{
		$$ = $1
	}
	| SUPPORT
	{
		$$ = $1
	}
	| SYMMETRIC
	{
		$$ = $1
	}
	| SYSID
	{
		$$ = $1
	}
	| SYSTEM_P
	{
		$$ = $1
	}
	| SYSTEM_USER
	{
		$$ = $1
	}
	| TABLE
	{
		$$ = $1
	}
	| TABLES
	{
		$$ = $1
	}
	| TABLESAMPLE
	{
		$$ = $1
	}
	| TABLESPACE
	{
		$$ = $1
	}
	| TARGET
	{
		$$ = $1
	}
	| TEMP
	{
		$$ = $1
	}
	| TEMPLATE
	{
		$$ = $1
	}
	| TEMPORARY
	{
		$$ = $1
	}
	| TEXT_P
	{
		$$ = $1
	}
	| THEN
	{
		$$ = $1
	}
	| TIES
	{
		$$ = $1
	}
	| TIME
	{
		$$ = $1
	}
	| TIMESTAMP
	{
		$$ = $1
	}
	| TRAILING
	{
		$$ = $1
	}
	| TRANSACTION
	{
		$$ = $1
	}
	| TRANSFORM
	{
		$$ = $1
	}
	| TREAT
	{
		$$ = $1
	}
	| TRIGGER
	{
		$$ = $1
	}
	| TRIM
	{
		$$ = $1
	}
	| TRUE_P
	{
		$$ = $1
	}
	| TRUNCATE
	{
		$$ = $1
	}
	| TRUSTED
	{
		$$ = $1
	}
	| TYPE_P
	{
		$$ = $1
	}
	| TYPES_P
	{
		$$ = $1
	}
	| UESCAPE
	{
		$$ = $1
	}
	| UNBOUNDED
	{
		$$ = $1
	}
	| UNCOMMITTED
	{
		$$ = $1
	}
	| UNCONDITIONAL
	{
		$$ = $1
	}
	| UNENCRYPTED
	{
		$$ = $1
	}
	| UNIQUE
	{
		$$ = $1
	}
	| UNKNOWN
	{
		$$ = $1
	}
	| UNLISTEN
	{
		$$ = $1
	}
	| UNLOGGED
	{
		$$ = $1
	}
	| UNTIL
	{
		$$ = $1
	}
	| UPDATE
	{
		$$ = $1
	}
	| USER
	{
		$$ = $1
	}
	| USING
	{
		$$ = $1
	}
	| VACUUM
	{
		$$ = $1
	}
	| VALID
	{
		$$ = $1
	}
	| VALIDATE
	{
		$$ = $1
	}
	| VALIDATOR
	{
		$$ = $1
	}
	| VALUE_P
	{
		$$ = $1
	}
	| VALUES
	{
		$$ = $1
	}
	| VARCHAR
	{
		$$ = $1
	}
	| VARIADIC
	{
		$$ = $1
	}
	| VERBOSE
	{
		$$ = $1
	}
	| VERSION_P
	{
		$$ = $1
	}
	| VIEW
	{
		$$ = $1
	}
	| VIEWS
	{
		$$ = $1
	}
	| VIRTUAL
	{
		$$ = $1
	}
	| VOLATILE
	{
		$$ = $1
	}
	| WHEN
	{
		$$ = $1
	}
	| WHITESPACE_P
	{
		$$ = $1
	}
	| WORK
	{
		$$ = $1
	}
	| WRAPPER
	{
		$$ = $1
	}
	| WRITE
	{
		$$ = $1
	}
	| XML_P
	{
		$$ = $1
	}
	| XMLATTRIBUTES
	{
		$$ = $1
	}
	| XMLCONCAT
	{
		$$ = $1
	}
	| XMLELEMENT
	{
		$$ = $1
	}
	| XMLEXISTS
	{
		$$ = $1
	}
	| XMLFOREST
	{
		$$ = $1
	}
	| XMLNAMESPACES
	{
		$$ = $1
	}
	| XMLPARSE
	{
		$$ = $1
	}
	| XMLPI
	{
		$$ = $1
	}
	| XMLROOT
	{
		$$ = $1
	}
	| XMLSERIALIZE
	{
		$$ = $1
	}
	| XMLTABLE
	{
		$$ = $1
	}
	| YES_P
	{
		$$ = $1
	}
	| ZONE
	{
		$$ = $1
	}
	;

%%
