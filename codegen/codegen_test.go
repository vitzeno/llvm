package codegen

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vitzeno/llvm-test/parser"
)

// compileSource runs the full parse, check, generate pipeline over a source
// string and returns the LLVM IR as text
func compileSource(t *testing.T, src string) string {
	t.Helper()

	lexer := parser.NewLexer(strings.NewReader(src))
	parseErrs := parser.YYParse(lexer)
	require.Zero(t, parseErrs, "parse failed: %v", lexer.Errors())
	require.Empty(t, lexer.Errors())

	checker := parser.NewChecker()
	diags := checker.Check(lexer.Root())
	require.Empty(t, diags, "checker rejected program")

	module, err := New().Generate(lexer.Root())
	require.NoError(t, err)
	return module.String()
}

// runSource compiles a source string and executes the generated IR under
// lli, returning the program's stdout. A timeout guards against broken
// control flow hanging the suite.
func runSource(t *testing.T, src string) string {
	t.Helper()

	if _, err := exec.LookPath("lli"); err != nil {
		t.Skip("lli not found on PATH, skipping execution test")
	}

	llPath := filepath.Join(t.TempDir(), "prog.ll")
	require.NoError(t, os.WriteFile(llPath, []byte(compileSource(t, src)), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "lli", llPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	require.NoError(t, ctx.Err(), "program timed out; generated control flow is likely broken")
	require.NoError(t, err, "lli failed: %s", stderr.String())
	return stdout.String()
}

func TestExecution(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{
			name: "int arithmetic",
			src:  `print(2 + 3 * 4);`,
			want: "14\n",
		},
		{
			name: "double arithmetic",
			src:  `print(1.5 * 2.0);`,
			want: "3\n",
		},
		{
			name: "mixed promotion",
			src:  `let a: int = 2; let b: double = 1.5; print(a + b);`,
			want: "3.5\n",
		},
		{
			name: "unary minus",
			src:  `print(-3); print(-(1.5));`,
			want: "-3\n-1.5\n",
		},
		{
			name: "integer division truncates",
			src:  `print(7 / 2);`,
			want: "3\n",
		},
		{
			name: "double division",
			src:  `print(7.0 / 2.0);`,
			want: "3.5\n",
		},
		{
			name: "if taken",
			src:  `let a: int = 4; if (a > 2) { print(1); } else { print(2); } print(3);`,
			want: "1\n3\n",
		},
		{
			name: "else taken",
			src:  `let a: int = 1; if (a > 2) { print(1); } else { print(2); } print(3);`,
			want: "2\n3\n",
		},
		{
			name: "if without else continues after",
			src:  `let a: int = 1; if (a > 2) { print(1); } print(3);`,
			want: "3\n",
		},
		{
			name: "while loops until condition fails",
			src:  `let a: int = 0; while (a < 5) { a = a + 1; } print(a);`,
			want: "5\n",
		},
		{
			name: "while body with multiple statements",
			src: `let a: int = 0;
let sum: int = 0;
while (a < 4) {
    sum = sum + a;
    a = a + 1;
}
print(sum);`,
			want: "6\n",
		},
		{
			name: "while never entered",
			src:  `let a: int = 9; while (a < 5) { a = a + 1; } print(a);`,
			want: "9\n",
		},
		{
			name: "nested while",
			src: `let i: int = 0;
let total: int = 0;
while (i < 3) {
    let j: int = 0;
    while (j < 3) {
        total = total + 1;
        j = j + 1;
    }
    i = i + 1;
}
print(total);`,
			want: "9\n",
		},
		{
			name: "function call",
			src:  `func add(a: int, b: int): int { return a + b; } print(add(2, 3));`,
			want: "5\n",
		},
		{
			name: "call before declaration",
			src:  `print(triple(4)); func triple(x: int): int { return 3 * x; }`,
			want: "12\n",
		},
		{
			name: "recursion",
			src: `func fact(n: int): int {
    if (n <= 1) {
        return 1;
    }
    return n * fact(n - 1);
}
print(fact(6));`,
			want: "720\n",
		},
		{
			name: "void function side effects",
			src:  `func show(x: double): void { print(x); } show(2.5); show(4);`,
			want: "2.5\n4\n",
		},
		{
			name: "early return in void function",
			src: `func f(x: int): void {
    if (x > 0) {
        print(1);
        return;
    }
    print(2);
}
f(5);
f(-5);`,
			want: "1\n2\n",
		},
		{
			name: "both branches return",
			src: `func sign(x: int): int {
    if (x >= 0) {
        return 1;
    } else {
        return -1;
    }
}
print(sign(3));
print(sign(-3));`,
			want: "1\n-1\n",
		},
		{
			name: "shadowing",
			src: `let x: int = 1;
if (x > 0) {
    let x: int = 2;
    print(x);
}
print(x);`,
			want: "2\n1\n",
		},
		{
			name: "bool printing and logic",
			src:  `let ok: bool = 2 < 3; print(ok); print(ok && false); print(ok || false); print(2 == 3);`,
			want: "1\n0\n1\n0\n",
		},
		{
			name: "comparison operators",
			src:  `print(2 <= 2); print(3 >= 4); print(1 != 2); print(1.5 < 2.5);`,
			want: "1\n0\n1\n1\n",
		},
		{
			name: "int argument promotes to double parameter",
			src:  `func half(x: double): double { return x / 2.0; } print(half(7));`,
			want: "3.5\n",
		},
		{
			name: "double division by dynamic zero is IEEE",
			src:  `func div(x: double, y: double): double { return x / y; } print(div(1.0, 0.0));`,
			want: "inf\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, runSource(t, tc.src))
		})
	}
}

// TestClangCompatibility proves the emitted IR also survives a real
// compile-and-link, not just the lli interpreter
func TestClangCompatibility(t *testing.T) {
	if _, err := exec.LookPath("clang"); err != nil {
		t.Skip("clang not found on PATH, skipping")
	}

	src := `func fib(n: int): int {
    if (n < 2) {
        return n;
    }
    return fib(n - 1) + fib(n - 2);
}
print(fib(10));`

	dir := t.TempDir()
	llPath := filepath.Join(dir, "prog.ll")
	binPath := filepath.Join(dir, "prog")
	require.NoError(t, os.WriteFile(llPath, []byte(compileSource(t, src)), 0o644))

	out, err := exec.Command("clang", "-Wno-override-module", llPath, "-o", binPath).CombinedOutput()
	require.NoError(t, err, "clang failed: %s", out)

	got, err := exec.Command(binPath).Output()
	require.NoError(t, err)
	assert.Equal(t, "55\n", string(got))
}

// TestGeneratedIRIsValid runs every source fixture through opt's verifier
// when available
func TestGeneratedIRIsValid(t *testing.T) {
	if _, err := exec.LookPath("opt"); err != nil {
		t.Skip("opt not found on PATH, skipping")
	}

	fixtures, err := filepath.Glob("../testdata/*.test")
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	for _, fixture := range fixtures {
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			src, err := os.ReadFile(fixture)
			require.NoError(t, err)

			llPath := filepath.Join(t.TempDir(), "prog.ll")
			require.NoError(t, os.WriteFile(llPath, []byte(compileSource(t, string(src))), 0o644))

			out, err := exec.Command("opt", "-passes=verify", "-S", llPath, "-o", os.DevNull).CombinedOutput()
			require.NoError(t, err, "verifier rejected IR: %s", out)
		})
	}
}
