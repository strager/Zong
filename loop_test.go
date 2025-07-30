package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Loop Integration Tests

// skips 2

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
