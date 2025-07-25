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
	Init(input)
	NextToken()
	ast := ParseProgram()

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
