package parser

import (
	"fmt"
	"strings"
)

type Ast any

// Block is an ordered list of statements: the whole program or the body of
// an if/else/while
type Block struct {
	Stmts []Ast
}

type BinaryExpr struct {
	Op  string
	Lhs Ast
	Rhs Ast
}

type UnaryExpr struct {
	Expr Ast
}

type ParenExpr struct {
	Expr Ast
}

type Variable struct {
	Name string
}

type Number struct {
	Value string
}

type Assignment struct {
	Variable string
	Expr     Ast
}

type Reassignment struct {
	Variable string
	Expr     Ast
}

type StdPrint struct {
	Expr Ast
}

type IfStatement struct {
	Cond     Ast
	ThenStmt Ast
	ElseStmt Ast
}

type WhileStatement struct {
	Cond Ast
	Body Ast
}

// PrintAST prints the AST starting from the given node
func PrintAST(a Ast, indentLevel int) {
	indent := strings.Repeat("\t", indentLevel)
	switch e := a.(type) {
	case *Block:
		for _, stmt := range e.Stmts {
			PrintAST(stmt, indentLevel)
		}
	case *BinaryExpr:
		fmt.Printf("%sBinaryExpr(%s)\n", indent, e.Op)
		PrintAST(e.Lhs, indentLevel+1)
		PrintAST(e.Rhs, indentLevel+1)
	case *UnaryExpr:
		fmt.Printf("%sUnaryExpr\n", indent)
		PrintAST(e.Expr, indentLevel+1)
	case *ParenExpr:
		fmt.Printf("%sParenExpr\n", indent)
		PrintAST(e.Expr, indentLevel+1)
	case *Variable:
		fmt.Printf("%sVariable(%s)\n", indent, e.Name)
	case *Number:
		fmt.Printf("%sNumber(%s)\n", indent, e.Value)
	case *Assignment:
		fmt.Printf("%sAssignment(%s)\n", indent, e.Variable)
		PrintAST(e.Expr, indentLevel+1)
	case *Reassignment:
		fmt.Printf("%sReassignment(%s)\n", indent, e.Variable)
		PrintAST(e.Expr, indentLevel+1)
	case *StdPrint:
		fmt.Printf("%sStdPrint\n", indent)
		PrintAST(e.Expr, indentLevel+1)
	case *IfStatement:
		fmt.Printf("%sIfStatement\n", indent)
		PrintAST(e.Cond, indentLevel+1)
		fmt.Printf("%sThen\n", indent)
		PrintAST(e.ThenStmt, indentLevel+1)
		if e.ElseStmt != nil {
			fmt.Printf("%sElse\n", indent)
			PrintAST(e.ElseStmt, indentLevel+1)
		}
	case *WhileStatement:
		fmt.Printf("%sWhileStatement\n", indent)
		PrintAST(e.Cond, indentLevel+1)
		PrintAST(e.Body, indentLevel+1)
	default:
		fmt.Println("Unknown node type")
	}
}
