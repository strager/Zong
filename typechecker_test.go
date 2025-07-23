package main

import (
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

func TestNewTypeChecker(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	be.True(t, tc != nil)
	be.Equal(t, st, tc.symbolTable)
	be.Equal(t, 0, len(tc.errors))
}

func TestCheckExpressionInteger(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Create integer node
	intNode := &ASTNode{
		Kind:    NodeInteger,
		Integer: 42,
	}

	// Check expression
	exprType, err := CheckExpression(intNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType)
}

func TestCheckExpressionVariableAssigned(t *testing.T) {
	st := NewSymbolTable()
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	st.AssignVariable("x")

	tc := NewTypeChecker(st)

	// Create variable reference node
	varNode := &ASTNode{
		Kind:   NodeIdent,
		String: "x",
	}

	// Check expression
	exprType, err := CheckExpression(varNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType)
}

func TestCheckExpressionVariableNotDeclared(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Create variable reference node
	varNode := &ASTNode{
		Kind:   NodeIdent,
		String: "undefined",
	}

	// Check expression
	_, err := CheckExpression(varNode, tc)
	be.True(t, err != nil)
	be.Equal(t, "error: variable 'undefined' used before declaration", err.Error())
}

func TestCheckExpressionVariableNotAssigned(t *testing.T) {
	st := NewSymbolTable()
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	tc := NewTypeChecker(st)

	// Create variable reference node
	varNode := &ASTNode{
		Kind:   NodeIdent,
		String: "x",
	}

	// Check expression
	_, err = CheckExpression(varNode, tc)
	be.True(t, err != nil)
	be.Equal(t, "error: variable 'x' used before assignment", err.Error())
}

func TestCheckExpressionBinaryArithmetic(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

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
	exprType, err := CheckExpression(binaryNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType)
}

func TestCheckExpressionBinaryComparison(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

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
	exprType, err := CheckExpression(binaryNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType) // Comparison returns I64 (0 or 1)
}

func TestCheckExpressionAddressOf(t *testing.T) {
	st := NewSymbolTable()
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	st.AssignVariable("x")

	tc := NewTypeChecker(st)

	// Create address-of expression: x&
	addrNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "&",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "x"},
		},
	}

	// Check expression
	exprType, err := CheckExpression(addrNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypePointer, exprType.Kind)
	be.Equal(t, TypeI64, exprType.Child)
}

func TestCheckExpressionDereference(t *testing.T) {
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	err := st.DeclareVariable("ptr", ptrType)
	be.Err(t, err, nil)
	st.AssignVariable("ptr")

	tc := NewTypeChecker(st)

	// Create dereference expression: ptr*
	derefNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "ptr"},
		},
	}

	// Check expression
	exprType, err := CheckExpression(derefNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType)
}

func TestCheckExpressionDereferenceNonPointer(t *testing.T) {
	st := NewSymbolTable()
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)
	st.AssignVariable("x")

	tc := NewTypeChecker(st)

	// Create dereference expression: x*
	derefNode := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "x"},
		},
	}

	// Check expression
	_, err = CheckExpression(derefNode, tc)
	be.True(t, err != nil)
	be.Equal(t, "error: cannot dereference non-pointer type I64", err.Error())
}

func TestCheckExpressionFunctionCall(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Create function call: print(42)
	callNode := &ASTNode{
		Kind: NodeCall,
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "print"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	// Check expression
	exprType, err := CheckExpression(callNode, tc)
	be.Err(t, err, nil)
	be.Equal(t, TypeI64, exprType)
}

func TestCheckExpressionUnknownFunction(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Create function call: unknown(42)
	callNode := &ASTNode{
		Kind: NodeCall,
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "unknown"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	// Check expression
	_, err := CheckExpression(callNode, tc)
	be.True(t, err != nil)
	be.Equal(t, "error: unknown function 'unknown'", err.Error())
}

func TestCheckAssignmentValid(t *testing.T) {
	st := NewSymbolTable()
	err := st.DeclareVariable("x", TypeI64)
	be.Err(t, err, nil)

	tc := NewTypeChecker(st)

	// Create assignment nodes: x = 42
	lhs := &ASTNode{Kind: NodeIdent, String: "x"}
	rhs := &ASTNode{Kind: NodeInteger, Integer: 42}

	// Check assignment
	err = CheckAssignment(lhs, rhs, tc)
	be.Err(t, err, nil)

	// Verify variable is now assigned
	symbol := st.LookupVariable("x")
	be.Equal(t, true, symbol.Assigned)
}

func TestCheckAssignmentUndeclaredVariable(t *testing.T) {
	st := NewSymbolTable()
	tc := NewTypeChecker(st)

	// Create assignment nodes: undefined = 42
	lhs := &ASTNode{Kind: NodeIdent, String: "undefined"}
	rhs := &ASTNode{Kind: NodeInteger, Integer: 42}

	// Check assignment
	err := CheckAssignment(lhs, rhs, tc)
	be.True(t, err != nil)
	be.Equal(t, "error: variable 'undefined' used before declaration", err.Error())
}

func TestCheckAssignmentPointerDereference(t *testing.T) {
	st := NewSymbolTable()
	ptrType := &TypeNode{Kind: TypePointer, Child: TypeI64}
	err := st.DeclareVariable("ptr", ptrType)
	be.Err(t, err, nil)
	st.AssignVariable("ptr")

	tc := NewTypeChecker(st)

	// Create assignment nodes: ptr* = 42
	lhs := &ASTNode{
		Kind: NodeUnary,
		Op:   "*",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "ptr"},
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
	st := BuildSymbolTable(ast)
	err := CheckProgram(ast, st)
	be.Err(t, err, nil)
}

func TestCheckProgramError(t *testing.T) {
	// Parse: { var x I64; print(x); } (use before assignment)
	input := []byte("{ var x I64; print(x); }\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Build symbol table and check program
	st := BuildSymbolTable(ast)
	err := CheckProgram(ast, st)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "variable 'x' used before assignment"))
}
