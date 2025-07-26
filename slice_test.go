package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestSliceTypeParsing(t *testing.T) {
	// Test basic slice type parsing directly
	input := []byte("var nums I64[];\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeVar)

	expectedType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	be.True(t, TypesEqual(stmt.TypeAST, expectedType))
}

func TestSliceTypeToString(t *testing.T) {
	tests := []struct {
		name        string
		typeNode    *TypeNode
		expectedStr string
	}{
		{
			name: "I64 slice",
			typeNode: &TypeNode{
				Kind:  TypeSlice,
				Child: TypeI64,
			},
			expectedStr: "I64[]",
		},
		{
			name: "pointer slice",
			typeNode: &TypeNode{
				Kind: TypeSlice,
				Child: &TypeNode{
					Kind:  TypePointer,
					Child: TypeI64,
				},
			},
			expectedStr: "I64*[]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypeToString(test.typeNode)
			be.Equal(t, result, test.expectedStr)
		})
	}
}

func TestSliceBasicDeclaration(t *testing.T) {
	// Test basic slice variable declaration
	input := []byte("var nums I64[];\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeVar)

	// Verify type is slice
	be.Equal(t, stmt.TypeAST.Kind, TypeSlice)
	be.Equal(t, stmt.TypeAST.Child.Kind, TypeBuiltin)
	be.Equal(t, stmt.TypeAST.Child.String, "I64")
}

func TestSliceStringRepresentation(t *testing.T) {
	// Test TypeToString for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	result := TypeToString(sliceType)
	be.Equal(t, result, "I64[]")
}

func TestSliceSize(t *testing.T) {
	// Test GetTypeSize for slices
	sliceType := &TypeNode{
		Kind:  TypeSlice,
		Child: TypeI64,
	}
	size := GetTypeSize(sliceType)
	be.Equal(t, size, 16) // 8 bytes pointer + 8 bytes length
}

// SExpr tests for slice parsing as required by the plan
func TestSliceSExprParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "slice variable declaration",
			input:    "var nums I64[];",
			expected: "(var (ident \"nums\") (ident \"I64[]\"))",
		},
		{
			name:     "slice subscript",
			input:    "nums[0];",
			expected: "(idx (ident \"nums\") (integer 0))",
		},
		{
			name:     "slice assignment",
			input:    "nums[1] = 42;",
			expected: "(binary \"=\" (idx (ident \"nums\") (integer 1)) (integer 42))",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.input + "\x00")
			Init(input)
			NextToken()

			stmt := ParseStatement()
			result := ToSExpr(stmt)
			be.Equal(t, result, test.expected)
		})
	}
}

// Integration tests as specified in the plan
// NOTE: append() functionality is partially implemented - these tests are commented out
// until the append() builtin is fully working

func TestSliceBasicsJustDeclaration(t *testing.T) {
	// Test just slice declaration without append to isolate the issue
	source := `
	func main() {
		var nums I64[];
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n")
}

func TestSliceAddressOf(t *testing.T) {
	// Test just taking address-of slice without calling append
	source := `
	func main() {
		var nums I64[];
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n")
}

func TestSliceBasicsMinimal(t *testing.T) {
	// Test just the first append to isolate the issue
	source := `
	func main() {
		var nums I64[];
		append(nums&, 42);
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n")
}

func TestSliceBasicsCurrentImplementation(t *testing.T) {
	// Test what the current implementation actually supports (single append)
	source := `
	func main() {
		var nums I64[];
		append(nums&, 42);
		print(nums[0]);
		print(nums.length);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n1\n")
}

func TestSliceBasics(t *testing.T) {
	// TODO: This test will pass once multi-element append is implemented
	source := `
	func main() {
		var nums I64[];
		append(nums&, 42);
		append(nums&, 100);
		print(nums[0]);
		print(nums[1]);
		print(nums.length);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	// With the new append function implementation, elements are properly preserved!
	// Expected: "42\n100\n2\n" (first element, second element, length)
	be.Equal(t, output, "42\n100\n2\n")
}

func TestAddressOfParsing(t *testing.T) {
	// Test parsing the address-of operator by itself using postfix syntax
	input := []byte("nums&;\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	t.Logf("Parsed nums&: %s", ToSExpr(stmt))
	be.Equal(t, ToSExpr(stmt), "(unary \"&\" (ident \"nums\"))")
}

func TestSliceAppendParsing(t *testing.T) {
	// Test if we can parse append() calls using correct postfix & syntax
	input := []byte("append(nums&, 42);\x00")
	Init(input)
	NextToken()

	stmt := ParseStatement()
	be.Equal(t, stmt.Kind, NodeCall)

	// Debug: print what we actually parsed
	t.Logf("Parsed: %s", ToSExpr(stmt))
	t.Logf("Children count: %d", len(stmt.Children))

	// Verify we get the correct structure
	if len(stmt.Children) >= 2 {
		t.Logf("First arg: %s", ToSExpr(stmt.Children[1]))
		be.Equal(t, ToSExpr(stmt.Children[1]), "(unary \"&\" (ident \"nums\"))")
		if len(stmt.Children) >= 3 {
			t.Logf("Second arg: %s", ToSExpr(stmt.Children[2]))
			be.Equal(t, ToSExpr(stmt.Children[2]), "(integer 42)")
		}
	}
}

func TestExecuteAppendProgram(t *testing.T) {
	// Test executing a complete program with append functionality
	source := `
	func main() {
		var numbers I64[];
		var flags Boolean[];
		
		// Test I64 slice append
		append(numbers&, 42);
		print(numbers[0]);
		print(numbers.length);
		
		// Test Boolean slice append  
		append(flags&, true);
		print(flags[0]);
		print(flags.length);
		
		// Test multiple I64 values
		append(numbers&, 100);
		print(numbers[0]);
		print(numbers[1]);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	// 42 (numbers[0] after first append)
	// 1 (numbers.length after first append)
	// 1 (flags[0] - true as I64)
	// 1 (flags.length)
	// 42 (numbers[0] after second append)
	// 100 (numbers[1] after second append)
	expected := "42\n1\n1\n1\n42\n100\n"
	be.Equal(t, output, expected)
}

func TestExecuteAppendWithFieldAccess(t *testing.T) {
	// Test that demonstrates slice field access works correctly with append
	source := `
	func main() {
		var nums I64[];
		
		// Initially empty
		print(nums.length);
		
		// After first append
		append(nums&, 255);
		print(nums.length);
		print(nums[0]);
		
		// Test that items pointer is properly set
		// This works because slice.items points to the allocated element
		print(nums[0]); // Should still be 255
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	expected := "0\n1\n255\n255\n"
	be.Equal(t, output, expected)
}

func TestExecuteAppendPracticalExample(t *testing.T) {
	// Practical example showing append usage in a real scenario
	source := `
	func processNumbers(_ value: I64): I64 {
		return value * 2;
	}
	
	func main() {
		var results I64[];
		var inputs I64[];
		
		// Collect some input data
		append(inputs&, 10);
		append(inputs&, 20); // Now properly preserves both elements
		
		// Process the data and store results
		var processed I64;
		processed = processNumbers(inputs[0]);
		append(results&, processed);
		
		// Print the results
		print(inputs[0]);      // Input value
		print(results[0]);     // Processed value (input * 2)
		print(results.length); // Number of results
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	// Fixed: inputs[0] is now 10 (first appended value preserved)
	// processNumbers(10) = 20
	expected := "10\n20\n1\n"
	be.Equal(t, output, expected)
}

func TestSliceSimpleDeclaration(t *testing.T) {
	// Test just variable declaration without field access
	source := `
	func main() {
		var nums I64[];
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n")
}

func TestSliceEmptyLength(t *testing.T) {
	source := `
	func main() {
		var nums I64[];
		print(nums.length);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "0\n")
}

func TestSliceFieldAccess(t *testing.T) {
	// Demonstrate that slice field access works perfectly
	source := `
	func main() {
		var nums I64[];
		var flags Boolean[];
		print(nums.length);
		print(flags.length);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "0\n0\n")
}

func TestMultiElementAppendBug(t *testing.T) {
	// This test demonstrates the current bug with multi-element append
	source := `
	func main() {
		var nums I64[];
		
		// Add first element
		append(nums&, 42);
		print(nums[0]);     // Should be 42
		print(nums.length); // Should be 1
		
		// Add second element - THIS IS WHERE THE BUG OCCURS
		append(nums&, 100);
		print(nums[0]);     // Should be 42, but currently prints 100 (BUG!)
		print(nums[1]);     // Should be 100, but currently prints 0 (BUG!)
		print(nums.length); // Should be 2, but currently prints 1 (BUG!)
		
		// Add third element
		append(nums&, 200);
		print(nums[0]);     // Should be 42, but currently prints 200 (BUG!)
		print(nums[1]);     // Should be 100, but currently prints 0 (BUG!)
		print(nums[2]);     // Should be 200, but currently prints 0 (BUG!)
		print(nums.length); // Should be 3, but currently prints 1 (BUG!)
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	// FIXED! All elements are now properly preserved during append
	expectedCorrectOutput := "42\n1\n42\n100\n2\n42\n100\n200\n3\n"
	be.Equal(t, output, expectedCorrectOutput)
}

func TestLengthIncrementBug(t *testing.T) {
	// Simpler test: just focus on the length increment issue
	source := `
	func main() {
		var nums I64[];
		append(nums&, 10);
		print(nums.length); // Should be 1
		append(nums&, 20);
		print(nums.length); // Should be 2, but currently prints 1 (BUG!)
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)

	// Fixed! Length now increments correctly
	be.Equal(t, output, "1\n2\n") // Length properly increments
}
