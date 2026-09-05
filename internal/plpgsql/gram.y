%{
package plpgsql
%}

%union {
 tok token
 text string
 expr *Expression
 stmt Statement
 stmts []Statement
 block *Block
 loop *Loop
 scope *scope
 elsifs []Elsif
 whens []When
 exceptions []Exception
 conditions []string
 otherwise caseElse
}

%token <tok> WORD STRING NUMBER SYMBOL LABEL
%token <tok> BEGIN DECLARE END IF THEN ELSE ELSIF CASE WHEN LOOP WHILE FOR FOREACH
%token <tok> EXCEPTION RETURN RAISE ASSERT PERFORM EXECUTE GET OPEN FETCH MOVE CLOSE
%token <tok> EXIT CONTINUE NULL CALL DO COMMIT ROLLBACK
%type <block> block
%type <scope> scope
%type <stmt> statement
%type <stmts> statements else_part
%type <expr> if_expr case_expr
%type <elsifs> elsif_parts
%type <whens> when_parts
%type <exceptions> exception_part handlers
%type <conditions> conditions
%type <text> end_label
%type <loop> while_header for_header foreach_header
%type <otherwise> case_else
%type <tok> declaration_name

%%
program: directives block optional_semi { yylex.(*parser).result = $2 };
directives: | directives '#' WORD { yylex.(*parser).directive($3) };
optional_semi: | ';';
scope: { $$ = yylex.(*parser).pushScope("", false) }
 | LABEL { $$ = yylex.(*parser).pushScope($1.text, false) };
block: scope declarations BEGIN statements exception_part END end_label {
 p := yylex.(*parser)
 p.endScope($1, $7)
 $$ = &Block{Base: Base{$3.line}, Label: $1.label, Body: $4, Exceptions: $5}
};
declarations: | DECLARE declaration_list;
declaration_list: | declaration_list DECLARE | declaration_list declaration_name { yylex.(*parser).declaration($2) };
declaration_name: WORD | ELSIF | EXCEPTION | RETURN | RAISE | ASSERT | PERFORM | GET | OPEN | FETCH | MOVE | CLOSE | EXIT | CONTINUE | CALL | DO | COMMIT | ROLLBACK;
end_label: { $$ = "" } | declaration_name { $$ = $1.text };
statements: { $$ = nil }
 | statements statement { $$ = $1; if $2 != nil { $$ = append($$, $2) } };
statement:
 block ';' { $$ = $1 }
 | NULL ';' { $$ = nil }
 | IF if_expr statements elsif_parts else_part END IF ';' { $$ = &If{Base: Base{$1.line}, Cond: $2, Then: $3, Elsifs: $4, Else: $5} }
 | CASE case_expr when_parts case_else END CASE ';' { $$ = yylex.(*parser).caseStatement($1, $2, $3, $4) }
 | scope LOOP { $1.loop = true } statements END LOOP end_label ';' {
  p := yylex.(*parser); p.endScope($1, $7)
  $$ = &Loop{Base: Base{$2.line}, Kind: "loop", Label: $1.label, Body: $4}
 }
 | scope WHILE while_header statements END LOOP end_label ';' {
  p := yylex.(*parser); p.endScope($1, $7)
  $3.Line = $2.line; $3.Label = $1.label; $3.Body = $4; $$ = $3
 }
 | scope FOR for_header statements END LOOP end_label ';' {
  p := yylex.(*parser); p.endScope($1, $7)
  $3.Line = $2.line; $3.Label = $1.label; $3.Body = $4; $$ = $3
 }
 | scope FOREACH foreach_header statements END LOOP end_label ';' {
  p := yylex.(*parser); p.endScope($1, $7)
  $3.Line = $2.line; $3.Label = $1.label; $3.Body = $4; $$ = $3
 }
 | RETURN { $$ = yylex.(*parser).returnStatement($1) }
 | RAISE { $$ = yylex.(*parser).raiseStatement($1) }
 | ASSERT { $$ = yylex.(*parser).assertStatement($1) }
 | PERFORM { $$ = yylex.(*parser).performStatement($1) }
 | EXECUTE { $$ = yylex.(*parser).executeStatement($1) }
 | GET { $$ = yylex.(*parser).diagnostics($1) }
 | OPEN { $$ = yylex.(*parser).openStatement($1) }
 | FETCH { $$ = yylex.(*parser).fetchStatement($1, false) }
 | MOVE { $$ = yylex.(*parser).fetchStatement($1, true) }
 | CLOSE { $$ = yylex.(*parser).closeStatement($1) }
 | EXIT { $$ = yylex.(*parser).exitStatement($1, true) }
 | CONTINUE { $$ = yylex.(*parser).exitStatement($1, false) }
 | CALL { $$ = yylex.(*parser).callStatement($1, true) }
 | DO { $$ = yylex.(*parser).callStatement($1, false) }
 | COMMIT { $$ = yylex.(*parser).transaction($1, false) }
 | ROLLBACK { $$ = yylex.(*parser).transaction($1, true) }
 | WORD { $$ = yylex.(*parser).sqlStatement($1) };
if_expr: { $$ = yylex.(*parser).expression(2, "then") };
elsif_parts: { $$ = nil }
 | elsif_parts ELSIF if_expr statements { $$ = append($1, Elsif{Line: $2.line, Cond: $3, Body: $4}) };
else_part: { $$ = nil } | ELSE statements { $$ = $2 };
case_expr: { $$ = yylex.(*parser).caseExpression() };
when_parts: WHEN if_expr statements { $$ = []When{{Line: $1.line, Expr: $2, Body: $3}} }
 | when_parts WHEN if_expr statements { $$ = append($1, When{Line: $2.line, Expr: $3, Body: $4}) };
case_else: { $$ = caseElse{} } | ELSE statements { $$ = caseElse{present: true, body: $2} };
while_header: { p := yylex.(*parser); p.currentScope().loop = true; $$ = &Loop{Kind: "while", Cond: p.expression(2, "loop")} };
for_header: { $$ = yylex.(*parser).forHeader() };
foreach_header: { $$ = yylex.(*parser).foreachHeader() };
exception_part: { $$ = nil }
 | EXCEPTION { yylex.(*parser).exceptionVariables($1) } handlers { $$ = $3 };
handlers: WHEN conditions statements { $$ = []Exception{{Conditions: $2, Action: $3}} }
 | handlers WHEN conditions statements { $$ = append($1, Exception{Conditions: $3, Action: $4}) };
conditions: { $$ = yylex.(*parser).conditions() };
%%
