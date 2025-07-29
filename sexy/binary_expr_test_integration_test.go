package sexy

import (
	"os"
	"testing"

	"github.com/nalgeon/be"
)

func TestExtractTestCases_BinaryExprTestMd(t *testing.T) {
	// Read the actual binary_expr_test.md file
	content, err := os.ReadFile("binary_expr_test.md")
	be.Err(t, err, nil)

	// Extract test cases
	testCases, err := ExtractTestCases(string(content))
	be.Err(t, err, nil)

	// We expect multiple test cases from the file
	be.True(t, len(testCases) > 5)

	// Test a few specific cases to verify parsing
	var plusTest *TestCase
	var precedenceTest *TestCase

	for i := range testCases {
		tc := &testCases[i]
		if tc.Name == "+" {
			plusTest = tc
		}
		if tc.Name == "operator precedence + *" {
			precedenceTest = tc
		}
	}

	// Verify the "+" test
	be.True(t, plusTest != nil)
	be.Equal(t, plusTest.Input, "1 + 2")
	be.Equal(t, plusTest.InputType, InputTypeZongExpr)
	be.Equal(t, len(plusTest.Assertions), 1)
	be.Equal(t, plusTest.Assertions[0].Type, AssertionTypeAST)
	be.Equal(t, plusTest.Assertions[0].Content, "(binary \"+\" 1 2)")
	be.Equal(t, plusTest.Assertions[0].ParsedSexy.String(), "(binary \"+\" 1 2)")

	// Verify the precedence test
	be.True(t, precedenceTest != nil)
	be.Equal(t, precedenceTest.Input, "1 + 2 * 3")
	be.Equal(t, precedenceTest.InputType, InputTypeZongExpr)
	be.Equal(t, len(precedenceTest.Assertions), 1)
	be.Equal(t, precedenceTest.Assertions[0].Type, AssertionTypeAST)

	// Verify the parsed structure has the expected format
	assertion := precedenceTest.Assertions[0].ParsedSexy
	be.Equal(t, assertion.Type, NodeList)
	be.Equal(t, len(assertion.Items), 4) // (binary "+" 1 (binary "*" 2 3))
	be.Equal(t, assertion.Items[0].Text, "binary")
	be.Equal(t, assertion.Items[1].Text, "+")
	be.Equal(t, assertion.Items[2].Text, "1")
	be.Equal(t, assertion.Items[3].Type, NodeList) // The nested (binary "*" 2 3)
}

func TestExtractTestCases_AllBinaryExprTests(t *testing.T) {
	// Read the binary_expr_test.md file
	content, err := os.ReadFile("binary_expr_test.md")
	be.Err(t, err, nil)

	// Extract all test cases
	testCases, err := ExtractTestCases(string(content))
	be.Err(t, err, nil)

	// Verify that all test cases have proper structure
	for _, tc := range testCases {
		// Each test should have a name
		be.True(t, tc.Name != "")

		// Each test should have input
		be.True(t, tc.Input != "")
		be.Equal(t, tc.InputType, InputTypeZongExpr)

		// Each test should have at least one assertion
		be.True(t, len(tc.Assertions) >= 1)

		// Each assertion should be parsed successfully
		for _, assertion := range tc.Assertions {
			be.Equal(t, assertion.Type, AssertionTypeAST)
			be.True(t, assertion.Content != "")
			be.True(t, assertion.ParsedSexy != nil)

			// The parsed Sexy should be a valid structure
			be.True(t, assertion.ParsedSexy.Type != NodeType(0))
		}
	}
}
