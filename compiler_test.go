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
	cmd = exec.Command("wasm-objdump", "-d", "-h", wasmFilePath)
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
	if err != nil || output != expected {
		absPath, _ := filepath.Abs(wasmFile)
		t.Logf("WASM file path: %s", absPath)

		if watContent, watErr := convertWasmToWat(wasmBytes, wasmFile); watErr == nil {
			t.Logf("Human-readable WASM (WAT format):\n%s", watContent)
		} else {
			t.Logf("Could not convert to WAT: %v", watErr)
		}
	}
	be.Err(t, err, nil)
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

// Helper function to append test cases to Sexy format files
func appendSexyTest(filename, testName, input, inputType, expected string) {
	// Create test directory if it doesn't exist
	testDir := "test"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create test directory: %v\n", err)
		return
	}

	filePath := filepath.Join(testDir, filename)

	// Open file for appending (create if doesn't exist)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	// Write the test case in Sexy format
	content := fmt.Sprintf("### Test: %s\n```%s\n%s\n```\n```execute\n%s\n```\n\n",
		testName, inputType, input, expected)

	if _, err := file.WriteString(content); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to file %s: %v\n", filePath, err)
	}
}

// Test: print(42)

// Verify WASM was generated

// Execute and verify output

// Test: print(42 + 8)

// Test: print((10 + 5) * 2 - 3)

// Test: print(1 + 2 * 3) - should be 7, not 9

// Test deeply nested expression: print(((2 + 3) * 4 - 8) / 2 + 1)
// Should be: (5 * 4 - 8) / 2 + 1 = (20 - 8) / 2 + 1 = 12 / 2 + 1 = 6 + 1 = 7

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
	localCtx := &LocalContext{
		Variables: []LocalVarInfo{},
	}

	// Create assignment to undefined variable
	node := &ASTNode{
		Kind: NodeBinary,
		Op:   "=",
		Children: []*ASTNode{
			{Kind: NodeIdent, String: "undefinedVar"},
			{Kind: NodeInteger, Integer: 42},
		},
	}

	EmitExpression(&buf, node, localCtx)
}

func TestEmitExpressionInvalidAssignmentTarget(t *testing.T) {
	defer func() {
		r := recover()
		be.True(t, r != nil)
		if r != nil {
			be.True(t, strings.Contains(r.(string), "Invalid assignment target - must be variable, field access, pointer dereference, or slice index"))
		}
	}()

	var buf bytes.Buffer
	localCtx := &LocalContext{
		Variables: []LocalVarInfo{},
	}

	// Create assignment to integer literal (invalid)
	node := &ASTNode{
		Kind: NodeBinary,
		Op:   "=",
		Children: []*ASTNode{
			{Kind: NodeInteger, Integer: 10}, // Invalid LHS
			{Kind: NodeInteger, Integer: 42},
		},
	}

	EmitExpression(&buf, node, localCtx)
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
	localCtx := &LocalContext{
		Variables: []LocalVarInfo{},
	}

	operand := &ASTNode{
		Kind:   NodeIdent,
		String: "undefinedVar",
	}

	EmitAddressOf(&buf, operand, localCtx)
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
	symbol := &SymbolInfo{
		Name:     "localVar",
		Type:     &TypeNode{Kind: TypeBuiltin, String: "I64"},
		Assigned: false,
	}
	localCtx := &LocalContext{
		Variables: []LocalVarInfo{
			{
				Symbol:  symbol,
				Storage: VarStorageLocal, // Not VarStorageTStack
				Address: 0,
			},
		},
	}

	operand := &ASTNode{
		Kind:   NodeIdent,
		String: "localVar",
		Symbol: symbol,
	}

	EmitAddressOf(&buf, operand, localCtx)
}

// TestStackVariableAddressAccess removed - now covered by test/TestAddressOfOperations_test.md

// Helper function to test that compilation fails with type checking error
func expectTypeError(t *testing.T, code string, expectedError string) {
	input := []byte(code + "\x00")

	// Parse
	Init(input)
	NextToken()
	ast := ParseStatement()

	// Should parse successfully
	if ast == nil {
		t.Fatal("Failed to parse source code")
	}

	// Compilation should fail with type error
	defer func() {
		r := recover()
		be.True(t, r != nil)
		errorMsg := fmt.Sprintf("%v", r)
		be.True(t, strings.Contains(errorMsg, expectedError))
	}()

	CompileToWASM(ast)
}

func TestTypeCheckingErrors(t *testing.T) {
	tests := []struct {
		name          string
		code          string
		expectedError string
	}{
		{
			name:          "variable used before declaration",
			code:          "print(undefined_var)",
			expectedError: "undefined symbol 'undefined_var'",
		},
		{
			name:          "variable used before assignment",
			code:          "{ var x I64; print(x); }",
			expectedError: "variable 'x' used before assignment",
		},
		{
			name:          "duplicate variable declaration",
			code:          "{ var x I64; var x I64; }",
			expectedError: "variable 'x' already declared",
		},
		{
			name:          "invalid assignment target",
			code:          "42 = 10",
			expectedError: "left side of assignment must be a variable, field access, or dereferenced pointer",
		},
		{
			name:          "dereference non-pointer",
			code:          "{ var x I64; x = 42; print(x*); }",
			expectedError: "cannot dereference non-pointer type I64",
		},
		{
			name:          "unknown function call",
			code:          "unknown_func(42)",
			expectedError: "undefined symbol 'unknown_func'",
		},
		{
			name:          "print with wrong argument count",
			code:          "print()",
			expectedError: "print() function expects 1 argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectTypeError(t, tt.code, tt.expectedError)
		})
	}
}

// Struct Integration Tests - Execute programs and verify output

// Test that struct fields are zero-initialized by default

// Test that struct variables work alongside regular I64 variables

// Test basic if statement

// Test if-else statement

// Test else-if chain

// Test nested if statements

// Test if statement with false condition (should print nothing)
// TestIfStatementFalse removed - now covered by test/statements_test.md
