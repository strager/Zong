// Semantic analysis and type system tests
//
// Tests symbol tables, type checking, and type system utilities.
// Covers variable scoping, type validation, and semantic error detection.

package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// =============================================================================
// TYPE SYSTEM UTILITIES TESTS
// =============================================================================

func TestTypesEqual(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a, b     *TypeNode
		expected bool
	}{
		{
			name:     "same builtin types",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: true,
		},
		{
			name:     "different builtin types",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypeBuiltin, String: "Boolean"},
			expected: false,
		},
		{
			name:     "different kinds",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: false,
		},
		{
			name:     "same pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: true,
		},
		{
			name:     "different pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "Boolean"}},
			expected: false,
		},
		{
			name:     "nested pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypesEqual(test.a, test.b)
			be.Equal(t, test.expected, result)
		})
	}
}

func TestIsWASMI64Type(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		t        *TypeNode
		expected bool
	}{
		{
			name:     "nil type",
			t:        nil,
			expected: false,
		},
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: true,
		},
		{
			name:     "Boolean builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Boolean"},
			expected: true,
		},
		{
			name:     "unsupported builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "String"},
			expected: false,
		},
		{
			name:     "pointer type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: false, // pointers are now i32, not i64
		},
		{
			name:     "pointer to unsupported type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "String"}},
			expected: false, // pointers are now i32, not i64
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isWASMI64Type(test.t)
			be.Equal(t, test.expected, result)
		})
	}
}

func TestGetTypeSize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		t        *TypeNode
		expected int
	}{
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: 8,
		},
		{
			name:     "Boolean builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Boolean"},
			expected: 8,
		},
		{
			name:     "unknown builtin defaults to 8",
			t:        &TypeNode{Kind: TypeBuiltin, String: "UnknownType"},
			expected: 8,
		},
		{
			name:     "pointer type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: 8,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetTypeSize(test.t)
			be.Equal(t, test.expected, result)
		})
	}
}

func TestTypeToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		t        *TypeNode
		expected string
	}{
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: "I64",
		},
		{
			name:     "Boolean builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Boolean"},
			expected: "Boolean",
		},
		{
			name:     "pointer to I64",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: "I64*",
		},
		{
			name:     "pointer to Boolean",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "Boolean"}},
			expected: "Boolean*",
		},
		{
			name:     "pointer to pointer",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			expected: "I64**",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypeToString(test.t)
			be.Equal(t, test.expected, result)
		})
	}
}

func TestSliceTypeToString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		typeNode    *TypeNode
		expectedStr string
	}{
		{
			name: "I64 slice",
			typeNode: &TypeNode{
				Kind:  TypeSlice,
				Child: TypeI64,
			},
			expectedStr: "I64[]",
		},
		{
			name: "pointer slice",
			typeNode: &TypeNode{
				Kind: TypeSlice,
				Child: &TypeNode{
					Kind:  TypePointer,
					Child: TypeI64,
				},
			},
			expectedStr: "I64*[]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypeToString(test.typeNode)
			be.Equal(t, result, test.expectedStr)
		})
	}
}

func TestSliceStringRepresentation(t *testing.T) {
	t.Parallel()
	// Test TypeToString for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	result := TypeToString(sliceType)
	be.Equal(t, result, "I64[]")
}

func TestSliceSize(t *testing.T) {
	t.Parallel()
	// Test GetTypeSize for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	size := GetTypeSize(sliceType)
	be.Equal(t, size, 16) // 8 bytes pointer + 8 bytes length
}

func TestStructTypeSize(t *testing.T) {
	t.Parallel()
	// Create a simple struct type
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []Parameter{
			{Name: "x", Type: TypeI64, Offset: 0},
			{Name: "y", Type: TypeI64, Offset: 8},
		},
	}

	size := GetTypeSize(structType)
	be.Equal(t, size, 16) // 8 bytes for x + 8 bytes for y
}

func TestTypeUtilityFunctions(t *testing.T) {
	t.Parallel()
	// Test TypesEqual
	if !TypesEqual(TypeI64, TypeI64) {
		t.Error("TypeI64 should equal itself")
	}

	if TypesEqual(TypeI64, TypeBool) {
		t.Error("TypeI64 should not equal TypeBool")
	}

	i64Ptr := &TypeNode{Kind: TypePointer, Child: TypeI64}
	i64Ptr2 := &TypeNode{Kind: TypePointer, Child: TypeI64}
	if !TypesEqual(i64Ptr, i64Ptr2) {
		t.Error("I64* types should be equal")
	}

	boolPtr := &TypeNode{Kind: TypePointer, Child: TypeBool}
	if TypesEqual(i64Ptr, boolPtr) {
		t.Error("I64* and Boolean* should not be equal")
	}

	// Test TypeToString
	be.Equal(t, "I64", TypeToString(TypeI64))

	be.Equal(t, "I64*", TypeToString(i64Ptr))

	i64PtrPtr := &TypeNode{Kind: TypePointer, Child: i64Ptr}
	be.Equal(t, "I64**", TypeToString(i64PtrPtr))

	// Test GetTypeSize
	be.Equal(t, 8, GetTypeSize(TypeI64))

	be.Equal(t, 8, GetTypeSize(TypeBool))

	be.Equal(t, 8, GetTypeSize(i64Ptr))

	// Test isWASMI64Type
	if !isWASMI64Type(TypeI64) {
		t.Error("I64 should be a WASM I64 type")
	}

	if !isWASMI64Type(TypeBool) {
		t.Error("Boolean should be a WASM I64 type")
	}

	if !isWASMI32Type(i64Ptr) {
		t.Error("I64* should be a WASM I32 type")
	}

	unknownType := &TypeNode{Kind: TypeBuiltin, String: "string"}
	if isWASMI64Type(unknownType) {
		t.Error("string type should not be a WASM I64 type")
	}
}

// =============================================================================
// SYMBOL TABLE TESTS
// =============================================================================

func TestNewSymbolTable(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	be.True(t, st != nil)
	be.Equal(t, 0, len(st.GetAllVariables()))
}

func TestDeclareVariable(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Declare a variable
	symbol := st.DeclareVariable("x", TypeI64)
	be.True(t, symbol != nil)

	variables := st.GetAllVariables()
	be.Equal(t, 1, len(variables))
	be.Equal(t, "x", variables[0].Name)
	be.Equal(t, TypeI64, variables[0].Type)
	be.Equal(t, false, variables[0].Assigned)
}

func TestDeclareVariableDuplicate(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Declare a variable
	st.DeclareVariable("x", TypeI64)

	// Try to declare the same variable again
	symbol := st.DeclareVariable("x", TypeI64)
	be.True(t, st.Errors.HasErrors())
	be.True(t, symbol == nil)
	be.Equal(t, "error: variable 'x' already declared", st.Errors.String())
}

func TestLookupVariable(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Lookup non-existent variable
	symbol := st.LookupVariable("x")
	be.True(t, symbol == nil)

	// Declare and lookup variable
	st.DeclareVariable("x", TypeI64)

	symbol = st.LookupVariable("x")
	be.True(t, symbol != nil)
	be.Equal(t, "x", symbol.Name)
	be.Equal(t, TypeI64, symbol.Type)
	be.Equal(t, false, symbol.Assigned)
}

func TestBuildSymbolTableSimple(t *testing.T) {
	t.Parallel()
	// Parse: var x: I64;
	input := []byte("var x: I64;\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

	// Verify symbol table
	variables := st.GetAllVariables()
	be.Equal(t, 1, len(variables))
	be.Equal(t, "x", variables[0].Name)
	be.Equal(t, TypeI64, variables[0].Type)
	be.Equal(t, false, variables[0].Assigned)
}

func TestBuildSymbolTableMultiple(t *testing.T) {
	t.Parallel()
	// Parse: { var x: I64; var y: I64; }
	input := []byte("{ var x: I64; var y: I64; }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

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
	t.Parallel()
	// Parse: { var x: I64; var ptr: I64*; }
	input := []byte("{ var x: I64; var ptr: I64*; }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

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

func TestBuildSymbolTableReportsUnknownStructTypes(t *testing.T) {
	t.Parallel()
	// Parse: { var x: I64; var y String; }
	input := []byte("{ var x: I64; var y String; }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)

	// Should report error for unknown struct type "String"
	be.True(t, st.Errors.HasErrors())
	// TODO(strager): Only report one error.
	be.Equal(t, 2, st.Errors.Count())
	be.Equal(t, "error: undefined symbol 'String'\nerror: undefined symbol 'String'", st.Errors.String())

	// Should still include both variables in symbol table
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Check that both variables are present regardless of order
	var xVar, yVar *SymbolInfo
	for _, v := range variables {
		if v.Name == "x" {
			xVar = v
		} else if v.Name == "y" {
			yVar = v
		}
	}

	be.True(t, xVar != nil)
	be.Equal(t, TypeI64, xVar.Type)
	be.True(t, yVar != nil)
	be.Equal(t, "String", yVar.Type.String) // Still parsed as struct type, just unresolved
}

func TestVariableShadowingInNestedBlocks(t *testing.T) {
	t.Parallel()
	// Parse: { var x: I64; { var x: I64; } }
	input := []byte("{ var x: I64; { var x: I64; } }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

	// Should have both variables but only outer one is accessible at top level
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Lookup should find the outer variable
	outerX := st.LookupVariable("x")
	be.True(t, outerX != nil)
	be.Equal(t, "x", outerX.Name)
}

func TestFunctionParameterShadowing(t *testing.T) {
	t.Parallel()
	// Parse: func test(x: I64) { var x: I64; }
	input := []byte("func test(x: I64) { var x: I64; }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

	// Should have both parameter and local variable
	variables := st.GetAllVariables()
	be.Equal(t, 2, len(variables))

	// Function should be declared
	testFunc := st.LookupFunction("test")
	be.True(t, testFunc != nil)
	be.Equal(t, "test", testFunc.Name)
}

func TestNestedBlockScoping(t *testing.T) {
	t.Parallel()
	// Parse: { var outer: I64; { var middle: I64; { var inner: I64; } } }
	input := []byte("{ var outer: I64; { var middle: I64; { var inner: I64; } } }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

	// Should have all three variables
	variables := st.GetAllVariables()
	be.Equal(t, 3, len(variables))

	// Check that outer variable is accessible
	outerVar := st.LookupVariable("outer")
	be.True(t, outerVar != nil)
	be.Equal(t, "outer", outerVar.Name)
}

func TestFunctionScopingWithLocalVariables(t *testing.T) {
	t.Parallel()
	// Parse: func test() { var local: I64; } var global: I64;
	input := []byte("func test() { var local: I64; } var global: I64;\x00")
	l := NewLexer(input)
	l.NextToken()

	// Parse function
	funcAST := ParseStatement(l)
	// Parse global variable
	varAST := ParseStatement(l)

	// Create block containing both
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{funcAST, varAST},
	}

	// Build symbol table
	st := BuildSymbolTable(blockAST)
	be.True(t, !st.Errors.HasErrors())

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
	t.Parallel()
	// Parse: { var x: I64; { var x: I64; { var x: I64; } } }
	input := []byte("{ var x: I64; { var x: I64; { var x: I64; } } }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table
	st := BuildSymbolTable(ast)
	be.True(t, !st.Errors.HasErrors())

	// Should have three x variables at different scope levels
	variables := st.GetAllVariables()
	be.Equal(t, 3, len(variables))

	// All should be named "x"
	for _, v := range variables {
		be.Equal(t, "x", v.Name)
		be.Equal(t, TypeI64, v.Type)
	}
}

func TestStructSymbolTable(t *testing.T) {
	t.Parallel()
	input := []byte(`struct Point(x: I64, y: I64);
	var p Point;
	var q Point;
	\x00`)
	l := NewLexer(input)
	l.NextToken()

	// Parse struct declaration
	structAST := ParseStatement(l)
	// Parse variable declarations
	varAST1 := ParseStatement(l)
	varAST2 := ParseStatement(l)

	// Create a block containing all statements
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{structAST, varAST1, varAST2},
	}

	// Build symbol table
	symbolTable := BuildSymbolTable(blockAST)
	be.True(t, !symbolTable.Errors.HasErrors())

	// Check that Point struct is declared
	pointStruct := symbolTable.LookupStruct("Point")
	be.True(t, pointStruct != nil)
	be.Equal(t, pointStruct.String, "Point")
	be.Equal(t, len(pointStruct.Fields), 2)
	be.Equal(t, pointStruct.Fields[0].Name, "x")
	be.Equal(t, pointStruct.Fields[1].Name, "y")

	// Check field offsets
	be.Equal(t, pointStruct.Fields[0].Offset, uint32(0))
	be.Equal(t, pointStruct.Fields[1].Offset, uint32(8))

	// Check that variables are declared with struct type
	pVar := symbolTable.LookupVariable("p")
	be.True(t, pVar != nil)
	be.Equal(t, pVar.Type.Kind, TypeStruct)
	be.Equal(t, pVar.Type.String, "Point")

	qVar := symbolTable.LookupVariable("q")
	be.True(t, qVar != nil)
	be.Equal(t, qVar.Type.Kind, TypeStruct)
	be.Equal(t, qVar.Type.String, "Point")

	// NEW: Check that struct fields now have symbol table entries
	xField := pointStruct.Fields[0]
	yField := pointStruct.Fields[1]
	be.True(t, xField.Symbol != nil)
	be.True(t, yField.Symbol != nil)
	be.Equal(t, xField.Symbol.Name, "x")
	be.Equal(t, yField.Symbol.Name, "y")
	be.Equal(t, xField.Symbol.Type, TypeI64)
	be.Equal(t, yField.Symbol.Type, TypeI64)
	be.Equal(t, xField.Symbol.Assigned, true) // Fields are always "assigned"
	be.Equal(t, yField.Symbol.Assigned, true)
}

// =============================================================================
// TYPE CHECKING TESTS
// =============================================================================

func TestCheckExpressionInteger(t *testing.T) {
	t.Parallel()
	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create integer node
	intNode := &ASTNode{
		Kind:    NodeInteger,
		Integer: 42,
	}

	// Check expression
	CheckExpression(intNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeIntegerNode, intNode.TypeAST)
}

func TestCheckExpressionVariableAssigned(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	symbol := st.DeclareVariable("x", TypeI64)
	symbol.Assigned = true

	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create variable reference node with symbol reference
	varNode := &ASTNode{
		Kind:   NodeIdent,
		String: "x",
		Symbol: symbol,
	}

	// Check expression
	CheckExpression(varNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeI64, varNode.TypeAST)
}

func TestCheckExpressionBinaryArithmetic(t *testing.T) {
	t.Parallel()
	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create binary expression: 42 + 10
	binaryNode := &ASTNode{
		Kind: NodeBinary,
		Op:   "+",
		Children: []*ASTNode{
			{Kind: NodeInteger, Integer: 42},
			{Kind: NodeInteger, Integer: 10},
		},
	}

	// Check expression
	CheckExpression(binaryNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeI64, binaryNode.TypeAST)
}

func TestCheckExpressionBinaryComparison(t *testing.T) {
	t.Parallel()
	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create binary expression: 42 == 10
	binaryNode := &ASTNode{
		Kind: NodeBinary,
		Op:   "==",
		Children: []*ASTNode{
			{Kind: NodeInteger, Integer: 42},
			{Kind: NodeInteger, Integer: 10},
		},
	}

	// Check expression
	CheckExpression(binaryNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeBool, binaryNode.TypeAST) // Comparison returns Bool
}

func TestCheckExpressionAddressOf(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	symbol := st.DeclareVariable("x", TypeI64)
	symbol.Assigned = true

	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create address-of expression: x& with symbol reference
	addrNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "&",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "x", Symbol: symbol},
		},
	}

	// Check expression
	CheckExpression(addrNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypePointer, addrNode.TypeAST.Kind)
	be.Equal(t, TypeI64, addrNode.TypeAST.Child)
}

func TestCheckExpressionDereference(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	symbol := st.DeclareVariable("ptr", ptrType)
	symbol.Assigned = true

	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create dereference expression: ptr* with symbol reference
	derefNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "ptr", Symbol: symbol},
		},
	}

	// Check expression
	CheckExpression(derefNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeI64, derefNode.TypeAST)
}

func TestCheckExpressionFunctionCall(t *testing.T) {
	t.Parallel()
	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create function call: print(42)
	callNode := &ASTNode{
		Kind: NodeCall,
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "print"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	// Check expression
	CheckExpression(callNode, tc)
	be.True(t, !tc.Errors.HasErrors())
	be.Equal(t, TypeI64, callNode.TypeAST)
}

func TestCheckAssignmentValid(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	symbol := st.DeclareVariable("x", TypeI64)

	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create assignment nodes: x = 42 with symbol reference
	lhs := &ASTNode{Kind: NodeIdent, String: "x", Symbol: symbol}
	rhs := &ASTNode{Kind: NodeInteger, Integer: 42}

	// Check assignment
	CheckAssignment(lhs, rhs, tc)
	be.True(t, !tc.Errors.HasErrors())

	// Verify variable is now assigned
	be.Equal(t, true, symbol.Assigned)
}

func TestCheckAssignmentPointerDereference(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	symbol := st.DeclareVariable("ptr", ptrType)
	symbol.Assigned = true

	typeTable := NewTypeTable()
	tc := NewTypeChecker(typeTable)

	// Create assignment nodes: ptr* = 42 with symbol reference
	lhs := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "ptr", Symbol: symbol},
		},
	}
	rhs := &ASTNode{Kind: NodeInteger, Integer: 42}

	// Check assignment
	CheckAssignment(lhs, rhs, tc)
	be.True(t, !tc.Errors.HasErrors())
}

func TestCheckProgramSuccess(t *testing.T) {
	t.Parallel()
	// Parse: { var x: I64; x = 42; print(x); }
	input := []byte("{ var x: I64; x = 42; print(x); }\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseStatement(l)

	// Build symbol table and check program
	symbolTable := BuildSymbolTable(ast)
	be.True(t, !symbolTable.Errors.HasErrors())
	errors := CheckProgram(ast, symbolTable.typeTable)
	be.True(t, !errors.HasErrors())
}

func TestStringLiteralTypeChecking(t *testing.T) {
	t.Parallel()
	input := []byte(`var s: U8[] = "hello";` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Build symbol table and run type checking
	symbolTable := BuildSymbolTable(ast)
	be.True(t, !symbolTable.Errors.HasErrors())
	errors := CheckProgram(ast, symbolTable.typeTable)
	be.True(t, !errors.HasErrors()) // Should not error

	// Check that string literal has correct type
	varDecl := ast.Children[0]
	stringLiteral := varDecl.Children[1]
	be.True(t, stringLiteral.TypeAST != nil)
	be.Equal(t, stringLiteral.TypeAST.Kind, TypeSlice)
	be.Equal(t, stringLiteral.TypeAST.Child.String, "U8")
}

// =============================================================================
// TYPE RESOLUTION TESTS
// =============================================================================

func TestResolveTypeBuiltin(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Builtin types should return unchanged
	resolved := ResolveType(TypeI64, st)
	be.Equal(t, TypeI64, resolved)

	resolved = ResolveType(TypeBool, st)
	be.Equal(t, TypeBool, resolved)

	resolved = ResolveType(TypeU8, st)
	be.Equal(t, TypeU8, resolved)
}

func TestResolveTypeStruct(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Define a struct
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []Parameter{
			{Name: "x", Type: TypeI64},
			{Name: "y", Type: TypeI64},
		},
	}
	st.DeclareStruct(structType)

	// Create a reference to the struct
	structRef := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
	}

	// Resolve should return the complete struct definition
	resolved := ResolveType(structRef, st)
	be.Equal(t, structType, resolved)
	be.Equal(t, 2, len(resolved.Fields))
	be.Equal(t, "x", resolved.Fields[0].Name)
	be.Equal(t, "y", resolved.Fields[1].Name)
}

func TestResolveTypePointer(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Define a struct
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []Parameter{
			{Name: "x", Type: TypeI64},
			{Name: "y", Type: TypeI64},
		},
	}
	st.DeclareStruct(structType)

	// Create a pointer to struct reference
	ptrToStructRef := &TypeNode{
		Kind: TypePointer,
		Child: &TypeNode{
			Kind:   TypeStruct,
			String: "Point",
		},
	}

	// Resolve should return pointer to complete struct definition
	resolved := ResolveType(ptrToStructRef, st)
	be.Equal(t, TypePointer, resolved.Kind)
	be.Equal(t, structType, resolved.Child)
}

func TestResolveTypeSlice(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Define a struct
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []Parameter{
			{Name: "x", Type: TypeI64},
			{Name: "y", Type: TypeI64},
		},
	}
	st.DeclareStruct(structType)

	// Create a slice of struct reference
	sliceOfStructRef := &TypeNode{
		Kind: TypeSlice,
		Child: &TypeNode{
			Kind:   TypeStruct,
			String: "Point",
		},
	}

	// Resolve should return slice of complete struct definition
	resolved := ResolveType(sliceOfStructRef, st)
	be.Equal(t, TypeSlice, resolved.Kind)
	be.Equal(t, structType, resolved.Child)
}

func TestResolveTypeNestedPointers(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Define a struct
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []Parameter{
			{Name: "x", Type: TypeI64},
			{Name: "y", Type: TypeI64},
		},
	}
	st.DeclareStruct(structType)

	// Create a pointer to pointer to struct reference
	ptrToPtrToStructRef := &TypeNode{
		Kind: TypePointer,
		Child: &TypeNode{
			Kind: TypePointer,
			Child: &TypeNode{
				Kind:   TypeStruct,
				String: "Point",
			},
		},
	}

	// Resolve should work recursively
	resolved := ResolveType(ptrToPtrToStructRef, st)
	be.Equal(t, TypePointer, resolved.Kind)
	be.Equal(t, TypePointer, resolved.Child.Kind)
	be.Equal(t, structType, resolved.Child.Child)
}

func TestResolveTypeUnknownStruct(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Create reference to unknown struct
	unknownStructRef := &TypeNode{
		Kind:   TypeStruct,
		String: "UnknownStruct",
	}

	// Resolve should return the original reference (may be forward reference)
	resolved := ResolveType(unknownStructRef, st)
	be.Equal(t, unknownStructRef, resolved)
}

func TestResolveTypeNil(t *testing.T) {
	t.Parallel()
	st := NewSymbolTable()

	// Nil should return nil
	resolved := ResolveType(nil, st)
	be.True(t, resolved == nil)
}
