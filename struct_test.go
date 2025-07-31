package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestStructSymbolTable(t *testing.T) {
	input := []byte(`struct Point { var x I64; var y I64; }
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
}

func TestStructTypeSize(t *testing.T) {
	// Create a simple struct type
	structType := &TypeNode{
		Kind:   TypeStruct,
		String: "Point",
		Fields: []StructField{
			{Name: "x", Type: TypeI64, Offset: 0},
			{Name: "y", Type: TypeI64, Offset: 8},
		},
	}

	size := GetTypeSize(structType)
	be.Equal(t, size, 16) // 8 bytes for x + 8 bytes for y
}
