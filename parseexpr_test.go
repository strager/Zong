package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestParseLiterals(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"42\x00", "(integer 42)"},
		{"\"hello\"\x00", "(string \"hello\")"},
		{"myVar\x00", "(ident \"myVar\")"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseBinaryOperations(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2\x00", "(binary \"+\" (integer 1) (integer 2))"},
		{"x == y\x00", "(binary \"==\" (ident \"x\") (ident \"y\"))"},
		{"\"a\" + \"b\"\x00", "(binary \"+\" (string \"a\") (string \"b\"))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 * 3\x00", "(binary \"+\" (integer 1) (binary \"*\" (integer 2) (integer 3)))"},
		{"(1 + 2) * 3\x00", "(binary \"*\" (binary \"+\" (integer 1) (integer 2)) (integer 3))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseComplexExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x + y * z\x00", "(binary \"+\" (ident \"x\") (binary \"*\" (ident \"y\") (ident \"z\")))"},
		{"a == b + c\x00", "(binary \"==\" (ident \"a\") (binary \"+\" (ident \"b\") (ident \"c\")))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseAdditionalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"5 - 3\x00", "(binary \"-\" (integer 5) (integer 3))"},
		{"8 / 2\x00", "(binary \"/\" (integer 8) (integer 2))"},
		{"10 % 3\x00", "(binary \"%\" (integer 10) (integer 3))"},
		{"a != b\x00", "(binary \"!=\" (ident \"a\") (ident \"b\"))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseNestedParentheses(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"((1 + 2))\x00", "(binary \"+\" (integer 1) (integer 2))"},
		{"(x + y) * (a - b)\x00", "(binary \"*\" (binary \"+\" (ident \"x\") (ident \"y\")) (binary \"-\" (ident \"a\") (ident \"b\")))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseMixedOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 + 3\x00", "(binary \"+\" (binary \"+\" (integer 1) (integer 2)) (integer 3))"},
		{"2 * 3 * 4\x00", "(binary \"*\" (binary \"*\" (integer 2) (integer 3)) (integer 4))"},
		{"1 + 2 * 3 + 4\x00", "(binary \"+\" (binary \"+\" (integer 1) (binary \"*\" (integer 2) (integer 3))) (integer 4))"},
		{"a == b + c * d\x00", "(binary \"==\" (ident \"a\") (binary \"+\" (ident \"b\") (binary \"*\" (ident \"c\") (ident \"d\"))))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseFunctionCalls(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"f()\x00", "(call (ident \"f\"))"},
		{"print(\"hello\")\x00", "(call (ident \"print\") (string \"hello\"))"},
		{"atan2(y, x)\x00", "(call (ident \"atan2\") (ident \"y\") (ident \"x\"))"},
		{"Point(x: 1, y: 2)\x00", "(call (ident \"Point\") \"x\" (integer 1) \"y\" (integer 2))"},
		{"httpGet(\"http://example.com\", headers: h)\x00", "(call (ident \"httpGet\") (string \"http://example.com\") \"headers\" (ident \"h\"))"},
		{"(foo)()\x00", "(call (ident \"foo\"))"},
		{"arr[0](x)\x00", "(call (idx (ident \"arr\") (integer 0)) (ident \"x\"))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseSubscript(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x[y]\x00", "(idx (ident \"x\") (ident \"y\"))"},
		{"arr[0]\x00", "(idx (ident \"arr\") (integer 0))"},
		{"matrix[i][j]\x00", "(idx (idx (ident \"matrix\") (ident \"i\")) (ident \"j\"))"},
		{"items[x + 1]\x00", "(idx (ident \"items\") (binary \"+\" (ident \"x\") (integer 1)))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseUnaryNot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"!x\x00", "(unary \"!\" (ident \"x\"))"},
		{"!true\x00", "(unary \"!\" (boolean true))"},
		{"!!x\x00", "(unary \"!\" (unary \"!\" (ident \"x\")))"},
		{"!(x == y)\x00", "(unary \"!\" (binary \"==\" (ident \"x\") (ident \"y\")))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseComplexExpressionsCombined(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"f(x)[0]\x00", "(idx (call (ident \"f\") (ident \"x\")) (integer 0))"},
		{"!arr[i]\x00", "(unary \"!\" (idx (ident \"arr\") (ident \"i\")))"},
		// TODO: Fix parsing complex expressions in named parameters
		// {"func(a: 1 + 2)\x00", "(call (ident \"func\") \"a\" (binary \"+\" (integer 1) (integer 2)))"},
		{"x[y] + z\x00", "(binary \"+\" (idx (ident \"x\") (ident \"y\")) (ident \"z\"))"},
		{"!f() == true\x00", "(binary \"==\" (unary \"!\" (call (ident \"f\"))) (boolean true))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseAddressOfOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x&\x00", "(unary \"&\" (ident \"x\"))"},
		{"(x + y)&\x00", "(unary \"&\" (binary \"+\" (ident \"x\") (ident \"y\")))"},
		{"arr[0]&\x00", "(unary \"&\" (idx (ident \"arr\") (integer 0)))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestParseDereferenceOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ptr*\x00", "(unary \"*\" (ident \"ptr\"))"},
		{"(ptr + 1)*\x00", "(unary \"*\" (binary \"+\" (ident \"ptr\") (integer 1)))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestPointerOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x& + 1\x00", "(binary \"+\" (unary \"&\" (ident \"x\")) (integer 1))"},
		{"1 + x&\x00", "(binary \"+\" (integer 1) (unary \"&\" (ident \"x\")))"},
		{"(x + 1)&\x00", "(unary \"&\" (binary \"+\" (ident \"x\") (integer 1)))"},
		{"ptr* + 1\x00", "(binary \"+\" (unary \"*\" (ident \"ptr\")) (integer 1))"},
		{"1 + ptr*\x00", "(binary \"+\" (integer 1) (unary \"*\" (ident \"ptr\")))"},
		{"(ptr + 1)*\x00", "(unary \"*\" (binary \"+\" (ident \"ptr\") (integer 1)))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

func TestComplexPointerExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"x&*\x00", "(unary \"*\" (unary \"&\" (ident \"x\")))"},
		{"x*&\x00", "(unary \"&\" (unary \"*\" (ident \"x\")))"},
		{"ptr*& + 1\x00", "(binary \"+\" (unary \"&\" (unary \"*\" (ident \"ptr\"))) (integer 1))"},
		{"arr[0]&*\x00", "(unary \"*\" (unary \"&\" (idx (ident \"arr\") (integer 0))))"},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		ast := ParseExpression()
		result := ToSExpr(ast)

		be.Equal(t, result, test.expected)
	}
}

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

// Tests for additional edge cases in parsing
func TestParseExpressionRightAssociativity(t *testing.T) {
	// Test right-associativity of assignment operator
	input := []byte("a = b = c\x00")
	Init(input)
	NextToken()

	result := ParseExpression()
	be.True(t, result != nil)

	// Verify the structure represents right-associativity: a = (b = c)
	be.Equal(t, NodeBinary, result.Kind)
	be.Equal(t, "=", result.Op)

	be.True(t, result.Children != nil)
	be.Equal(t, 2, len(result.Children))

	// Right child should be another assignment
	rightChild := result.Children[1]
	be.Equal(t, NodeBinary, rightChild.Kind)
	be.Equal(t, "=", rightChild.Op)
}

func TestParseExpressionOperatorPrecedence(t *testing.T) {
	// Test operator precedence: multiplication before addition
	input := []byte("a + b * c\x00")
	Init(input)
	NextToken()

	result := ParseExpression()
	be.True(t, result != nil)

	// Should parse as: a + (b * c)
	be.Equal(t, NodeBinary, result.Kind)
	be.Equal(t, "+", result.Op)

	be.True(t, result.Children != nil)
	be.Equal(t, 2, len(result.Children))

	// Right child should be multiplication
	rightChild := result.Children[1]
	be.Equal(t, NodeBinary, rightChild.Kind)
	be.Equal(t, "*", rightChild.Op)
}

// Test for handling pointer dereference expressions
func TestParseExpressionPointerDereference(t *testing.T) {
	input := []byte("ptr*\x00") // Postfix dereference, not prefix
	Init(input)
	NextToken()

	result := ParseExpression()
	be.True(t, result != nil)
	if result == nil {
		return
	}

	be.Equal(t, NodeUnary, result.Kind)
	be.Equal(t, "*", result.Op)
}

// Test for handling address-of expressions
func TestParseExpressionAddressOf(t *testing.T) {
	input := []byte("x&\x00") // Postfix address-of, not prefix
	Init(input)
	NextToken()

	result := ParseExpression()
	be.True(t, result != nil)
	if result == nil {
		return
	}

	be.Equal(t, NodeUnary, result.Kind)
	be.Equal(t, "&", result.Op)
}
