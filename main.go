package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vitzeno/llvm-test/codegen"
	"github.com/vitzeno/llvm-test/parser"
)

func main() {
	sourceFile := flag.String("in", "", "input file")
	emitLLVM := flag.Bool("emit-llvm", false, "emit LLVM IR")
	output := flag.String("o", "", "output file for LLVM IR (defaults to stdout)")
	printAST := flag.Bool("print-ast", false, "print the AST")
	flag.Parse()

	if *sourceFile == "" {
		fmt.Println("input file is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	file, err := os.Open(*sourceFile)
	if err != nil {
		fmt.Printf("cannot open %s: %v\n", *sourceFile, err)
		os.Exit(1)
	}
	defer file.Close()

	parser.YYDebug = 0
	lexer := parser.NewLexer(file)
	errCount := parser.YYParse(lexer)
	for _, diag := range lexer.Errors() {
		fmt.Println(diag.String())
	}
	if errCount != 0 || len(lexer.Errors()) > 0 {
		fmt.Println("parsing failed, found error(s) in source file")
		os.Exit(1)
	}

	root := lexer.Root()
	if *printAST {
		parser.PrintAST(root, 0)
	}

	checker := parser.NewChecker()
	if diags := checker.Check(root); len(diags) > 0 {
		for i := range diags {
			fmt.Println(diags[i].String())
		}
		os.Exit(1)
	}

	if !*emitLLVM {
		return
	}

	module, err := codegen.New().Generate(root)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out := os.Stdout
	if *output != "" {
		out, err = os.Create(*output)
		if err != nil {
			fmt.Printf("cannot create %s: %v\n", *output, err)
			os.Exit(1)
		}
		defer out.Close()
	}
	if _, err := module.WriteTo(out); err != nil {
		fmt.Printf("cannot write module: %v\n", err)
		os.Exit(1)
	}
}
