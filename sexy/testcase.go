package sexy

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// InputType represents the type of input code fence in a Sexy test
type InputType string

const (
	InputTypeZongExpr    InputType = "zong-expr"
	InputTypeZongProgram InputType = "zong-program"
)

// AssertionType represents the type of assertion code fence in a Sexy test
type AssertionType string

const (
	AssertionTypeAST          AssertionType = "ast"
	AssertionTypeASTSym       AssertionType = "ast-sym"
	AssertionTypeTypes        AssertionType = "types"
	AssertionTypeExecute      AssertionType = "execute"
	AssertionTypeCompileError AssertionType = "compile-error"
	AssertionTypeWasmLocals   AssertionType = "wasm-locals"
	AssertionTypeInput        AssertionType = "input"
)

// Assertion represents a single assertion in a Sexy test
type Assertion struct {
	Type       AssertionType // The type of assertion (ast, ast-sym, types)
	Content    string        // The raw content of the assertion code fence
	ParsedSexy *Node         // The parsed Sexy expression from the assertion content
}

// TestCase represents a complete Sexy test case extracted from Markdown
type TestCase struct {
	Name       string      // The test name from the heading (after "Test: ")
	Input      string      // The raw input code from the input fence
	InputType  InputType   // The type of input fence (zong-expr, zong-program)
	InputData  string      // The stdin input data from input fence (if any)
	Assertions []Assertion // All assertions for this test case
}

// ExtractTestCases parses a Markdown document and extracts all Sexy test cases
func ExtractTestCases(markdownContent string) ([]TestCase, error) {
	md := goldmark.New()
	source := []byte(markdownContent)

	// Parse the markdown document
	doc := md.Parser().Parse(text.NewReader(source))

	var testCases []TestCase
	var currentTestCase *TestCase

	// Walk through all nodes in the document
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := node.(type) {
		case *ast.Heading:
			// Check if this is a test heading
			if n.Level >= 1 && n.Level <= 6 {
				headingText := extractTextFromNode(n, source)
				if strings.HasPrefix(headingText, "Test: ") {
					// Validate the previous test case before saving it
					if currentTestCase != nil {
						if err := validateTestCase(currentTestCase, source); err != nil {
							return ast.WalkStop, err
						}
						testCases = append(testCases, *currentTestCase)
					}

					// Start a new test case
					testName := strings.TrimPrefix(headingText, "Test: ")
					currentTestCase = &TestCase{
						Name:       testName,
						Assertions: []Assertion{},
					}
				}
			}

		case *ast.FencedCodeBlock:
			language := string(n.Language(source))
			content := extractCodeBlockContent(n, source)
			lineNum := getLineNumber(n, source)

			// Check for fences outside test cases
			if currentTestCase == nil {
				// Only allow Sexy fences inside test cases - error on any fence outside
				if language != "" {
					if isInputFence(language) || isAssertionFence(language) {
						return ast.WalkStop, fmt.Errorf("line %d: %s fence found outside of test case", lineNum, language)
					} else {
						return ast.WalkStop, fmt.Errorf("line %d: unknown fence language '%s' found outside of test case", lineNum, language)
					}
				}
				// Allow code blocks with no language specified
				return ast.WalkContinue, nil
			}

			// Check for unknown fence languages within test cases
			if language != "" && !isInputFence(language) && !isAssertionFence(language) {
				return ast.WalkStop, fmt.Errorf("line %d: unknown fence language '%s' in test '%s'", lineNum, language, currentTestCase.Name)
			}

			// Check if this is an input fence
			if isInputFence(language) {
				if currentTestCase.Input != "" {
					return ast.WalkStop, fmt.Errorf("line %d: multiple input fences found in test '%s'", lineNum, currentTestCase.Name)
				}
				currentTestCase.Input = strings.TrimRight(content, "\n")
				currentTestCase.InputType = InputType(language)
			} else if isAssertionFence(language) {
				if language == string(AssertionTypeInput) {
					// Handle input fence - store input data for execution tests
					if currentTestCase.InputData != "" {
						return ast.WalkStop, fmt.Errorf("line %d: multiple input fences found in test '%s'", lineNum, currentTestCase.Name)
					}
					currentTestCase.InputData = content
				} else {
					// Handle other assertion fences
					assertion := Assertion{
						Type:    AssertionType(language),
						Content: strings.TrimRight(content, "\n"),
					}

					if assertion.Type != AssertionTypeExecute && assertion.Type != AssertionTypeCompileError {
						parsedSexy, parseErr := Parse(assertion.Content)
						if parseErr != nil {
							return ast.WalkStop, fmt.Errorf("line %d: failed to parse Sexy assertion in test '%s': %w", lineNum, currentTestCase.Name, parseErr)
						}
						assertion.ParsedSexy = parsedSexy
					}

					currentTestCase.Assertions = append(currentTestCase.Assertions, assertion)
				}
			}
		}

		return ast.WalkContinue, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking markdown AST: %w", err)
	}

	// Validate and save the last test case
	if currentTestCase != nil {
		if err := validateTestCase(currentTestCase, source); err != nil {
			return nil, err
		}
		testCases = append(testCases, *currentTestCase)
	}

	return testCases, nil
}

// extractTextFromNode extracts plain text content from a markdown node
func extractTextFromNode(node ast.Node, source []byte) string {
	var buf bytes.Buffer

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if text, ok := n.(*ast.Text); ok {
				buf.Write(text.Segment.Value(source))
			}
		}
		return ast.WalkContinue, nil
	})

	return buf.String()
}

// extractCodeBlockContent extracts the content from a fenced code block
func extractCodeBlockContent(codeBlock *ast.FencedCodeBlock, source []byte) string {
	var buf bytes.Buffer

	for i := 0; i < codeBlock.Lines().Len(); i++ {
		line := codeBlock.Lines().At(i)
		buf.Write(line.Value(source))
	}

	return buf.String()
}

// isInputFence checks if the language indicates an input fence
func isInputFence(language string) bool {
	return language == string(InputTypeZongExpr) || language == string(InputTypeZongProgram)
}

// isAssertionFence checks if the language indicates an assertion fence
func isAssertionFence(language string) bool {
	return language == string(AssertionTypeAST) ||
		language == string(AssertionTypeASTSym) ||
		language == string(AssertionTypeTypes) ||
		language == string(AssertionTypeExecute) ||
		language == string(AssertionTypeCompileError) ||
		language == string(AssertionTypeWasmLocals) ||
		language == string(AssertionTypeInput)
}

// validateTestCase ensures a test case has both input and at least one assertion
func validateTestCase(testCase *TestCase, source []byte) error {
	if testCase.Input == "" {
		return fmt.Errorf("test '%s' has no input fence", testCase.Name)
	}
	if len(testCase.Assertions) == 0 {
		return fmt.Errorf("test '%s' has no assertion fences", testCase.Name)
	}
	return nil
}

// getLineNumber calculates the line number of a given AST node
func getLineNumber(node ast.Node, source []byte) int {
	if node.Lines().Len() == 0 {
		return 1
	}
	// Count newlines before the node's start position
	startPos := node.Lines().At(0).Start
	lineNum := 1
	for i := 0; i < startPos && i < len(source); i++ {
		if source[i] == '\n' {
			lineNum++
		}
	}
	return lineNum
}
