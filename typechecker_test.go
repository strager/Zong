package main

import (
	"testing"

	"github.com/nalgeon/be"
)

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
