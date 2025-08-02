// Code generation and WASM compilation tests
//
// Tests WASM generation from typed ASTs (typed AST → executable WASM).
// Covers WASM utilities, string compilation, and execution helpers for integration tests.

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

// =============================================================================
// WASM UTILITY TESTS
// =============================================================================

func TestWriteByte(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	writeByte(&buf, 0x42)
	writeByte(&buf, 0xFF)

	be.True(t, bytes.Equal(buf.Bytes(), []byte{0x42, 0xFF}))
}

func TestWriteBytes(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := []byte{0x01, 0x02, 0x03}
	writeBytes(&buf, data)

	be.True(t, bytes.Equal(buf.Bytes(), data))
}

func TestWriteLEB128(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    uint32
		expected []byte
	}{
		{0, []byte{0x00}},
		{127, []byte{0x7F}},
		{128, []byte{0x80, 0x01}},
		{300, []byte{0xAC, 0x02}},
		{16384, []byte{0x80, 0x80, 0x01}},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		writeLEB128(&buf, test.input)
		be.True(t, bytes.Equal(buf.Bytes(), test.expected))
	}
}

func TestWriteLEB128Signed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    int64
		expected []byte
	}{
		{0, []byte{0x00}},
		{1, []byte{0x01}},
		{-1, []byte{0x7F}},
		{127, []byte{0xFF, 0x00}},
		{-128, []byte{0x80, 0x7F}},
		{128, []byte{0x80, 0x01}},
		{-129, []byte{0xFF, 0x7E}},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		writeLEB128Signed(&buf, test.input)
		be.True(t, bytes.Equal(buf.Bytes(), test.expected))
	}
}

func TestEmitWASMHeader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	EmitWASMHeader(&buf)

	// WASM magic number (0x00 0x61 0x73 0x6D) + version (0x01 0x00 0x00 0x00)
	be.True(t, bytes.Equal(buf.Bytes(), []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}))
}

func TestEmitImportSection(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	EmitImportSection(&buf)

	result := buf.Bytes()

	// Should start with import section ID (0x02)
	be.Equal(t, result[0], byte(0x02))

	// Should contain "env", "print", and "print_bytes" strings
	be.True(t, containsBytes(result, []byte("env")))
	be.True(t, containsBytes(result, []byte("print")))
	be.True(t, containsBytes(result, []byte("print_bytes")))
}

func TestEmitFunctionSection(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	EmitFunctionSection(&buf, []*ASTNode{}) // empty functions list

	result := buf.Bytes()

	// Should start with function section ID (0x03)
	be.Equal(t, result[0], byte(0x03))
}

func TestEmitExportSection(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	EmitExportSection(&buf)

	result := buf.Bytes()

	// Should start with export section ID (0x07)
	be.Equal(t, result[0], byte(0x07))

	// Should contain "main" string
	be.True(t, containsBytes(result, []byte("main")))
}

func containsBytes(haystack, needle []byte) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if bytes.Equal(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func TestEmitExpression(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ast      *ASTNode
		expected []byte
	}{
		{
			name: "integer constant",
			ast: &ASTNode{
				Kind:    NodeInteger,
				Integer: 42,
				TypeAST: TypeI64, // Set explicit type for WASM emission test
			},
			expected: []byte{I64_CONST, 42}, // i64.const 42
		},
		{
			name: "simple addition",
			ast: &ASTNode{
				Kind:    NodeBinary,
				Op:      "+",
				TypeAST: TypeI64, // Result type for WASM emission test
				Children: []*ASTNode{
					{Kind: NodeInteger, Integer: 1, TypeAST: TypeI64},
					{Kind: NodeInteger, Integer: 2, TypeAST: TypeI64},
				},
			},
			expected: []byte{I64_CONST, 1, I64_CONST, 2, I64_ADD}, // i64.const 1, i64.const 2, i64.add
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			localCtx := &LocalContext{
				Variables: []LocalVarInfo{},
			}
			EmitExpression(&buf, test.ast, localCtx)
			be.True(t, bytes.Equal(buf.Bytes(), test.expected))
		})
	}
}

func TestCompileToWASMIntegration(t *testing.T) {
	// Test parsing and compiling a simple expression
	input := []byte("42 + 8\x00")
	l := NewLexer(input)
	l.NextToken()

	ast := ParseExpression(l)
	be.Equal(t, ToSExpr(ast), "(binary \"+\" (integer 42) (integer 8))")

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) >= 8)

	// Verify WASM header
	be.True(t, bytes.Equal(wasmBytes[:8], []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}))

	t.Logf("Successfully compiled '42 + 8' to %d bytes of WASM", len(wasmBytes))
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

// =============================================================================
// STRING LITERAL COMPILATION TESTS
// =============================================================================

func TestStringCollection(t *testing.T) {
	input := []byte(`var s U8[] = "hello"; var t U8[] = "world"; var u U8[] = "hello";` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Collect strings
	strings := collectStringLiterals(ast)

	// Should have 2 unique strings (hello, world) due to deduplication
	be.Equal(t, len(strings), 2)

	// Check string contents and addresses
	foundHello := false
	foundWorld := false
	for _, str := range strings {
		if str.Content == "hello" {
			foundHello = true
			be.Equal(t, str.Length, uint32(5))
		} else if str.Content == "world" {
			foundWorld = true
			be.Equal(t, str.Length, uint32(5))
		}
	}
	be.True(t, foundHello)
	be.True(t, foundWorld)
}

func TestDataSectionSize(t *testing.T) {
	strings := []StringLiteral{
		{Content: "hello", Address: 0, Length: 5},
		{Content: "world", Address: 5, Length: 5},
	}

	totalSize := calculateDataSectionSize(strings)
	be.Equal(t, totalSize, uint32(10))
}

func TestWASMCompilationSuccess(t *testing.T) {
	// Test that compilation doesn't crash, even if execution fails
	input := []byte(`func main() { var s U8[] = "hello"; }` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WASM compilation panicked: %v", r)
		}
	}()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

func TestDataSectionInWASM(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "hello"; }` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	wasmBytes := CompileToWASM(ast)

	// Check that "hello" appears in the WASM bytes (data section)
	wasmString := string(wasmBytes)
	be.True(t, strings.Contains(wasmString, "hello"))

	// Check for data section ID (0x0B)
	foundDataSection := false
	for _, b := range wasmBytes {
		if b == 0x0B {
			foundDataSection = true
			break
		}
	}
	be.True(t, foundDataSection)
}

func TestGlobalStringAddresses(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "test"; }` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Compile to populate global string addresses
	CompileToWASM(ast)

	// Check that global string addresses were populated
	be.True(t, globalStringAddresses != nil)
	be.True(t, len(globalStringAddresses) > 0)

	address, exists := globalStringAddresses["test"]
	be.True(t, exists)
	be.Equal(t, address, uint32(0)) // First string should be at address 0
}

func TestMultipleStringAddresses(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "first"; var t U8[] = "second"; }` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	CompileToWASM(ast)

	firstAddr, exists1 := globalStringAddresses["first"]
	secondAddr, exists2 := globalStringAddresses["second"]

	be.True(t, exists1)
	be.True(t, exists2)
	be.True(t, firstAddr != secondAddr) // Should have different addresses
}

func TestEmptyString(t *testing.T) {
	input := []byte(`func main() { var s U8[] = ""; }` + "\x00")
	l := NewLexer(input)
	l.NextToken()
	ast := ParseProgram(l)

	// Should not crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Empty string handling panicked: %v", r)
		}
	}()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)

	// Check that empty string is recorded
	address, exists := globalStringAddresses[""]
	be.True(t, exists)
	be.Equal(t, address, uint32(0)) // Empty string should be at address 0
}

// =============================================================================
// WASM EXECUTION HELPERS (for integration tests)
// =============================================================================

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
	l := NewLexer(input)
	l.NextToken()
	ast := ParseExpression(l)

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
