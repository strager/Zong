package sexy

import (
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

func TestExtractTestCases_BasicTest(t *testing.T) {
	markdown := `# Binary expressions

## Test: +
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```" + `

## Test: -
` + "```zong-expr" + `
1 - 2
` + "```" + `
` + "```ast" + `
(binary "-" 1 2)
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 2)

	// First test case
	tc1 := testCases[0]
	be.Equal(t, tc1.Name, "+")
	be.Equal(t, tc1.Input, "1 + 2")
	be.Equal(t, tc1.InputType, InputTypeZongExpr)
	be.Equal(t, len(tc1.Assertions), 1)
	be.Equal(t, tc1.Assertions[0].Type, AssertionTypeAST)
	be.Equal(t, tc1.Assertions[0].Content, `(binary "+" 1 2)`)
	be.Equal(t, tc1.Assertions[0].ParsedSexy.String(), `(binary "+" 1 2)`)

	// Second test case
	tc2 := testCases[1]
	be.Equal(t, tc2.Name, "-")
	be.Equal(t, tc2.Input, "1 - 2")
	be.Equal(t, tc2.InputType, InputTypeZongExpr)
	be.Equal(t, len(tc2.Assertions), 1)
	be.Equal(t, tc2.Assertions[0].Type, AssertionTypeAST)
	be.Equal(t, tc2.Assertions[0].Content, `(binary "-" 1 2)`)
	be.Equal(t, tc2.Assertions[0].ParsedSexy.String(), `(binary "-" 1 2)`)
}

func TestExtractTestCases_MultipleAssertions(t *testing.T) {
	markdown := `## Test: multiple assertions
` + "```zong-expr" + `
x + y
` + "```" + `
` + "```ast" + `
(binary "+" (var "x") (var "y"))
` + "```" + `
` + "```ast-sym" + `
(binary "+" #x=(var "x") #y=(var "y"))
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)

	tc := testCases[0]
	be.Equal(t, tc.Name, "multiple assertions")
	be.Equal(t, tc.Input, "x + y")
	be.Equal(t, tc.InputType, InputTypeZongExpr)
	be.Equal(t, len(tc.Assertions), 2)

	// First assertion
	be.Equal(t, tc.Assertions[0].Type, AssertionTypeAST)
	be.Equal(t, tc.Assertions[0].Content, `(binary "+" (var "x") (var "y"))`)
	be.Equal(t, tc.Assertions[0].ParsedSexy.String(), `(binary "+" (var "x") (var "y"))`)

	// Second assertion
	be.Equal(t, tc.Assertions[1].Type, AssertionTypeASTSym)
	be.Equal(t, tc.Assertions[1].Content, `(binary "+" #x=(var "x") #y=(var "y"))`)
	be.Equal(t, tc.Assertions[1].ParsedSexy.String(), `(binary "+" #x=(var "x") #y=(var "y"))`)
}

func TestExtractTestCases_DifferentInputTypes(t *testing.T) {
	markdown := `## Test: zong-program input
` + "```zong-program" + `
func main() { print(42); }
` + "```" + `
` + "```ast" + `
(program (func "main" [] [(call "print" [42])]))
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)

	tc := testCases[0]
	be.Equal(t, tc.Name, "zong-program input")
	be.Equal(t, tc.Input, "func main() { print(42); }")
	be.Equal(t, tc.InputType, InputTypeZongProgram)
	be.Equal(t, len(tc.Assertions), 1)
	be.Equal(t, tc.Assertions[0].Type, AssertionTypeAST)
}

func TestExtractTestCases_DifferentAssertionTypes(t *testing.T) {
	markdown := `## Test: different assertions
` + "```zong-expr" + `
x
` + "```" + `
` + "```ast" + `
(var "x")
` + "```" + `
` + "```ast-sym" + `
#x=(var "x")
` + "```" + `
` + "```types" + `
{x: I64}
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)

	tc := testCases[0]
	be.Equal(t, len(tc.Assertions), 3)

	be.Equal(t, tc.Assertions[0].Type, AssertionTypeAST)
	be.Equal(t, tc.Assertions[1].Type, AssertionTypeASTSym)
	be.Equal(t, tc.Assertions[2].Type, AssertionTypeTypes)
}

func TestExtractTestCases_EmptyFile(t *testing.T) {
	markdown := ""

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 0)
}

func TestExtractTestCases_NoTestCases(t *testing.T) {
	markdown := `# Some document

This is just regular markdown content.

## Regular heading

No test cases here.`

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 0)
}

func TestExtractTestCases_NoTestCasesWithUnknownFence(t *testing.T) {
	markdown := `# Some document

This is just regular markdown content.

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

## Regular heading

No test cases here.`

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "unknown fence language 'go' found outside of test case"))
}

func TestExtractTestCases_InvalidSexyAssertion(t *testing.T) {
	markdown := `## Test: invalid sexy
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(unclosed list
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "failed to parse Sexy assertion"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

// Error condition tests

func TestExtractTestCases_FenceOutsideTestCase(t *testing.T) {
	tests := []struct {
		name      string
		markdown  string
		fenceType string
	}{
		{
			"zong-expr fence outside test",
			"# Document\n\n```zong-expr\n1 + 2\n```\n",
			"zong-expr",
		},
		{
			"zong-program fence outside test",
			"# Document\n\n```zong-program\nfunc main() {}\n```\n",
			"zong-program",
		},
		{
			"ast fence outside test",
			"# Document\n\n```ast\n(binary \"+\" 1 2)\n```\n",
			"ast",
		},
		{
			"ast-sym fence outside test",
			"# Document\n\n```ast-sym\n#x=(var \"x\")\n```\n",
			"ast-sym",
		},
		{
			"types fence outside test",
			"# Document\n\n```types\n{x: I64}\n```\n",
			"types",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ExtractTestCases(test.markdown)
			be.True(t, err != nil)
			be.True(t, strings.Contains(err.Error(), test.fenceType+" fence found outside of test case"))
			be.True(t, strings.Contains(err.Error(), "line"))
		})
	}
}

func TestExtractTestCases_UnknownFenceLanguageInTest(t *testing.T) {
	// Unknown fence languages should cause errors even within test cases
	markdown := `## Test: with unknown fence
` + "```python" + `
print("hello")
` + "```" + `
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "unknown fence language 'python'"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

func TestExtractTestCases_TestMissingInputFence(t *testing.T) {
	markdown := `## Test: no input
` + "```ast" + `
(binary "+" 1 2)
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "test 'no input' has no input fence"))
}

func TestExtractTestCases_TestMissingAssertionFence(t *testing.T) {
	markdown := `## Test: no assertions
` + "```zong-expr" + `
1 + 2
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "test 'no assertions' has no assertion fences"))
}

func TestExtractTestCases_MultipleInputFences(t *testing.T) {
	markdown := `## Test: multiple inputs
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```zong-expr" + `
3 + 4
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "multiple input fences found"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

func TestExtractTestCases_UnknownFenceOutsideTest(t *testing.T) {
	markdown := `# Document with unknown code block

` + "```go" + `
func main() {}
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "unknown fence language 'go' found outside of test case"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

func TestExtractTestCases_UnknownFenceInTest(t *testing.T) {
	markdown := `## Test: test with unknown fence
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```" + `

` + "```shell" + `
echo "more code"
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "unknown fence language 'shell'"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

func TestExtractTestCases_AllowFencesWithoutLanguage(t *testing.T) {
	// Code blocks without language specification should be allowed
	markdown := `# Document with generic code block

` + "```" + `
some code without language
` + "```" + `

## Test: valid test
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```" + `

` + "```" + `
more code without language in test
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)
	be.Equal(t, testCases[0].Name, "valid test")
	be.Equal(t, testCases[0].Input, "1 + 2")
	be.Equal(t, len(testCases[0].Assertions), 1)
}

func TestExtractTestCases_LineNumberAccuracy(t *testing.T) {
	// Test that line numbers are reported correctly for fences outside test cases
	markdown := `# Title
Line 2
Line 3

` + "```zong-expr" + `
this should fail - fence outside any test
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	t.Logf("Error message: %s", err.Error())
	// This should be detected as a fence outside test case
	be.True(t, strings.Contains(err.Error(), "fence found outside"))
	be.True(t, strings.Contains(err.Error(), "line"))
}

func TestExtractTestCases_ErrorInSecondTest(t *testing.T) {
	markdown := `## Test: first test
` + "```zong-expr" + `
1 + 2
` + "```" + `
` + "```ast" + `
(binary "+" 1 2)
` + "```" + `

## Test: second test missing input
` + "```ast" + `
(binary "-" 1 2)
` + "```"

	_, err := ExtractTestCases(markdown)
	be.True(t, err != nil)
	be.True(t, strings.Contains(err.Error(), "test 'second test missing input' has no input fence"))
}

// This test removed - we now error on unknown fence languages

func TestExtractTestCases_InputFence(t *testing.T) {
	markdown := `## Test: input fence test
` + "```zong-program" + `
func main() {
    var line U8[] = read_line();
    print_bytes(line);
}
` + "```" + `
` + "```input" + `
hello world

` + "```" + `
` + "```execute" + `
hello world
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)

	tc := testCases[0]
	be.Equal(t, tc.Name, "input fence test")
	be.Equal(t, tc.Input, "func main() {\n    var line U8[] = read_line();\n    print_bytes(line);\n}")
	be.Equal(t, tc.InputType, InputTypeZongProgram)
	be.Equal(t, tc.InputData, "hello world\n\n")
	be.Equal(t, len(tc.Assertions), 1)

	// Only assertion should be execute
	be.Equal(t, tc.Assertions[0].Type, AssertionTypeExecute)
	be.Equal(t, tc.Assertions[0].Content, "hello world")
}

func TestExtractTestCases_ComplexSexyExpressions(t *testing.T) {
	markdown := `## Test: complex expression
` + "```zong-expr" + `
x + yyy * 2
` + "```" + `
` + "```ast" + `
(binary "+"
 (var "x")
 (binary "*"
  (var "yyy")
  (integer 2)))
` + "```"

	testCases, err := ExtractTestCases(markdown)
	be.Err(t, err, nil)
	be.Equal(t, len(testCases), 1)

	tc := testCases[0]
	be.Equal(t, len(tc.Assertions), 1)

	// Verify the parsed Sexy structure
	assertion := tc.Assertions[0]
	be.Equal(t, assertion.Type, AssertionTypeAST)

	// Check that it's parsed as a list
	be.Equal(t, assertion.ParsedSexy.Type, NodeList)
	be.Equal(t, len(assertion.ParsedSexy.Items), 4)

	// First item should be "binary" symbol
	be.Equal(t, assertion.ParsedSexy.Items[0].Type, NodeSymbol)
	be.Equal(t, assertion.ParsedSexy.Items[0].Text, "binary")

	// Second item should be "+" string
	be.Equal(t, assertion.ParsedSexy.Items[1].Type, NodeString)
	be.Equal(t, assertion.ParsedSexy.Items[1].Text, "+")

	// Third item should be another list (var "x")
	be.Equal(t, assertion.ParsedSexy.Items[2].Type, NodeList)

	// Fourth item should be another list (binary "*" ...)
	be.Equal(t, assertion.ParsedSexy.Items[3].Type, NodeList)
}
