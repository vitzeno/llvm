package codegen

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"

	"github.com/vitzeno/llvm-test/parser"
)

// varSlot is a stack slot for a named variable
type varSlot struct {
	alloca *ir.InstAlloca
	typ    parser.Type
}

// Generator lowers a type-checked AST to an LLVM IR module. It assumes the
// program has already passed the semantic checker: variables resolve, types
// line up and non-void functions return on every path, so no error paths for
// those conditions exist here.
type Generator struct {
	module *ir.Module
	printf *ir.Func

	functions map[string]*ir.Func
	sigs      map[string]parser.FuncSig

	fn            *ir.Func
	block         *ir.Block
	currentReturn parser.Type
	scopes        []map[string]varSlot

	fmtInt    *ir.Global // "%d\n" for int and bool
	fmtDouble *ir.Global // "%g\n" for double

	labelID int
}

// New creates an empty Generator
func New() *Generator {
	return &Generator{
		functions: make(map[string]*ir.Func),
		sigs:      make(map[string]parser.FuncSig),
	}
}

// Generate lowers a whole program. Top-level statements become the body of
// an implicit main function; function declarations become their own LLVM
// functions and are pre-declared so call order does not matter.
func (g *Generator) Generate(root parser.Ast) (m *ir.Module, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("codegen: %v", r)
		}
	}()

	block, ok := root.(*parser.Block)
	if !ok {
		return nil, fmt.Errorf("codegen: program root is not a block")
	}

	g.module = ir.NewModule()
	switch runtime.GOOS {
	case "windows":
		g.module.TargetTriple = "x86_64-pc-windows-msvc"
	case "darwin":
		g.module.TargetTriple = "x86_64-apple-darwin"
	case "linux":
		g.module.TargetTriple = "x86_64-pc-linux-gnu"
	default:
		return nil, fmt.Errorf("codegen: unsupported OS target %q", runtime.GOOS)
	}

	g.printf = g.module.NewFunc("printf", types.I32, ir.NewParam("", types.NewPointer(types.I8)))
	g.printf.Sig.Variadic = true

	g.fmtInt = g.newFmtString(".fmt.int", "%d\n\x00")
	g.fmtDouble = g.newFmtString(".fmt.double", "%g\n\x00")

	// declare every function up front so calls resolve regardless of
	// declaration order
	for _, stmt := range block.Stmts {
		if fn, ok := stmt.(*parser.FuncDecl); ok {
			g.declareFunc(fn)
		}
	}

	// top-level statements form the body of main
	mainFn := g.module.NewFunc("main", types.I32)
	g.fn = mainFn
	g.block = mainFn.NewBlock("entry")
	g.currentReturn = parser.TypeUnknown
	g.scopes = []map[string]varSlot{make(map[string]varSlot)}
	for _, stmt := range block.Stmts {
		if _, ok := stmt.(*parser.FuncDecl); ok {
			continue
		}
		if g.block.Term != nil {
			break
		}
		g.genStmt(stmt)
	}
	if g.block.Term == nil {
		g.block.NewRet(constant.NewInt(types.I32, 0))
	}

	for _, stmt := range block.Stmts {
		if fn, ok := stmt.(*parser.FuncDecl); ok {
			g.genFunc(fn)
		}
	}

	return g.module, nil
}

func (g *Generator) newFmtString(name, contents string) *ir.Global {
	global := g.module.NewGlobalDef(name, constant.NewCharArrayFromString(contents))
	global.Immutable = true
	return global
}

func (g *Generator) declareFunc(decl *parser.FuncDecl) {
	params := make([]*ir.Param, len(decl.Params))
	sigParams := make([]parser.Type, len(decl.Params))
	for i, p := range decl.Params {
		params[i] = ir.NewParam(p.Name, irType(p.Type))
		sigParams[i] = p.Type
	}
	fn := g.module.NewFunc(decl.Name, irType(decl.ReturnType), params...)
	g.functions[decl.Name] = fn
	g.sigs[decl.Name] = parser.FuncSig{Params: sigParams, Return: decl.ReturnType}
}

func (g *Generator) genFunc(decl *parser.FuncDecl) {
	fn := g.functions[decl.Name]
	g.fn = fn
	g.block = fn.NewBlock("entry")
	g.currentReturn = decl.ReturnType
	// a function sees only its own parameters and locals
	g.scopes = []map[string]varSlot{make(map[string]varSlot)}

	for i, p := range decl.Params {
		slot := g.block.NewAlloca(irType(p.Type))
		slot.SetName(fmt.Sprintf("%s.addr", p.Name))
		g.block.NewStore(fn.Params[i], slot)
		g.declare(p.Name, varSlot{alloca: slot, typ: p.Type})
	}

	body := decl.Body.(*parser.Block)
	for _, stmt := range body.Stmts {
		if g.block.Term != nil {
			break
		}
		g.genStmt(stmt)
	}

	if g.block.Term == nil {
		if decl.ReturnType == parser.TypeVoid {
			g.block.NewRet(nil)
		} else {
			// the checker proved every path returns, so an open block
			// here can only be an unreachable merge block
			g.block.NewUnreachable()
		}
	}
}

func (g *Generator) pushScope() {
	g.scopes = append(g.scopes, make(map[string]varSlot))
}

func (g *Generator) popScope() {
	g.scopes = g.scopes[:len(g.scopes)-1]
}

func (g *Generator) declare(name string, slot varSlot) {
	g.scopes[len(g.scopes)-1][name] = slot
}

func (g *Generator) lookup(name string) varSlot {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if slot, ok := g.scopes[i][name]; ok {
			return slot
		}
	}
	panic(fmt.Sprintf("internal error: undefined variable %q survived the checker", name))
}

func (g *Generator) nextID() int {
	g.labelID++
	return g.labelID
}

// allocaVar creates a stack slot in the entry block of the current function.
// Allocas live in the entry block regardless of where the declaration
// appears so they dominate all uses after branches merge.
func (g *Generator) allocaVar(name string, t parser.Type) *ir.InstAlloca {
	entry := g.fn.Blocks[0]
	slot := entry.NewAlloca(irType(t))
	slot.SetName(fmt.Sprintf("%s.%d", name, g.nextID()))
	return slot
}

func (g *Generator) genStmt(a parser.Ast) {
	switch e := a.(type) {
	case *parser.Block:
		g.pushScope()
		for _, stmt := range e.Stmts {
			if g.block.Term != nil {
				break // statements after a return are unreachable
			}
			g.genStmt(stmt)
		}
		g.popScope()

	case *parser.Assignment:
		v, t := g.genExpr(e.Expr)
		v = g.coerce(v, t, e.Type)
		slot := g.allocaVar(e.Variable, e.Type)
		g.block.NewStore(v, slot)
		g.declare(e.Variable, varSlot{alloca: slot, typ: e.Type})

	case *parser.Reassignment:
		slot := g.lookup(e.Variable)
		v, t := g.genExpr(e.Expr)
		v = g.coerce(v, t, slot.typ)
		g.block.NewStore(v, slot.alloca)

	case *parser.StdPrint:
		g.genPrint(e)

	case *parser.IfStatement:
		g.genIf(e)

	case *parser.WhileStatement:
		g.genWhile(e)

	case *parser.ReturnStmt:
		if e.Expr == nil {
			g.block.NewRet(nil)
			return
		}
		v, t := g.genExpr(e.Expr)
		v = g.coerce(v, t, g.currentReturn)
		g.block.NewRet(v)

	default:
		// a bare expression statement, evaluated for its side effects
		g.genExpr(a)
	}
}

func (g *Generator) genPrint(e *parser.StdPrint) {
	v, t := g.genExpr(e.Expr)
	var fmtStr *ir.Global
	switch t {
	case parser.TypeInt:
		fmtStr = g.fmtInt
	case parser.TypeDouble:
		fmtStr = g.fmtDouble
	case parser.TypeBool:
		v = g.block.NewZExt(v, types.I32)
		fmtStr = g.fmtInt
	default:
		panic(fmt.Sprintf("internal error: cannot print %s value", t))
	}
	zero := constant.NewInt(types.I32, 0)
	ptr := g.block.NewGetElementPtr(fmtStr.ContentType, fmtStr, zero, zero)
	g.block.NewCall(g.printf, ptr, v)
}

func (g *Generator) genIf(e *parser.IfStatement) {
	cond, _ := g.genExpr(e.Cond)

	id := g.nextID()
	thenB := g.fn.NewBlock(fmt.Sprintf("if.then.%d", id))
	var elseB *ir.Block
	if e.ElseStmt != nil {
		elseB = g.fn.NewBlock(fmt.Sprintf("if.else.%d", id))
	}
	mergeB := g.fn.NewBlock(fmt.Sprintf("if.end.%d", id))

	// without an else the false edge falls through to the merge block
	falseTarget := mergeB
	if elseB != nil {
		falseTarget = elseB
	}
	g.block.NewCondBr(cond, thenB, falseTarget)

	g.block = thenB
	g.genStmt(e.ThenStmt)
	if g.block.Term == nil {
		g.block.NewBr(mergeB)
	}

	if elseB != nil {
		g.block = elseB
		g.genStmt(e.ElseStmt)
		if g.block.Term == nil {
			g.block.NewBr(mergeB)
		}
	}

	g.block = mergeB
}

func (g *Generator) genWhile(e *parser.WhileStatement) {
	id := g.nextID()
	condB := g.fn.NewBlock(fmt.Sprintf("while.cond.%d", id))
	bodyB := g.fn.NewBlock(fmt.Sprintf("while.body.%d", id))
	endB := g.fn.NewBlock(fmt.Sprintf("while.end.%d", id))

	g.block.NewBr(condB)

	// the condition is emitted inside the loop header so the backedge
	// re-evaluates it on every iteration
	g.block = condB
	cond, _ := g.genExpr(e.Cond)
	g.block.NewCondBr(cond, bodyB, endB)

	g.block = bodyB
	g.genStmt(e.Body)
	if g.block.Term == nil {
		g.block.NewBr(condB)
	}

	g.block = endB
}

func (g *Generator) genExpr(a parser.Ast) (value.Value, parser.Type) {
	switch e := a.(type) {
	case *parser.Number:
		if e.IsFloat {
			v, err := strconv.ParseFloat(e.Value, 64)
			if err != nil {
				panic(fmt.Sprintf("internal error: invalid double literal %q", e.Value))
			}
			return constant.NewFloat(types.Double, v), parser.TypeDouble
		}
		v, err := strconv.ParseInt(e.Value, 10, 32)
		if err != nil {
			panic(fmt.Sprintf("internal error: invalid int literal %q", e.Value))
		}
		return constant.NewInt(types.I32, v), parser.TypeInt

	case *parser.BoolLit:
		return constant.NewBool(e.Value), parser.TypeBool

	case *parser.Variable:
		slot := g.lookup(e.Name)
		return g.block.NewLoad(irType(slot.typ), slot.alloca), slot.typ

	case *parser.ParenExpr:
		return g.genExpr(e.Expr)

	case *parser.UnaryExpr:
		v, t := g.genExpr(e.Expr)
		if t == parser.TypeDouble {
			return g.block.NewFNeg(v), parser.TypeDouble
		}
		return g.block.NewSub(constant.NewInt(types.I32, 0), v), parser.TypeInt

	case *parser.BinaryExpr:
		return g.genBinaryExpr(e)

	case *parser.CallExpr:
		fn := g.functions[e.Name]
		sig := g.sigs[e.Name]
		args := make([]value.Value, len(e.Args))
		for i, arg := range e.Args {
			v, t := g.genExpr(arg)
			args[i] = g.coerce(v, t, sig.Params[i])
		}
		return g.block.NewCall(fn, args...), sig.Return

	default:
		panic(fmt.Sprintf("internal error: unknown expression node %T", a))
	}
}

func (g *Generator) genBinaryExpr(e *parser.BinaryExpr) (value.Value, parser.Type) {
	lv, lt := g.genExpr(e.Lhs)
	rv, rt := g.genExpr(e.Rhs)

	switch e.Op {
	case "+", "-", "*", "/":
		if lt == parser.TypeDouble || rt == parser.TypeDouble {
			lv = g.coerce(lv, lt, parser.TypeDouble)
			rv = g.coerce(rv, rt, parser.TypeDouble)
			switch e.Op {
			case "+":
				return g.block.NewFAdd(lv, rv), parser.TypeDouble
			case "-":
				return g.block.NewFSub(lv, rv), parser.TypeDouble
			case "*":
				return g.block.NewFMul(lv, rv), parser.TypeDouble
			default:
				return g.block.NewFDiv(lv, rv), parser.TypeDouble
			}
		}
		switch e.Op {
		case "+":
			return g.block.NewAdd(lv, rv), parser.TypeInt
		case "-":
			return g.block.NewSub(lv, rv), parser.TypeInt
		case "*":
			return g.block.NewMul(lv, rv), parser.TypeInt
		default:
			return g.block.NewSDiv(lv, rv), parser.TypeInt
		}

	case "<", ">", "<=", ">=", "==", "!=":
		// bool operands are only legal for == and != per the checker
		if lt == parser.TypeBool {
			pred := map[string]enum.IPred{"==": enum.IPredEQ, "!=": enum.IPredNE}[e.Op]
			return g.block.NewICmp(pred, lv, rv), parser.TypeBool
		}
		if lt == parser.TypeDouble || rt == parser.TypeDouble {
			lv = g.coerce(lv, lt, parser.TypeDouble)
			rv = g.coerce(rv, rt, parser.TypeDouble)
			preds := map[string]enum.FPred{
				"<": enum.FPredOLT, ">": enum.FPredOGT,
				"<=": enum.FPredOLE, ">=": enum.FPredOGE,
				"==": enum.FPredOEQ, "!=": enum.FPredONE,
			}
			return g.block.NewFCmp(preds[e.Op], lv, rv), parser.TypeBool
		}
		preds := map[string]enum.IPred{
			"<": enum.IPredSLT, ">": enum.IPredSGT,
			"<=": enum.IPredSLE, ">=": enum.IPredSGE,
			"==": enum.IPredEQ, "!=": enum.IPredNE,
		}
		return g.block.NewICmp(preds[e.Op], lv, rv), parser.TypeBool

	case "&&":
		return g.block.NewAnd(lv, rv), parser.TypeBool

	case "||":
		return g.block.NewOr(lv, rv), parser.TypeBool

	default:
		panic(fmt.Sprintf("internal error: unknown operator %s", e.Op))
	}
}

// coerce converts a value to the expected type; the only conversion in the
// language is the implicit int to double promotion
func (g *Generator) coerce(v value.Value, from, to parser.Type) value.Value {
	if from == to {
		return v
	}
	if from == parser.TypeInt && to == parser.TypeDouble {
		return g.block.NewSIToFP(v, types.Double)
	}
	panic(fmt.Sprintf("internal error: cannot coerce %s to %s", from, to))
}

func irType(t parser.Type) types.Type {
	switch t {
	case parser.TypeInt:
		return types.I32
	case parser.TypeDouble:
		return types.Double
	case parser.TypeBool:
		return types.I1
	case parser.TypeVoid:
		return types.Void
	default:
		panic(fmt.Sprintf("internal error: no LLVM type for %s", t))
	}
}
