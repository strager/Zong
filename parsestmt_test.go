package main

import "testing"

func TestParseIfStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "if x { y; }\x00",
			expected: "(if (ident \"x\") (ident \"y\"))",
		},
		{
			input:    "if 1 + 2 { 3; }\x00",
			expected: "(if (binary \"+\" (integer 1) (integer 2)) (integer 3))",
		},
		{
			input:    "if foo == bar { return 42; }\x00",
			expected: "(if (binary \"==\" (ident \"foo\") (ident \"bar\")) (return (integer 42)))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

func TestParseVarStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "var x int;\x00",
			expected: "(var (ident \"x\") (ident \"int\"))",
		},
		{
			input:    "var name string;\x00",
			expected: "(var (ident \"name\") (ident \"string\"))",
		},
		{
			input:    "var count MyType;\x00",
			expected: "(var (ident \"count\") (ident \"MyType\"))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

func TestParseBlockStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "{ }\x00",
			expected: "(block)",
		},
		{
			input:    "{ x; }\x00",
			expected: "(block (ident \"x\"))",
		},
		{
			input:    "{ 1; 2; }\x00",
			expected: "(block (integer 1) (integer 2))",
		},
		{
			input:    "{ var x int; return x; }\x00",
			expected: "(block (var (ident \"x\") (ident \"int\")) (return (ident \"x\")))",
		},
		{
			input:    "{ { } }\x00",
			expected: "(block (block))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

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
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

func TestParseLoopStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "loop { }\x00",
			expected: "(loop)",
		},
		{
			input:    "loop { x; }\x00",
			expected: "(loop (ident \"x\"))",
		},
		{
			input:    "loop { break; }\x00",
			expected: "(loop (break))",
		},
		{
			input:    "loop { continue; }\x00",
			expected: "(loop (continue))",
		},
		{
			input:    "loop { var i int; if i == 10 { break; } }\x00",
			expected: "(loop (var (ident \"i\") (ident \"int\")) (if (binary \"==\" (ident \"i\") (integer 10)) (break)))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
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
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
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
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

func TestParseExpressionStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "x;\x00",
			expected: "(ident \"x\")",
		},
		{
			input:    "42;\x00",
			expected: "(integer 42)",
		},
		{
			input:    "a + b;\x00",
			expected: "(binary \"+\" (ident \"a\") (ident \"b\"))",
		},
		{
			input:    "x * y + z;\x00",
			expected: "(binary \"+\" (binary \"*\" (ident \"x\") (ident \"y\")) (ident \"z\"))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}

func TestComplexStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "if x > 0 { var y int; return y + 1; }\x00",
			expected: "(if (binary \">\" (ident \"x\") (integer 0)) (var (ident \"y\") (ident \"int\")) (return (binary \"+\" (ident \"y\") (integer 1))))",
		},
		{
			input:    "loop { if done { break; } continue; }\x00",
			expected: "(loop (if (ident \"done\") (break)) (continue))",
		},
		{
			input:    "{ if a { { b; } } }\x00",
			expected: "(block (if (ident \"a\") (block (ident \"b\"))))",
		},
	}

	for _, test := range tests {
		Init([]byte(test.input))
		NextToken()
		result := ParseStatement()
		actual := ToSExpr(result)
		if actual != test.expected {
			t.Errorf("Input: %q\nExpected: %s\nActual: %s", test.input, test.expected, actual)
		}
	}
}
