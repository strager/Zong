package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Phase 3 Tests: Advanced Returns

// Test I64* return types

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

// Test struct parameter parsing
func TestStructParameterParsing(t *testing.T) {
	source := `func test(_ testP: Point): I64 { return 42; }`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	expected := `(func "test" ((param "testP" "Point*" positional)) "I64" (block (return (integer 42))))`
	result := ToSExpr(ast)
	be.Equal(t, result, expected)
}
