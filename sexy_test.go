package main

import (
	"os"
	"testing"

	"github.com/nalgeon/be"
	"github.com/strager/zong/sexy"
)

func TestSexyBinaryExpressions(t *testing.T) {
	// Read the binary_expr_test.md file
	content, err := os.ReadFile("test/binary_expr_test.md")
	be.Err(t, err, nil)

	// Extract test cases
	testCases, err := sexy.ExtractTestCases(string(content))
	be.Err(t, err, nil)

	// Generate a subtest for each test case
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Parse the Zong expression
			input := tc.Input + "\x00" // Null-terminate as required by Zong parser
			Init([]byte(input))
			NextToken()
			ast := ParseExpression()

			// For each assertion, match the AST against the Sexy pattern
			for i, assertion := range tc.Assertions {
				t.Run("assertion_"+string(rune('a'+i)), func(t *testing.T) {
					if assertion.Type == sexy.AssertionTypeAST {
						assertPatternMatch(t, ast, assertion.ParsedSexy, "root")
					}
				})
			}
		})
	}
}

// assertPatternMatch recursively matches a Zong AST node against a Sexy pattern,
// providing detailed error messages with path information when mismatches occur.
func assertPatternMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if zongAST == nil && sexyPattern == nil {
		return
	}

	if zongAST == nil {
		t.Errorf("At %s: expected AST node, got nil", path)
		return
	}

	if sexyPattern == nil {
		t.Errorf("At %s: expected nil, got AST node %v", path, zongAST.Kind)
		return
	}

	// Handle different Zong AST node types and their corresponding Sexy patterns
	switch zongAST.Kind {
	case NodeInteger:
		assertIntegerMatch(t, zongAST, sexyPattern, path)
	case NodeIdent:
		assertIdentMatch(t, zongAST, sexyPattern, path)
	case NodeString:
		assertStringMatch(t, zongAST, sexyPattern, path)
	case NodeBinary:
		assertBinaryMatch(t, zongAST, sexyPattern, path)
	default:
		t.Errorf("At %s: unsupported Zong AST node type: %v", path, zongAST.Kind)
	}
}

// assertIntegerMatch matches Zong NodeInteger against Sexy NodeInteger
func assertIntegerMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeInteger {
		t.Errorf("At %s: expected Sexy integer, got %v", path, sexyPattern.Type)
		return
	}

	expectedValue := sexyPattern.Text
	actualValue := intToString(zongAST.Integer)

	be.Equal(t, actualValue, expectedValue, "At %s: integer value mismatch", path)
}

// assertStringMatch matches Zong NodeString against Sexy patterns
// Sexy patterns for strings are lists like (string "value")
func assertStringMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for string literal, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (string \"value\") pattern with 2 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "string"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "string" {
		t.Errorf("At %s: expected 'string' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the string value
	if sexyPattern.Items[1].Type != sexy.NodeString {
		t.Errorf("At %s: expected string for string value, got %v", path, sexyPattern.Items[1].Type)
		return
	}

	be.Equal(t, zongAST.String, sexyPattern.Items[1].Text, "At %s: string value mismatch", path)
}

// assertIdentMatch matches Zong NodeIdent (variable references) against Sexy patterns
// In Sexy patterns, variable references appear as (var "name") lists
func assertIdentMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	// For variable references like "x" in Zong, the Sexy pattern should be (var "x")
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for variable reference, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (var \"name\") pattern with 2 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "var"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "var" {
		t.Errorf("At %s: expected 'var' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the variable name as a string
	if sexyPattern.Items[1].Type != sexy.NodeString {
		t.Errorf("At %s: expected string for variable name, got %v", path, sexyPattern.Items[1].Type)
		return
	}

	be.Equal(t, zongAST.String, sexyPattern.Items[1].Text, "At %s: variable name mismatch", path)
}

// assertBinaryMatch matches Zong NodeBinary against Sexy NodeList patterns
// Zong binary expressions map to Sexy patterns like (binary "+" left right)
func assertBinaryMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for binary expression, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 4 {
		t.Errorf("At %s: expected (binary \"op\" left right) pattern with 4 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "binary"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "binary" {
		t.Errorf("At %s: expected 'binary' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the operator as a string
	if sexyPattern.Items[1].Type != sexy.NodeString {
		t.Errorf("At %s: expected string for operator, got %v", path, sexyPattern.Items[1].Type)
		return
	}

	be.Equal(t, zongAST.Op, sexyPattern.Items[1].Text, "At %s: operator mismatch", path)

	// Check that we have the expected number of children
	if len(zongAST.Children) != 2 {
		t.Errorf("At %s: expected 2 children for binary expression, got %d", path, len(zongAST.Children))
		return
	}

	// Recursively match left child (third item in pattern)
	leftPattern := sexyPattern.Items[2]
	rightPattern := sexyPattern.Items[3]

	// Handle simple atoms (integers, strings, variables) vs nested lists
	assertPatternMatch(t, zongAST.Children[0], leftPattern, path+".left")
	assertPatternMatch(t, zongAST.Children[1], rightPattern, path+".right")
}
