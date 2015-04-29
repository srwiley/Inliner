# Inliner

Inliner is a Go language pre-processor intended to be used with Go tool's 'generate' facility to inline simple functions, loops with integer counters and static literal bounds, and assertions. Benchmarks comparing inlined vs non-inlined functions can show speed increases ranging from a few fold, to up to ten fold or more. Here is the output of benchmarks from testfiles/inlined_test.go of this repository.

Benchmark1_2xLocalNotInlined	 3000000	       411 ns/op
Benchmark1_2xLocalInlined		30000000	        48.2 ns/op

Benchmark2_2xLoopedNotInlined	 1000000	      2330 ns/op
Benchmark2_2xLoopedInlined	 1000000	      1489 ns/op
Benchmark2_2xLoopedUnwound	10000000	       206 ns/op

Benchmark3_1xLoopNotUnwound	10000000	       147 ns/op
Benchmark3_1xLoopUnwound	30000000	        48.6 ns/op

Benchmark4_2xLoopNotUnwound	20000000	        79.7 ns/op
Benchmark4_2xLoopUnwound	200000000	         9.57 ns/op


Inlining a  function:

Inliner can inline both local and global functions. Inlineable functions must not return a value or define a receiver. Local functions must be declared by assignment to a variable with the ":=" token. The variable name must match the filter regular expression, which defaults to “_$”. Variables should not be declared inside the inlineable function if it is to be used more than once in a code block. Multiple function arguments are allowed. Type compatibility is not checked, but will be caught during the Go build phase. Notice that once inlined, the original function and its calls are commented out, but remain in the code.

Example: 

{Source: 
func Foo() {
	sum := 0.0
	bar_ := func(x float64) {
		sum *= x
	}
	
	bar_(1.0)
	bar_(2.0)
	bar_(3.0)
	fmt.Println("sum:", sum)
}

Inlined:
func Foo() {
	sum := 0.0
	 /* bar_ := func(x float64) {
		sum *= x
	} /* inlined func */ 

	sum *= (1.0) // inlined bar_(1.0)
	sum *= (2.0) // inlined bar_(2.0)
	sum *= (3.0) // inlined bar_(3.0)
	fmt.Println("sum:", sum)
}
}
Unwinding a static loop:

For a loop to be unwound,  it must declare an integer variable at the start of the for statement using the “:=” token with a static integer literal on the right side. The variable name must match the filter regular expression. The condition statement must be a simple "<" or "<=" token with the integer variable on the left side and a static integer literal on the right. The for statement must increment the integer variable with a "++" token.

Example:

Source:
func Foo() {
	for i_ := 0; i_ < 3; i_++ {
		fmt.Println("i:", i_)
	}
}

Inlined:
func Foo() {
	 /* for i_ := 0; i_ < 3; i_++ { /* unwound */ 
		fmt.Println("i:", (0))
		fmt.Println("i:", (1))
		fmt.Println("i:", (2)) /* } */ 
}

 Asserts:

The assertion feature works by defining two new keywords, "affirm_" and "deny_". These words take one or two arguments. The second argument, if present, must be a string, which defines the failure action. If the second argument is not present, the default failure action is “return”. Unlike the function inlining and loop unwinding feature, which will run and produce the same results whether inlined or not, asserts will not compile unless inliner processes the source file.

If first the argument is a boolean and is false ,"affirm_" will execute the failure action, if true, "deny_" will execute the failure action.

If the first argument is not a boolean and is nil, ,"affirm_" will execute the failure action, if not nil, "deny_" will execute the failure action. 


Example:

`Source:
func Foo() {
	number := "3141"
	n, err := strconv.Atoi(number)
	deny_(err)
	affirm_(n == 3141)
}

Inlined:
func Foo() {
	number := "3141"
	n, err := strconv.Atoi(number)
	/* deny_(err) /* inlined assert */
	if err != nil {
		return
	} /* */
	/* affirm_(n == 3141) /* inlined assert */
	if (n == 3141) == false {
		return
	} /* */
}`

More complex examples, tests, and benchmarks can be found in the testfiles folder. Assuming you have Go version 1.4 or later installed, you can run the examples as follows: Download the repository. Build inliner.go. Move to the testfiles directory and type:

go generate; go test -test.benchmark=”.”

Generate directives: 

Inliner is intended to work with the Go tool's generate feature introduced in Go version 1.4. In the generate directive, you must provide an input file, an output file, and, optionally, a regular expression to filter function names and loop counter variables. The default filter matches names ending with an underscore. 

A compiled version of inliner must be available either in the system PATH variable or directly referenced by the generate directive. For example, testfile/main.go expects an inliner executable in it's parent folder.

//go:generate -command inline ../inliner
//go:generate inline -out asserts_inlined.go -in asserts.go
//go:generate gofmt -w=true asserts_inlined.go

Source files intended for inlining should be prevented from being compiled by Go build by placing this directive before the package declaration:

// +build generate

About inliner :

Inliner uses the go language “ast” (abstract syntax tree) package to parse source code. It will perform multiple passes over the source code until all inlineable declarations are resolved, including nested inlineable func declarations, code blocks within an inlineable function's scope, and nested static integer loops. It does not check type compatibility between inlineable function arguments and their call statements. Any such errors will be caught during the Go build phase.

Inliner defines a “BlockOperator” type to provide a simple plugin-like architecture. Three BlockOperator types are included in this version of inliner: “functInline”, “unwindStaticLoop”, and “assertInline”. (See inliner.go.) To disable any feature, it can be removed from the BlockOperator slice passed to the “BlockVisitor”  (inliner.go , line 405) . Additional features may be added by defining new BlockVisitor types, and adding it to the BlockOperator slice on line 

