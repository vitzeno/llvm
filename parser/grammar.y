%{
package parser

%}

%union{
String string
Ast Ast
Asts []Ast
Params []Param
Type Type
}

%token<String> NUMBER IDENTIFIER SEPARATOR ASSIGN LET IF LT LTE GT GTE EQ NE OR AND ELSE WHILE PRINT
%token FUNC RETURN TYPE_INT TYPE_DOUBLE TYPE_BOOL TYPE_VOID TRUE FALSE

%type <Ast> program statements statement expression assignment reassignment print control_flow while_statement func_decl return_stmt
%type <Asts> args arg_list
%type <Params> params param_list
%type <Type> type_name return_type

%nonassoc NO_ELSE
%nonassoc ELSE
%right ASSIGN
%left OR
%left AND
%left LT GT LTE GTE EQ NE
%left '+' '-'
%left '*' '/'
%right UMINUS
%left '(' ')'

%start program

%%
program :  /* empty */ { root := &Block{}; $$ = root; YYlex.(*Lexer).rootAst = root }
     | program statement { blk := $1.(*Block); blk.Stmts = append(blk.Stmts, $2); $$ = blk }
     | program func_decl { blk := $1.(*Block); blk.Stmts = append(blk.Stmts, $2); $$ = blk }
     ;

statement:
     expression SEPARATOR
     | assignment SEPARATOR
     | reassignment SEPARATOR
     | print SEPARATOR
     | return_stmt SEPARATOR
     | control_flow
     | while_statement
     ;

statements:
     statement statements { blk := $2.(*Block); blk.Stmts = append([]Ast{$1}, blk.Stmts...); $$ = blk }
     | statement { $$ = &Block{Stmts: []Ast{$1}} }
     ;

expression:
     NUMBER { $$ = newNumber($1) }
    | TRUE { $$ = &BoolLit{Value: true} }
    | FALSE { $$ = &BoolLit{Value: false} }
    | IDENTIFIER { $$ = &Variable{$1} }
    | IDENTIFIER '(' args ')' { $$ = &CallExpr{Name: $1, Args: $3} }
    | expression '+' expression { $$ = &BinaryExpr{Op: "+", Lhs: $1, Rhs: $3} }
    | expression '-' expression { $$ = &BinaryExpr{Op: "-", Lhs: $1, Rhs: $3} }
    | expression '*' expression { $$ = &BinaryExpr{Op: "*", Lhs: $1, Rhs: $3} }
    | expression '/' expression { $$ = &BinaryExpr{Op: "/", Lhs: $1, Rhs: $3} }
    | expression LT expression { $$ = &BinaryExpr{Op: "<", Lhs: $1, Rhs: $3} }
    | expression GT expression { $$ = &BinaryExpr{Op: ">", Lhs: $1, Rhs: $3} }
    | expression LTE expression { $$ = &BinaryExpr{Op: "<=", Lhs: $1, Rhs: $3} }
    | expression GTE expression { $$ = &BinaryExpr{Op: ">=", Lhs: $1, Rhs: $3} }
    | expression EQ expression { $$ = &BinaryExpr{Op: "==", Lhs: $1, Rhs: $3} }
    | expression NE expression { $$ = &BinaryExpr{Op: "!=", Lhs: $1, Rhs: $3} }
    | expression OR expression { $$ = &BinaryExpr{Op: "||", Lhs: $1, Rhs: $3} }
    | expression AND expression { $$ = &BinaryExpr{Op: "&&", Lhs: $1, Rhs: $3} }
    | '(' expression ')'  { $$ = &ParenExpr{$2} }
    | '-' expression %prec UMINUS { $$ = &UnaryExpr{$2} }
    ;

type_name:
     TYPE_INT { $$ = TypeInt }
     | TYPE_DOUBLE { $$ = TypeDouble }
     | TYPE_BOOL { $$ = TypeBool }
     ;

return_type:
     type_name { $$ = $1 }
     | TYPE_VOID { $$ = TypeVoid }
     ;

assignment:
     LET IDENTIFIER ':' type_name ASSIGN expression { $$ = &Assignment{Variable: $2, Type: $4, Expr: $6} }
     ;

reassignment:
     IDENTIFIER ASSIGN expression { $$ = &Reassignment{Variable: $1, Expr: $3} }
     ;

print:
     PRINT '(' expression ')' { $$ = &StdPrint{Expr: $3} }
     ;

control_flow:
     IF '(' expression ')' '{' statements '}'  %prec NO_ELSE { $$ = &IfStatement{Cond: $3, ThenStmt: $6, ElseStmt: nil} }
     | IF '(' expression ')' '{' statements '}' ELSE '{' statements '}' { $$ = &IfStatement{Cond: $3, ThenStmt: $6, ElseStmt: $10} }
     ;

while_statement:
     WHILE '(' expression ')' '{' statements '}' { $$ = &WhileStatement{Cond: $3, Body: $6} }
     ;

func_decl:
     FUNC IDENTIFIER '(' params ')' ':' return_type '{' statements '}' { $$ = &FuncDecl{Name: $2, Params: $4, ReturnType: $7, Body: $9} }
     ;

params:
     /* empty */ { $$ = nil }
     | param_list
     ;

param_list:
     IDENTIFIER ':' type_name { $$ = []Param{{Name: $1, Type: $3}} }
     | param_list ',' IDENTIFIER ':' type_name { $$ = append($1, Param{Name: $3, Type: $5}) }
     ;

return_stmt:
     RETURN expression { $$ = &ReturnStmt{Expr: $2} }
     | RETURN { $$ = &ReturnStmt{} }
     ;

args:
     /* empty */ { $$ = nil }
     | arg_list
     ;

arg_list:
     expression { $$ = []Ast{$1} }
     | arg_list ',' expression { $$ = append($1, $3) }
     ;

%%
