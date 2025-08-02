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
	// Test TypeToString for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	result := TypeToString(sliceType)
	be.Equal(t, result, "I64[]")
}

func TestSliceSize(t *testing.T) {
	// Test GetTypeSize for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	size := GetTypeSize(sliceType)
	be.Equal(t, size, 16) // 8 bytes pointer + 8 bytes length
}

func TestStructTypeSize(t *testing.T) {
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

func TestStructSymbolTable(t *testing.T) {
	input := []byte(`struct Point(x: I64, y: I64);
	var p Point;
	var q Point;
	\x00`)
	Init(input)
	NextToken()

	// Parse struct declaration
	structAST := ParseStatement()
	// Parse variable declarations
	varAST1 := ParseStatement()
	varAST2 := ParseStatement()

	// Create a block containing all statements
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{structAST, varAST1, varAST2},
	}

	// Build symbol table
	symbolTable := BuildSymbolTable(blockAST)

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
	tc := NewTypeChecker()

	// Create integer node
	intNode := &ASTNode{
		Kind:    NodeInteger,
		Integer: 42,
	}

	// Check expression
	err := CheckExpression(intNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeIntegerNode, intNode.TypeAST)
}

func TestCheckExpressionVariableAssigned(t *testing.T) {
	st := NewSymbolTable()
	symbol, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	symbol.Assigned = true

	tc := NewTypeChecker()

	// Create variable reference node with symbol reference
	varNode := &ASTNode{
		Kind:   NodeIdent,
		String: "x",
		Symbol: symbol,
	}

	// Check expression
	err = CheckExpression(varNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, varNode.TypeAST)
}

func TestCheckExpressionBinaryArithmetic(t *testing.T) {
	tc := NewTypeChecker()

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
	err := CheckExpression(binaryNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, binaryNode.TypeAST)
}

func TestCheckExpressionBinaryComparison(t *testing.T) {
	tc := NewTypeChecker()

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
	err := CheckExpression(binaryNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeBool, binaryNode.TypeAST) // Comparison returns Bool
}

func TestCheckExpressionAddressOf(t *testing.T) {
	st := NewSymbolTable()
	symbol, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	symbol.Assigned = true

	tc := NewTypeChecker()

	// Create address-of expression: x& with symbol reference
	addrNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "&",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "x", Symbol: symbol},
		},
	}

	// Check expression
	err = CheckExpression(addrNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypePointer, addrNode.TypeAST.Kind)
	be.Equal(t, TypeI64, addrNode.TypeAST.Child)
}

func TestCheckExpressionDereference(t *testing.T) {
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	symbol, err := st.DeclareVariable("ptr", ptrType)
	be.Err(t, err, nil)
	symbol.Assigned = true

	tc := NewTypeChecker()

	// Create dereference expression: ptr* with symbol reference
	derefNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "ptr", Symbol: symbol},
		},
	}

	// Check expression
	err = CheckExpression(derefNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, derefNode.TypeAST)
}

func TestCheckExpressionFunctionCall(t *testing.T) {
	tc := NewTypeChecker()

	// Create function call: print(42)
	callNode := &ASTNode{
		Kind: NodeCall,
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "print"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	// Check expression
	err := CheckExpression(callNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, callNode.TypeAST)
}

func TestCheckAssignmentValid(t *testing.T) {
	st := NewSymbolTable()
	symbol, err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	tc := NewTypeChecker()

	// Create assignment nodes: x = 42 with symbol reference
	lhs := &ASTNode{Kind: NodeIdent, String: "x", Symbol: symbol}
	rhs := &ASTNode{Kind: NodeInteger, Integer: 42}

	// Check assignment
	err = CheckAssignment(lhs, rhs, tc)
	be.Err(t, err, nil)

	// Verify variable is now assigned
	be.Equal(t, true, symbol.Assigned)
}

func TestCheckAssignmentPointerDereference(t *testing.T) {
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	symbol, err := st.DeclareVariable("ptr", ptrType)
	be.Err(t, err, nil)
	symbol.Assigned = true

	tc := NewTypeChecker()

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
	err = CheckAssignment(lhs, rhs, tc)
	be.Err(t, err, nil)
}

func TestCheckProgramSuccess(t *testing.T) {
	// Parse: { var x I64; x = 42; print(x); }
	input := []byte("{ var x I64; x = 42; print(x); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table and check program
	_ = BuildSymbolTable(ast)
	err := CheckProgram(ast)
	be.Err(t, err, nil)
}

func TestStringLiteralTypeChecking(t *testing.T) {
	input := []byte(`var s U8[] = "hello";` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Build symbol table and run type checking
	_ = BuildSymbolTable(ast)
	err := CheckProgram(ast)
	be.Err(t, err, nil) // Should not error

	// Check that string literal has correct type
	varDecl := ast.Children[0]
	stringLiteral := varDecl.Children[1]
	be.True(t, stringLiteral.TypeAST != nil)
	be.Equal(t, stringLiteral.TypeAST.Kind, TypeSlice)
	be.Equal(t, stringLiteral.TypeAST.Child.String, "U8")
}
