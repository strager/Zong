package main

import (
	"bytes"
	"testing"
)

func TestWriteByte(t *testing.T) {
	var buf bytes.Buffer
	writeByte(&buf, 0x42)
	writeByte(&buf, 0xFF)

	expected := []byte{0x42, 0xFF}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Expected %v, got %v", expected, buf.Bytes())
	}
}

func TestWriteBytes(t *testing.T) {
	var buf bytes.Buffer
	data := []byte{0x01, 0x02, 0x03}
	writeBytes(&buf, data)

	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("Expected %v, got %v", data, buf.Bytes())
	}
}

func TestWriteLEB128(t *testing.T) {
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
		if !bytes.Equal(buf.Bytes(), test.expected) {
			t.Errorf("LEB128(%d): expected %v, got %v", test.input, test.expected, buf.Bytes())
		}
	}
}

func TestWriteLEB128Signed(t *testing.T) {
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
		if !bytes.Equal(buf.Bytes(), test.expected) {
			t.Errorf("LEB128Signed(%d): expected %v, got %v", test.input, test.expected, buf.Bytes())
		}
	}
}

func TestEmitWASMHeader(t *testing.T) {
	var buf bytes.Buffer
	EmitWASMHeader(&buf)

	// WASM magic number (0x00 0x61 0x73 0x6D) + version (0x01 0x00 0x00 0x00)
	expected := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
	if !bytes.Equal(buf.Bytes(), expected) {
		t.Errorf("Expected %v, got %v", expected, buf.Bytes())
	}
}

func TestEmitImportSection(t *testing.T) {
	var buf bytes.Buffer
	EmitImportSection(&buf)

	result := buf.Bytes()

	// Should start with import section ID (0x02)
	if result[0] != 0x02 {
		t.Errorf("Expected section ID 0x02, got 0x%02x", result[0])
	}

	// Should contain "env" and "print" strings
	if !containsBytes(result, []byte("env")) {
		t.Error("Expected import section to contain 'env'")
	}
	if !containsBytes(result, []byte("print")) {
		t.Error("Expected import section to contain 'print'")
	}
}

func TestEmitFunctionSection(t *testing.T) {
	var buf bytes.Buffer
	EmitFunctionSection(&buf)

	result := buf.Bytes()

	// Should start with function section ID (0x03)
	if result[0] != 0x03 {
		t.Errorf("Expected section ID 0x03, got 0x%02x", result[0])
	}
}

func TestEmitExportSection(t *testing.T) {
	var buf bytes.Buffer
	EmitExportSection(&buf)

	result := buf.Bytes()

	// Should start with export section ID (0x07)
	if result[0] != 0x07 {
		t.Errorf("Expected section ID 0x07, got 0x%02x", result[0])
	}

	// Should contain "main" string
	if !containsBytes(result, []byte("main")) {
		t.Error("Expected export section to contain 'main'")
	}
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
			},
			expected: []byte{I64_CONST, 42}, // i64.const 42
		},
		{
			name: "simple addition",
			ast: &ASTNode{
				Kind: NodeBinary,
				Op:   "+",
				Children: []*ASTNode{
					{Kind: NodeInteger, Integer: 1},
					{Kind: NodeInteger, Integer: 2},
				},
			},
			expected: []byte{I64_CONST, 1, I64_CONST, 2, I64_ADD}, // i64.const 1, i64.const 2, i64.add
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			EmitExpression(&buf, test.ast)
			if !bytes.Equal(buf.Bytes(), test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, buf.Bytes())
			}
		})
	}
}

func TestCompileToWASM(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ast   *ASTNode
	}{
		{
			name:  "integer constant",
			input: "42",
			ast: &ASTNode{
				Kind:    NodeInteger,
				Integer: 42,
			},
		},
		{
			name:  "simple addition",
			input: "1 + 2",
			ast: &ASTNode{
				Kind: NodeBinary,
				Op:   "+",
				Children: []*ASTNode{
					{Kind: NodeInteger, Integer: 1},
					{Kind: NodeInteger, Integer: 2},
				},
			},
		},
		{
			name:  "complex expression",
			input: "(10 + 5) * 2 - 3",
			ast: &ASTNode{
				Kind: NodeBinary,
				Op:   "-",
				Children: []*ASTNode{
					{
						Kind: NodeBinary,
						Op:   "*",
						Children: []*ASTNode{
							{
								Kind: NodeBinary,
								Op:   "+",
								Children: []*ASTNode{
									{Kind: NodeInteger, Integer: 10},
									{Kind: NodeInteger, Integer: 5},
								},
							},
							{Kind: NodeInteger, Integer: 2},
						},
					},
					{Kind: NodeInteger, Integer: 3},
				},
			},
		},
		{
			name:  "print function call",
			input: "print(42)",
			ast: &ASTNode{
				Kind: NodeCall,
				Children: []*ASTNode{
					{Kind: NodeIdent, String: "print"},
					{Kind: NodeInteger, Integer: 42},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wasmBytes := CompileToWASM(test.ast)

			// Basic validation: check WASM magic number and version
			if len(wasmBytes) < 8 {
				t.Fatal("WASM output too short")
			}

			expectedHeader := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
			if !bytes.Equal(wasmBytes[:8], expectedHeader) {
				t.Errorf("Invalid WASM header. Expected %v, got %v", expectedHeader, wasmBytes[:8])
			}

			// Check that we have some content beyond the header
			if len(wasmBytes) <= 8 {
				t.Error("WASM output contains only header")
			}

			t.Logf("Generated %d bytes of WASM for input: %s", len(wasmBytes), test.input)
		})
	}
}

func TestCompileToWASMIntegration(t *testing.T) {
	// Test parsing and compiling a simple expression
	input := []byte("42 + 8\x00")
	Init(input)
	NextToken()

	ast := ParseExpression()
	expectedSExpr := "(binary \"+\" (integer 42) (integer 8))"
	if ToSExpr(ast) != expectedSExpr {
		t.Errorf("Expected AST %s, got %s", expectedSExpr, ToSExpr(ast))
	}

	wasmBytes := CompileToWASM(ast)
	if len(wasmBytes) < 8 {
		t.Fatal("WASM output too short")
	}

	// Verify WASM header
	expectedHeader := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
	if !bytes.Equal(wasmBytes[:8], expectedHeader) {
		t.Errorf("Invalid WASM header. Expected %v, got %v", expectedHeader, wasmBytes[:8])
	}

	t.Logf("Successfully compiled '42 + 8' to %d bytes of WASM", len(wasmBytes))
}
