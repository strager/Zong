package main

import (
	"github.com/nalgeon/be"
	"testing"
)

// Test function declaration parsing
func TestFunctionDeclarationParsing(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "void function no parameters",
			source:   "func test() {}",
			expected: `(func "test" () void (block))`,
		},
		{
			name:     "function with I64 return type",
			source:   "func add(): I64 {}",
			expected: `(func "add" () "I64" (block))`,
		},
		{
			name:     "function with positional parameters",
			source:   "func add(_ a: I64, _ b: I64): I64 {}",
			expected: `(func "add" ((param "a" "I64" positional) (param "b" "I64" positional)) "I64" (block))`,
		},
		{
			name:     "function with named parameters",
			source:   "func test(x: I64, y: I64) {}",
			expected: `(func "test" ((param "x" "I64" named) (param "y" "I64" named)) void (block))`,
		},
		{
			name:     "function with body",
			source:   "func test() { var x I64; }",
			expected: `(func "test" () void (block (var (ident "x") (ident "I64"))))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()

			result := ToSExpr(ast)
			be.Equal(t, result, tt.expected)
		})
	}
}

// Test basic function compilation
func TestBasicFunctionCompilation(t *testing.T) {
	source := `func test(): I64 {
		return 42;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement() // Parse single function

	// Test WASM compilation doesn't panic
	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test simple function with parameters
func TestFunctionWithParameters(t *testing.T) {
	source := `func add(_ a: I64, _ b: I64): I64 {
		return a + b;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test simple main function first
func TestSimpleMain(t *testing.T) {
	source := `func main() {
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)

	executeWasmAndVerify(t, wasmBytes, "42\n")
}

// Test end-to-end function execution with main
func TestEndToEndFunctionExecution(t *testing.T) {
	source := `func add(_ a: I64, _ b: I64): I64 {
		return a + b;
	}
	
	func main() {
		print(add(5, 3));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram() // Parse multiple functions

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)

	// Execute and verify output
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "8\n")
}

// Test multiple functions with various signatures
func TestMultipleFunctions(t *testing.T) {
	source := `func double(_ x: I64): I64 {
		return x * 2;
	}
	
	func triple(_ x: I64): I64 {
		return x * 3;
	}
	
	func main() {
		print(double(5));
		print(triple(4));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "10\n12\n")
}

// Test void function (no return value)
func TestVoidFunction(t *testing.T) {
	source := `func printTwice(_ x: I64) {
		print(x);
		print(x);
	}
	
	func main() {
		printTwice(7);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "7\n7\n")
}

// Test function with complex expressions
func TestFunctionWithComplexExpressions(t *testing.T) {
	source := `func compute(_ a: I64, _ b: I64, _ c: I64): I64 {
		return (a + b) * c - 10;
	}
	
	func main() {
		print(compute(3, 4, 5));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "25\n") // (3+4)*5-10 = 35-10 = 25
}

// Test nested function calls
func TestNestedFunctionCalls(t *testing.T) {
	source := `func add(_ a: I64, _ b: I64): I64 {
		return a + b;
	}
	
	func multiply(_ a: I64, _ b: I64): I64 {
		return a * b;
	}
	
	func main() {
		print(add(multiply(2, 3), multiply(4, 5)));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "26\n") // (2*3) + (4*5) = 6 + 20 = 26
}

// Phase 2 Tests: Named Parameters

// Test named parameters parsing
func TestNamedParameterParsing(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "function with named parameters",
			source:   "func test(x: I64, y: I64): I64 {}",
			expected: `(func "test" ((param "x" "I64" named) (param "y" "I64" named)) "I64" (block))`,
		},
		{
			name:     "mixed positional and named parameters",
			source:   "func test(_ a: I64, x: I64, y: I64): I64 {}",
			expected: `(func "test" ((param "a" "I64" positional) (param "x" "I64" named) (param "y" "I64" named)) "I64" (block))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()

			result := ToSExpr(ast)
			be.Equal(t, result, tt.expected)
		})
	}
}

// Test named parameter function calls
func TestNamedParameterCalls(t *testing.T) {
	source := `func greet(name: I64, age: I64) {
		print(name);
		print(age);
	}
	
	func main() {
		greet(name: 42, age: 25);
		greet(age: 30, name: 50); 
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "42\n25\n50\n30\n")
}

// Test mixed positional and named parameters
func TestMixedParameters(t *testing.T) {
	source := `func compute(_ base: I64, multiplier: I64, offset: I64): I64 {
		return base * multiplier + offset;
	}
	
	func main() {
		print(compute(5, multiplier: 3, offset: 10));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "25\n") // 5 * 3 + 10 = 25
}

// Test function that returns a struct
func TestFunctionReturningStruct(t *testing.T) {
	source := `struct Point { var x I64; var y I64; }
	
	func createPoint(_ xVal: I64, _ yVal: I64): Point {
		var p Point;
		p.x = xVal;
		p.y = yVal;
		return p;
	}
	
	func main() {
		var result Point;
		result = createPoint(10, 20);
		print(result.x);
		print(result.y);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "10\n20\n")
}

func TestFunctionStructParamCopies(t *testing.T) {
	source := `
		struct S { var i I64; }

		func f(_ s: S) {
			s.i = 3;
			print(s.i);
		}

		func main() {
			var ss S;
			ss.i = 2;
			print(ss.i);
			f(ss);
			print(ss.i);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "2\n3\n2\n")
}
