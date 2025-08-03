// Cross-cutting integration tests via Sexy framework
//
// Tests end-to-end compiler behavior using declarative S-expression patterns.
// Runs tests from test/*.md files covering AST validation, execution, and error testing.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nalgeon/be"
	"github.com/strager/zong/sexy"
)

func TestSexyAllTests(t *testing.T) {
	// Find all test files in the test/ directory
	testFiles, err := filepath.Glob("test/*_test.md")
	be.Err(t, err, nil)

	// Run tests for each file
	for _, testFile := range testFiles {
		// Extract a clean test name from the file path
		fileName := filepath.Base(testFile)
		testName := strings.TrimSuffix(fileName, ".md")

		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			// Read the test file
			content, err := os.ReadFile(testFile)
			be.Err(t, err, nil)

			// Extract test cases
			testCases, err := sexy.ExtractTestCases(string(content))
			be.Err(t, err, nil)

			// Generate a subtest for each test case
			for _, tc := range testCases {
				t.Run(tc.Name, func(t *testing.T) {
					t.Parallel()
					// Parse the Zong input based on input type, but skip for compile-error tests
					// since they handle their own parsing with error recovery
					var ast *ASTNode
					hasCompileErrorAssertion := false
					for _, assertion := range tc.Assertions {
						if assertion.Type == sexy.AssertionTypeCompileError {
							hasCompileErrorAssertion = true
							break
						}
					}

					if !hasCompileErrorAssertion {
						input := tc.Input + "\x00" // Null-terminate as required by Zong parser
						l := NewLexer([]byte(input))
						l.NextToken()

						switch tc.InputType {
						case sexy.InputTypeZongExpr:
							ast = ParseExpression(l)
						case sexy.InputTypeZongProgram:
							ast = ParseProgram(l)
						default:
							t.Fatalf("Unknown input type: %s", tc.InputType)
						}
					}

					// For each assertion, match the AST against the Sexy pattern or execute the code
					for i, assertion := range tc.Assertions {
						t.Run("assertion_"+string(rune('a'+i)), func(t *testing.T) {
							t.Parallel()
							if assertion.Type == sexy.AssertionTypeAST {
								assertPatternMatch(t, ast, assertion.ParsedSexy, "root")
							} else if assertion.Type == sexy.AssertionTypeExecute {
								assertExecutionMatch(t, ast, assertion.Content, tc.InputType)
							} else if assertion.Type == sexy.AssertionTypeCompileError {
								assertCompileErrorMatch(t, tc.Input, assertion.Content, tc.InputType)
							} else if assertion.Type == sexy.AssertionTypeWasmLocals {
								assertWasmLocalsMatch(t, ast, assertion.ParsedSexy)
							}
						})
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
	case NodeBoolean:
		assertBooleanMatch(t, zongAST, sexyPattern, path)
	case NodeBinary:
		assertBinaryMatch(t, zongAST, sexyPattern, path)
	case NodeUnary:
		assertUnaryMatch(t, zongAST, sexyPattern, path)
	case NodeCall:
		assertCallMatch(t, zongAST, sexyPattern, path)
	case NodeIndex:
		assertIndexMatch(t, zongAST, sexyPattern, path)
	case NodeDot:
		assertDotMatch(t, zongAST, sexyPattern, path)
	case NodeVar:
		assertVarMatch(t, zongAST, sexyPattern, path)
	case NodeIf:
		assertIfMatch(t, zongAST, sexyPattern, path)
	case NodeBlock:
		assertBlockMatch(t, zongAST, sexyPattern, path)
	case NodeReturn:
		assertReturnMatch(t, zongAST, sexyPattern, path)
	case NodeFunc:
		assertFuncMatch(t, zongAST, sexyPattern, path)
	case NodeStruct:
		assertStructMatch(t, zongAST, sexyPattern, path)
	case NodeLoop:
		assertLoopMatch(t, zongAST, sexyPattern, path)
	case NodeBreak:
		assertBreakMatch(t, zongAST, sexyPattern, path)
	case NodeContinue:
		assertContinueMatch(t, zongAST, sexyPattern, path)
	default:
		t.Errorf("At %s: unsupported Zong AST node type: %v", path, zongAST.Kind)
	}
}

// assertIntegerMatch matches Zong NodeInteger against Sexy NodeInteger
func assertIntegerMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeInteger {
		t.Errorf("At %s: expected integer, got type %v", path, sexyPattern.Type)
		return
	}
	actualValue := intToString(zongAST.Integer)
	if actualValue != sexyPattern.Text {
		t.Errorf("At %s: expected integer %s, got %v", path, actualValue, sexyPattern.Text)
	}
}

// assertBooleanMatch matches Zong NodeBoolean against Sexy patterns
// Sexy patterns for booleans are lists like true or false
func assertBooleanMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeSymbol {
		t.Errorf("At %s: expected symbol for boolean value, got type %v", path, sexyPattern.Items[1].Type)
		return
	}
	actualValue := "false"
	if zongAST.Boolean {
		actualValue = "true"
	}
	if actualValue != sexyPattern.Text {
		t.Errorf("At %s: expected %s, got %v", path, sexyPattern.Text, actualValue)
		return
	}
}

// assertStringMatch matches Zong NodeString against Sexy patterns
// Sexy patterns for strings are lists like (string "value")
func assertStringMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for string literal, got type %v", path, sexyPattern.Type)
		return
	}
	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (string \"value\") pattern with 2 items, got %d", path, len(sexyPattern.Items))
		return
	}
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "string" {
		t.Errorf("At %s: expected 'string' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}
	expected := sexyPattern.Items[1]
	if expected.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for string value, got type %v", path, expected.Type)
		return
	}
	if expected.Text != zongAST.String {
		t.Errorf("At %s: expected string %#v, got %#v", path, expected.Text, zongAST.String)
		return
	}
}

// assertIdentMatch matches Zong NodeIdent (variable references) against Sexy patterns
// In Sexy patterns, variable references appear as (var "name") lists
func assertIdentMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	// For variable references like "x" in Zong, the Sexy pattern should be (var "x")
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for variable reference, got type %v", path, sexyPattern.Type)
		return
	}
	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (var \"name\") pattern with 2 items, got %d", path, len(sexyPattern.Items))
		return
	}
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "var" {
		t.Errorf("At %s: expected 'var' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}
	expected := sexyPattern.Items[1]
	if expected.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for variable name, got %v", path, expected.Type)
		return
	}
	if expected.Text != zongAST.String {
		t.Errorf("At %s: expected variable name %v, got %v", path, expected.Text, zongAST.String)
		return
	}
}

// assertBinaryMatch matches Zong NodeBinary against Sexy NodeList patterns
// Zong binary expressions map to Sexy patterns like (binary "+" left right)
func assertBinaryMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for binary expression, got type %v", path, sexyPattern.Type)
		return
	}
	if len(sexyPattern.Items) != 4 {
		t.Errorf("At %s: expected (binary \"op\" left right) pattern with 4 items, got %d", path, len(sexyPattern.Items))
		return
	}
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "binary" {
		t.Errorf("At %s: expected 'binary' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}
	expectedOp := sexyPattern.Items[1]
	if expectedOp.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for operator, got %v", path, expectedOp.Type)
		return
	}
	if expectedOp.Text != zongAST.Op {
		t.Errorf("At %s: expected operator %s, got %s", path, expectedOp.Text, zongAST.Op)
		return
	}
	if len(zongAST.Children) != 2 {
		t.Errorf("At %s: expected 2 children for binary expression, got %d", path, len(zongAST.Children))
		return
	}
	assertPatternMatch(t, zongAST.Children[0], sexyPattern.Items[2], path+".left")
	assertPatternMatch(t, zongAST.Children[1], sexyPattern.Items[3], path+".right")
}

// assertUnaryMatch matches Zong NodeUnary against Sexy NodeList patterns
// Zong unary expressions map to Sexy patterns like (unary "!" operand)
func assertUnaryMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for unary expression, got type %v", path, sexyPattern.Type)
		return
	}
	if len(sexyPattern.Items) != 3 {
		t.Errorf("At %s: expected (unary \"op\" operand) pattern with 3 items, got %d", path, len(sexyPattern.Items))
		return
	}
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "unary" {
		t.Errorf("At %s: expected 'unary' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}
	expectedOp := sexyPattern.Items[1]
	if expectedOp.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for operator, got %v", path, expectedOp.Type)
		return
	}
	if zongAST.Op != expectedOp.Text {
		t.Errorf("At %s: expected operator %s, got %s", path, expectedOp.Text, zongAST.Op)
		return
	}
	if len(zongAST.Children) != 1 {
		t.Errorf("At %s: expected 1 child for unary expression, got %d", path, len(zongAST.Children))
		return
	}
	assertPatternMatch(t, zongAST.Children[0], sexyPattern.Items[2], path+".operand")
}

// assertCallMatch matches Zong NodeCall against Sexy NodeList patterns
// Zong function calls map to Sexy patterns like (call function_expr arg1 arg2 ...)
// For named parameters: (call function_expr "param1" value1 "param2" value2)
func assertCallMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for function call, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) < 2 {
		t.Errorf("At %s: expected (call function ...) pattern with at least 2 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "call"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "call" {
		t.Errorf("At %s: expected 'call' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the function expression (usually a variable)
	functionPattern := sexyPattern.Items[1]
	if len(zongAST.Children) == 0 {
		t.Errorf("At %s: expected at least 1 child for function call (the function), got %d", path, len(zongAST.Children))
		return
	}
	assertPatternMatch(t, zongAST.Children[0], functionPattern, path+".function")

	// Build expected pattern by interleaving parameter names and values
	expectedItems := sexyPattern.Items[2:] // Skip "call" and function
	actualArgs := zongAST.Children[1:]     // Skip function

	// Build the actual pattern that ToSExpr would generate
	var actualPattern []interface{}
	for i, arg := range actualArgs {
		// Add parameter name if it exists
		if i < len(zongAST.ParameterNames) && zongAST.ParameterNames[i] != "" {
			actualPattern = append(actualPattern, zongAST.ParameterNames[i])
		}
		actualPattern = append(actualPattern, arg)
	}

	if len(expectedItems) != len(actualPattern) {
		t.Errorf("At %s: expected %d items in pattern, got %d items in actual pattern",
			path, len(expectedItems), len(actualPattern))
		return
	}

	// Match the interleaved pattern
	for i, expectedItem := range expectedItems {
		actualItem := actualPattern[i]

		if actualArg, ok := actualItem.(*ASTNode); ok {
			// This is an argument value - match it
			assertPatternMatch(t, actualArg, expectedItem, path+".item"+string(rune('0'+i)))
		} else if paramName, ok := actualItem.(string); ok {
			// This is a parameter name - check that expected item is a string with matching value
			if expectedItem.Type != sexy.NodeString {
				t.Errorf("At %s.item%d: expected string for parameter name, got %v", path, i, expectedItem.Type)
				return
			}
			if expectedItem.Text != paramName {
				t.Errorf("At %s.item%d: expected parameter name '%s', got '%s'", path, i, expectedItem.Text, paramName)
				return
			}
		}
	}
}

// assertIndexMatch matches Zong NodeIndex against Sexy NodeList patterns
// Zong array/slice subscripts map to Sexy patterns like (idx array_expr index_expr)
func assertIndexMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for index expression, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 3 {
		t.Errorf("At %s: expected (idx array index) pattern with 3 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "idx"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "idx" {
		t.Errorf("At %s: expected 'idx' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Check that we have the expected number of children
	if len(zongAST.Children) != 2 {
		t.Errorf("At %s: expected 2 children for index expression, got %d", path, len(zongAST.Children))
		return
	}

	// Recursively match array expression (second item in pattern)
	arrayPattern := sexyPattern.Items[1]
	indexPattern := sexyPattern.Items[2]

	assertPatternMatch(t, zongAST.Children[0], arrayPattern, path+".array")
	assertPatternMatch(t, zongAST.Children[1], indexPattern, path+".index")
}

// assertDotMatch matches Zong NodeDot against Sexy NodeList patterns
// Zong struct field access maps to Sexy patterns like (dot struct_expr "field_name")
func assertDotMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for dot expression, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 3 {
		t.Errorf("At %s: expected (dot struct \"field\") pattern with 3 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "dot"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "dot" {
		t.Errorf("At %s: expected 'dot' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Third item should be the field name as a string
	if sexyPattern.Items[2].Type != sexy.NodeString {
		t.Errorf("At %s: expected string for field name, got %v", path, sexyPattern.Items[2].Type)
		return
	}

	// Check that we have the expected number of children
	if len(zongAST.Children) != 1 {
		t.Errorf("At %s: expected 1 child for dot expression, got %d", path, len(zongAST.Children))
		return
	}

	// Match the field name
	if zongAST.FieldName != sexyPattern.Items[2].Text {
		t.Errorf("At %s: field name mismatch, expected %s, got %s", path, sexyPattern.Items[2].Text, zongAST.FieldName)
		return
	}

	// Recursively match struct expression (second item in pattern)
	structPattern := sexyPattern.Items[1]
	assertPatternMatch(t, zongAST.Children[0], structPattern, path+".struct")
}

// assertVarMatch matches Zong NodeVar against Sexy NodeList patterns
// Zong variable declarations map to Sexy patterns like (var var_name type_expr)
// The type comes from TypeAST field, not from children
func assertVarMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for var declaration, got %v", path, sexyPattern.Type)
		return
	}

	expectedItems := 3 // (var name type)
	if len(zongAST.Children) > 1 {
		expectedItems = 4 // (var name type init_expr) for initialized variables
	}

	if len(sexyPattern.Items) != expectedItems {
		t.Errorf("At %s: expected (var name type [init]) pattern with %d items, got %d", path, expectedItems, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "var-decl"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "var-decl" {
		t.Errorf("At %s: expected 'var-decl' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Check that we have at least the variable name
	if len(zongAST.Children) < 1 {
		t.Errorf("At %s: expected at least 1 child for var declaration (variable name), got %d", path, len(zongAST.Children))
		return
	}

	// Match variable name (first child)
	namePattern := sexyPattern.Items[1]
	if namePattern.Type != sexy.NodeString {
		t.Errorf("At %s: expected var pattern \"var_name\", got %v", path, namePattern.Type)
		return
	}
	if zongAST.Children[0].String != namePattern.Text {
		t.Errorf("At %s: var name mismatch, expected %s, got %s", path, namePattern.Text, zongAST.Children[0].String)
		return
	}

	// Match type (from TypeAST field, represented as "type_string")
	typePattern := sexyPattern.Items[2]
	if typePattern.Type != sexy.NodeString {
		t.Errorf("At %s: expected type pattern \"type_name\", got %v", path, typePattern.Type)
		return
	}

	if zongAST.TypeAST == nil {
		t.Errorf("At %s: variable declaration missing TypeAST", path)
		return
	}

	actualTypeName := TypeToString(zongAST.TypeAST)
	expectedTypeName := typePattern.Text
	if actualTypeName != expectedTypeName {
		t.Errorf("At %s: type name mismatch, expected %s, got %s", path, expectedTypeName, actualTypeName)
		return
	}

	// Match initialization expression if present
	if len(zongAST.Children) > 1 {
		initPattern := sexyPattern.Items[3]
		assertPatternMatch(t, zongAST.Children[1], initPattern, path+".init")
	}
}

// assertBlockMatch matches Zong NodeBlock against Sexy patterns
// Zong blocks can map to:
// - (block [stmt1 stmt2 ...]) for explicit blocks
// - [stmt1 stmt2 ...] for program-level statement lists
func assertBlockMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	var expectedStmts *sexy.Node

	if sexyPattern.Type == sexy.NodeArray && path == "root" {
		// Program-level pattern: [stmt1 stmt2 ...]
		expectedStmts = sexyPattern
	} else if sexyPattern.Type == sexy.NodeList {
		// Explicit block pattern: (block [stmt1 stmt2 ...])
		if len(sexyPattern.Items) != 2 {
			t.Errorf("At %s: expected (block [...]) pattern, got %d", path, len(sexyPattern.Items))
			return
		}

		// First item should be the symbol "block"
		if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "block" {
			t.Errorf("At %s: expected 'block' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
			return
		}

		expectedStmts = sexyPattern.Items[1]
	} else {
		t.Errorf("At %s: expected Sexy array or list for block, got %v", path, sexyPattern.Type)
		return
	}

	// Match the statements
	assertNodeArray(t, zongAST.Children, expectedStmts, path)
}

// assertReturnMatch matches Zong NodeReturn against Sexy NodeList patterns
// Zong return statements map to Sexy patterns like (return expr) or (return) for void
func assertReturnMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for return statement, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) < 1 || len(sexyPattern.Items) > 2 {
		t.Errorf("At %s: expected (return [expr]) pattern with 1-2 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "return"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "return" {
		t.Errorf("At %s: expected 'return' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Match return expression if present
	if len(sexyPattern.Items) == 2 {
		if len(zongAST.Children) != 1 {
			t.Errorf("At %s: expected 1 child for return with expression, got %d", path, len(zongAST.Children))
			return
		}
		exprPattern := sexyPattern.Items[1]
		assertPatternMatch(t, zongAST.Children[0], exprPattern, path+".expr")
	} else {
		if len(zongAST.Children) != 0 {
			t.Errorf("At %s: expected 0 children for void return, got %d", path, len(zongAST.Children))
			return
		}
	}
}

// assertIfMatch matches Zong NodeIf against Sexy NodeList patterns
// Zong if statements map to Sexy patterns like:
// - (if condition then_stmts) for simple if
// - (if condition then_stmts else_condition else_stmts) for if-else chains
// - (if condition then_stmts nil else_stmts) for if-else without else-if
func assertIfMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for if statement, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) < 3 {
		t.Errorf("At %s: expected (if condition then_stmts ...) pattern with at least 3 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "if"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "if" {
		t.Errorf("At %s: expected 'if' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Check that we have the expected number of children (condition + then_stmts + optional else parts)
	if len(zongAST.Children) < 2 {
		t.Errorf("At %s: expected at least 2 children for if statement (condition + then_stmts), got %d", path, len(zongAST.Children))
		return
	}

	// Match condition (second item in pattern)
	conditionPattern := sexyPattern.Items[1]
	assertPatternMatch(t, zongAST.Children[0], conditionPattern, path+".condition")

	// Match then statements (third item in pattern)
	thenPattern := sexyPattern.Items[2]
	if thenPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array for then statements, got %v (pattern: %v)", path, thenPattern.Type, thenPattern.String())
		return
	}

	// Then statements are in the array pattern
	expectedThenStmts := thenPattern.Items

	// In the AST, the then statements are in a block at Children[1]
	if len(zongAST.Children) < 2 {
		t.Errorf("At %s: expected at least 2 children for if statement (condition + then block), got %d", path, len(zongAST.Children))
		return
	}

	thenBlock := zongAST.Children[1]
	if thenBlock.Kind != NodeBlock {
		t.Errorf("At %s: expected NodeBlock for then statements, got %v", path, thenBlock.Kind)
		return
	}

	// The then statements are the children of the block
	thenStmts := thenBlock.Children

	if len(expectedThenStmts) != len(thenStmts) {
		t.Errorf("At %s: expected %d then statements, got %d", path, len(expectedThenStmts), len(thenStmts))
		return
	}

	for i, expectedStmt := range expectedThenStmts {
		actualStmt := thenStmts[i]
		assertPatternMatch(t, actualStmt, expectedStmt, path+".then"+string(rune('0'+i)))
	}

	// Handle else-if and else parts if present
	if len(sexyPattern.Items) > 3 {
		// For simple if-else: (if condition then_stmts nil else_block)
		if len(sexyPattern.Items) == 5 &&
			sexyPattern.Items[3].Type == sexy.NodeSymbol &&
			sexyPattern.Items[3].Text == "nil" {

			// Simple if-else pattern
			elsePattern := sexyPattern.Items[4]

			// In the AST, the else block is at Children[3] (after nil condition at Children[2])
			if len(zongAST.Children) < 4 {
				t.Errorf("At %s: expected else block in AST, got only %d children", path, len(zongAST.Children))
				return
			}

			elseBlock := zongAST.Children[3]
			if elseBlock == nil {
				t.Errorf("At %s: else block is nil", path)
				return
			}

			// The else pattern should be an array like the then pattern
			if elsePattern.Type != sexy.NodeArray {
				t.Errorf("At %s: expected array for else pattern, got %v", path, elsePattern.Type)
				return
			}

			// Match the statements in the else block
			expectedElseStmts := elsePattern.Items
			actualElseStmts := elseBlock.Children

			if len(expectedElseStmts) != len(actualElseStmts) {
				t.Errorf("At %s: expected %d else statements, got %d", path, len(expectedElseStmts), len(actualElseStmts))
				return
			}

			for i, expectedStmt := range expectedElseStmts {
				actualStmt := actualElseStmts[i]
				assertPatternMatch(t, actualStmt, expectedStmt, path+".else"+string(rune('0'+i)))
			}
		}
		// TODO: Handle more complex if-else-if chains later if needed
	}
}

// assertFuncMatch matches Zong NodeFunc against Sexy NodeList patterns
// Zong functions map to Sexy patterns like:
// - (func "name" [] nil []) for void function with no parameters and no body
// - (func "name" [] "I64" []) for function with return type but no body
// - (func "name" [(param "p1" "I64" positional)] "I64" []) for function with parameters
// - (func "name" [] nil [stmt1 stmt2]) for function with body
func assertFuncMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for function declaration, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 5 {
		t.Errorf("At %s: expected (func \"name\" params return_type body) pattern with 5 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "func"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "func" {
		t.Errorf("At %s: expected 'func' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the function name as a string
	namePattern := sexyPattern.Items[1]
	if namePattern.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for function name, got %v", path, namePattern.Type)
		return
	}

	if zongAST.FunctionName != namePattern.Text {
		t.Errorf("At %s: function name mismatch, expected %s, got %s", path, namePattern.Text, zongAST.FunctionName)
		return
	}

	// Third item should be the parameters array
	paramsPattern := sexyPattern.Items[2]
	if paramsPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array for function parameters, got %v (pattern: %v)", path, paramsPattern.Type, paramsPattern.String())
		return
	}

	// Match parameters
	expectedParams := paramsPattern.Items
	actualParams := zongAST.Parameters

	if len(expectedParams) != len(actualParams) {
		t.Errorf("At %s: expected %d parameters, got %d", path, len(expectedParams), len(actualParams))
		return
	}

	for i, expectedParam := range expectedParams {
		actualParam := actualParams[i]

		// Each parameter should be a list like (param "name" "type" positional/named)
		if expectedParam.Type != sexy.NodeList {
			t.Errorf("At %s.param%d: expected list for parameter, got %v", path, i, expectedParam.Type)
			continue
		}

		if len(expectedParam.Items) != 4 {
			t.Errorf("At %s.param%d: expected (param \"name\" \"type\" kind) with 4 items, got %d", path, i, len(expectedParam.Items))
			continue
		}

		// First item should be "param"
		if expectedParam.Items[0].Type != sexy.NodeSymbol || expectedParam.Items[0].Text != "param" {
			t.Errorf("At %s.param%d: expected 'param' symbol, got %v with text '%s'", path, i, expectedParam.Items[0].Type, expectedParam.Items[0].Text)
			continue
		}

		// Second item should be parameter name
		if expectedParam.Items[1].Type != sexy.NodeString {
			t.Errorf("At %s.param%d: expected string for parameter name, got %v", path, i, expectedParam.Items[1].Type)
			continue
		}
		if actualParam.Name != expectedParam.Items[1].Text {
			t.Errorf("At %s.param%d: parameter name mismatch, expected %s, got %s", path, i, expectedParam.Items[1].Text, actualParam.Name)
			continue
		}

		// Third item should be parameter type
		if expectedParam.Items[2].Type != sexy.NodeString {
			t.Errorf("At %s.param%d: expected string for parameter type, got %v", path, i, expectedParam.Items[2].Type)
			continue
		}
		actualTypeName := TypeToString(actualParam.Type)
		if actualTypeName != expectedParam.Items[2].Text {
			t.Errorf("At %s.param%d: parameter type mismatch, expected %s, got %s", path, i, expectedParam.Items[2].Text, actualTypeName)
			continue
		}

		// Fourth item should be parameter kind (positional/named)
		if expectedParam.Items[3].Type != sexy.NodeSymbol {
			t.Errorf("At %s.param%d: expected symbol for parameter kind, got %v", path, i, expectedParam.Items[3].Type)
			continue
		}
		expectedKind := expectedParam.Items[3].Text
		actualKind := "positional"
		if actualParam.IsNamed {
			actualKind = "named"
		}
		if actualKind != expectedKind {
			t.Errorf("At %s.param%d: parameter kind mismatch, expected %s, got %s", path, i, expectedKind, actualKind)
			continue
		}
	}

	// Fourth item should be the return type (string or nil)
	returnTypePattern := sexyPattern.Items[3]
	if returnTypePattern.Type == sexy.NodeSymbol && returnTypePattern.Text == "nil" {
		// Function has no return type (void)
		if zongAST.ReturnType != nil {
			t.Errorf("At %s: expected void return type (nil), got %s", path, TypeToString(zongAST.ReturnType))
		}
	} else if returnTypePattern.Type == sexy.NodeString {
		// Function has a return type
		if zongAST.ReturnType == nil {
			t.Errorf("At %s: expected return type '%s', got void (nil)", path, returnTypePattern.Text)
		} else {
			actualReturnType := TypeToString(zongAST.ReturnType)
			if actualReturnType != returnTypePattern.Text {
				t.Errorf("At %s: return type mismatch, expected %s, got %s", path, returnTypePattern.Text, actualReturnType)
				return
			}
		}
	} else {
		t.Errorf("At %s: expected string or nil for return type, got %v", path, returnTypePattern.Type)
		return
	}

	// Fifth item should be the function body (array of statements)
	bodyPattern := sexyPattern.Items[4]
	// Function statements are now directly in Children
	assertNodeArray(t, zongAST.Children, bodyPattern, path+".body")
}

// assertStructMatch matches Zong NodeStruct against Sexy NodeList patterns
// Zong structs map to Sexy patterns like:
// - (struct "Name" [(field "x" "I64") (field "y" "I64")])
func assertStructMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for struct declaration, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 3 {
		t.Errorf("At %s: expected (struct \"name\" fields) pattern with 3 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "struct"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "struct" {
		t.Errorf("At %s: expected 'struct' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the struct name as a string
	namePattern := sexyPattern.Items[1]
	if namePattern.Type != sexy.NodeString {
		t.Errorf("At %s: expected string for struct name, got %v", path, namePattern.Type)
		return
	}

	// For NodeStruct, the struct name is stored in the String field
	if zongAST.String != namePattern.Text {
		t.Errorf("At %s: struct name mismatch, expected %s, got %s", path, namePattern.Text, zongAST.String)
		return
	}

	// Third item should be the fields array
	fieldsPattern := sexyPattern.Items[2]
	if fieldsPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array for struct fields, got %v (pattern: %v)", path, fieldsPattern.Type, fieldsPattern.String())
		return
	}

	// Match fields
	expectedFields := fieldsPattern.Items

	// For NodeStruct, fields are now stored in StructFields metadata (no longer as Children)
	actualFieldDecls := zongAST.StructFields

	if len(expectedFields) != len(actualFieldDecls) {
		t.Errorf("At %s: expected %d fields, got %d", path, len(expectedFields), len(actualFieldDecls))
		return
	}

	for i, expectedField := range expectedFields {
		actualFieldDecl := actualFieldDecls[i]

		// Each field should be a list like (field "name" "type")
		if expectedField.Type != sexy.NodeList {
			t.Errorf("At %s.field%d: expected list for field, got %v", path, i, expectedField.Type)
			continue
		}

		if len(expectedField.Items) != 3 {
			t.Errorf("At %s.field%d: expected (field \"name\" \"type\") with 3 items, got %d", path, i, len(expectedField.Items))
			continue
		}

		// First item should be "field"
		if expectedField.Items[0].Type != sexy.NodeSymbol || expectedField.Items[0].Text != "field" {
			t.Errorf("At %s.field%d: expected 'field' symbol, got %v with text '%s'", path, i, expectedField.Items[0].Type, expectedField.Items[0].Text)
			continue
		}

		// Second item should be field name
		if expectedField.Items[1].Type != sexy.NodeString {
			t.Errorf("At %s.field%d: expected string for field name, got %v", path, i, expectedField.Items[1].Type)
			continue
		}

		// actualFieldDecl is now a Parameter struct (not AST node)
		actualFieldName := actualFieldDecl.Name
		if actualFieldName != expectedField.Items[1].Text {
			t.Errorf("At %s.field%d: field name mismatch, expected %s, got %s", path, i, expectedField.Items[1].Text, actualFieldName)
			continue
		}

		// Third item should be field type
		if expectedField.Items[2].Type != sexy.NodeString {
			t.Errorf("At %s.field%d: expected string for field type, got %v", path, i, expectedField.Items[2].Type)
			continue
		}

		if actualFieldDecl.Type == nil {
			t.Errorf("At %s.field%d: field declaration missing Type", path, i)
			continue
		}

		actualTypeName := TypeToString(actualFieldDecl.Type)
		if actualTypeName != expectedField.Items[2].Text {
			t.Errorf("At %s.field%d: field type mismatch, expected %s, got %s", path, i, expectedField.Items[2].Text, actualTypeName)
			continue
		}
	}
}

// assertLoopMatch matches Zong NodeLoop against Sexy NodeList patterns
// Zong loops map to Sexy patterns like (loop [stmt1 stmt2 ...])
func assertLoopMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for loop, got %v", path, sexyPattern.Type)
		return
	}
	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (loop [...]) pattern, got %d", path, len(sexyPattern.Items))
		return
	}
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "loop" {
		t.Errorf("At %s: expected 'loop' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}
	assertNodeArray(t, zongAST.Children, sexyPattern.Items[1], path+".body")
}

func assertNodeArray(t *testing.T, zongASTs []*ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array, got %v (pattern: %v)", path, sexyPattern.Type, sexyPattern.String())
		return
	}
	if len(sexyPattern.Items) != len(zongASTs) {
		t.Errorf("At %s: expected %d nodes, got %d", path, len(sexyPattern.Items), len(zongASTs))
		return
	}
	for i := range sexyPattern.Items {
		assertPatternMatch(t, zongASTs[i], sexyPattern.Items[i], path+"."+string(rune('0'+i)))
	}
}

// assertBreakMatch matches Zong NodeBreak against Sexy NodeSymbol patterns
// Zong break statements map to Sexy patterns like (break)
func assertBreakMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeSymbol {
		t.Errorf("At %s: expected Sexy symbol for break statement, got %v", path, sexyPattern.Type)
		return
	}
	if sexyPattern.Text != "break" {
		t.Errorf("At %s: expected 'break' symbol, got '%s'", path, sexyPattern.Text)
		return
	}
}

// assertContinueMatch matches Zong NodeContinue against Sexy NodeSymbol patterns
// Zong continue statements map to Sexy patterns like (continue)
func assertContinueMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeSymbol {
		t.Errorf("At %s: expected Sexy symbol for continue statement, got %v", path, sexyPattern.Type)
		return
	}
	if sexyPattern.Text != "continue" {
		t.Errorf("At %s: expected 'continue' symbol, got '%s'", path, sexyPattern.Text)
		return
	}
}

// assertExecutionMatch compiles and executes the given AST, comparing output to expected result
func assertExecutionMatch(t *testing.T, ast *ASTNode, expectedOutput string, inputType sexy.InputType) {
	t.Helper()

	// Compile the AST to WASM
	wasmBytes := CompileToWASM(ast)
	if len(wasmBytes) == 0 {
		t.Fatal("Failed to compile AST to WASM - no bytes generated")
	}

	// Execute the WASM and capture output
	actualOutput, err := executeWasm(t, wasmBytes)
	if err != nil {
		t.Fatalf("WASM execution failed: %v", err)
	}

	// Normalize expected output - add newline if not empty and doesn't already have one
	normalizedExpected := expectedOutput
	if normalizedExpected != "" && !strings.HasSuffix(normalizedExpected, "\n") {
		normalizedExpected += "\n"
	}

	// Compare actual output with expected output
	if actualOutput != normalizedExpected {
		// Dump WASM for debugging when execution fails
		watOutput := dumpWasmForDebugging(t, wasmBytes)
		t.Errorf("Execution output mismatch:\n  Expected: %q\n  Actual:   %q\n\nGenerated WASM:\n%s", normalizedExpected, actualOutput, watOutput)
	}
}

// assertCompileErrorMatch attempts to compile the given input and verifies it produces the expected error
func assertCompileErrorMatch(t *testing.T, input string, expectedError string, inputType sexy.InputType) {
	t.Helper()

	// Parse the input
	input = input + "\x00" // Null-terminate as required by Zong parser
	l := NewLexer([]byte(input))
	l.NextToken()

	var ast *ASTNode

	// Parse the input
	switch inputType {
	case sexy.InputTypeZongExpr:
		ast = ParseExpression(l)
	case sexy.InputTypeZongProgram:
		ast = ParseProgram(l)
	default:
		t.Fatalf("Unknown input type: %s", inputType)
	}

	// Check for lexer/parser errors first
	if l.Errors.HasErrors() {
		if expectedError == "" {
			// Test passes - we expected some error and got one
			return
		}
		// Check if any error message exactly matches the expected error
		errorMsg := l.Errors.String()
		if strings.TrimSpace(errorMsg) != strings.TrimSpace(expectedError) {
			t.Errorf("Compilation error mismatch:\n  Expected error: %q\n  Actual errors: %q", expectedError, errorMsg)
		}
		return
	}

	// If parsing succeeded, check for type errors and then try to compile
	if ast != nil {
		// Build symbol table and check for type errors
		symbolTable := BuildSymbolTable(ast)
		if symbolTable.Errors.HasErrors() {
			// Check if we expected some error and got one
			if expectedError == "" {
				t.Errorf("Unexpected symbol resolution errors: %s", symbolTable.Errors.String())
				return
			}
			// Check if any error message exactly matches the expected error
			errorMsg := symbolTable.Errors.String()
			if strings.TrimSpace(errorMsg) != strings.TrimSpace(expectedError) {
				t.Errorf("Symbol resolution error mismatch:\n  Expected error: %q\n  Actual errors: %q", expectedError, errorMsg)
			}
			return
		}
		typeErrors := CheckProgram(ast, symbolTable.typeTable)

		// Check for type errors first
		if typeErrors.HasErrors() {
			if expectedError == "" {
				// Test passes - we expected some error and got one
				return
			}
			// Check if any type error message exactly matches the expected error
			errorMsg := typeErrors.String()
			if strings.TrimSpace(errorMsg) != strings.TrimSpace(expectedError) {
				t.Errorf("Compilation error mismatch:\n  Expected error: %q\n  Actual type errors: %q", expectedError, errorMsg)
			}
			return
		}
	} else {
		// Parsing failed but no errors recorded - this shouldn't happen
		if expectedError == "" {
			// Test passes - we expected some error and got one
			return
		}
		t.Errorf("Parsing failed but no errors recorded in lexer.Errors")
	}
}

// assertWasmLocalsMatch compares collected local variables against expected pattern
// Pattern format: [(local "name" "type" storage address)]
func assertWasmLocalsMatch(t *testing.T, ast *ASTNode, expectedPattern *sexy.Node) {
	t.Helper()

	// Collect actual local variables from the AST
	actualLocals, _ := collectLocalVariables(ast)

	// Expected pattern should be an array
	if expectedPattern.Type != sexy.NodeArray {
		t.Errorf("Expected array for wasm-locals pattern, got %v", expectedPattern.Type)
		return
	}

	expectedLocals := expectedPattern.Items

	// Check count matches
	if len(actualLocals) != len(expectedLocals) {
		t.Errorf("Expected %d local variables, got %d", len(expectedLocals), len(actualLocals))
		return
	}

	// Check each local variable
	for i, expectedLocal := range expectedLocals {
		actualLocal := actualLocals[i]

		// Each expected local should be a list: (local "name" "type" storage address)
		if expectedLocal.Type != sexy.NodeList {
			t.Errorf("Expected list for local variable %d, got %v", i, expectedLocal.Type)
			continue
		}

		if len(expectedLocal.Items) != 5 {
			t.Errorf("Expected (local \"name\" \"type\" storage address) with 5 items for local %d, got %d", i, len(expectedLocal.Items))
			continue
		}

		// Check "local" symbol
		if expectedLocal.Items[0].Type != sexy.NodeSymbol || expectedLocal.Items[0].Text != "local" {
			t.Errorf("Expected 'local' symbol for local %d, got %v with text '%s'", i, expectedLocal.Items[0].Type, expectedLocal.Items[0].Text)
			continue
		}

		// Check variable name
		if expectedLocal.Items[1].Type != sexy.NodeString {
			t.Errorf("Expected string for variable name at local %d, got %v", i, expectedLocal.Items[1].Type)
			continue
		}
		expectedName := expectedLocal.Items[1].Text
		actualName := actualLocal.Symbol.Name
		if actualName != expectedName {
			t.Errorf("Local %d: expected name %s, got %s", i, expectedName, actualName)
			continue
		}

		// Check variable type
		if expectedLocal.Items[2].Type != sexy.NodeString {
			t.Errorf("Expected string for variable type at local %d, got %v", i, expectedLocal.Items[2].Type)
			continue
		}
		expectedType := expectedLocal.Items[2].Text
		actualType := TypeToString(actualLocal.Symbol.Type)
		if actualType != expectedType {
			t.Errorf("Local %d (%s): expected type %s, got %s", i, actualName, expectedType, actualType)
			continue
		}

		// Check storage type
		if expectedLocal.Items[3].Type != sexy.NodeSymbol {
			t.Errorf("Expected symbol for storage type at local %d, got %v", i, expectedLocal.Items[3].Type)
			continue
		}
		expectedStorage := expectedLocal.Items[3].Text
		var actualStorage string
		switch actualLocal.Storage {
		case VarStorageLocal:
			actualStorage = "local"
		case VarStorageTStack:
			actualStorage = "tstack"
		case VarStorageParameterLocal:
			actualStorage = "local" // Parameter locals are shown as "local"
		default:
			actualStorage = "unknown"
		}
		if actualStorage != expectedStorage {
			t.Errorf("Local %d (%s): expected storage %s, got %s", i, actualName, expectedStorage, actualStorage)
			continue
		}

		// Check address
		if expectedLocal.Items[4].Type != sexy.NodeInteger {
			t.Errorf("Expected integer for address at local %d, got %v", i, expectedLocal.Items[4].Type)
			continue
		}
		expectedAddress := expectedLocal.Items[4].Text
		actualAddress := intToString(int64(actualLocal.Address))
		if actualAddress != expectedAddress {
			t.Errorf("Local %d (%s): expected address %s, got %s", i, actualName, expectedAddress, actualAddress)
			continue
		}
	}
}

// dumpWasmForDebugging writes WASM bytes to a temporary file and converts to WAT format for debugging
func dumpWasmForDebugging(t *testing.T, wasmBytes []byte) string {
	t.Helper()

	// Write WASM to temporary file
	tmpFile, err := ioutil.TempFile("", "debug_*.wasm")
	if err != nil {
		return fmt.Sprintf("Failed to create temp file for WASM dump: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = tmpFile.Write(wasmBytes)
	if err != nil {
		return fmt.Sprintf("Failed to write WASM to temp file: %v", err)
	}
	tmpFile.Close()

	// Convert to WAT format
	watOutput, err := convertWasmToWat(wasmBytes, tmpFile.Name())
	if err != nil {
		return fmt.Sprintf("Failed to convert WASM to WAT: %v\nRaw WASM bytes (%d): %x", err, len(wasmBytes), wasmBytes)
	}

	return watOutput
}
