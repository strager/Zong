package main

import (
	"strings"
	"testing"

	"github.com/nalgeon/be"
)

// Test 2: String literal type checking
func TestStringLiteralTypeChecking(t *testing.T) {
	input := []byte(`var s U8[] = "hello";` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Build symbol table and run type checking
	_ = BuildSymbolTable(ast)
	err := CheckProgram(ast)
	be.Err(t, err, nil) // Should not error

	// Check that string literal has correct type
	varDecl := ast.Children[0]
	stringLiteral := varDecl.Children[1]
	be.True(t, stringLiteral.TypeAST != nil)
	be.Equal(t, stringLiteral.TypeAST.Kind, TypeSlice)
	be.Equal(t, stringLiteral.TypeAST.Child.String, "U8")
}

// Test 3: String collection during compilation
func TestStringCollection(t *testing.T) {
	input := []byte(`var s U8[] = "hello"; var t U8[] = "world"; var u U8[] = "hello";` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

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

// Test 4: Data section size calculation
func TestDataSectionSize(t *testing.T) {
	strings := []StringLiteral{
		{Content: "hello", Address: 0, Length: 5},
		{Content: "world", Address: 5, Length: 5},
	}

	totalSize := calculateDataSectionSize(strings)
	be.Equal(t, totalSize, uint32(10))
}

// Test 5: WASM compilation succeeds (without execution)
func TestWASMCompilationSuccess(t *testing.T) {
	// Test that compilation doesn't crash, even if execution fails
	input := []byte(`func main() { var s U8[] = "hello"; }` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WASM compilation panicked: %v", r)
		}
	}()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test 6: Data section is included in WASM output
func TestDataSectionInWASM(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "hello"; }` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

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

// Test 7: Global string addresses are populated
func TestGlobalStringAddresses(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "test"; }` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Compile to populate global string addresses
	CompileToWASM(ast)

	// Check that global string addresses were populated
	be.True(t, globalStringAddresses != nil)
	be.True(t, len(globalStringAddresses) > 0)

	address, exists := globalStringAddresses["test"]
	be.True(t, exists)
	be.Equal(t, address, uint32(0)) // First string should be at address 0
}

// Test 8: Multiple different string literals get different addresses
func TestMultipleStringAddresses(t *testing.T) {
	input := []byte(`func main() { var s U8[] = "first"; var t U8[] = "second"; }` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	CompileToWASM(ast)

	firstAddr, exists1 := globalStringAddresses["first"]
	secondAddr, exists2 := globalStringAddresses["second"]

	be.True(t, exists1)
	be.True(t, exists2)
	be.True(t, firstAddr != secondAddr) // Should have different addresses
}

// Test 9: Empty string handling
func TestEmptyString(t *testing.T) {
	input := []byte(`func main() { var s U8[] = ""; }` + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

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
