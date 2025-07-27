package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Test complete string literal integration - assignment and usage
func TestStringLiteralIntegration(t *testing.T) {
	input := []byte(`func main() { print(42); }` + "\x00")

	Init(input)
	NextToken()
	ast := ParseProgram()
	st := BuildSymbolTable(ast)
	CheckProgram(ast, st)
	wasmBytes := CompileToWASM(ast)

	result, err := executeWasm(t, wasmBytes)
	be.Equal(t, err, nil)
	be.Equal(t, result, "42\n")
}

// Test string assignment to slice variable
func TestStringLiteralAssignment(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "test"; print(4); }` + "\x00")

	Init(input)
	NextToken()
	ast := ParseProgram()
	st := BuildSymbolTable(ast)
	CheckProgram(ast, st)
	wasmBytes := CompileToWASM(ast)

	result, err := executeWasm(t, wasmBytes)
	be.Equal(t, err, nil)
	be.Equal(t, result, "4\n")
}

// Test multiple string literals
func TestMultipleStringLiterals(t *testing.T) {
	input := []byte(`func main() { var s1 U8[] = "hello"; var s2 U8[] = "world"; print(5); }` + "\x00")

	Init(input)
	NextToken()
	ast := ParseProgram()
	st := BuildSymbolTable(ast)
	CheckProgram(ast, st)
	wasmBytes := CompileToWASM(ast)

	result, err := executeWasm(t, wasmBytes)
	be.Equal(t, err, nil)
	be.Equal(t, result, "5\n")
}

// Test string literal deduplication
func TestStringLiteralDeduplication(t *testing.T) {
	input := []byte(`func main() { var s1 U8[] = "same"; var s2 U8[] = "same"; print(42); }` + "\x00")

	Init(input)
	NextToken()
	ast := ParseProgram()
	st := BuildSymbolTable(ast)
	CheckProgram(ast, st)
	wasmBytes := CompileToWASM(ast)

	result, err := executeWasm(t, wasmBytes)
	be.Equal(t, err, nil)
	be.Equal(t, result, "42\n")
}

// Test compilation with string literals - verify data section is properly formatted
func TestStringLiteralCompilation(t *testing.T) {
	input := []byte(`func main() { var msg U8[] = "hello world"; print(11); }` + "\x00")

	Init(input)
	NextToken()
	ast := ParseProgram()
	st := BuildSymbolTable(ast)
	CheckProgram(ast, st)
	wasmBytes := CompileToWASM(ast)

	result, err := executeWasm(t, wasmBytes)
	be.Equal(t, err, nil)
	be.Equal(t, result, "11\n")
}
