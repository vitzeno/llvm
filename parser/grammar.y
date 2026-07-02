%{
package parser

%}

%union{
String string
Ast Ast 
}

%token<String> NUMBER IDENTIFIER SEPARATOR ASSIGN LET IF LT LTE GT GTE EQ NE OR AND ELSE WHILE PRINT
%token FUNC RETURN TYPE_INT TYPE_DOUBLE TYPE_BOOL TYPE_VOID TRUE FALSE

%type <Ast> statements statement expression assignment reassignment print control_flow while_statement

%nonassoc NO_ELSE
%nonassoc ELSE
%right ASSIGN
%left LT GT LTE GTE EQ NE OR AND
%left '+' '-'
%left '*' '/'
%right UMINUS
%left '(' ')'

%start program

%%
program :  /* empty */
     | program statement { YYlex.(*Lexer).evalAst($2); PrintAST($2, 0)} // replace this with a call to codegen for code generation
     ;

statement:
     expression SEPARATOR
     | assignment SEPARATOR
     | reassignment SEPARATOR
     | print SEPARATOR
     | control_flow
     | while_statement
     ;

statements:
     statement statements
     | statement
     ;

expression:
     NUMBER { $$ = &Number{$1} }
    | IDENTIFIER { $$ = &Variable{$1} }
    | expression '+' expression { $$ = &BinaryExpr{Op: "+", Lhs: $1, Rhs: $3} }
    | expression '-' expression { $$ = &BinaryExpr{Op: "-", Lhs: $1, Rhs: $3} }
    | expression '*' expression { $$ = &BinaryExpr{Op: "*", Lhs: $1, Rhs: $3} }
    | expression '/' expression { $$ = &BinaryExpr{Op: "/", Lhs: $1, Rhs: $3} }
    | expression LT expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression GT expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression LTE expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression GTE expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression EQ expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression NE expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression OR expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | expression AND expression { $$ = &BinaryExpr{Op: $2, Lhs: $1, Rhs: $3} }
    | '(' expression ')'  { $$ = &ParenExpr{$2} }
    | '-' expression %prec UMINUS { $$ = &UnaryExpr{$2} }
    ;
    
assignment:
     LET IDENTIFIER ASSIGN expression { $$ = &Assignment{Variable: $2, Expr: $4} }
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

%%
