package main

import (
	"fmt"
	"testing"
)

// formatHex converts bytes to readable hexadecimal representation
func formatHex(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	result := ""
	for i, b := range data {
		if i > 0 && i%16 == 0 {
			result += "\n"
		}
		if i > 0 && i%8 == 0 && i%16 != 0 {
			result += "  "
		}
		if i > 0 && i%16 != 0 {
			result += " "
		}
		result += fmt.Sprintf("%02X", b)
	}
	return result
}

// debugCompilePointerTest compiles a pointer test case and outputs detailed debugging info
func debugCompilePointerTest(t *testing.T, testCode string) {
	t.Logf("=== Debugging pointer test case ===")
	t.Logf("Code: %s", testCode)

	// Parse the code
	input := []byte(testCode + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	if ast == nil {
		t.Fatal("Failed to parse test code")
	}

	t.Logf("AST: %s", ToSExpr(ast))

	// Build symbol table
	symbolTable := BuildSymbolTable(ast)
	t.Logf("Symbol table built successfully")

	// Collect local variables
	locals, frameSize := collectLocalVariables(ast, symbolTable)
	t.Logf("Frame size: %d bytes", frameSize)

	for i, local := range locals {
		t.Logf("  Local[%d]: %s, Type: %s, Storage: %v, Address: %d",
			i, local.Name, TypeToString(local.Type), local.Storage, local.Address)
	}

	// Try to compile - this should reveal where the error occurs
	defer func() {
		if r := recover(); r != nil {
			t.Logf("COMPILATION PANIC: %v", r)
			// Don't re-panic so we can see the debug output
		}
	}()

	wasmBytes := CompileToWASM(ast)

	if len(wasmBytes) > 0 {
		t.Logf("Successfully compiled %d bytes of WASM", len(wasmBytes))
		t.Logf("WASM bytes (hex):\n%s", formatHex(wasmBytes))

		// Try to execute it
		output, err := executeWasm(t, wasmBytes)
		if err != nil {
			t.Logf("Execution failed: %v", err)
		} else {
			t.Logf("Output: %q", output)
		}
	}
}

// TestDebugBasicPointerOperations tests the most basic pointer operations to identify issues
func TestDebugBasicPointerOperations(t *testing.T) {
	testCases := []string{
		// Test 1: Simple variable address
		"{ var x I64; x = 42; print(x&); }",

		// Test 2: Simple pointer assignment and dereference
		"{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }",

		// Test 3: Pointer modification
		"{ var x I64; var ptr I64*; x = 42; ptr = x&; ptr* = 99; print(x); }",
	}

	for i, testCode := range testCases {
		t.Run(fmt.Sprintf("test_%d", i+1), func(t *testing.T) {
			debugCompilePointerTest(t, testCode)
		})
	}
}

// TestDebugCompareWorkingVsFailingCases compares working address-of with failing pointer operations
func TestDebugCompareWorkingVsFailingCases(t *testing.T) {
	workingCases := []string{
		// These should work (based on existing tests)
		"{ var x I64; x = 42; print(x&); }",       // Address-of variable
		"{ var x I64; x = 5; print((x + 10)&); }", // Address-of expression
	}

	failingCases := []string{
		// These are expected to fail with i32/i64 mismatch
		"{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }", // Basic pointer dereference
		"{ var x I64; var ptr I64*; x = 42; ptr = x&; ptr* = 99; }",   // Pointer assignment
	}

	t.Log("=== WORKING CASES ===")
	for i, testCode := range workingCases {
		t.Run(fmt.Sprintf("working_%d", i+1), func(t *testing.T) {
			debugCompilePointerTest(t, testCode)
		})
	}

	t.Log("\n=== FAILING CASES ===")
	for i, testCode := range failingCases {
		t.Run(fmt.Sprintf("failing_%d", i+1), func(t *testing.T) {
			debugCompilePointerTest(t, testCode)
		})
	}
}

// TestDebugWASMInstructionAnalysis specifically analyzes the WASM instructions generated
func TestDebugWASMInstructionAnalysis(t *testing.T) {
	// Test the specific case from the issue description
	testCode := "{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }"

	t.Logf("=== WASM Instruction Analysis ===")
	t.Logf("Test case: %s", testCode)

	// Parse
	input := []byte(testCode + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	if ast == nil {
		t.Fatal("Failed to parse")
	}

	// Build symbol table and collect locals
	symbolTable := BuildSymbolTable(ast)
	locals, _ := collectLocalVariables(ast, symbolTable)

	t.Logf("Locals collected:")
	for i, local := range locals {
		t.Logf("  [%d] %s: %s, storage=%v, addr=%d",
			i, local.Name, TypeToString(local.Type), local.Storage, local.Address)
	}

	// Now let's manually trace through the EmitExpression calls to see where the issue occurs
	t.Logf("\n=== Manual WASM generation trace ===")

	// Let's look at the AST structure
	t.Logf("AST structure: %s", ToSExpr(ast))

	// The issue is likely in the print(ptr*) part
	// Let's trace what happens when we try to emit code for ptr*

	defer func() {
		if r := recover(); r != nil {
			t.Logf("ERROR during compilation: %v", r)

			// Try to analyze which specific instruction caused the issue
			t.Logf("This suggests the i32/i64 mismatch occurs during:")
			t.Logf("  1. Pointer dereference (ptr*)")
			t.Logf("  2. Memory load operations")
			t.Logf("  3. Type conversion between i32 pointers and i64 addresses")
		}
	}()

	wasmBytes := CompileToWASM(ast)

	if len(wasmBytes) > 0 {
		t.Logf("Successfully compiled to WASM")
		t.Logf("Total bytes: %d", len(wasmBytes))
		t.Logf("Hex dump:\n%s", formatHex(wasmBytes))

		// Analyze the code section specifically
		analyzeWASMCodeSection(t, wasmBytes)
	}
}

// analyzeWASMCodeSection attempts to find and analyze the code section
func analyzeWASMCodeSection(t *testing.T, wasmBytes []byte) {
	t.Logf("\n=== WASM Code Section Analysis ===")

	// Look for the code section (section ID 0x0A)
	for i := 0; i < len(wasmBytes)-1; i++ {
		if wasmBytes[i] == 0x0A {
			t.Logf("Found code section at offset %d (0x%02X)", i, i)

			// Try to extract some instructions from after the section header
			if i+10 < len(wasmBytes) {
				instructions := wasmBytes[i : i+min(50, len(wasmBytes)-i)]
				t.Logf("Code section bytes:\n%s", formatHex(instructions))

				// Look for specific WASM opcodes we expect
				analyzeInstructions(t, instructions)
			}
			break
		}
	}
}

// analyzeInstructions looks for specific WASM opcodes in the byte stream
func analyzeInstructions(t *testing.T, instructions []byte) {
	t.Logf("\n=== Instruction Analysis ===")

	opcodeNames := map[byte]string{
		I32_CONST:        "I32_CONST",
		I32_WRAP_I64:     "I32_WRAP_I64",
		I64_CONST:        "I64_CONST",
		I64_ADD:          "I64_ADD",
		I64_LOAD:         "I64_LOAD",
		I64_STORE:        "I64_STORE",
		I64_EXTEND_I32_S: "I64_EXTEND_I32_S",
		GLOBAL_GET:       "GLOBAL_GET",
		GLOBAL_SET:       "GLOBAL_SET",
		LOCAL_GET:        "LOCAL_GET",
		LOCAL_SET:        "LOCAL_SET",
		CALL:             "CALL",
		END:              "END",
	}

	for i, b := range instructions {
		if name, found := opcodeNames[b]; found {
			t.Logf("  Offset %d: %s (0x%02X)", i, name, b)
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestDebugTypeConversions specifically tests the type conversion logic
func TestDebugTypeConversions(t *testing.T) {
	// Test what happens with different type scenarios
	testCases := []struct {
		name string
		code string
		desc string
	}{
		{
			"i64_variable",
			"{ var x I64; x = 42; print(x); }",
			"Basic I64 variable (should work)",
		},
		{
			"i64_pointer_variable",
			"{ var ptr I64*; }",
			"I64 pointer variable declaration only",
		},
		{
			"address_of_i64",
			"{ var x I64; x = 42; print(x&); }",
			"Address-of I64 variable (should work)",
		},
		{
			"pointer_assignment",
			"{ var x I64; var ptr I64*; x = 42; ptr = x&; }",
			"Pointer assignment (no dereference)",
		},
		{
			"pointer_dereference",
			"{ var x I64; var ptr I64*; x = 42; ptr = x&; print(ptr*); }",
			"Pointer dereference (likely fails here)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing: %s", tc.desc)
			debugCompilePointerTest(t, tc.code)
		})
	}
}

// TestDebugSpecificWASMGeneration tests the specific WASM generation functions
func TestDebugSpecificWASMGeneration(t *testing.T) {
	t.Log("=== Testing specific WASM generation functions ===")

	// Test the functions that deal with type conversions
	testTypeHelpers(t)
	testAddressOfGeneration(t)
	testPointerDereferenceGeneration(t)
}

func testTypeHelpers(t *testing.T) {
	t.Log("\n--- Type Helper Functions ---")

	// Test type checking functions
	i64Type := &TypeNode{Kind: TypeBuiltin, String: "I64"}
	ptrType := &TypeNode{Kind: TypePointer, Child: i64Type}

	t.Logf("I64 type -> isWASMI64Type: %v, isWASMI32Type: %v",
		isWASMI64Type(i64Type), isWASMI32Type(i64Type))
	t.Logf("I64* type -> isWASMI64Type: %v, isWASMI32Type: %v",
		isWASMI64Type(ptrType), isWASMI32Type(ptrType))
}

func testAddressOfGeneration(t *testing.T) {
	t.Log("\n--- Address-of Generation Test ---")

	t.Logf("Address-of generation test would emit address calculation for variable 'x'")
	t.Logf("This is working in existing tests, so the issue is likely elsewhere")
}

func testPointerDereferenceGeneration(t *testing.T) {
	t.Log("\n--- Pointer Dereference Generation Test ---")

	t.Logf("Pointer dereference test: This is where the i32/i64 mismatch likely occurs")
	t.Logf("The issue is probably in the sequence:")
	t.Logf("  1. Load pointer value (i32)")
	t.Logf("  2. Convert to i64 for memory address")
	t.Logf("  3. Load from memory")

	// The actual issue is likely in the EmitExpression function for NodeUnary with "*" operator
	// Looking at main.go lines 559-566, the code does:
	// 1. EmitExpression(buf, node.Children[0], locals) // Get the pointer value (i32)
	// 2. writeByte(buf, I64_EXTEND_I32_S)              // Convert i32 pointer to i64 address
	// 3. writeByte(buf, I64_LOAD)                      // Load i64 from memory

	t.Logf("Expected WASM instruction sequence:")
	t.Logf("  LOCAL_GET ptr_index  ; Load pointer (i32)")
	t.Logf("  I64_EXTEND_I32_S     ; Convert i32 to i64 for memory address")
	t.Logf("  I64_LOAD             ; Load i64 value from memory")
}
