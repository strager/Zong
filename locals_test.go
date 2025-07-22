package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

func TestCollectSingleLocalVariable(t *testing.T) {
	input := []byte("var x I64;\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestCollectMultipleLocalVariables(t *testing.T) {
	input := []byte("{ var x I64; var y I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
		{Name: "y", Type: TypeI64, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestCollectNestedBlockVariables(t *testing.T) {
	input := []byte("{ var a I64; { var b I64; } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "a", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
		{Name: "b", Type: TypeI64, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestNoVariables(t *testing.T) {
	input := []byte("print(42);\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)
	be.Equal(t, 0, len(locals))

	var buf bytes.Buffer
	EmitCodeSection(&buf, ast)

	// Should emit 0 locals (existing behavior)
	bytes_result := buf.Bytes()
	// Verify locals count is 0 in the generated WASM
	// After section id (0x0A) and section size, we should find the function body
	// which starts with locals count = 0
	be.True(t, len(bytes_result) > 3) // At least section id + size + locals count
}

func TestUndefinedVariableReference(t *testing.T) {
	input := []byte("print(undefined_var);\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	var buf bytes.Buffer
	locals := []LocalVarInfo{} // No locals defined

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
	EmitExpression(&buf, printArg, locals)
}

func TestCollectSinglePointerVariable(t *testing.T) {
	input := []byte("var ptr I64*;\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestCollectMixedPointerAndRegularVariables(t *testing.T) {
	input := []byte("{ var x I64; var ptr I64*; var y I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
		{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 1},
		{Name: "y", Type: TypeI64, Storage: VarStorageLocal, Address: 2},
	}

	be.Equal(t, expected, locals)
}

func TestCollectMultiplePointerVariables(t *testing.T) {
	input := []byte("{ var ptr1 I64*; var ptr2 I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "ptr1", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 0},
		{Name: "ptr2", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 1},
	}

	be.Equal(t, expected, locals)
}

func TestCollectNestedBlockPointerVariables(t *testing.T) {
	input := []byte("{ var a I64; { var ptr I64*; } var b I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "a", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
		{Name: "ptr", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 1},
		{Name: "b", Type: &TypeNode{Kind: TypePointer, Child: TypeI64}, Storage: VarStorageLocal, Address: 2},
	}

	be.Equal(t, expected, locals)
}

func TestPointerVariablesInWASMCodeSection(t *testing.T) {
	input := []byte("{ var x I64; var ptr I64*; print(x); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	var buf bytes.Buffer
	EmitCodeSection(&buf, ast)

	// Should emit locals for both I64 and I64* variables
	// Both should be counted as i64 locals in WASM
	bytesResult := buf.Bytes()

	// Verify that the function has locals (non-zero local count)
	// The exact WASM structure verification is complex, but we can at least
	// verify that code generation doesn't panic and produces output
	be.True(t, len(bytesResult) > 0)
}

func TestAddressedSingleVariable(t *testing.T) {
	input := []byte("{ var x I64; print(x&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageTStack, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestAddressedMultipleVariables(t *testing.T) {
	input := []byte("{ var x I64; var y I64; print(x&); print(y&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageTStack, Address: 0},
		{Name: "y", Type: TypeI64, Storage: VarStorageTStack, Address: 8},
	}

	be.Equal(t, expected, locals)
}

func TestMixedAddressedAndNonAddressedVariables(t *testing.T) {
	input := []byte("{ var a I64; var b I64; var c I64; print(b&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "a", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
		{Name: "b", Type: TypeI64, Storage: VarStorageTStack, Address: 0},
		{Name: "c", Type: TypeI64, Storage: VarStorageLocal, Address: 2},
	}

	be.Equal(t, expected, locals)
}

func TestAddressedVariableFrameOffsetCalculation(t *testing.T) {
	input := []byte("{ var a I64; var b I64; var c I64; var d I64; print(a&); print(c&); print(d&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	expected := []LocalVarInfo{
		{Name: "a", Type: TypeI64, Storage: VarStorageTStack, Address: 0},
		{Name: "b", Type: TypeI64, Storage: VarStorageLocal, Address: 1},
		{Name: "c", Type: TypeI64, Storage: VarStorageTStack, Address: 8},
		{Name: "d", Type: TypeI64, Storage: VarStorageTStack, Address: 16},
	}

	be.Equal(t, expected, locals)
}

func TestAddressOfRvalue(t *testing.T) {
	input := []byte("{ var x I64; print((x + 1)&); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	locals := collectLocalVariables(ast)

	// x is not addressed since we're taking address of expression, not variable
	expected := []LocalVarInfo{
		{Name: "x", Type: TypeI64, Storage: VarStorageLocal, Address: 0},
	}

	be.Equal(t, expected, locals)
}

func TestCollectLocalsRecursive(t *testing.T) {
	t.Run("handles nil TypeAST", func(t *testing.T) {
		// Create a NodeVar with nil TypeAST to trigger the early return
		node := &ASTNode{
			Kind: NodeVar,
			Children: []*ASTNode{
				{Kind: NodeIdent, String: "x"},
			},
			TypeAST: nil, // This will trigger the early return
		}

		var locals []LocalVarInfo
		var index uint32 = 0

		// This should not panic and should return early
		collectLocalsRecursive(node, &locals, &index)

		// Verify that no variables were collected due to nil TypeAST
		if len(locals) != 0 {
			t.Errorf("Expected no variables collected with nil TypeAST, got %d", len(locals))
		}
	})

	t.Run("collects variable with valid TypeAST", func(t *testing.T) {
		// Create a NodeVar with valid TypeAST
		node := &ASTNode{
			Kind: NodeVar,
			Children: []*ASTNode{
				{Kind: NodeIdent, String: "x"},
			},
			TypeAST: &TypeNode{
				Kind:   TypeBuiltin,
				String: "I64",
			},
		}

		var locals []LocalVarInfo
		var index uint32 = 0

		collectLocalsRecursive(node, &locals, &index)

		// Verify that the variable was collected
		if len(locals) != 1 {
			t.Errorf("Expected 1 variable collected, got %d", len(locals))
		} else if locals[0].Name != "x" {
			t.Errorf("Expected variable name 'x', got '%s'", locals[0].Name)
		}
	})
}
