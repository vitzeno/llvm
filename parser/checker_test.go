package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkSource parses and type-checks a source string, returning the
// diagnostics from both phases
func checkSource(t *testing.T, src string) []Diagnostic {
	t.Helper()
	lexer := NewLexer(strings.NewReader(src))
	parseErrs := YYParse(lexer)
	require.Zero(t, parseErrs, "parse failed: %v", lexer.Errors())
	require.Empty(t, lexer.Errors())

	checker := NewChecker()
	return checker.Check(lexer.Root())
}

func messages(diags []Diagnostic) []string {
	out := make([]string, len(diags))
	for i := range diags {
		out[i] = diags[i].Message
	}
	return out
}

func TestCheckerAccepts(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"arithmetic", `let a: int = 1 + 2 * 3;`},
		{"promotion", `let a: double = 1 + 2.5;`},
		{"int to double let", `let a: double = 4;`},
		{"comparisons", `let ok: bool = 1 < 2 && 3.5 >= 2 || 1 != 2;`},
		{"bool equality", `let ok: bool = true == false;`},
		{"if else", `let a: int = 1; if (a > 0) { print(a); } else { print(0); }`},
		{"while", `let a: int = 0; while (a < 3) { a = a + 1; }`},
		{"shadowing in nested scope", `let x: int = 1; if (x > 0) { let x: int = 2; print(x); }`},
		{"function", `func add(a: int, b: int): int { return a + b; } print(add(1, 2));`},
		{"forward reference", `print(f(1)); func f(x: int): int { return x; }`},
		{"recursion", `func fact(n: int): int { if (n <= 1) { return 1; } return n * fact(n - 1); }`},
		{"void function statement", `func show(x: int): void { print(x); } show(3);`},
		{"void explicit return", `func f(): void { return; } f();`},
		{"both branches return", `func f(x: int): int { if (x > 0) { return 1; } else { return 0; } }`},
		{"int arg promotes to double param", `func f(x: double): double { return x; } print(f(2));`},
		{"dynamic divisor allowed", `func f(x: double): double { return 1.0 / x; }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Empty(t, checkSource(t, tc.src))
		})
	}
}

func TestCheckerRejects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		src     string
		wantMsg string
	}{
		{"undefined variable", `print(z);`, "undefined variable: z"},
		{"undefined variable reassignment", `z = 1;`, "undefined variable: z"},
		{"redefinition same scope", `let a: int = 1; let a: int = 2;`, `variable "a" already defined`},
		{"param redefined at body top level", `func f(x: int): void { let x: int = 1; }`, `variable "x" already defined`},
		{"division by literal zero", `let a: int = 5 / 0;`, "division by zero"},
		{"division by parenthesised zero", `let a: double = 5.0 / (0.0);`, "division by zero"},
		{"let type mismatch", `let a: int = 1.5;`, `type mismatch: cannot assign double to int variable "a"`},
		{"bool to int", `let a: int = true;`, `type mismatch: cannot assign bool to int variable "a"`},
		{"reassignment type mismatch", `let a: int = 1; a = 2.5;`, `type mismatch: cannot assign double to int variable "a"`},
		{"non-bool if condition", `if (1) { print(1); }`, "if condition must be bool, got int"},
		{"non-bool while condition", `while (2.5) { print(1); }`, "while condition must be bool, got double"},
		{"bool arithmetic", `let a: bool = true + false;`, "operator + requires numeric operands, got bool and bool"},
		{"logical on ints", `let a: bool = 1 && 2;`, "operator && requires bool operands, got int and int"},
		{"compare bool with int", `let a: bool = true == 1;`, "cannot compare bool and int"},
		{"unary minus on bool", `let a: int = -true;`, "unary minus requires a numeric operand, got bool"},
		{"undefined function", `print(f(1));`, "undefined function: f"},
		{"wrong arity", `func f(x: int): int { return x; } print(f(1, 2));`, `wrong number of arguments to "f": expected 1, got 2`},
		{"argument type mismatch", `func f(x: int): int { return x; } print(f(1.5));`, `argument 1 to "f": cannot use double as int`},
		{"double arg to int param", `func f(x: int): int { return x; } let a: int = f(2.5);`, `argument 1 to "f": cannot use double as int`},
		{"void as value", `func f(): void { print(1); } let a: int = f();`, `type mismatch: cannot assign void to int variable "a"`},
		{"print void", `func f(): void { print(1); } print(f());`, "cannot print a void expression"},
		{"missing return", `func f(x: int): int { if (x > 0) { return 1; } }`, `function "f": not all paths return a value`},
		{"while does not guarantee return", `func f(x: int): int { while (x > 0) { return 1; } }`, `function "f": not all paths return a value`},
		{"return outside function", `return 1;`, "return outside of function"},
		{"bare return in int function", `func f(): int { return; }`, "missing return value in int function"},
		{"value return in void function", `func f(): void { return 1; }`, "void function cannot return a value"},
		{"duplicate function", `func f(): void { print(1); } func f(): void { print(2); }`, `function "f" already defined`},
		{"reserved function name", `func main(): void { print(1); }`, `function name "main" is reserved`},
		{"int literal overflow", `let a: int = 99999999999;`, "invalid number: 99999999999"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := checkSource(t, tc.src)
			assert.Contains(t, messages(diags), tc.wantMsg)
		})
	}
}

func TestCheckerShadowingIsNotRedefinition(t *testing.T) {
	src := `
let x: int = 1;
if (x > 0) {
    let x: double = 2.5;
    print(x);
}
while (x < 5) {
    let x: int = 0;
    x = x + 1;
}
print(x);
`
	assert.Empty(t, checkSource(t, src))
}
