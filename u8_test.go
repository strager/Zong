package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestU8BasicVariableDeclaration(t *testing.T) {
	program := `
func main() {
	var x U8 = 42;
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

func TestU8Arithmetic(t *testing.T) {
	program := `
func main() {
	var a U8 = 10;
	var b U8 = 5;
	print(a + b);
	print(a - b);
	print(a * b);
	print(a / b);
	print(a % b);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "15\n5\n50\n2\n0\n")
}

func TestU8Comparisons(t *testing.T) {
	program := `
func main() {
	var a U8 = 10;
	var b U8 = 5;
	var c U8 = 10;
	
	if (a == c) {
		print(1);
	}
	if (a != b) {
		print(2);
	}
	if (a > b) {
		print(3);
	}
	if (b < a) {
		print(4);
	}
	if (a >= c) {
		print(5);
	}
	if (b <= a) {
		print(6);
	}
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "1\n2\n3\n4\n5\n6\n")
}

func TestU8MaxValue(t *testing.T) {
	program := `
func main() {
	var x U8 = 255;
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
	be.Equal(t, result, "255\n")
}

func TestU8MinValue(t *testing.T) {
	program := `
func main() {
	var x U8 = 0;
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
	be.Equal(t, result, "0\n")
}

func TestU8Assignment(t *testing.T) {
	program := `
func main() {
	var x U8;
	x = 123;
	print(x);
	x = 200;
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
	be.Equal(t, result, "123\n200\n")
}

func TestU8SliceSimple(t *testing.T) {
	program := `
func main() {
	var slice U8[];
	print(42);
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

func TestU8SliceDeclarationOnly(t *testing.T) {
	// Test just the slice declaration without append
	program := `
func main() {
	var slice U8[];
	print(123);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "123\n")
}

func TestI64SliceWithAppend(t *testing.T) {
	// Test I64 slice for comparison - using correct postfix & syntax
	program := `
func main() {
	var slice I64[];
	append(slice&, 10);
	print(slice[0]);
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

func TestI64SliceMultipleAppend(t *testing.T) {
	// Test I64 slice with multiple appends for comparison
	program := `
func main() {
	var slice I64[];
	append(slice&, 10);
	append(slice&, 20);
	append(slice&, 30);
	
	print(slice[0]);
	print(slice[1]);
	print(slice[2]);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n20\n30\n")
}

func TestU8SliceWithAppend(t *testing.T) {
	program := `
func main() {
	var slice U8[];
	append(slice&, 10);
	print(slice[0]);
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

func TestU8SliceMultipleAppend(t *testing.T) {
	program := `
func main() {
	var slice U8[];
	append(slice&, 10);
	append(slice&, 20);
	append(slice&, 30);
	
	print(slice[0]);
	print(slice[1]);
	print(slice[2]);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n20\n30\n")
}

func TestU8SliceMinMaxValues(t *testing.T) {
	program := `
func main() {
	var slice U8[];
	append(slice&, 0);
	append(slice&, 255);
	
	print(slice[0]);
	print(slice[1]);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "0\n255\n")
}

func TestU8SliceOutOfRange(t *testing.T) {
	// Test that values outside U8 range (0-255) are rejected
	program := `
func main() {
	var slice U8[];
	append(slice&, 256);  // This should fail
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// This should panic during compilation due to out-of-range value
	defer func() {
		if r := recover(); r != nil {
			// Expected panic
			be.True(t, true)
		} else {
			t.Error("Expected panic for out-of-range U8 value")
		}
	}()

	CompileToWASM(ast)
}
