package main

import (
	"github.com/nalgeon/be"
	"testing"
)

// Phase 3 Tests: Advanced Returns

// Test I64* return types
func TestI64PointerReturns(t *testing.T) {
	source := `func getPointer(): I64* {
		var x I64;
		x = 42;
		return x&;
	}
	
	func main() {
		var ptr I64*;
		ptr = getPointer();
		print(ptr*);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n")
}

// Test parsing I64* return type
func TestI64PointerReturnParsing(t *testing.T) {
	source := `func getPointer(): I64* {
		return null;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	expected := `(func "getPointer" () "I64*" (block (return (ident "null"))))`
	result := ToSExpr(ast)
	be.Equal(t, result, expected)
}

// Test struct parameter passing by copy
func TestStructParameterPassing(t *testing.T) {
	source := `struct Point { var x I64; var y I64; }
	
	func processPoint(_ p: Point) {
		print(p.x);
		print(p.y);
	}
	
	func main() {
		var p Point;
		p.x = 10;
		p.y = 20;
		processPoint(p);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "10\n20\n")
}

// Test struct parameter parsing
func TestStructParameterParsing(t *testing.T) {
	source := `func test(_ p: Point): I64 { return 42; }`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	expected := `(func "test" ((param "p" "Point*" positional)) "I64" (block (return (integer 42))))`
	result := ToSExpr(ast)
	be.Equal(t, result, expected)
}
