package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestNewSymbolTable(t *testing.T) {
	st := NewSymbolTable()
	be.True(t, st != nil)
	be.Equal(t, 0, len(st.variables))
}

func TestDeclareVariable(t *testing.T) {
	st := NewSymbolTable()

	// Declare a variable
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	be.Equal(t, 1, len(st.variables))
	be.Equal(t, "x", st.variables[0].Name)
	be.Equal(t, TypeI64, st.variables[0].Type)
	be.Equal(t, false, st.variables[0].Assigned)
}

func TestDeclareVariableDuplicate(t *testing.T) {
	st := NewSymbolTable()

	// Declare a variable
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	// Try to declare the same variable again
	err = st.DeclareVariable("x", TypeI64)
	be.True(t, err != nil)
	be.Equal(t, "error: variable 'x' already declared", err.Error())
}

func TestLookupVariable(t *testing.T) {
	st := NewSymbolTable()

	// Lookup non-existent variable
	symbol := st.LookupVariable("x")
	be.True(t, symbol == nil)

	// Declare and lookup variable
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	symbol = st.LookupVariable("x")
	be.True(t, symbol != nil)
	be.Equal(t, "x", symbol.Name)
	be.Equal(t, TypeI64, symbol.Type)
	be.Equal(t, false, symbol.Assigned)
}

func TestAssignVariable(t *testing.T) {
	st := NewSymbolTable()

	// Declare variable
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	// Check initially not assigned
	symbol := st.LookupVariable("x")
	be.Equal(t, false, symbol.Assigned)

	// Assign variable
	st.AssignVariable("x")

	// Check now assigned
	symbol = st.LookupVariable("x")
	be.True(t, symbol.Assigned)
}

func TestAssignVariableNotDeclared(t *testing.T) {
	st := NewSymbolTable()

	// Try to assign non-existent variable - should panic
	defer func() {
		r := recover()
		be.True(t, r != nil)
		errorMsg := r.(string)
		be.Equal(t, "error: variable 'undefined' used before declaration", errorMsg)
	}()

	st.AssignVariable("undefined")
}

func TestBuildSymbolTableSimple(t *testing.T) {
	// Parse: var x I64;
	input := []byte("var x I64;\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Verify symbol table
	be.Equal(t, 1, len(st.variables))
	be.Equal(t, "x", st.variables[0].Name)
	be.Equal(t, TypeI64, st.variables[0].Type)
	be.Equal(t, false, st.variables[0].Assigned)
}

func TestBuildSymbolTableMultiple(t *testing.T) {
	// Parse: { var x I64; var y I64; }
	input := []byte("{ var x I64; var y I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Verify symbol table
	be.Equal(t, 2, len(st.variables))
	be.Equal(t, "x", st.variables[0].Name)
	be.Equal(t, TypeI64, st.variables[0].Type)
	be.Equal(t, "y", st.variables[1].Name)
	be.Equal(t, TypeI64, st.variables[1].Type)
}

func TestBuildSymbolTableWithPointers(t *testing.T) {
	// Parse: { var x I64; var ptr I64*; }
	input := []byte("{ var x I64; var ptr I64*; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Verify symbol table
	be.Equal(t, 2, len(st.variables))
	be.Equal(t, "x", st.variables[0].Name)
	be.Equal(t, TypeI64, st.variables[0].Type)
	be.Equal(t, "ptr", st.variables[1].Name)
	be.Equal(t, TypePointer, st.variables[1].Type.Kind)
	be.Equal(t, TypeI64, st.variables[1].Type.Child)
}

func TestBuildSymbolTableIgnoresUnsupportedTypes(t *testing.T) {
	// Parse: { var x I64; var y string; }
	input := []byte("{ var x I64; var y string; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should only include I64 variable
	be.Equal(t, 1, len(st.variables))
	be.Equal(t, "x", st.variables[0].Name)
	be.Equal(t, TypeI64, st.variables[0].Type)
}

func TestBuildSymbolTableNested(t *testing.T) {
	// Parse: { var a I64; { var b I64; } }
	input := []byte("{ var a I64; { var b I64; } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should include both variables (function-scoped in WebAssembly)
	be.Equal(t, 2, len(st.variables))
	be.Equal(t, "a", st.variables[0].Name)
	be.Equal(t, "b", st.variables[1].Name)
}
