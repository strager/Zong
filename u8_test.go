package main

import (
	"testing"

	"github.com/nalgeon/be"
)

// Test just the slice declaration without append

// Test I64 slice for comparison - using correct postfix & syntax

// Test I64 slice with multiple appends for comparison

func TestU8SliceOutOfRange(t *testing.T) {
	// Test that values outside U8 range (0-255) are rejected
	program := `
func main() {
	var slice U8[];
	append(slice&, 256);  // This should fail
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	// This should panic during compilation due to out-of-range value
	defer func() {
		if r := recover(); r != nil {
			// Expected panic
			be.True(t, true)
		} else {
			t.Error("Expected panic for out-of-range U8 value")
		}
	}()

	CompileToWASM(ast)
}
