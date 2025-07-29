package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestVariableShadowingEndToEnd(t *testing.T) {
	source := `
		func main() {
			var x I64;
			x = 10;
			print(x);
			{
				var x I64;
				x = 20;
				print(x);
			}
			print(x);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "10\n20\n10\n")
}

func TestFunctionParameterShadowingEndToEnd(t *testing.T) {
	source := `
		func test(x: I64) {
			print(x);
			{
				var x I64;
				x = 99;
				print(x);
			}
			print(x);
		}
		
		func main() {
			test(x: 42);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n99\n42\n")
}

func TestDeepNestedShadowingEndToEnd(t *testing.T) {
	source := `
		func main() {
			var x I64;
			x = 1;
			print(x);
			{
				var x I64;
				x = 2;
				print(x);
				{
					var x I64;
					x = 3;
					print(x);
					{
						var x I64;
						x = 4;
						print(x);
					}
					print(x);
				}
				print(x);
			}
			print(x);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "1\n2\n3\n4\n3\n2\n1\n")
}

func TestShadowingWithDifferentTypes(t *testing.T) {
	source := `
		func main() {
			var x I64;
			x = 42;
			print(x);
			{
				var x Boolean;
				x = true;
				print(x);
			}
			print(x);
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "42\n1\n42\n")
}
