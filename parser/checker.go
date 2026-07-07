package parser

import (
	"fmt"
	"strconv"
)

// FuncSig describes a function's parameter and return types
type FuncSig struct {
	Params []Type
	Return Type
}

// reservedFuncNames are names the code generator claims for itself
var reservedFuncNames = map[string]bool{
	"main":   true,
	"printf": true,
}

// Checker performs semantic analysis over a parsed program: scope and
// definition resolution, type checking, and return-path analysis. It never
// evaluates anything; it only computes the static type of each expression.
type Checker struct {
	scopes        []map[string]Type
	functions     map[string]FuncSig
	currentReturn Type
	inFunction    bool
	errors        []Diagnostic
}

// NewChecker creates a checker with an empty global scope
func NewChecker() *Checker {
	return &Checker{
		scopes:    []map[string]Type{make(map[string]Type)},
		functions: make(map[string]FuncSig),
	}
}

// Errors returns the diagnostics accumulated during Check
func (c *Checker) Errors() []Diagnostic {
	return c.errors
}

func (c *Checker) errorf(format string, args ...any) {
	c.errors = append(c.errors, Diagnostic{
		Message:  fmt.Sprintf(format, args...),
		Severity: Error,
	})
}

// Check validates a whole program. Function signatures are registered before
// any body is checked so functions may reference each other regardless of
// declaration order.
func (c *Checker) Check(root Ast) []Diagnostic {
	block, ok := root.(*Block)
	if !ok {
		c.errorf("internal error: program root is not a block")
		return c.errors
	}

	for _, stmt := range block.Stmts {
		fn, ok := stmt.(*FuncDecl)
		if !ok {
			continue
		}
		if reservedFuncNames[fn.Name] {
			c.errorf("function name %q is reserved", fn.Name)
			continue
		}
		if _, exists := c.functions[fn.Name]; exists {
			c.errorf("function %q already defined", fn.Name)
			continue
		}
		params := make([]Type, len(fn.Params))
		for i, p := range fn.Params {
			params[i] = p.Type
		}
		c.functions[fn.Name] = FuncSig{Params: params, Return: fn.ReturnType}
	}

	for _, stmt := range block.Stmts {
		if fn, ok := stmt.(*FuncDecl); ok {
			c.checkFunc(fn)
			continue
		}
		c.checkStmt(stmt)
	}

	return c.errors
}

func (c *Checker) pushScope() {
	c.scopes = append(c.scopes, make(map[string]Type))
}

func (c *Checker) popScope() {
	c.scopes = c.scopes[:len(c.scopes)-1]
}

// declare adds a name to the innermost scope; shadowing an outer scope is
// allowed, redeclaring within the same scope is not
func (c *Checker) declare(name string, t Type) {
	scope := c.scopes[len(c.scopes)-1]
	if _, exists := scope[name]; exists {
		c.errorf("variable %q already defined", name)
		return
	}
	scope[name] = t
}

// lookup resolves a name against the scope stack, innermost first
func (c *Checker) lookup(name string) (Type, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if t, ok := c.scopes[i][name]; ok {
			return t, true
		}
	}
	return TypeUnknown, false
}

// assignable reports whether a value of type "from" can be used where type
// "to" is expected; int implicitly promotes to double
func assignable(from, to Type) bool {
	return from == to || (from == TypeInt && to == TypeDouble)
}

func isNumeric(t Type) bool {
	return t == TypeInt || t == TypeDouble
}

func (c *Checker) checkFunc(fn *FuncDecl) {
	prevReturn, prevIn := c.currentReturn, c.inFunction
	c.currentReturn, c.inFunction = fn.ReturnType, true
	defer func() { c.currentReturn, c.inFunction = prevReturn, prevIn }()

	// a function body sees only its own parameters and locals: top-level
	// variables live in main's stack frame and are not addressable from
	// other functions. Parameters share a scope with the body's top level,
	// so a top-level let cannot reuse a parameter name (it may shadow in a
	// nested block)
	prevScopes := c.scopes
	c.scopes = []map[string]Type{make(map[string]Type)}
	defer func() { c.scopes = prevScopes }()
	for _, p := range fn.Params {
		if p.Type == TypeVoid {
			c.errorf("function %q: parameter %q cannot be void", fn.Name, p.Name)
			continue
		}
		c.declare(p.Name, p.Type)
	}

	body, ok := fn.Body.(*Block)
	if !ok {
		c.errorf("internal error: function %q body is not a block", fn.Name)
		return
	}
	for _, stmt := range body.Stmts {
		c.checkStmt(stmt)
	}

	if fn.ReturnType != TypeVoid && !alwaysReturns(body) {
		c.errorf("function %q: not all paths return a value", fn.Name)
	}
}

func (c *Checker) checkStmt(a Ast) {
	switch e := a.(type) {
	case *Block:
		c.pushScope()
		for _, stmt := range e.Stmts {
			c.checkStmt(stmt)
		}
		c.popScope()

	case *Assignment:
		t := c.checkExpr(e.Expr)
		if t != TypeUnknown && !assignable(t, e.Type) {
			c.errorf("type mismatch: cannot assign %s to %s variable %q", t, e.Type, e.Variable)
		}
		c.declare(e.Variable, e.Type)

	case *Reassignment:
		declared, ok := c.lookup(e.Variable)
		if !ok {
			c.errorf("undefined variable: %s", e.Variable)
			c.checkExpr(e.Expr)
			return
		}
		t := c.checkExpr(e.Expr)
		if t != TypeUnknown && !assignable(t, declared) {
			c.errorf("type mismatch: cannot assign %s to %s variable %q", t, declared, e.Variable)
		}

	case *StdPrint:
		t := c.checkExpr(e.Expr)
		if t == TypeVoid {
			c.errorf("cannot print a void expression")
		}

	case *IfStatement:
		if t := c.checkExpr(e.Cond); t != TypeUnknown && t != TypeBool {
			c.errorf("if condition must be bool, got %s", t)
		}
		c.checkStmt(e.ThenStmt)
		if e.ElseStmt != nil {
			c.checkStmt(e.ElseStmt)
		}

	case *WhileStatement:
		if t := c.checkExpr(e.Cond); t != TypeUnknown && t != TypeBool {
			c.errorf("while condition must be bool, got %s", t)
		}
		c.checkStmt(e.Body)

	case *ReturnStmt:
		if !c.inFunction {
			c.errorf("return outside of function")
			if e.Expr != nil {
				c.checkExpr(e.Expr)
			}
			return
		}
		if e.Expr == nil {
			if c.currentReturn != TypeVoid {
				c.errorf("missing return value in %s function", c.currentReturn)
			}
			return
		}
		if c.currentReturn == TypeVoid {
			c.errorf("void function cannot return a value")
			c.checkExpr(e.Expr)
			return
		}
		t := c.checkExpr(e.Expr)
		if t != TypeUnknown && !assignable(t, c.currentReturn) {
			c.errorf("type mismatch: cannot return %s from %s function", t, c.currentReturn)
		}

	case *FuncDecl:
		// the grammar only allows function declarations at the top level,
		// which Check handles directly
		c.errorf("internal error: nested function declaration %q", e.Name)

	default:
		// a bare expression statement; a void call is legitimate here
		c.checkExpr(a)
	}
}

func (c *Checker) checkExpr(a Ast) Type {
	switch e := a.(type) {
	case *Number:
		if e.IsFloat {
			if _, err := strconv.ParseFloat(e.Value, 64); err != nil {
				c.errorf("invalid number: %s", e.Value)
				return TypeUnknown
			}
			return TypeDouble
		}
		if _, err := strconv.ParseInt(e.Value, 10, 32); err != nil {
			c.errorf("invalid number: %s", e.Value)
			return TypeUnknown
		}
		return TypeInt

	case *BoolLit:
		return TypeBool

	case *Variable:
		t, ok := c.lookup(e.Name)
		if !ok {
			c.errorf("undefined variable: %s", e.Name)
			return TypeUnknown
		}
		return t

	case *ParenExpr:
		return c.checkExpr(e.Expr)

	case *UnaryExpr:
		t := c.checkExpr(e.Expr)
		if t == TypeUnknown {
			return TypeUnknown
		}
		if !isNumeric(t) {
			c.errorf("unary minus requires a numeric operand, got %s", t)
			return TypeUnknown
		}
		return t

	case *BinaryExpr:
		return c.checkBinaryExpr(e)

	case *CallExpr:
		sig, ok := c.functions[e.Name]
		if !ok {
			c.errorf("undefined function: %s", e.Name)
			for _, arg := range e.Args {
				c.checkExpr(arg)
			}
			return TypeUnknown
		}
		if len(e.Args) != len(sig.Params) {
			c.errorf("wrong number of arguments to %q: expected %d, got %d", e.Name, len(sig.Params), len(e.Args))
			for _, arg := range e.Args {
				c.checkExpr(arg)
			}
			return sig.Return
		}
		for i, arg := range e.Args {
			t := c.checkExpr(arg)
			if t != TypeUnknown && !assignable(t, sig.Params[i]) {
				c.errorf("argument %d to %q: cannot use %s as %s", i+1, e.Name, t, sig.Params[i])
			}
		}
		return sig.Return

	default:
		c.errorf("internal error: unknown expression node %T", a)
		return TypeUnknown
	}
}

func (c *Checker) checkBinaryExpr(e *BinaryExpr) Type {
	lt := c.checkExpr(e.Lhs)
	rt := c.checkExpr(e.Rhs)
	if lt == TypeUnknown || rt == TypeUnknown {
		// an operand already failed; avoid cascading errors
		return TypeUnknown
	}

	switch e.Op {
	case "+", "-", "*", "/":
		if !isNumeric(lt) || !isNumeric(rt) {
			c.errorf("operator %s requires numeric operands, got %s and %s", e.Op, lt, rt)
			return TypeUnknown
		}
		if e.Op == "/" && isZeroLiteral(e.Rhs) {
			c.errorf("division by zero")
			return TypeUnknown
		}
		if lt == TypeDouble || rt == TypeDouble {
			return TypeDouble
		}
		return TypeInt

	case "<", ">", "<=", ">=":
		if !isNumeric(lt) || !isNumeric(rt) {
			c.errorf("operator %s requires numeric operands, got %s and %s", e.Op, lt, rt)
			return TypeUnknown
		}
		return TypeBool

	case "==", "!=":
		bothNumeric := isNumeric(lt) && isNumeric(rt)
		bothBool := lt == TypeBool && rt == TypeBool
		if !bothNumeric && !bothBool {
			c.errorf("cannot compare %s and %s", lt, rt)
			return TypeUnknown
		}
		return TypeBool

	case "&&", "||":
		if lt != TypeBool || rt != TypeBool {
			c.errorf("operator %s requires bool operands, got %s and %s", e.Op, lt, rt)
			return TypeUnknown
		}
		return TypeBool

	default:
		c.errorf("internal error: unknown operator %s", e.Op)
		return TypeUnknown
	}
}

// isZeroLiteral reports whether an expression is a literal zero divisor,
// looking through parentheses. Dynamic zero divisors are deliberately not
// detected: double division follows IEEE-754 semantics at runtime
func isZeroLiteral(a Ast) bool {
	switch e := a.(type) {
	case *ParenExpr:
		return isZeroLiteral(e.Expr)
	case *Number:
		v, err := strconv.ParseFloat(e.Value, 64)
		return err == nil && v == 0
	default:
		return false
	}
}

// alwaysReturns reports whether every execution path through a statement
// ends in a return
func alwaysReturns(a Ast) bool {
	switch e := a.(type) {
	case *Block:
		for _, stmt := range e.Stmts {
			if alwaysReturns(stmt) {
				return true
			}
		}
		return false
	case *ReturnStmt:
		return true
	case *IfStatement:
		return e.ElseStmt != nil && alwaysReturns(e.ThenStmt) && alwaysReturns(e.ElseStmt)
	default:
		return false
	}
}
