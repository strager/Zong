package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// TestParseLiterals removed - duplicates test/expressions_test.md

// TestParseBinaryOperations removed - duplicates test/binary_expr_test.md

// TestParseOperatorPrecedence removed - duplicates test/binary_expr_test.md

// TestParseComplexExpressions removed - duplicates test/expressions_test.md

// TestParseAdditionalOperators removed - duplicates test/binary_expr_test.md and test/expressions_test.md

// TestParseNestedParentheses removed - duplicates test/expressions_test.md

// TestParseMixedOperatorPrecedence removed - duplicates test/binary_expr_test.md

// TestParseFunctionCalls removed - duplicates test/expressions_test.md

// TestParseSubscript removed - duplicates test/expressions_test.md

// TestParseUnaryNot removed - duplicates test/expressions_test.md

// TestParseComplexExpressionsCombined removed - duplicates test/expressions_test.md

// TestParseAddressOfOperator removed - duplicates test/expressions_test.md

// TestParseDereferenceOperator removed - duplicates test/expressions_test.md

// TestPointerOperatorPrecedence removed - duplicates test/expressions_test.md

// TestComplexPointerExpressions removed - duplicates test/expressions_test.md

// Tests for parser edge cases
func TestParseExpressionMalformedFunctionCall(t *testing.T) {
	// Test function call parsing with malformed arguments
	input := []byte("func(arg1 arg2\x00") // Missing comma between args
	Init(input)
	NextToken()

	// Should handle malformed function call gracefully without panic
	result := ParseExpression()
	be.True(t, result != nil)
}

func TestParsePrimaryUnknownToken(t *testing.T) {
	// Test parsing with unexpected token types
	input := []byte("{\x00") // LBRACE is not handled by parsePrimary directly
	Init(input)
	NextToken()

	result := parsePrimary()
	// Should handle unknown tokens gracefully
	be.True(t, result != nil)
}

func TestParseTypeExpressionNonIdentToken(t *testing.T) {
	// Test type parsing with non-identifier token
	input := []byte("123\x00") // INT token instead of IDENT
	Init(input)
	NextToken()

	result := parseTypeExpression()
	be.Equal(t, nil, result)
}

// TestParseExpressionRightAssociativity removed - now covered by test/parsing_comprehensive_test.md

// TestParseExpressionOperatorPrecedence removed - duplicates test/expressions_test.md and test/binary_expr_test.md

// TestParseExpressionPointerDereference removed - duplicates test/expressions_test.md

// TestParseExpressionAddressOf removed - duplicates test/expressions_test.md
