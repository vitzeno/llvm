parser-gen:
	@# -p YY changes yacc prefix to upper case, this exports it for golang packages to access
	@echo "Generating parser..."
	@goyacc -p YY -o parser/parser.go -v parser/parser.output parser/grammar.y
	@echo "Done."

run:
	@go run main.go -in=asm/mock -print-ast

emit:
	@go run main.go -in=asm/mock -emit-llvm -o asm/mock.ll

lli: emit
	@lli asm/mock.ll

clang: emit
	@clang -Wno-override-module asm/mock.ll -o asm/mock.out

run-bin:
	@./asm/mock.out

test:
	@go test -v ./...
