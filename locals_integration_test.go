package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestIntegrationVariablesInExpressions(t *testing.T) {
	// Test that variables work in expressions
	input := []byte("{ var a I64; var b I64; a = 10; b = 20; print(a + b); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Check locals collection
	locals := collectLocalVariables(ast)
	be.Equal(t, 2, len(locals))
	be.Equal(t, "a", locals[0].Name)
	be.Equal(t, "b", locals[1].Name)

	// Compile and execute WASM
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "30\n") // 10 + 20 = 30
}

func TestIntegrationNestedVariableScoping(t *testing.T) {
	// Test nested blocks with variables (WebAssembly has function-level scope)
	input := []byte("{ var x I64; x = 42; { var y I64; y = x; print(y); } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)
	be.Equal(t, 2, len(locals))

	// Both variables should be available at function level
	be.Equal(t, "x", locals[0].Name)
	be.Equal(t, "y", locals[1].Name)

	// Compile and execute WASM - should print the value of y (which was assigned from x)
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "42\n") // y = x = 42
}

func TestIntegrationMixedTypes(t *testing.T) {
	// Test that non-I64 types are ignored (as per the plan)
	input := []byte("{ var x I64; var y string; x = 42; print(x); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)
	// Only I64 variable should be collected
	be.Equal(t, 1, len(locals))
	be.Equal(t, "x", locals[0].Name)
	be.Equal(t, "I64", locals[0].Type)

	// Compile and execute WASM - should print the value of x
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "42\n") // x = 42
}

func TestIntegrationComplexVariableCalculations(t *testing.T) {
	// Test complex calculations with multiple variables
	input := []byte("{ var x I64; var y I64; var result I64; x = 15; y = 3; result = x * y + 5; print(result); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)
	be.Equal(t, 3, len(locals))

	// Compile and execute WASM - should calculate 15 * 3 + 5 = 50
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "50\n")
}

func TestIntegrationVariableReassignment(t *testing.T) {
	// Test variable reassignment
	input := []byte("{ var counter I64; counter = 5; counter = counter + 10; print(counter); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)
	be.Equal(t, 1, len(locals))
	be.Equal(t, "counter", locals[0].Name)

	// Compile and execute WASM - should calculate 5 + 10 = 15
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "15\n")
}

func TestIntegrationComprehensiveDemo(t *testing.T) {
	// Comprehensive test showing all local variable features working together
	input := []byte(`{
		var a I64;
		var b I64;
		var temp I64;
		var final I64;

		a = 8;
		b = 3;
		temp = a * b;        // temp = 24
		final = temp + a - b; // final = 24 + 8 - 3 = 29
		print(final);
	}` + "\x00")

	Init(input)
	NextToken()
	ast := ParseStatement()

	// Verify all variables are collected
	locals := collectLocalVariables(ast)
	be.Equal(t, 4, len(locals))

	expectedNames := []string{"a", "b", "temp", "final"}
	for i, local := range locals {
		be.Equal(t, expectedNames[i], local.Name)
		be.Equal(t, "I64", local.Type)
		be.Equal(t, uint32(i), local.Index)
	}

	// Execute and verify the complex calculation
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "29\n") // (8 * 3) + 8 - 3 = 24 + 8 - 3 = 29
}
