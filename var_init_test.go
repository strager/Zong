package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Tests for variable declaration with initialization syntax

func TestBasicVariableInitialization(t *testing.T) {
	program := `
func main() {
	var x I64 = 42;
	print(x);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "42\n")
}

func TestMultipleVariableInitialization(t *testing.T) {
	program := `
func main() {
	var x I64 = 10;
	var y I64 = 20;
	var z I64 = x + y;
	print(z);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "30\n")
}

func TestMixedInitializedAndUninitializedVars(t *testing.T) {
	program := `
func main() {
	var x I64 = 5;
	var y I64;
	y = x * 2;
	print(y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n")
}

func TestBooleanVariableInitialization(t *testing.T) {
	program := `
func main() {
	var flag Boolean = true;
	print(flag);
	var flag2 Boolean = false;
	print(flag2);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "1\n0\n")
}

func TestVariableInitializationWithExpressions(t *testing.T) {
	program := `
func main() {
	var a I64 = 3;
	var b I64 = 4;
	var hypotenuse I64 = a * a + b * b;
	print(hypotenuse);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "25\n")
}

func TestPointerVariableInitialization(t *testing.T) {
	program := `
func main() {
	var x I64 = 42;
	var ptr I64* = x&;
	print(ptr*);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "42\n")
}

func TestEquivalenceWithSeparateAssignment(t *testing.T) {
	// Test that 'var x I64 = 5;' is equivalent to 'var x I64; x = 5;'
	program1 := `
func main() {
	var x I64 = 5;
	var y I64 = x * 2;
	print(y);
}
`
	program2 := `
func main() {
	var x I64;
	x = 5;
	var y I64;
	y = x * 2;
	print(y);
}
`
	// Test program1
	input1 := []byte(program1 + "\x00")
	Init(input1)
	NextToken()
	ast1 := ParseProgram()
	wasmBytes1 := CompileToWASM(ast1)
	result1, err1 := executeWasm(t, wasmBytes1)
	be.Err(t, err1, nil)

	// Test program2
	input2 := []byte(program2 + "\x00")
	Init(input2)
	NextToken()
	ast2 := ParseProgram()
	wasmBytes2 := CompileToWASM(ast2)
	result2, err2 := executeWasm(t, wasmBytes2)
	be.Err(t, err2, nil)

	// Both should produce the same result
	be.Equal(t, result1, result2)
	be.Equal(t, result1, "10\n")
}
