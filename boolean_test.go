package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Boolean Tests

func TestBooleanParsing(t *testing.T) {
	source := `
		func main() {
			var x Boolean;
			x = true;
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Check that the AST contains boolean nodes
	expected := `(block (func "main" () void (block (var (ident "x") (ident "Boolean")) (binary "=" (ident "x") (boolean true)))))`
	result := ToSExpr(ast)
	be.Equal(t, result, expected)
}

func TestBooleanTypeChecking(t *testing.T) {
	source := `
		func main() {
			var x Boolean;
			x = true;
			var y I64;
			y = x; // This should fail type checking
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Should fail type checking
	err := CheckProgram(ast)
	be.Equal(t, err != nil, true)
}
