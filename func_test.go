package main

import (
	"github.com/nalgeon/be"
	"testing"
)

// Test function declaration parsing
func TestFunctionDeclarationParsing(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "void function no parameters",
			source:   "func test() {}",
			expected: `(func "test" () void (block))`,
		},
		{
			name:     "function with I64 return type",
			source:   "func add(): I64 {}",
			expected: `(func "add" () "I64" (block))`,
		},
		{
			name:     "function with positional parameters",
			source:   "func add(_ addA: I64, _ addB: I64): I64 {}",
			expected: `(func "add" ((param "addA" "I64" positional) (param "addB" "I64" positional)) "I64" (block))`,
		},
		{
			name:     "function with named parameters",
			source:   "func test(testX: I64, testY: I64) {}",
			expected: `(func "test" ((param "testX" "I64" named) (param "testY" "I64" named)) void (block))`,
		},
		{
			name:     "function with body",
			source:   "func test() { var x I64; }",
			expected: `(func "test" () void (block (var (ident "x") (ident "I64"))))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()

			result := ToSExpr(ast)
			be.Equal(t, result, tt.expected)
		})
	}
}

// Test basic function compilation
func TestBasicFunctionCompilation(t *testing.T) {
	source := `func test(): I64 {
		return 42;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement() // Parse single function

	// Test WASM compilation doesn't panic
	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test simple function with parameters
func TestFunctionWithParameters(t *testing.T) {
	source := `func add(_ addA2: I64, _ addB2: I64): I64 {
		return addA2 + addB2;
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseStatement()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)
}

// Test simple main function first
func TestSimpleMain(t *testing.T) {
	source := `func main() {
		print(42);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)

	executeWasmAndVerify(t, wasmBytes, "42\n")
}

// Test end-to-end function execution with main
func TestEndToEndFunctionExecution(t *testing.T) {
	source := `func add(_ addA3: I64, _ addB3: I64): I64 {
		return addA3 + addB3;
	}
	
	func main() {
		print(add(5, 3));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram() // Parse multiple functions

	wasmBytes := CompileToWASM(ast)
	be.True(t, len(wasmBytes) > 0)

	// Execute and verify output
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "8\n")
}

// Test multiple functions with various signatures
func TestMultipleFunctions(t *testing.T) {
	source := `func double(_ doubleX: I64): I64 {
		return doubleX * 2;
	}
	
	func triple(_ tripleX: I64): I64 {
		return tripleX * 3;
	}
	
	func main() {
		print(double(5));
		print(triple(4));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "10\n12\n")
}

// Test void function (no return value)
func TestVoidFunction(t *testing.T) {
	source := `func printTwice(_ printTwiceX: I64) {
		print(printTwiceX);
		print(printTwiceX);
	}
	
	func main() {
		printTwice(7);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "7\n7\n")
}

// Test function with complex expressions
func TestFunctionWithComplexExpressions(t *testing.T) {
	source := `func compute(_ computeA: I64, _ computeB: I64, _ computeC: I64): I64 {
		return (computeA + computeB) * computeC - 10;
	}
	
	func main() {
		print(compute(3, 4, 5));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "25\n") // (3+4)*5-10 = 35-10 = 25
}

// Test nested function calls
func TestNestedFunctionCalls(t *testing.T) {
	source := `func add(_ addA4: I64, _ addB4: I64): I64 {
		return addA4 + addB4;
	}
	
	func multiply(_ multiplyA: I64, _ multiplyB: I64): I64 {
		return multiplyA * multiplyB;
	}
	
	func main() {
		print(add(multiply(2, 3), multiply(4, 5)));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "26\n") // (2*3) + (4*5) = 6 + 20 = 26
}

// Phase 2 Tests: Named Parameters

// Test named parameters parsing
func TestNamedParameterParsing(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "function with named parameters",
			source:   "func test(testX2: I64, testY2: I64): I64 {}",
			expected: `(func "test" ((param "testX2" "I64" named) (param "testY2" "I64" named)) "I64" (block))`,
		},
		{
			name:     "mixed positional and named parameters",
			source:   "func test(_ testA: I64, testX3: I64, testY3: I64): I64 {}",
			expected: `(func "test" ((param "testA" "I64" positional) (param "testX3" "I64" named) (param "testY3" "I64" named)) "I64" (block))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.source + "\x00")
			Init(input)
			NextToken()
			ast := ParseStatement()

			result := ToSExpr(ast)
			be.Equal(t, result, tt.expected)
		})
	}
}

// Test named parameter function calls
func TestNamedParameterCalls(t *testing.T) {
	source := `func greet(greetName: I64, greetAge: I64) {
		print(greetName);
		print(greetAge);
	}
	
	func main() {
		greet(greetName: 42, greetAge: 25);
		greet(greetAge: 30, greetName: 50); 
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "42\n25\n50\n30\n")
}

// Test mixed positional and named parameters
func TestMixedParameters(t *testing.T) {
	source := `func compute(_ computeBase: I64, computeMultiplier: I64, computeOffset: I64): I64 {
		return computeBase * computeMultiplier + computeOffset;
	}
	
	func main() {
		print(compute(5, computeMultiplier: 3, computeOffset: 10));
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "25\n") // 5 * 3 + 10 = 25
}

// Test function that returns a struct
func TestFunctionReturningStruct(t *testing.T) {
	source := `struct Point { var x I64; var y I64; }
	
	func createPoint(_ createPointXVal: I64, _ createPointYVal: I64): Point {
		var p Point;
		p.x = createPointXVal;
		p.y = createPointYVal;
		return p;
	}
	
	func main() {
		var result Point;
		result = createPoint(10, 20);
		print(result.x);
		print(result.y);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "10\n20\n")
}

func TestFunctionStructParamCopies(t *testing.T) {
	source := `
		struct S { var i I64; }

		func f(_ fS: S) {
			fS.i = 3;
			print(fS.i);
		}

		func main() {
			var ss S;
			ss.i = 2;
			print(ss.i);
			f(ss);
			print(ss.i);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "2\n3\n2\n")
}

// Test function return field access (e.g., make_point(x: 1, y: 2).x)
func TestFunctionReturnFieldAccess(t *testing.T) {
	source := `struct Point { var x I64; var y I64; }
	
	func makePoint(pointX: I64, pointY: I64): Point {
		var p Point;
		p.x = pointX;
		p.y = pointY;
		return p;
	}
	
	func main() {
		print(makePoint(pointX: 10, pointY: 20).x);
		print(makePoint(pointX: 30, pointY: 40).y);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// This should now work with the fixed implementation
	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "10\n40\n")
}

// Test both variable field access and function return field access in one program
func TestMixedFieldAccess(t *testing.T) {
	source := `struct Point { var x I64; var y I64; }
	
	func makePoint(pointX: I64, pointY: I64): Point {
		var newP Point;
		newP.x = pointX;
		newP.y = pointY;
		return newP;
	}
	
	func main() {
		// Test variable field access
		var mainP Point;
		mainP.x = 100;
		mainP.y = 200;
		print(mainP.x);
		print(mainP.y);
		
		// Test function return field access
		print(makePoint(pointX: 300, pointY: 400).x);
		print(makePoint(pointX: 500, pointY: 600).y);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "100\n200\n300\n600\n")
}

// Test nested struct definitions and field access
func TestNestedStructs(t *testing.T) {
	source := `struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
	func main() {
		var person Person;
		person.name = 100;
		person.age = 25;
		
		// Set nested struct fields
		person.address.state = 42;
		person.address.zipCode = 12345;
		
		// Read nested struct fields
		print(person.name);
		print(person.address.state);
		print(person.address.zipCode);
		print(person.age);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "100\n42\n12345\n25\n")
}

// Test nested struct initialization
func TestNestedStructInitialization(t *testing.T) {
	source := `struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
	func main() {
		var person Person;
		var addr Address;
		
		// Initialize address separately
		addr.state = 99;
		addr.zipCode = 54321;
		
		// Assign nested struct
		person.name = 200;
		person.address = addr;
		person.age = 30;
		
		print(person.name);
		print(person.address.state);
		print(person.address.zipCode);
		print(person.age);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "200\n99\n54321\n30\n")
}

// Test function returning nested struct
func TestNestedStructFunctionReturn(t *testing.T) {
	source := `struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
	func createAddress(addrState: I64, addrZip: I64): Address {
		var addr Address;
		addr.state = addrState;
		addr.zipCode = addrZip;
		return addr;
	}
	
	func createPerson(personName: I64, personAge: I64): Person {
		var p Person;
		p.name = personName;
		p.age = personAge;
		p.address = createAddress(addrState: 77, addrZip: 98765);
		return p;
	}
	
	func main() {
		// Test function return field access with nested structs
		print(createPerson(personName: 300, personAge: 35).name);
		print(createPerson(personName: 400, personAge: 40).address.state);
		print(createPerson(personName: 500, personAge: 45).address.zipCode);
		print(createPerson(personName: 600, personAge: 50).age);
	}`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	executeWasmAndVerify(t, wasmBytes, "300\n77\n98765\n50\n")
}
