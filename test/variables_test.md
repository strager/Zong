# Variables Tests

Tests for variable declarations, initialization, scoping, and shadowing.

## Variable Declarations

## Test: identifier
```zong-expr
myVar
```
```ast
(var "myVar")
```

## Test: variable declaration with type
```zong-program
var x I64;
```
```ast
[(var-decl "x" "I64")]
```

## Test: variable declaration string type
```zong-program
var name U8[];
```
```ast
[(var-decl "name" "U8[]")]
```

## Test: variable declaration custom type
```zong-program
var count MyType;
```
```ast
[(var-decl "count" "MyType")]
```

## Test: pointer variable declaration
```zong-program
var ptr I64*;
```
```ast
[(var-decl "ptr" "I64*")]
```

## Test: slice variable declaration
```zong-program
var data I64[];
```
```ast
[(var-decl "data" "I64[]")]
```

## Test: U8 variable declaration  
```zong-program
var b U8;
```
```ast
[(var-decl "b" "U8")]
```

## Test: U8 slice declaration
```zong-program
var data U8[];
```
```ast
[(var-decl "data" "U8[]")]
```

## Variable Initialization

## Test: integer initialization
```zong-program
var x I64 = 42;
```
```ast
[(var-decl "x" "I64" 42)]
```

## Test: boolean initialization
```zong-program
var flag Boolean = true;
```
```ast
[(var-decl "flag" "Boolean" true)]
```

## Test: U8 slice initialization
```zong-program
var name U8[] = "hello";
```
```ast
[(var-decl "name" "U8[]" (string "hello"))]
```

## Test: variable initialization with expression
```zong-program
var result I64 = x + y;
```
```ast
[(var-decl "result" "I64" (binary "+" (var "x") (var "y")))]
```

## Test: initialization with function call
```zong-program
var value I64 = getValue();
```
```ast
[(var-decl "value" "I64" (call (var "getValue")))]
```

## Test: initialization without semicolon
```zong-program
var count I64 = 0
```
```ast
[(var-decl "count" "I64" 0)]
```

## Test: pointer initialization
```zong-program
var ptr I64* = ptr&;
```
```ast
[(var-decl "ptr" "I64*" (unary "&" (var "ptr")))]
```

## Variable Scoping

## Variable Shadowing

## Test: variable shadowing end to end
```zong-program
func main() {
    var x I64;
    x = 10;
    print(x);
    {
        var x I64;
        x = 20;
        print(x);
    }
    print(x);
}
```
```execute
10
20
10
```

## Test: function parameter shadowing end to end
```zong-program
func test(x: I64) {
    print(x);
    {
        var x I64;
        x = 99;
        print(x);
    }
    print(x);
}

func main() {
    test(x: 42);
}
```
```execute
42
99
42
```

## Test: deep nested shadowing end to end
```zong-program
func main() {
    var x I64;
    x = 1;
    print(x);
    {
        var x I64;
        x = 2;
        print(x);
        {
            var x I64;
            x = 3;
            print(x);
            {
                var x I64;
                x = 4;
                print(x);
            }
            print(x);
        }
        print(x);
    }
    print(x);
}
```
```execute
1
2
3
4
3
2
1
```

## Test: shadowing with different types
```zong-program
func main() {
    var x I64;
    x = 42;
    print(x);
    {
        var x Boolean;
        x = true;
        print(x);
    }
    print(x);
}
```
```execute
42
1
42
```

## WASM Local Variable Tests (from locals_test.md)

## Test: single local variable
```zong-program
func main() { var x I64; }
```
```wasm-locals
[(local "x" "I64" local 0)]
```

## Test: multiple local variables
```zong-program
func main() { var x I64; var y I64; }
```
```wasm-locals
[(local "x" "I64" local 0) (local "y" "I64" local 1)]
```

## Test: nested block variables
```zong-program
func main() { var a I64; { var b I64; } }
```
```wasm-locals
[(local "a" "I64" local 0) (local "b" "I64" local 1)]
```

## Test: no variables
```zong-program
func main() { print(42); }
```
```wasm-locals
[]
```

## Test: single pointer variable
```zong-program
func main() { var ptr I64*; }
```
```wasm-locals
[(local "ptr" "I64*" local 0)]
```

## Test: addressed single variable
```zong-program
func main() { var x I64; print(x&); }
```
```wasm-locals
[(local "x" "I64" tstack 0)]
```

## Variable Execution Tests (from extracted_execution_test.md)

## Test: basic variable initialization
```zong-program
func main() {
	var x I64 = 42;
	print(x);
}
```
```execute
42
```

## Test: boolean variable initialization
```zong-program
func main() {
	var flag Boolean = true;
	print(flag);
	var flag2 Boolean = false;
	print(flag2);
}
```
```execute
1
0
```

## Test: multiple variable initialization
```zong-program
func main() {
	var x I64 = 10;
	var y I64 = 20;
	var z I64 = x + y;
	print(z);
}
```
```execute
30
```

## Test: mixed initialized and uninitialized vars
```zong-program
func main() {
	var x I64 = 5;
	var y I64;
	y = x * 2;
	print(y);
}
```
```execute
10
```

## Test: variable initialization with expressions
```zong-program
func main() {
	var a I64 = 3;
	var b I64 = 4;
	var hypotenuse I64 = a * a + b * b;
	print(hypotenuse);
}
```
```execute
25
```

## Test: equivalence with separate assignment
```zong-program
func main() {
	var x I64 = 5;
	var y I64 = x * 2;
	print(y);
}
```
```execute
10
```

## Test: pointer variable initialization
```zong-program
func main() {
	var x I64 = 42;
	var ptr I64* = x&;
	print(ptr*);
}
```
```execute
42
```

## Variable TypeAST Tests (from TestTypeASTInCompilation_test.md)

## Test: I64 variable with TypeAST
```zong-program
{ var x I64; x = 42; print(x); }
```
```execute
42
```

## Test: Second I64 variable with TypeAST
```zong-program
{ var y I64; y = 7; print(y); }
```
```execute
7
```

## Test: Multiple types with TypeAST
```zong-program
{ var x I64; var y I64; var ptr I64*; x = 10; y = 0; ptr = x&; print(x); print(y); print(ptr*); }
```
```execute
10
0
10
```

## Additional Variable Execution Tests (from execution_test.md)

## Test: full program execution
```zong-program
func main() {
    var x I64 = 10;
    print(x * 2);
}
```
```execute
20
```

## Test: no output expected
```zong-program
func main() {
    var x I64 = 10;
    x = x * 2;
}
```
```execute

```

## Integration Variable Tests (from more_test.md)

## Test: integration complex variable calculations
```zong-program
{ var x I64; var y I64; var result I64; x = 15; y = 3; result = x * y + 5; print(result); }
```
```execute
50
```

## Test: integration comprehensive demo
```zong-program
{
	var a I64;
	var b I64;
	var temp I64;
	var final I64;

	a = 8;
	b = 3;
	temp = a * b;        // temp = 24
	final = temp + a - b; // final = 24 + 8 - 3 = 29
	print(final);
}
```
```execute
29
```

## Test: integration mixed types
```zong-program
{ var x I64; var y Boolean; x = 42; y = true; print(x); }
```
```execute
42
```

## Test: integration nested variable scoping
```zong-program
{ var x I64; x = 42; { var y I64; y = x; print(y); } }
```
```execute
42
```

## Test: integration variable reassignment
```zong-program
{ var counter I64; counter = 5; counter = counter + 10; print(counter); }
```
```execute
15
```

## Test: integration variables in expressions
```zong-program
{ var a I64; var b I64; a = 10; b = 20; print(a + b); }
```
```execute
30
```

## Variable Compile Error Tests (from compile_error_test.md)

## Test: undefined variable error
```zong-expr
undefinedVar
```
```compile-error
error: undefined symbol 'undefinedVar'
```

## Test: variable used before assignment
```zong-program
func main() {
    var x I64;
    var y Boolean;
    x = y;
}
```
```compile-error
error: variable 'y' used before assignment
```

## Test: variable not assigned in expression  
```zong-program
func main() {
    var x I64;
    print(x);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: assignment to undeclared variable
```zong-program
func main() {
    undefined = 42;
}
```
```compile-error
error: undefined symbol 'undefined'
```

## Test: program with variable used before assignment
```zong-program
func main() {
    var x I64;
    print(x);
}
```
```compile-error
error: variable 'x' used before assignment
```

## Test: duplicate variable declaration
```zong-program
func main() {
    var x I64;
    var x I64;
}
```
```compile-error
error: variable 'x' already declared
```

## Test: assignment to undefined variable
```zong-program
func main() {
    undefinedVar = 42;
}
```
```compile-error
error: undefined symbol 'undefinedVar'
```

## Test: address of undefined variable
```zong-program
func main() {
    var ptr I64* = undefinedVar&;
}
```
```compile-error
error: undefined symbol 'undefinedVar'
```

## Variable Parser Robustness Tests (from parser_robustness_test.md)

## Test: var statement without variable name
```zong-expr
var ;
```
```compile-error

```

## Test: var statement without type
```zong-expr
var x ;
```
```compile-error

```

## Test: var statement with invalid type
```zong-expr
var x 123;
```
```compile-error

```

## Comprehensive Variable Expression Tests (from parsing_comprehensive_test.md)

## Test: expression statement with variable
```zong-program
x;
```
```ast
[(var "x")]
```
