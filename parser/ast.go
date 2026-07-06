package parser

import (
	"fmt"
	"strings"
)

type Ast any

// Type is the static type of an expression or declaration
type Type int

const (
	TypeUnknown Type = iota
	TypeInt
	TypeDouble
	TypeBool
	TypeVoid
)

func (t Type) String() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeDouble:
		return "double"
	case TypeBool:
		return "bool"
	case TypeVoid:
		return "void"
	default:
		return "unknown"
	}
}

// Block is an ordered list of statements: the whole program, a function
// body, or the body of an if/else/while
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
	Value   string
	IsFloat bool
}

// newNumber builds a Number literal, classifying it as int or double
// based on the presence of a decimal point in the source text
func newNumber(text string) *Number {
	return &Number{Value: text, IsFloat: strings.Contains(text, ".")}
}

type BoolLit struct {
	Value bool
}

type Assignment struct {
	Variable string
	Type     Type
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

type Param struct {
	Name string
	Type Type
}

type FuncDecl struct {
	Name       string
	Params     []Param
	ReturnType Type
	Body       Ast
}

type ReturnStmt struct {
	Expr Ast // nil for a bare "return;"
}

type CallExpr struct {
	Name string
	Args []Ast
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
	case *BoolLit:
		fmt.Printf("%sBoolLit(%t)\n", indent, e.Value)
	case *Assignment:
		fmt.Printf("%sAssignment(%s: %s)\n", indent, e.Variable, e.Type)
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
	case *FuncDecl:
		params := make([]string, len(e.Params))
		for i, p := range e.Params {
			params[i] = fmt.Sprintf("%s: %s", p.Name, p.Type)
		}
		fmt.Printf("%sFuncDecl(%s(%s): %s)\n", indent, e.Name, strings.Join(params, ", "), e.ReturnType)
		PrintAST(e.Body, indentLevel+1)
	case *ReturnStmt:
		fmt.Printf("%sReturn\n", indent)
		if e.Expr != nil {
			PrintAST(e.Expr, indentLevel+1)
		}
	case *CallExpr:
		fmt.Printf("%sCall(%s)\n", indent, e.Name)
		for _, arg := range e.Args {
			PrintAST(arg, indentLevel+1)
		}
	default:
		fmt.Println("Unknown node type")
	}
}
