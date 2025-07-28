package main

import (
	"github.com/nalgeon/be"
	"testing"
)

// Boolean Tests

func TestBooleanLiterals(t *testing.T) {
	source := `
		func main() {
			var t Boolean;
			var f Boolean;
			t = true;
			f = false;
			print(t);
			print(f);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n0\n")
}

func TestBooleanComparisons(t *testing.T) {
	source := `
		func main() {
			var x I64;
			var result Boolean;
			x = 5;
			
			result = x == 5;
			print(result);
			
			result = x != 5;
			print(result);
			
			result = x > 3;
			print(result);
			
			result = x < 3;
			print(result);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n0\n1\n0\n")
}

func TestBooleanInIfStatements(t *testing.T) {
	source := `
		func main() {
			var flag Boolean;
			flag = true;
			
			if flag {
				print(1);
			}
			
			flag = false;
			if flag {
				print(2);
			} else {
				print(3);
			}
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n3\n")
}

func TestBooleanFunctionParameters(t *testing.T) {
	source := `
		func checkFlag(flag: Boolean): I64 {
			if flag {
				return 1;
			}
			return 0;
		}
		
		func main() {
			print(checkFlag(flag: true));
			print(checkFlag(flag: false));
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n0\n")
}

func TestBooleanLoops(t *testing.T) {
	source := `
		func main() {
			var i I64;
			var keepGoing Boolean;
			i = 0;
			keepGoing = true;
			
			loop {
				if i >= 3 {
					keepGoing = false;
				}
				
				if keepGoing {
					print(i);
					i = i + 1;
				} else {
					break;
				}
			}
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "0\n1\n2\n")
}

func TestBooleanParsing(t *testing.T) {
	source := `
		func main() {
			var x Boolean;
			x = true;
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Check that the AST contains boolean nodes
	expected := `(block (func "main" () void (block (var (ident "x") (ident "Boolean")) (binary "=" (ident "x") (boolean true)))))`
	result := ToSExpr(ast)
	be.Equal(t, result, expected)
}

func TestBooleanTypeChecking(t *testing.T) {
	source := `
		func main() {
			var x Boolean;
			x = true;
			var y I64;
			y = x; // This should fail type checking
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Should fail type checking
	err := CheckProgram(ast)
	be.Equal(t, err != nil, true)
}

func TestBooleanReturnType(t *testing.T) {
	source := `
		func getTrue(): Boolean {
			return true;
		}
		
		func main() {
			print(getTrue());
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n")
}
