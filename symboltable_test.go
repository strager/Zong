package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestNewSymbolTable(t *testing.T) {
	st := NewSymbolTable()
	be.True(t, st != nil)
	be.Equal(t, 0, len(st.GetAllVariables()))
}

func TestDeclareVariable(t *testing.T) {
	st := NewSymbolTable()

	// Declare a variable
	symbol, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	be.True(t, symbol != nil)

	variables := st.GetAllVariables()
	be.Equal(t, 1, len(variables))
	be.Equal(t, "x", variables[0].Name)
	be.Equal(t, TypeI64, variables[0].Type)
	be.Equal(t, false, variables[0].Assigned)
}

func TestDeclareVariableDuplicate(t *testing.T) {
	st := NewSymbolTable()

	// Declare a variable
	_, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	// Try to declare the same variable again
	symbol, err := st.DeclareVariable("x", TypeI64)
	be.True(t, err != nil)
	be.True(t, symbol == nil)
	be.Equal(t, "error: variable 'x' already declared", err.Error())
}

func TestLookupVariable(t *testing.T) {
	st := NewSymbolTable()

	// Lookup non-existent variable
	symbol := st.LookupVariable("x")
	be.True(t, symbol == nil)

	// Declare and lookup variable
	_, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	symbol = st.LookupVariable("x")
	be.True(t, symbol != nil)
	be.Equal(t, "x", symbol.Name)
	be.Equal(t, TypeI64, symbol.Type)
	be.Equal(t, false, symbol.Assigned)
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
	variables := st.GetAllVariables()
	be.Equal(t, 1, len(variables))
	be.Equal(t, "x", variables[0].Name)
	be.Equal(t, TypeI64, variables[0].Type)
	be.Equal(t, false, variables[0].Assigned)
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
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))
	// Check that both variables exist (order may vary due to hierarchical structure)
	names := make(map[string]bool)
	for _, v := range variables {
		names[v.Name] = true
		be.Equal(t, TypeI64, v.Type)
	}
	be.True(t, names["x"])
	be.True(t, names["y"])
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
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))
	// Check that both variables exist with correct types (order may vary)
	var ptrVar, xVar *SymbolInfo
	for _, v := range variables {
		if v.Name == "ptr" {
			ptrVar = v
		} else if v.Name == "x" {
			xVar = v
		}
	}
	be.True(t, ptrVar != nil)
	be.Equal(t, TypePointer, ptrVar.Type.Kind)
	be.Equal(t, TypeI64, ptrVar.Type.Child)
	be.True(t, xVar != nil)
	be.Equal(t, TypeI64, xVar.Type)
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
	variables := st.GetAllVariables()
	be.Equal(t, 1, len(variables))
	be.Equal(t, "x", variables[0].Name)
	be.Equal(t, TypeI64, variables[0].Type)
}

func TestVariableShadowingInNestedBlocks(t *testing.T) {
	// Parse: { var x I64; { var x I64; } }
	input := []byte("{ var x I64; { var x I64; } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should have both variables but only outer one is accessible at top level
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Lookup should find the outer variable
	outerX := st.LookupVariable("x")
	be.True(t, outerX != nil)
	be.Equal(t, "x", outerX.Name)
}

func TestFunctionParameterShadowing(t *testing.T) {
	// Parse: func test(x: I64) { var x I64; }
	input := []byte("func test(x: I64) { var x I64; }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should have both parameter and local variable
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Function should be declared
	testFunc := st.LookupFunction("test")
	be.True(t, testFunc != nil)
	be.Equal(t, "test", testFunc.Name)
}

func TestNestedBlockScoping(t *testing.T) {
	// Parse: { var outer I64; { var middle I64; { var inner I64; } } }
	input := []byte("{ var outer I64; { var middle I64; { var inner I64; } } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should have all three variables
	variables := st.GetAllVariables()
	be.Equal(t, 3, len(variables))

	// Check that outer variable is accessible
	outerVar := st.LookupVariable("outer")
	be.True(t, outerVar != nil)
	be.Equal(t, "outer", outerVar.Name)
}

func TestFunctionScopingWithLocalVariables(t *testing.T) {
	// Parse: func test() { var local I64; } var global I64;
	input := []byte("func test() { var local I64; } var global I64;\x00")
	Init(input)
	NextToken()

	// Parse function
	funcAST := ParseStatement()
	// Parse global variable
	varAST := ParseStatement()

	// Create block containing both
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{funcAST, varAST},
	}

	// Build symbol table
	st := BuildSymbolTable(blockAST)

	// Should have both variables
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Should have function
	testFunc := st.LookupFunction("test")
	be.True(t, testFunc != nil)

	// Global variable should be accessible
	globalVar := st.LookupVariable("global")
	be.True(t, globalVar != nil)
	be.Equal(t, "global", globalVar.Name)
}

func TestMultipleShadowingLevels(t *testing.T) {
	// Parse: { var x I64; { var x I64; { var x I64; } } }
	input := []byte("{ var x I64; { var x I64; { var x I64; } } }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should have three x variables at different scope levels
	variables := st.GetAllVariables()
	be.Equal(t, 3, len(variables))

	// All should be named "x"
	for _, v := range variables {
		be.Equal(t, "x", v.Name)
		be.Equal(t, TypeI64, v.Type)
	}
}
