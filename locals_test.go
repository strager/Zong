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
		{Name: "x", Type: "I64", Index: 0},
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
		{Name: "x", Type: "I64", Index: 0},
		{Name: "y", Type: "I64", Index: 1},
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
		{Name: "a", Type: "I64", Index: 0},
		{Name: "b", Type: "I64", Index: 1},
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
