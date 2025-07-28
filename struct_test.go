package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestStructDeclarationParsing(t *testing.T) {
	input := []byte("struct Point { var x I64; var y I64; }\x00")
	Init(input)
	NextToken()

	ast := ParseStatement()

	expectedSExpr := `(struct "Point" (var (ident "x") (ident "I64")) (var (ident "y") (ident "I64")))`
	actualSExpr := ToSExpr(ast)

	be.Equal(t, actualSExpr, expectedSExpr)
}

func TestStructTypeInVariableDeclaration(t *testing.T) {
	input := []byte("var p Point;\x00")
	Init(input)
	NextToken()

	ast := ParseStatement()

	expectedSExpr := `(var (ident "p") (ident "Point"))`
	actualSExpr := ToSExpr(ast)

	be.Equal(t, actualSExpr, expectedSExpr)
}

func TestFieldAccessParsing(t *testing.T) {
	input := []byte("p.x\x00")
	Init(input)
	NextToken()

	ast := ParseExpression()

	expectedSExpr := `(dot (ident "p") "x")`
	actualSExpr := ToSExpr(ast)

	be.Equal(t, actualSExpr, expectedSExpr)
}

func TestFieldAssignmentParsing(t *testing.T) {
	input := []byte("p.x = 42\x00")
	Init(input)
	NextToken()

	ast := ParseExpression()

	expectedSExpr := `(binary "=" (dot (ident "p") "x") (integer 42))`
	actualSExpr := ToSExpr(ast)

	be.Equal(t, actualSExpr, expectedSExpr)
}

func TestComplexStructExpression(t *testing.T) {
	input := []byte("p.x + q.y\x00")
	Init(input)
	NextToken()

	ast := ParseExpression()

	expectedSExpr := `(binary "+" (dot (ident "p") "x") (dot (ident "q") "y"))`
	actualSExpr := ToSExpr(ast)

	be.Equal(t, actualSExpr, expectedSExpr)
}

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

func TestStructTypeChecking(t *testing.T) {
	input := []byte(`struct Point { var x I64; var y I64; }
	var p Point;
	p.x = 42;
	print(p.y);
	\x00`)
	Init(input)
	NextToken()

	// Parse all statements
	structAST := ParseStatement()
	varAST := ParseStatement()
	assignAST := ParseStatement()
	printAST := ParseStatement()

	// Create a block containing all statements
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{structAST, varAST, assignAST, printAST},
	}

	// Build symbol table and perform type checking
	_ = BuildSymbolTable(blockAST)
	err := CheckProgram(blockAST)

	// Should not have any type errors
	be.Err(t, err, nil)
}

func TestFieldAccessTypeError(t *testing.T) {
	input := []byte("var x I64; x = 42; x.field;\x00")
	Init(input)
	NextToken()

	// Parse statements
	varAST := ParseStatement()
	assignAST := ParseStatement()
	accessAST := ParseStatement()

	// Create a block containing all statements
	blockAST := &ASTNode{
		Kind:     NodeBlock,
		Children: []*ASTNode{varAST, assignAST, accessAST},
	}

	// Build symbol table and perform type checking
	_ = BuildSymbolTable(blockAST)
	err := CheckProgram(blockAST)

	// Should have a type error (cannot access field of non-struct type)
	be.True(t, err != nil)
}
