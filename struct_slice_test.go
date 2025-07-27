package main

import (
	"testing"

	"github.com/nalgeon/be"
)

func TestStructSliceBasics(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	print(points.length);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "0\n")
}

func TestStructSliceAppend(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 10;
	p.y = 20;
	append(points&, p);
	print(points.length);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "1\n")
}

func TestStructSliceAppendMultiple(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p1 Point;
	p1.x = 10;
	p1.y = 20;
	
	var p2 Point;
	p2.x = 30;
	p2.y = 40;
	
	append(points&, p1);
	append(points&, p2);
	print(points.length);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "2\n")
}

func TestStructSliceIndexing(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 10;
	p.y = 20;
	append(points&, p);
	
	var retrieved Point;
	retrieved = points[0];
	print(retrieved.x);
	print(retrieved.y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n20\n")
}

func TestStructSliceFieldAccessAtIndex(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 10;
	p.y = 20;
	append(points&, p);
	
	// Read field at index
	print(points[0].x);
	print(points[0].y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n20\n")
}

func TestStructSliceFieldAssignmentAtIndex(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 10;
	p.y = 20;
	append(points&, p);
	
	// Assign field at index
	points[0].x = 100;
	points[0].y = 200;
	
	// Read back the modified values
	print(points[0].x);
	print(points[0].y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "100\n200\n")
}

func TestStructSliceWholeStructAssignmentAtIndex(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p1 Point;
	p1.x = 10;
	p1.y = 20;
	append(points&, p1);
	
	// Create a new point
	var p2 Point;
	p2.x = 100;
	p2.y = 200;
	
	// Assign whole struct at index
	points[0] = p2;
	
	// Read back the values
	print(points[0].x);
	print(points[0].y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "100\n200\n")
}

func TestStructSliceMultipleElementsFieldAccess(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p1 Point;
	p1.x = 10;
	p1.y = 20;
	
	var p2 Point;
	p2.x = 30;
	p2.y = 40;
	
	var p3 Point;
	p3.x = 50;
	p3.y = 60;
	
	append(points&, p1);
	append(points&, p2);
	append(points&, p3);
	
	// Access fields of different elements
	print(points[0].x);
	print(points[1].y);
	print(points[2].x);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "10\n40\n50\n")
}

func TestStructSliceComplexOperations(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	
	// Add several points manually (instead of while loop)
	var p0 Point;
	p0.x = 0;
	p0.y = 0;
	append(points&, p0);
	
	var p1 Point;
	p1.x = 10;
	p1.y = 20;
	append(points&, p1);
	
	var p2 Point;
	p2.x = 20;
	p2.y = 40;
	append(points&, p2);
	
	// Modify middle element's fields
	points[1].x = 999;
	points[1].y = 888;
	
	// Print all values manually
	print(points[0].x);
	print(points[0].y);
	print(points[1].x);
	print(points[1].y);
	print(points[2].x);
	print(points[2].y);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "0\n0\n999\n888\n20\n40\n")
}

func TestStructSliceWithDifferentStructSize(t *testing.T) {
	program := `
struct Rectangle { var x I64; var y I64; var width I64; var height I64; }

func main() {
	var rects Rectangle[];
	var r Rectangle;
	r.x = 10;
	r.y = 20;
	r.width = 100;
	r.height = 200;
	
	append(rects&, r);
	
	print(rects.length);
	print(rects[0].x);
	print(rects[0].y);
	print(rects[0].width);
	print(rects[0].height);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "1\n10\n20\n100\n200\n")
}

func TestStructSliceNestedFieldAccess(t *testing.T) {
	program := `
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 42;
	p.y = 84;
	append(points&, p);
	
	// Test nested expressions with field access
	var sum I64;
	sum = points[0].x + points[0].y;
	print(sum);
	
	// Test assignment with expression
	points[0].x = points[0].y + 100;
	print(points[0].x);
}
`
	input := []byte(program + "\x00")
	Init(input)
	NextToken()
	ast := ParseProgram()

	wasmBytes := CompileToWASM(ast)
	result, err := executeWasm(t, wasmBytes)
	be.Err(t, err, nil)
	be.Equal(t, result, "126\n184\n")
}
