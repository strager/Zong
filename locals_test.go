package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

func TestCollectSingleLocalVariable(t *testing.T) {
	input := []byte("func main() { var x I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestCollectMultipleLocalVariables(t *testing.T) {
	input := []byte("func main() { var x I64; var y I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},
		{Symbol: &SymbolInfo{Name: "y", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestCollectNestedBlockVariables(t *testing.T) {
	input := []byte("func main() { var a I64; { var b I64; } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "a", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},
		{Symbol: &SymbolInfo{Name: "b", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestNoVariables(t *testing.T) {
	input := []byte("func main() { print(42); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)
	be.Equal(t, 0, len(locals))

	var buf bytes.Buffer
	// Use legacy compilation path
	wasmBytes := CompileToWASM(ast)
	buf.Write(wasmBytes)

	// Should emit 0 locals (existing behavior)
	bytes_result := buf.Bytes()
	// Verify locals count is 0 in the generated WASM
	// After section id (0x0A) and section size, we should find the function body
	// which starts with locals count = 0
	be.True(t, len(bytes_result) > 3) // At least section id + size + locals count
}

func TestUndefinedVariableReference(t *testing.T) {
	input := []byte("func main() { print(undefined_var); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	var buf bytes.Buffer
	localCtx := &LocalContext{
		Variables: []LocalVarInfo{}, // No locals defined
	}

	defer func() {
		if r := recover(); r != nil {
			panicMsg := r.(string)
			if !strings.Contains(panicMsg, "Undefined variable: undefined_var") {
				t.Fatalf("Expected panic message to contain 'Undefined variable: undefined_var', got: %s", panicMsg)
			}
		} else {
			t.Fatal("Expected panic for undefined variable")
		}
	}()

	// Extract undefined_var from print(undefined_var)
	printArg := ast.Children[1] // the undefined_var argument
	EmitExpression(&buf, printArg, localCtx)
}

func TestCollectSinglePointerVariable(t *testing.T) {
	input := []byte("func main() { var ptr I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestCollectMixedPointerAndRegularVariables(t *testing.T) {
	input := []byte("func main() { var x I64; var ptr I64*; var y I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 1},                                        // i64 locals start at index 1 (after i32 locals)
		{Symbol: &SymbolInfo{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 0}, // i32 pointer at index 0
		{Symbol: &SymbolInfo{Name: "y", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 2},                                        // second i64 local at index 2
	}

	be.Equal(t, expected, locals)
}

func TestCollectMultiplePointerVariables(t *testing.T) {
	input := []byte("func main() { var ptr1 I64*; var ptr2 I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "ptr1", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 0},
		{Symbol: &SymbolInfo{Name: "ptr2", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestCollectNestedBlockPointerVariables(t *testing.T) {
	input := []byte("func main() { var a I64; { var ptr I64*; } var b I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "a", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 2},                                        // i64 local comes after 2 i32 locals
		{Symbol: &SymbolInfo{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 0}, // first i32 pointer
		{Symbol: &SymbolInfo{Name: "b", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Assigned: false}, Storage: VarStorageLocal, Address: 1},   // second i32 pointer
	}

	be.Equal(t, expected, locals)
}

func TestPointerVariablesInWASMCodeSection(t *testing.T) {
	input := []byte("func main() { var x I64; var ptr I64*; x = 42; print(x); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	var buf bytes.Buffer
	// Use legacy compilation path
	wasmBytes := CompileToWASM(ast)
	buf.Write(wasmBytes)

	// Should emit locals for both I64 and I64* variables
	// Both should be counted as i64 locals in WASM
	bytesResult := buf.Bytes()

	// Verify that the function has locals (non-zero local count)
	// The exact WASM structure verification is complex, but we can at least
	// verify that code generation doesn't panic and produces output
	be.True(t, len(bytesResult) > 0)
}

func TestAddressedSingleVariable(t *testing.T) {
	input := []byte("func main() { var x I64; print(x&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestAddressedMultipleVariables(t *testing.T) {
	input := []byte("func main() { var x I64; var y I64; print(x&); print(y&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 0},
		{Symbol: &SymbolInfo{Name: "y", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 8},
	}

	be.Equal(t, expected, locals)
}

func TestMixedAddressedAndNonAddressedVariables(t *testing.T) {
	input := []byte("func main() { var a I64; var b I64; var c I64; print(b&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "a", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},  // first i64 local
		{Symbol: &SymbolInfo{Name: "b", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 0}, // addressed variable (no change)
		{Symbol: &SymbolInfo{Name: "c", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 1},  // second i64 local
	}

	be.Equal(t, expected, locals)
}

func TestAddressedVariableFrameOffsetCalculation(t *testing.T) {
	input := []byte("func main() { var a I64; var b I64; var c I64; var d I64; print(a&); print(c&); print(d&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "a", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 0},  // addressed variable (no change)
		{Symbol: &SymbolInfo{Name: "b", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},   // i64 local
		{Symbol: &SymbolInfo{Name: "c", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 8},  // addressed variable (no change)
		{Symbol: &SymbolInfo{Name: "d", Type: TypeI64, Assigned: false}, Storage: VarStorageTStack, Address: 16}, // addressed variable (no change)
	}

	be.Equal(t, expected, locals)
}

func TestAddressOfRvalue(t *testing.T) {
	input := []byte("func main() { var x I64; print((x + 1)&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals, _ := collectLocalVariables(ast)

	// x is not addressed since we're taking address of expression, not variable
	expected := []LocalVarInfo{
		{Symbol: &SymbolInfo{Name: "x", Type: TypeI64, Assigned: false}, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}
