target triple = "x86_64-pc-linux-gnu"

@.fmt.int = constant [4 x i8] c"%d\0A\00"
@.fmt.double = constant [4 x i8] c"%g\0A\00"

declare i32 @printf(i8* %0, ...)

define i32 @square(i32 %x) {
entry:
	%x.addr = alloca i32
	store i32 %x, i32* %x.addr
	%0 = load i32, i32* %x.addr
	%1 = load i32, i32* %x.addr
	%2 = mul i32 %0, %1
	ret i32 %2
}

define i32 @main() {
entry:
	%a.1 = alloca i32
	store i32 5, i32* %a.1
	%b.2 = alloca i32
	store i32 5, i32* %b.2
	%0 = load i32, i32* %a.1
	%1 = load i32, i32* %b.2
	%2 = add i32 %0, %1
	%3 = icmp slt i32 %2, 12
	%c.4 = alloca i32
	%ratio.6 = alloca double
	br i1 %3, label %if.then.3, label %if.else.3

if.then.3:
	%4 = load i32, i32* %a.1
	%5 = getelementptr [4 x i8], [4 x i8]* @.fmt.int, i32 0, i32 0
	%6 = call i32 (i8*, ...) @printf(i8* %5, i32 %4)
	br label %if.end.3

if.else.3:
	%7 = getelementptr [4 x i8], [4 x i8]* @.fmt.int, i32 0, i32 0
	%8 = call i32 (i8*, ...) @printf(i8* %7, i32 25)
	br label %if.end.3

if.end.3:
	store i32 0, i32* %c.4
	br label %while.cond.5

while.cond.5:
	%9 = load i32, i32* %c.4
	%10 = icmp slt i32 %9, 10
	br i1 %10, label %while.body.5, label %while.end.5

while.body.5:
	%11 = load i32, i32* %c.4
	%12 = load i32, i32* %a.1
	%13 = add i32 %11, %12
	store i32 %13, i32* %c.4
	br label %while.cond.5

while.end.5:
	%14 = load i32, i32* %c.4
	%15 = getelementptr [4 x i8], [4 x i8]* @.fmt.int, i32 0, i32 0
	%16 = call i32 (i8*, ...) @printf(i8* %15, i32 %14)
	%17 = load i32, i32* %c.4
	%18 = call i32 @square(i32 %17)
	%19 = getelementptr [4 x i8], [4 x i8]* @.fmt.int, i32 0, i32 0
	%20 = call i32 (i8*, ...) @printf(i8* %19, i32 %18)
	%21 = load i32, i32* %c.4
	%22 = sitofp i32 %21 to double
	%23 = fdiv double %22, 4.0
	store double %23, double* %ratio.6
	%24 = load double, double* %ratio.6
	%25 = getelementptr [4 x i8], [4 x i8]* @.fmt.double, i32 0, i32 0
	%26 = call i32 (i8*, ...) @printf(i8* %25, double %24)
	ret i32 0
}
