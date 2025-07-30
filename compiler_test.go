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
// TestEmitExpressionUndefinedVariable removed - now covered by test/compile_error_test.md

// TestEmitExpressionInvalidAssignmentTarget removed - now covered by test/compile_error_test.md

// TestEmitAddressOfUndefinedVariable removed - now covered by test/compile_error_test.md

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

// Struct Integration Tests - Execute programs and verify output

// Test that struct fields are zero-initialized by default

// Test that struct variables work alongside regular I64 variables

// Test basic if statement

// Test if-else statement

// Test else-if chain

// Test nested if statements

// Test if statement with false condition (should print nothing)
// TestIfStatementFalse removed - now covered by test/statements_test.md
