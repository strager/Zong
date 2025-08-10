# Struct parsing tests

## Test: struct declaration
```zong-program
struct Point(x: I64, y: I64);
```
```ast
[(struct "Point"
  [(field "x" "I64")
   (field "y" "I64")])]
```

## Test: struct type in variable declaration
```zong-program
var p Point;
```
```ast
[(var-decl "p" "Point")]
```

## Test: field access
```zong-expr
p.x
```
```ast
(dot (var "p") "x")
```

## Test: field assignment
```zong-expr
p.x = 42
```
```ast
(binary "=" (dot (var "p") "x") 42)
```

## Test: complex struct expression
```zong-expr
p.x + q.y
```
```ast
(binary "+" (dot (var "p") "x") (dot (var "q") "y"))
```

## Test: struct field access and assignment execution
```zong-program
struct Point(x: I64, y: I64);
func main() {
    var p Point;
    p.x = 42;
    p.y = 24;
    print(p.x);
    print(p.y);
}
```
```execute
42
24
```

## Advanced Struct Operations

## Test: nested struct operations
```zong-program
{
		struct Point(x: I64, y: I64);
		var p1 Point;
		var p2 Point;
		var p3 Point;
		
		p1.x = 1;
		p1.y = 2;
		p2.x = 3;
		p2.y = 4;
		
		p3.x = p1.x + p2.x;
		p3.y = p1.y * p2.y;
		
		print(p3.x);
		print(p3.y);
		print(p3.x + p3.y);
	}
```
```execute
4
8
12
```

## Test: struct basic field access
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point;
		p.x = 42;
		p.y = 84;
		print(p.x);
		print(p.y);
	}
```
```execute
42
84
```

## Test: struct field arithmetic
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point;
		p.x = 10;
		p.y = 20;
		print(p.x + p.y);
		print(p.x * p.y);
	}
```
```execute
30
200
```

## Test: struct field assignment chain
```zong-program
{
		struct Point(x: I64, y: I64);
		var p1 Point;
		var p2 Point;
		p1.x = 100;
		p2.x = p1.x + 50;
		p1.y = p2.x - p1.x;
		print(p1.x);
		print(p2.x);
		print(p1.y);
	}
```
```execute
100
150
50
```

## Test: struct field in expressions
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point;
		p.x = 15;
		p.y = 25;
		print(p.x < p.y);
		print(p.x > p.y);
		print(p.x == 15);
		print(p.y != 20);
	}
```
```execute
1
0
1
1
```

## Test: struct multiple instances
```zong-program
{
		struct Point(x: I64, y: I64);
		var p1 Point;
		var p2 Point;
		p1.x = 1;
		p1.y = 2;
		p2.x = 10;
		p2.y = 20;
		print(p1.x + p2.x);
		print(p1.y + p2.y);
	}
```
```execute
11
22
```

## Test: struct with mixed variable types
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point;
		var regular I64;
		
		regular = 100;
		p.x = regular / 4;
		p.y = regular / 2;
		
		print(regular);
		print(p.x);
		print(p.y);
		print(regular + p.x + p.y);
	}
```
```execute
100
25
50
175
```

## Test: struct with more fields
```zong-program
{
		struct Rectangle(width: I64, height: I64, depth: I64);
		var rect Rectangle;
		rect.width = 5;
		rect.height = 10;
		rect.depth = 3;
		print(rect.width);
		print(rect.height);
		print(rect.depth);
		print(rect.width * rect.height * rect.depth);
	}
```
```execute
5
10
3
150
```

## Test: struct zero initialization
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point;
		print(p.x);
		print(p.y);
	}
```
```execute
0
0
```

## Test: struct function-style initialization
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 2, y: 3);
		print(p.x);
		print(p.y);
	}
```
```execute
2
3
```

## Test: struct function-style initialization field order doesn't matter
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(y: 3, x: 2);
		print(p.x);
		print(p.y);
	}
```
```execute
2
3
```

## Test: struct initializer executes arguments in source code order
```zong-program
func print_and_return(_ x: I64): I64 {
	print(x);
	return x;
}
func main() {
	struct Point(x: I64, y: I64);
	var p Point = Point(y: print_and_return(3), x: print_and_return(2));
	print(p.x);
	print(p.y);
}
```
```execute
3
2
2
3
```

## Test: struct function-style initialization with different field types
```zong-program
{
		struct Mixed(flag: Boolean, count: I64);
		var m Mixed = Mixed(flag: true, count: 42);
		print(m.flag);
		print(m.count);
	}
```
```execute
1
42
```

## Test: struct function-style initialization with nested expressions
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 1 + 1, y: 3 * 4);
		print(p.x);
		print(p.y);
	}
```
```execute
2
12
```

## Test: struct function-style initialization in expressions
```zong-program
{
		struct Point(x: I64, y: I64);
		var sum I64 = Point(x: 1, y: 2).x + Point(x: 3, y: 4).y;
		print(sum);
	}
```
```execute
5
```

## Test: struct function-style initialization as function argument
```zong-program
{
		struct Point(x: I64, y: I64);
		print(Point(x: 5, y: 6).x);
	}
```
```execute
5
```

## Test: single field struct initialization
```zong-program
{
		struct Single(value: I64);
		var s Single = Single(value: 42);
		print(s.value);
	}
```
```execute
42
```

## Nested Struct Tests (from extracted_execution_test.md)

## Test: nested struct initialization
```zong-program
struct Address(state: I64, zipCode: I64);
struct Person(name: I64, address: Address, age: I64);

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
}
```
```execute
200
99
54321
30
```

## Test: nested structs
```zong-program
struct Address(state: I64, zipCode: I64);
struct Person(name: I64, address: Address, age: I64);

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
}
```
```execute
100
42
12345
25
```

## Struct Function-Style Initialization Error Tests

## Test: struct initialization missing required field
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 2);
	}
```
```compile-error
error: struct initialization expects 2 fields, got 1
error: struct initialization missing required field 'y'
```

## Test: struct initialization missing all fields
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point();
	}
```
```compile-error
error: struct initialization expects 2 fields, got 0
error: struct initialization missing required field 'x'
```

## Test: struct initialization with unknown field
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 1, y: 2, z: 3);
	}
```
```compile-error
error: struct initialization expects 2 fields, got 3
```

## Test: struct initialization with duplicate field
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 1, y: 2, x: 3);
	}
```
```compile-error
error: struct initialization has duplicate field 'x'
```

## Test: struct initialization with wrong field type
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: true, y: 2);
	}
```
```compile-error
error: struct initialization field 'x' expects type I64, got Boolean
```

## Test: struct initialization with unknown field name
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(x: 1, z: 2);
	}
```
```compile-error
error: struct initialization has unknown field 'z'
error: struct initialization missing required field 'y'
```

## Test: struct initialization with non-existent struct
```zong-program
{
		var p FakeStruct = FakeStruct(a: 1);
	}
```
```compile-error
error: undefined symbol 'FakeStruct'
error: undefined symbol 'FakeStruct'
```

## Test: struct initialization without named parameters
```zong-program
{
		struct Point(x: I64, y: I64);
		var p Point = Point(1, 2);
	}
```
```compile-error
error: struct initialization requires named parameters for all fields
```

## Struct Compile Error Tests (from compile_error_test.md)

## Test: field access on non-struct type
```zong-program
func main() {
    var x I64;
    x = 42;
    print(x.field);
}
```
```compile-error
error: cannot access field of non-struct type I64
```

## Test: function parameter 64-byte struct copy semantics

Regression test. Ensure a struct with >= 64 bytes can be passed as a parameter.

This will trigger the code path where the struct size (64) is encoded into a
constant for memory.copy when copying the struct to the stack.

```zong-program
// Define a struct that is exactly 64 bytes
struct BigStruct(
    a: I64, b: I64, c: I64, d: I64, e: I64, f: I64, g: I64, h: I64  // 8 * 8 = 64 bytes
);

func processStruct(s: BigStruct) {
    // Function receives a 64-byte struct by value
    print(s.a);
    print(s.b);
    print(s.h);
    
    // Modify the copy to test independence
    s.a = 999;
}

func main() {
    var s BigStruct = BigStruct(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8);
    
    // This function call will pass the 64-byte struct by value
    processStruct(s);
    
    // Original should be unchanged (copy semantics)
    print(s.a);
    print(s.h);
}
```
```execute
1
2
8
1
8
```

## Test: struct field access with offset 64

Regression test. Ensure fields at offset >= 64 bytes can be read from and
written to.

```zong-program
// Define a struct where a field is at offset 64
struct BigStruct(
    p1: I64, p2: I64, p3: I64, p4: I64, p5: I64, p6: I64, p7: I64, p8: I64,  // Padding.
    target: I64);     // This field will be at offset 64

func main() {
    var s BigStruct = BigStruct(
        p1: 1, p2: 2, p3: 3, p4: 4, p5: 5, p6: 6, p7: 7, p8: 8,
        target: 999);
    print(s.target);
    s.target = 777;
    print(s.target);
}
```
```execute
999
777
```
