package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type Ast any

type Root struct {
	Expr Ast
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

func (l *Lexer) evalAst(a Ast) (float64, error) {
	switch e := a.(type) {
	case *BinaryExpr:
		return l.evalBinaryExpr(e)
	case *UnaryExpr:
		return l.evalUnaryExpr(e)
	case *Root:
		return l.evalRoot(e)
	case *ParenExpr:
		return l.evalParenExpr(e)
	case *Variable:
		return l.evalVariable(e)
	case *Number:
		return l.evalNumber(e)
	case *Assignment:
		return l.evalAssignment(e)
	case *Reassignment:
		return l.evalReassignment(e)
	case *StdPrint:
		return l.evalStdPrint(e)
	case *IfStatement:
		return l.evalIfStatement(e)
	case *WhileStatement:
		return l.evalWhileStatement(e)
	default:
		return 0, fmt.Errorf("unknown node type")
	}
}

func (l *Lexer) evalBinaryExpr(e *BinaryExpr) (float64, error) {
	switch e.Op {
	case "+":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		return lhs + rhs, nil
	case "-":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		return lhs - rhs, nil
	case "*":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		return lhs * rhs, nil
	case "/":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if rhs == 0 {
			l.Error("division by zero")
			return 0, fmt.Errorf("division by zero")
		}
		return lhs / rhs, nil
	case ">":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs > rhs {
			return 1, nil
		}
		return 0, nil
	case "<":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs < rhs {
			return 1, nil
		}
		return 0, nil
	case "==":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs == rhs {
			return 1, nil
		}
		return 0, nil
	case "NE":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs != rhs {
			return 1, nil
		}
		return 0, nil
	case "OR":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs != 0 || rhs != 0 {
			return 1, nil
		}
		return 0, nil
	case "AND":
		lhs, err := l.evalAst(e.Lhs)
		if err != nil {
			return 0, err
		}
		rhs, err := l.evalAst(e.Rhs)
		if err != nil {
			return 0, err
		}

		if lhs != 0 && rhs != 0 {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown operator: %s", e.Op)
	}
}

func (l *Lexer) evalUnaryExpr(e *UnaryExpr) (float64, error) {
	result, err := l.evalAst(e.Expr)
	return -result, err
}

func (l *Lexer) evalRoot(e *Root) (float64, error) {
	return l.evalAst(e.Expr)
}

func (l *Lexer) evalParenExpr(e *ParenExpr) (float64, error) {
	return l.evalAst(e.Expr)
}

func (l *Lexer) evalVariable(e *Variable) (float64, error) {
	val, ok := l.variables[e.Name]
	if !ok {
		l.Error(fmt.Sprintf("undefined variable: %s", e.Name))
		return 0, fmt.Errorf("undefined variable: %s", e.Name)
	}
	return val.value, nil
}

func (l *Lexer) evalNumber(e *Number) (float64, error) {
	val, err := strconv.ParseFloat(e.Value, 64)
	if err != nil {
		l.Error(fmt.Sprintf("invalid number: %s", e.Value))
		return 0, fmt.Errorf("invalid number: %s", e.Value)
	}
	return val, nil
}

func (l *Lexer) evalAssignment(e *Assignment) (float64, error) {
	_, ok := l.variables[e.Variable]
	if ok {
		l.Error(fmt.Sprintf(`variable "%s" already defined`, e.Variable))
		return 0, fmt.Errorf(`variable "%s" already defined`, e.Variable)
	}

	result, err := l.evalAst(e.Expr)
	if err != nil {
		return 0, err
	}

	if !l.evalFailed {
		l.variables[e.Variable] = variable{location: nil, value: result}
	}
	return result, nil
}

func (l *Lexer) evalReassignment(e *Reassignment) (float64, error) {
	_, ok := l.variables[e.Variable]
	if !ok {
		l.Error(fmt.Sprintf("undefined variable: %s", e.Variable))
		return 0, fmt.Errorf("undefined variable: %s", e.Variable)
	}

	result, err := l.evalAst(e.Expr)
	if err != nil {
		return 0, err
	}

	if !l.evalFailed {
		l.variables[e.Variable] = variable{location: l.variables[e.Variable].location, value: result}
	}
	return result, nil
}

func (l *Lexer) evalStdPrint(e *StdPrint) (float64, error) {
	result, err := l.evalAst(e.Expr)
	if err != nil {
		return 0, err
	}
	if !l.evalFailed {
		fmt.Println(result)
	}
	return result, nil
}

func (l *Lexer) evalIfStatement(e *IfStatement) (float64, error) {
	result, err := l.evalAst(e.Cond)
	if err != nil {
		return 0, err
	}

	if result != 0 {
		return l.evalAst(e.ThenStmt)
	}
	if e.ElseStmt != nil {
		return l.evalAst(e.ElseStmt)
	}
	return 0, nil
}

func (l *Lexer) evalWhileStatement(e *WhileStatement) (float64, error) {
	result, err := l.evalAst(e.Cond)
	if err != nil {
		return 0, err
	}

	if result != 0 {
		_, err := l.evalAst(e.Body)
		if err != nil {
			return 0, err
		}
	}
	return 0, nil
}

// PrintAST prints the AST starting from the given node
func PrintAST(a Ast, indentLevel int) {
	switch e := a.(type) {
	case *BinaryExpr:
		fmt.Printf("%sBinaryExpr(%s)\n", strings.Repeat("\t", indentLevel), e.Op)
		PrintAST(e.Lhs, indentLevel+1)
		PrintAST(e.Rhs, indentLevel+1)
	case *UnaryExpr:
		fmt.Printf("%sUnaryExpr\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Expr, indentLevel+1)
	case *Root:
		fmt.Printf("%sAstRoot\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Expr, indentLevel+1)
	case *ParenExpr:
		fmt.Printf("%sParenExpr\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Expr, indentLevel+1)
	case *Variable:
		fmt.Printf("%sVariable(%s)\n", strings.Repeat("\t", indentLevel), e.Name)
	case *Number:
		fmt.Printf("%sNumber(%s)\n", strings.Repeat("\t", indentLevel), e.Value)
	case *Assignment:
		fmt.Printf("%sAssignment(%s)\n", strings.Repeat("\t", indentLevel), e.Variable)
		PrintAST(e.Expr, indentLevel+1)
	case *Reassignment:
		fmt.Printf("%sReassignment(%s)\n", strings.Repeat("\t", indentLevel), e.Variable)
		PrintAST(e.Expr, indentLevel+1)
	case *StdPrint:
		fmt.Printf("%sStdPrint\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Expr, indentLevel+1)
	case *IfStatement:
		fmt.Printf("%sIfStatement\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Cond, indentLevel+1)
		fmt.Printf("%sThen\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.ThenStmt, indentLevel+1)
		if e.ElseStmt != nil {
			fmt.Printf("%sElse\n", strings.Repeat("\t", indentLevel))
			PrintAST(e.ElseStmt, indentLevel+1)
		}
	case *WhileStatement:
		fmt.Printf("%sWhileStatement\n", strings.Repeat("\t", indentLevel))
		PrintAST(e.Cond, indentLevel+1)
		PrintAST(e.Body, indentLevel+1)
	default:
		fmt.Println("Unknown node type")
	}
}
