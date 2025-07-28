package main

import (
	"github.com/nalgeon/be"
	"testing"
)

// Loop Integration Tests

func TestBasicLoop(t *testing.T) {
	source := `
		func main() {
			var i I64;
			i = 0;
			loop {
				print(i);
				i = i + 1;
				if i >= 3 {
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

func TestNestedLoops(t *testing.T) {
	source := `
		func main() {
			var i I64;
			var j I64;
			i = 0;
			loop {
				j = 0;
				loop {
					print(j);
					j = j + 1;
					if j >= 2 {
						break;
					}
				}
				i = i + 1; 
				if i >= 2 {
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
	be.Equal(t, output, "0\n1\n0\n1\n")
}

func TestContinueStatement(t *testing.T) {
	source := `
		func main() {
			var i I64;
			i = 0;
			loop {
				i = i + 1;
				if i == 2 {
					continue;
				}
				print(i);
				if i >= 3 {
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
	be.Equal(t, output, "1\n3\n") // skips 2
}

func TestBreakOutsideLoop(t *testing.T) {
	source := `
		func main() {
			break;
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Should fail type checking
	err := CheckProgram(ast)
	be.Equal(t, err != nil, true)
	be.Equal(t, err.Error(), "error: break statement outside of loop")
}

func TestContinueOutsideLoop(t *testing.T) {
	source := `
		func main() {
			continue;
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// Should fail type checking
	err := CheckProgram(ast)
	be.Equal(t, err != nil, true)
	be.Equal(t, err.Error(), "error: continue statement outside of loop")
}

func TestBreakContinueInNestedLoops(t *testing.T) {
	source := `
		func main() {
			var i I64;
			var j I64;
			i = 0;
			loop {
				j = 0;
				loop {
					j = j + 1;
					if j == 2 {
						continue; // continue inner loop
					}
					if j == 4 {
						break; // break inner loop
					}
					print(j);
				}
				i = i + 1;
				if i >= 2 {
					break; // break outer loop
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
	be.Equal(t, output, "1\n3\n1\n3\n")
}

func TestLoopWithVariableModification(t *testing.T) {
	source := `
		func main() {
			var counter I64;
			var sum I64;
			counter = 1;
			sum = 0;
			loop {
				sum = sum + counter;
				counter = counter + 1;
				if counter > 5 {
					break;
				}
			}
			print(sum); // Should print 15 (1+2+3+4+5)
		}
	`

	input := []byte(source + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	output, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, output, "15\n")
}

func TestEmptyLoop(t *testing.T) {
	source := `
		func main() {
			var i I64;
			i = 0;
			loop {
				i = i + 1;
				if i >= 1 {
					break;
				}
			}
			print(i);
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

func TestNestedLoopBreakBug(t *testing.T) {
	source := `
		func main() {
			loop {
				if true {
					loop {
						if true {
							print(3);
							break;
						}
					}
					print(4);
				}
				print(5);
				break;
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
	be.Equal(t, output, "3\n4\n5\n")
}
