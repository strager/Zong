package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestSliceTypeParsing(t *testing.T) {
	// Test basic slice type parsing directly
	input := []byte("var nums I64[];\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeVar)

	expectedType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	be.True(t, TypesEqual(stmt.TypeAST, expectedType))
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

func TestSliceBasicDeclaration(t *testing.T) {
	// Test basic slice variable declaration
	input := []byte("var nums I64[];\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeVar)

	// Verify type is slice
	be.Equal(t, stmt.TypeAST.Kind, TypeSlice)
	be.Equal(t, stmt.TypeAST.Child.Kind, TypeBuiltin)
	be.Equal(t, stmt.TypeAST.Child.String, "I64")
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
