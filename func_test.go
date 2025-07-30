package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// TestFunctionDeclarationParsing removed - duplicates test/functions_test.md

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
	source := `func add(_ addA2: I64, _ addB2: I64): I64 {
		return addA2 + addB2;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test simple main function first

// Test end-to-end function execution with main

// Parse multiple functions

// Execute and verify output

// Test multiple functions with various signatures

// Test void function (no return value)

// Test function with complex expressions

// (3+4)*5-10 = 35-10 = 25

// Test nested function calls

// (2*3) + (4*5) = 6 + 20 = 26

// Phase 2 Tests: Named Parameters

// TestNamedParameterParsing removed - duplicates test/functions_test.md

// Test named parameter function calls

// Test mixed positional and named parameters

// 5 * 3 + 10 = 25

// Test function that returns a struct

// Test function return field access (e.g., make_point(x: 1, y: 2).x)

// This should now work with the fixed implementation

// Test both variable field access and function return field access in one program

// Test nested struct definitions and field access

// Test nested struct initialization

// Test function returning nested struct
