package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// TestParseIfStatement removed - duplicates test/statements_test.md

// TestParseIfElseStatement removed - duplicates test/statements_test.md

// TestParseVarStatement removed - duplicates test/statements_test.md

// TestParsePointerVariableDeclaration removed - duplicates test/statements_test.md

// TestParseBlockStatement removed - now covered by test/parsing_comprehensive_test.md

func TestParseReturnStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "return;\x00",
			expected: "(return)",
		},
		{
			input:    "return 42;\x00",
			expected: "(return (integer 42))",
		},
		{
			input:    "return x + y;\x00",
			expected: "(return (binary \"+\" (ident \"x\") (ident \"y\")))",
		},
		{
			input:    "return foo == bar;\x00",
			expected: "(return (binary \"==\" (ident \"foo\") (ident \"bar\")))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		be.Equal(t, actual, test.expected)
	}
}

func TestParseBreakStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "break;\x00",
			expected: "(break)",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		be.Equal(t, actual, test.expected)
	}
}

func TestParseContinueStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "continue;\x00",
			expected: "(continue)",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		be.Equal(t, actual, test.expected)
	}
}

// TestParseExpressionStatement removed - now covered by test/parsing_comprehensive_test.md

// TestComplexStatements removed - now covered by test/parsing_comprehensive_test.md

func TestVarTypeAST(t *testing.T) {
	tests := []struct {
		input        string
		expectedType *TypeNode
	}{
		{
			input:        "var x I64;\x00",
			expectedType: TypeI64,
		},
		{
			input:        "var flag Boolean;\x00",
			expectedType: TypeBool,
		},
		{
			input:        "var ptr I64*;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: TypeI64},
		},
		{
			input:        "var ptrPtr I64**;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: TypeI64}},
		},
		{
			input:        "var boolPtr Boolean*;\x00",
			expectedType: &TypeNode{Kind: TypePointer, Child: TypeBool},
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()

		be.Equal(t, NodeVar, result.Kind)
		if result.Kind != NodeVar {
			continue
		}

		be.True(t, result.TypeAST != nil)
		if result.TypeAST == nil {
			continue
		}

		be.True(t, TypesEqual(result.TypeAST, test.expectedType))
	}
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

func TestParseStatementErrorCases(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"IF without LBRACE", "if x == 1 ;\x00"},   // Missing {
		{"VAR without variable name", "var ;\x00"}, // Missing identifier
		{"VAR without type", "var x ;\x00"},        // Missing type
		{"LOOP without LBRACE", "loop ;\x00"},      // Missing {
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Init([]byte(tc.input))
			NextToken()

			result := ParseStatement()
			// Should handle malformed statements gracefully
			be.True(t, result != nil)
		})
	}
}

// Test for VAR statement with invalid type
func TestParseStatementVarWithInvalidType(t *testing.T) {
	input := []byte("var x 123;\x00") // 123 is not a valid type
	Init(input)
	NextToken()

	result := ParseStatement()
	// Should handle invalid type gracefully
	be.True(t, result != nil)
}
