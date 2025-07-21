package main

import (
	"testing"
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
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

		if result != test.expected {
			t.Errorf("Input: %s, Expected: %s, Got: %s", test.input, test.expected, result)
		}
	}
}
