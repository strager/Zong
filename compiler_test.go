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
