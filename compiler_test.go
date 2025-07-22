package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

// Helper function to convert WASM bytes to human-readable WAT format
func convertWasmToWat(wasmBytes []byte, wasmFilePath string) (string, error) {
	// Try wasm2wat first, then fallback to wasm-objdump
	cmd := exec.Command("wasm2wat", wasmFilePath)
	output, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	// Fallback to wasm-objdump
	cmd = exec.Command("wasm-objdump", "-d", wasmFilePath)
	output, err = cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(output)), nil
	}

	return "", fmt.Errorf("neither wasm2wat nor wasm-objdump available or failed")
}

// Helper function to parse and compile an expression to WASM
func compileExpression(t *testing.T, expression string) []byte {
	input := []byte(expression + "\x00")

	// Parse the expression
	Init(input)
	NextToken()
	ast := ParseExpression()

	// Compile to WASM
	wasmBytes := CompileToWASM(ast)
	return wasmBytes
}

// Helper function to execute WASM and capture output
func executeWasm(t *testing.T, wasmBytes []byte) (string, error) {
	// Create a temporary WASM file
	tempDir := t.TempDir()
	wasmFile := filepath.Join(tempDir, "test.wasm")

	err := os.WriteFile(wasmFile, wasmBytes, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write WASM file: %v", err)
	}

	return executeWasmFromFile(t, wasmFile)
}

// Helper function that executes WASM and provides debug info if the result doesn't match expected
func executeWasmAndVerify(t *testing.T, wasmBytes []byte, expected string) {
	t.Helper()

	// Create WASM file with test-specific name for easier debugging
	tempDir := t.TempDir()
	testName := strings.ReplaceAll(t.Name(), "/", "_")
	wasmFile := filepath.Join(tempDir, fmt.Sprintf("%s.wasm", testName))

	err := os.WriteFile(wasmFile, wasmBytes, 0644)
	if err != nil {
		t.Fatalf("failed to write WASM file: %v", err)
	}

	output, err := executeWasmFromFile(t, wasmFile)
	be.Err(t, err, nil)

	if output != expected {
		absPath, _ := filepath.Abs(wasmFile)
		t.Logf("WASM file path: %s", absPath)

		if watContent, watErr := convertWasmToWat(wasmBytes, wasmFile); watErr == nil {
			t.Logf("Human-readable WASM (WAT format):\n%s", watContent)
		} else {
			t.Logf("Could not convert to WAT: %v", watErr)
		}
	}

	be.Equal(t, output, expected)
}

// Execute WASM from an existing file
func executeWasmFromFile(t *testing.T, wasmFile string) (string, error) {
	// Build the Rust runtime if it doesn't exist
	runtimeBinary := "./wasmruntime/target/release/wasmruntime"
	if _, err := os.Stat(runtimeBinary); os.IsNotExist(err) {
		t.Log("Building Rust wasmruntime...")
		buildCmd := exec.Command("cargo", "build", "--release")
		buildCmd.Dir = "./wasmruntime"
		buildOutput, buildErr := buildCmd.CombinedOutput()
		if buildErr != nil {
			return "", fmt.Errorf("failed to build Rust runtime: %v\nOutput: %s", buildErr, buildOutput)
		}
	}

	// Execute the WASM file with the Rust runtime
	cmd := exec.Command(runtimeBinary, wasmFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("WASM execution failed: %v\nStderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func TestBasicPrintExpression(t *testing.T) {
	// Test: print(42)
	expression := "print(42)"
	wasmBytes := compileExpression(t, expression)

	// Verify WASM was generated
	be.True(t, len(wasmBytes) > 0)

	// Execute and verify output
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	be.Equal(t, output, "42\n")
}

func TestArithmeticPrint(t *testing.T) {
	// Test: print(42 + 8)
	expression := "print(42 + 8)"
	wasmBytes := compileExpression(t, expression)

	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	be.Equal(t, output, "50\n")
}

func TestComplexArithmetic(t *testing.T) {
	// Test: print((10 + 5) * 2 - 3)
	expression := "print((10 + 5) * 2 - 3)"
	wasmBytes := compileExpression(t, expression)

	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	be.Equal(t, output, "27\n")
}

func TestOperatorPrecedence(t *testing.T) {
	// Test: print(1 + 2 * 3) - should be 7, not 9
	expression := "print(1 + 2 * 3)"
	wasmBytes := compileExpression(t, expression)

	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	be.Equal(t, output, "7\n")
}

func TestDivisionAndModulo(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"print(20 / 4)", "5\n"},
		{"print(23 % 5)", "3\n"},
		{"print(15 / 3 + 2)", "7\n"}, // Division has higher precedence than addition
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			wasmBytes := compileExpression(t, test.expr)
			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

func TestComparisons(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"print(5 > 3)", "1\n"},  // true = 1 in WASM i64
		{"print(3 > 5)", "0\n"},  // false = 0 in WASM i64
		{"print(5 == 5)", "1\n"}, // true
		{"print(5 != 3)", "1\n"}, // true
		{"print(3 < 5)", "1\n"},  // true
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			wasmBytes := compileExpression(t, test.expr)
			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

func TestNestedExpressions(t *testing.T) {
	// Test deeply nested expression: print(((2 + 3) * 4 - 8) / 2 + 1)
	// Should be: (5 * 4 - 8) / 2 + 1 = (20 - 8) / 2 + 1 = 12 / 2 + 1 = 6 + 1 = 7
	expression := "print(((2 + 3) * 4 - 8) / 2 + 1)"
	wasmBytes := compileExpression(t, expression)

	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	be.Equal(t, output, "7\n")
}

func TestAddressOfOperations(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		// Test address of variable - should print some address value
		{"{ var x I64; x = 42; print(x&); }", "0\n"}, // First addressed variable at offset 0
		// Test multiple addressed variables
		{"{ var x I64; var y I64; x = 10; y = 20; print(x&); print(y&); }", "0\n8\n"}, // x at 0, y at 8
		// Test address of rvalue expression
		{"{ var x I64; x = 5; print((x + 10)&); }", "0\n"}, // Expression result stored at tstack=0
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			input := []byte(test.expr + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()
			wasmBytes := CompileToWASM(ast)

			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

func TestFullCapabilitiesDemo(t *testing.T) {
	// Comprehensive test of all supported operations
	tests := []struct {
		name     string
		expr     string
		expected string
		desc     string
	}{
		{"literals", "print(42)", "42\n", "Integer literals"},
		{"addition", "print(10 + 5)", "15\n", "Addition"},
		{"subtraction", "print(10 - 3)", "7\n", "Subtraction"},
		{"multiplication", "print(6 * 7)", "42\n", "Multiplication"},
		{"division", "print(20 / 4)", "5\n", "Division"},
		{"modulo", "print(17 % 5)", "2\n", "Modulo"},
		{"precedence", "print(2 + 3 * 4)", "14\n", "Operator precedence (mult before add)"},
		{"parentheses", "print((2 + 3) * 4)", "20\n", "Parentheses override precedence"},
		{"equal_true", "print(5 == 5)", "1\n", "Equality (true)"},
		{"equal_false", "print(5 == 3)", "0\n", "Equality (false)"},
		{"not_equal", "print(5 != 3)", "1\n", "Not equal"},
		{"greater_than", "print(5 > 3)", "1\n", "Greater than"},
		{"less_than", "print(3 < 5)", "1\n", "Less than"},
		{"complex", "print((10 + 5) * 2 - 3)", "27\n", "Complex nested expression"},
		{"mixed_ops", "print(20 / 4 + 3 * 2)", "11\n", "Mixed arithmetic with precedence"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing %s: %s", test.desc, test.expr)
			wasmBytes := compileExpression(t, test.expr)
			executeWasmAndVerify(t, wasmBytes, test.expected)
		})
	}
}

// TestPointerOperations tests comprehensive pointer functionality including:
// - Basic pointer assignment and dereferencing (ptr = var&, print(ptr*))
// - Bidirectional synchronization (modify via pointer, read via variable and vice versa)
// - Multiple pointers to the same target
// - Pointer dereferencing in arithmetic expressions
// - Assignment through pointer dereferencing (ptr* = value)
func TestPointerOperations(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
		desc     string
	}{
		{
			"basic_pointer_assignment",
			"{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }",
			"42\n",
			"Basic pointer: assign address, dereference to read value",
		},
		{
			"modify_via_pointer_read_via_var",
			"{ var x I64; var ptr I64*; x = 10; ptr = x&; ptr* = 99; print(x); }",
			"99\n",
			"Modify pointee via pointer, read via original variable",
		},
		{
			"modify_via_var_read_via_pointer",
			"{ var x I64; var ptr I64*; x = 25; ptr = x&; x = 77; print(ptr*); }",
			"77\n",
			"Modify via variable, read via pointer",
		},
		{
			"pointer_in_arithmetic",
			"{ var x I64; var ptr I64*; x = 7; ptr = x&; print(ptr* + 3); }",
			"10\n",
			"Use pointer dereference in arithmetic expression",
		},
		{
			"multiple_pointers_same_target",
			"{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 123; ptr1 = x&; ptr2 = x&; print(ptr1*); print(ptr2*); ptr1* = 456; print(ptr2*); }",
			"123\n123\n456\n",
			"Multiple pointers to same variable - modify via one, read via another",
		},
		{
			"sequential_pointer_ops",
			"{ var x I64; var ptr I64*; x = 100; ptr = x&; print(ptr*); ptr* = 200; print(x); }",
			"100\n200\n",
			"Sequential pointer operations on same variable",
		},
		{
			"pointer_in_expression",
			"{ var x I64; var y I64; var ptr I64*; x = 8; y = 7; ptr = x&; print(ptr* * y + 6); }",
			"62\n",
			"Use pointer dereference in complex expression: (8 * 7 + 6)",
		},
		{
			"pointer_modification_sequence",
			"{ var x I64; var ptr I64*; x = 5; ptr = x&; ptr* = ptr* + 1; print(x); ptr* = ptr* * 2; print(x); }",
			"6\n12\n",
			"Sequential modifications via pointer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing %s: %s", test.desc, test.expr)
			input := []byte(test.expr + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()
			wasmBytes := CompileToWASM(ast)

			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

// TestAdvancedPointerScenarios tests more complex pointer use cases including:
// - Address-of complex expressions stored on stack
// - Pointer chains and transitive modifications
// - Expression result addresses with proper stack frame management
func TestAdvancedPointerScenarios(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		expected string
		desc     string
	}{
		{
			"pointer_to_expression_result",
			"{ var x I64; x = 5; print((x + 10)&); print((x * 2)&); }",
			"0\n8\n",
			"Address-of expressions stored on stack at different offsets",
		},
		{
			"complex_pointer_assignment",
			"{ var a I64; var b I64; var c I64; var ptr I64*; a = 1; b = 2; c = 3; ptr = (a + b)&; print(ptr*); ptr = (b * c)&; print(ptr*); }",
			"3\n6\n",
			"Pointer to complex expression results",
		},
		{
			"pointer_chain_modification",
			"{ var x I64; var ptr1 I64*; var ptr2 I64*; x = 50; ptr1 = x&; ptr2 = ptr1; ptr2* = 75; print(x); print(ptr1*); }",
			"75\n75\n",
			"Chain of pointer assignments - modify through second pointer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Testing %s: %s", test.desc, test.expr)
			input := []byte(test.expr + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()
			wasmBytes := CompileToWASM(ast)

			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

func TestTypeASTInCompilation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "I64 variable with TypeAST",
			code:     "{ var x I64; x = 42; print(x); }",
			expected: "42\n",
		},
		{
			name:     "Second I64 variable with TypeAST",
			code:     "{ var y I64; y = 7; print(y); }",
			expected: "7\n",
		},
		{
			name:     "Pointer variable with TypeAST",
			code:     "{ var ptr I64*; var x I64; x = 99; ptr = x&; print(ptr*); }",
			expected: "99\n",
		},
		{
			name:     "Multiple types with TypeAST",
			code:     "{ var x I64; var y I64; var ptr I64*; x = 10; y = 0; ptr = x&; print(x); print(y); print(ptr*); }",
			expected: "10\n0\n10\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.code + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()

			// Verify TypeAST is populated in the parsed AST
			verifyTypeASTPopulated(t, ast)

			// Compile and execute
			wasmBytes := CompileToWASM(ast)
			output, err := executeWasm(t, wasmBytes)
			be.Err(t, err, nil)
			be.Equal(t, output, test.expected)
		})
	}
}

// Helper function to verify TypeAST is populated in variable declarations
func verifyTypeASTPopulated(t *testing.T, node *ASTNode) {
	if node == nil {
		return
	}

	switch node.Kind {
	case NodeVar:
		be.True(t, node.TypeAST != nil)
		// No need to verify string representation since we're not storing it anymore
	case NodeBlock, NodeIf, NodeLoop:
		for _, child := range node.Children {
			verifyTypeASTPopulated(t, child)
		}
	}
}

// Tests for EmitExpression edge cases
func TestEmitExpressionUndefinedVariable(t *testing.T) {
	defer func() {
		r := recover()
		be.True(t, r != nil)
		if r != nil {
			be.True(t, strings.Contains(r.(string), "Undefined variable"))
		}
	}()

	var buf bytes.Buffer
	locals := []LocalVarInfo{}

	// Create assignment to undefined variable
	node := &ASTNode{
		Kind: NodeBinary,
		Op:   "=",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "undefinedVar"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	EmitExpression(&buf, node, locals)
}

func TestEmitExpressionInvalidAssignmentTarget(t *testing.T) {
	defer func() {
		r := recover()
		be.True(t, r != nil)
		if r != nil {
			be.True(t, strings.Contains(r.(string), "Invalid assignment target"))
		}
	}()

	var buf bytes.Buffer
	locals := []LocalVarInfo{}

	// Create assignment to integer literal (invalid)
	node := &ASTNode{
		Kind: NodeBinary,
		Op:   "=",
		Children: []*ASTNode{
			{Kind: NodeInteger, Integer: 10}, // Invalid LHS
			{Kind: NodeInteger, Integer: 42},
		},
	}

	EmitExpression(&buf, node, locals)
}

// Tests for EmitAddressOf edge cases
func TestEmitAddressOfUndefinedVariable(t *testing.T) {
	defer func() {
		r := recover()
		be.True(t, r != nil)
		if r != nil {
			be.True(t, strings.Contains(r.(string), "Undefined variable"))
		}
	}()

	var buf bytes.Buffer
	locals := []LocalVarInfo{}

	operand := &ASTNode{
		Kind:   NodeIdent,
		String: "undefinedVar",
	}

	EmitAddressOf(&buf, operand, locals)
}

func TestEmitAddressOfNonAddressedVariable(t *testing.T) {
	defer func() {
		r := recover()
		be.True(t, r != nil)
		if r != nil {
			be.True(t, strings.Contains(r.(string), "not addressed"))
		}
	}()

	var buf bytes.Buffer
	locals := []LocalVarInfo{
		{
			Name:    "localVar",
			Type:    &TypeNode{Kind: TypeBuiltin, String: "I64"},
			Storage: VarStorageLocal, // Not VarStorageTStack
			Address: 0,
		},
	}

	operand := &ASTNode{
		Kind:   NodeIdent,
		String: "localVar",
	}

	EmitAddressOf(&buf, operand, locals)
}

// Test for stack variable access with address-of operator
func TestStackVariableAddressAccess(t *testing.T) {
	source := "{ var a I64; var b I64; a = 0; b = 0; print(a&); print(b&); print(a); print(b); }"

	// Parse the source code
	Init([]byte(source + "\x00"))
	NextToken()
	ast := ParseStatement()

	// Should parse successfully
	if ast == nil {
		t.Fatal("Failed to parse source code")
	}

	// Compile to WASM and execute
	wasmBytes := CompileToWASM(ast)
	if len(wasmBytes) == 0 {
		t.Fatal("Failed to compile to WASM")
	}

	// Execute and verify output (addresses should be different)
	output, err := executeWasm(t, wasmBytes)
	if err != nil {
		t.Fatalf("Failed to execute WASM: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("Expected at least 2 output lines, got %d: %v", len(lines), lines)
	}

	// Addresses should be different (distinct values)
	be.True(t, lines[0] != lines[1])
}
