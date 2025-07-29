package main

import (
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
			// Read the test file
			content, err := os.ReadFile(testFile)
			be.Err(t, err, nil)

			// Extract test cases
			testCases, err := sexy.ExtractTestCases(string(content))
			be.Err(t, err, nil)

			// Generate a subtest for each test case
			for _, tc := range testCases {
				t.Run(tc.Name, func(t *testing.T) {
					// Parse the Zong input based on input type
					input := tc.Input + "\x00" // Null-terminate as required by Zong parser
					Init([]byte(input))
					NextToken()

					var ast *ASTNode
					switch tc.InputType {
					case sexy.InputTypeZongExpr:
						ast = ParseExpression()
					case sexy.InputTypeZongProgram:
						ast = ParseProgram()
					default:
						t.Fatalf("Unknown input type: %s", tc.InputType)
					}

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
		t.Errorf("At %s: expected Sexy integer, got %v", path, sexyPattern.Type)
		return
	}

	expectedValue := sexyPattern.Text
	actualValue := intToString(zongAST.Integer)

	be.Equal(t, actualValue, expectedValue, "At %s: integer value mismatch", path)
}

// assertBooleanMatch matches Zong NodeBoolean against Sexy patterns
// Sexy patterns for booleans are lists like (boolean true) or (boolean false)
func assertBooleanMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for boolean literal, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 2 {
		t.Errorf("At %s: expected (boolean true/false) pattern with 2 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "boolean"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "boolean" {
		t.Errorf("At %s: expected 'boolean' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the boolean value as a symbol (true or false)
	if sexyPattern.Items[1].Type != sexy.NodeSymbol {
		t.Errorf("At %s: expected symbol for boolean value, got %v", path, sexyPattern.Items[1].Type)
		return
	}

	expectedValue := sexyPattern.Items[1].Text
	actualValue := "false"
	if zongAST.Boolean {
		actualValue = "true"
	}

	be.Equal(t, actualValue, expectedValue, "At %s: boolean value mismatch", path)
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

// assertUnaryMatch matches Zong NodeUnary against Sexy NodeList patterns
// Zong unary expressions map to Sexy patterns like (unary "!" operand)
func assertUnaryMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for unary expression, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) != 3 {
		t.Errorf("At %s: expected (unary \"op\" operand) pattern with 3 items, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "unary"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "unary" {
		t.Errorf("At %s: expected 'unary' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Second item should be the operator as a string
	if sexyPattern.Items[1].Type != sexy.NodeString {
		t.Errorf("At %s: expected string for operator, got %v", path, sexyPattern.Items[1].Type)
		return
	}

	be.Equal(t, zongAST.Op, sexyPattern.Items[1].Text, "At %s: operator mismatch", path)

	// Check that we have the expected number of children
	if len(zongAST.Children) != 1 {
		t.Errorf("At %s: expected 1 child for unary expression, got %d", path, len(zongAST.Children))
		return
	}

	// Recursively match operand (third item in pattern)
	operandPattern := sexyPattern.Items[2]
	assertPatternMatch(t, zongAST.Children[0], operandPattern, path+".operand")
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
	be.Equal(t, zongAST.FieldName, sexyPattern.Items[2].Text, "At %s: field name mismatch", path)

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
	be.Equal(t, zongAST.Children[0].String, namePattern.Text, "At %s: var name mismatch", path)

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
	be.Equal(t, actualTypeName, expectedTypeName, "At %s: type name mismatch", path)

	// Match initialization expression if present
	if len(zongAST.Children) > 1 {
		initPattern := sexyPattern.Items[3]
		assertPatternMatch(t, zongAST.Children[1], initPattern, path+".init")
	}
}

// assertBlockMatch matches Zong NodeBlock against Sexy patterns
// Zong blocks can map to:
// - (block stmt1 stmt2 ...) for explicit blocks
// - [stmt1 stmt2 ...] for program-level statement lists
func assertBlockMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	var expectedStmts []*sexy.Node

	if sexyPattern.Type == sexy.NodeArray {
		// Program-level pattern: [stmt1 stmt2 ...]
		expectedStmts = sexyPattern.Items
	} else if sexyPattern.Type == sexy.NodeList {
		// Explicit block pattern: (block stmt1 stmt2 ...)
		if len(sexyPattern.Items) < 1 {
			t.Errorf("At %s: expected (block ...) pattern with at least 1 item, got %d", path, len(sexyPattern.Items))
			return
		}

		// First item should be the symbol "block"
		if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "block" {
			t.Errorf("At %s: expected 'block' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
			return
		}

		expectedStmts = sexyPattern.Items[1:] // Skip "block"
	} else {
		t.Errorf("At %s: expected Sexy array or list for block, got %v", path, sexyPattern.Type)
		return
	}

	// Match the statements
	actualStmts := zongAST.Children

	if len(expectedStmts) != len(actualStmts) {
		t.Errorf("At %s: expected %d statements, got %d", path, len(expectedStmts), len(actualStmts))
		return
	}

	for i, expectedStmt := range expectedStmts {
		actualStmt := actualStmts[i]
		assertPatternMatch(t, actualStmt, expectedStmt, path+".stmt"+string(rune('0'+i)))
	}
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

	be.Equal(t, zongAST.FunctionName, namePattern.Text, "At %s: function name mismatch", path)

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
		be.Equal(t, actualParam.Name, expectedParam.Items[1].Text)

		// Third item should be parameter type
		if expectedParam.Items[2].Type != sexy.NodeString {
			t.Errorf("At %s.param%d: expected string for parameter type, got %v", path, i, expectedParam.Items[2].Type)
			continue
		}
		actualTypeName := TypeToString(actualParam.Type)
		be.Equal(t, actualTypeName, expectedParam.Items[2].Text)

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
		be.Equal(t, actualKind, expectedKind)
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
			be.Equal(t, actualReturnType, returnTypePattern.Text, "At %s: return type mismatch", path)
		}
	} else {
		t.Errorf("At %s: expected string or nil for return type, got %v", path, returnTypePattern.Type)
		return
	}

	// Fifth item should be the function body (array of statements)
	bodyPattern := sexyPattern.Items[4]
	if bodyPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array for function body, got %v (pattern: %v)", path, bodyPattern.Type, bodyPattern.String())
		return
	}

	expectedBodyStmts := bodyPattern.Items

	if zongAST.Body == nil {
		// Function has no body
		if len(expectedBodyStmts) != 0 {
			t.Errorf("At %s: expected %d body statements, got no body", path, len(expectedBodyStmts))
		}
	} else {
		// Function has a body - should be a block statement
		if zongAST.Body.Kind != NodeBlock {
			t.Errorf("At %s: expected NodeBlock for function body, got %v", path, zongAST.Body.Kind)
			return
		}

		actualBodyStmts := zongAST.Body.Children
		if len(expectedBodyStmts) != len(actualBodyStmts) {
			t.Errorf("At %s: expected %d body statements, got %d", path, len(expectedBodyStmts), len(actualBodyStmts))
			return
		}

		for i, expectedStmt := range expectedBodyStmts {
			actualStmt := actualBodyStmts[i]
			assertPatternMatch(t, actualStmt, expectedStmt, path+".body"+string(rune('0'+i)))
		}
	}
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
	be.Equal(t, zongAST.String, namePattern.Text, "At %s: struct name mismatch", path)

	// Third item should be the fields array
	fieldsPattern := sexyPattern.Items[2]
	if fieldsPattern.Type != sexy.NodeArray {
		t.Errorf("At %s: expected array for struct fields, got %v (pattern: %v)", path, fieldsPattern.Type, fieldsPattern.String())
		return
	}

	// Match fields
	expectedFields := fieldsPattern.Items

	// For NodeStruct, fields are stored as NodeVar children
	actualFieldDecls := zongAST.Children

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

		// actualFieldDecl should be a NodeVar with field name in its first child
		if actualFieldDecl.Kind != NodeVar {
			t.Errorf("At %s.field%d: expected NodeVar for field declaration, got %v", path, i, actualFieldDecl.Kind)
			continue
		}

		if len(actualFieldDecl.Children) == 0 {
			t.Errorf("At %s.field%d: field declaration missing name child", path, i)
			continue
		}

		actualFieldName := actualFieldDecl.Children[0].String
		be.Equal(t, actualFieldName, expectedField.Items[1].Text)

		// Third item should be field type
		if expectedField.Items[2].Type != sexy.NodeString {
			t.Errorf("At %s.field%d: expected string for field type, got %v", path, i, expectedField.Items[2].Type)
			continue
		}

		if actualFieldDecl.TypeAST == nil {
			t.Errorf("At %s.field%d: field declaration missing TypeAST", path, i)
			continue
		}

		actualTypeName := TypeToString(actualFieldDecl.TypeAST)
		be.Equal(t, actualTypeName, expectedField.Items[2].Text)
	}
}

// assertLoopMatch matches Zong NodeLoop against Sexy NodeList patterns
// Zong loops map to Sexy patterns like (loop stmt1 stmt2 ...)
func assertLoopMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeList {
		t.Errorf("At %s: expected Sexy list for loop, got %v", path, sexyPattern.Type)
		return
	}

	if len(sexyPattern.Items) < 1 {
		t.Errorf("At %s: expected (loop ...) pattern with at least 1 item, got %d", path, len(sexyPattern.Items))
		return
	}

	// First item should be the symbol "loop"
	if sexyPattern.Items[0].Type != sexy.NodeSymbol || sexyPattern.Items[0].Text != "loop" {
		t.Errorf("At %s: expected 'loop' symbol, got %v with text '%s'", path, sexyPattern.Items[0].Type, sexyPattern.Items[0].Text)
		return
	}

	// Match the statements
	expectedStmts := sexyPattern.Items[1:] // Skip "loop"
	actualStmts := zongAST.Children

	if len(expectedStmts) != len(actualStmts) {
		t.Errorf("At %s: expected %d statements, got %d", path, len(expectedStmts), len(actualStmts))
		return
	}

	for i, expectedStmt := range expectedStmts {
		actualStmt := actualStmts[i]
		assertPatternMatch(t, actualStmt, expectedStmt, path+".stmt"+string(rune('0'+i)))
	}
}

// assertBreakMatch matches Zong NodeBreak against Sexy NodeSymbol patterns
// Zong break statements map to Sexy patterns like (break)
func assertBreakMatch(t *testing.T, zongAST *ASTNode, sexyPattern *sexy.Node, path string) {
	if sexyPattern.Type != sexy.NodeSymbol {
		t.Errorf("At %s: expected Sexy symbol for break statement, got %v", path, sexyPattern.Type)
		return
	}

	// Pattern should be the symbol "break"
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

	// Pattern should be the symbol "continue"
	if sexyPattern.Text != "continue" {
		t.Errorf("At %s: expected 'continue' symbol, got '%s'", path, sexyPattern.Text)
		return
	}
}
