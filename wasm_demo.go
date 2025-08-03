package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run wasm_test.go '<expression>'")
		os.Exit(1)
	}

	expression := os.Args[1]
	input := []byte(expression + "\x00")

	// Parse the expression
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Build symbol table for type checking
	symbolTable := BuildSymbolTable(ast)
	if symbolTable.Errors.HasErrors() {
		fmt.Println("Symbol resolution errors:")
		fmt.Println(symbolTable.Errors.String())
		os.Exit(1)
	}

	// Perform type checking to collect type errors
	typeErrors := CheckProgram(ast, symbolTable.typeTable)

	// Check for any errors (parsing or type checking)
	hasErrors := false
	if l.Errors.HasErrors() {
		fmt.Printf("Parsing errors:\n%s\n", l.Errors.String())
		hasErrors = true
	}
	if typeErrors.HasErrors() {
		fmt.Printf("Type checking errors:\n%s\n", typeErrors.String())
		hasErrors = true
	}

	if hasErrors {
		os.Exit(1)
	}

	fmt.Printf("Input: %s\n", expression)
	fmt.Printf("AST: %s\n", ToSExpr(ast))

	// Compile to WASM
	wasmBytes := CompileToWASM(ast)
	fmt.Printf("Generated %d bytes of WASM\n", len(wasmBytes))

	// Write to file for inspection
	filename := "test.wasm"
	err := os.WriteFile(filename, wasmBytes, 0644)
	if err != nil {
		fmt.Printf("Error writing WASM file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote WASM to %s\n", filename)
}
