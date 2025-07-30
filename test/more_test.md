# Extracted execution tests

Generated from existing Go test files.

## Tests from compiler_test.go

### Test: nested struct operations
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct basic field access
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct field arithmetic
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct field assignment chain
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct field in expressions
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct multiple instances
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct with mixed variable types
```zong-program
{
		struct Point { var x I64; var y I64; }
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

### Test: struct with more fields
```zong-program
{
		struct Rectangle { var width I64; var height I64; var depth I64; }
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

### Test: struct zero initialization
```zong-program
{
		struct Point { var x I64; var y I64; }
		var p Point;
		print(p.x);
		print(p.y);
	}
```
```execute
0
0
```

## Tests from locals_integration_test.go

### Test: integration complex variable calculations
```zong-program
{ var x I64; var y I64; var result I64; x = 15; y = 3; result = x * y + 5; print(result); }
```
```execute
50
```

### Test: integration comprehensive demo
```zong-program
{
		var a I64;
		var b I64;
		var temp I64;
		var final I64;

		a = 8;
		b = 3;
		temp = a * b;        // temp = 24
		final = temp + a - b; // final = 24 + 8 - 3 = 29
		print(final);
	}
```
```execute
29
```

### Test: integration mixed types
```zong-program
{ var x I64; var y string; x = 42; print(x); }
```
```execute
42
```

### Test: integration nested variable scoping
```zong-program
{ var x I64; x = 42; { var y I64; y = x; print(y); } }
```
```execute
42
```

### Test: integration variable reassignment
```zong-program
{ var counter I64; counter = 5; counter = counter + 10; print(counter); }
```
```execute
15
```

### Test: integration variables in expressions
```zong-program
{ var a I64; var b I64; a = 10; b = 20; print(a + b); }
```
```execute
30
```

## Tests from slice_test.go

### Test: execute append practical example
```zong-program
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
	}
```
```execute
10
20
1
```

### Test: execute append program
```zong-program
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
	}
```
```execute
42
1
1
1
42
100
```

### Test: execute append with field access
```zong-program
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
	}
```
```execute
0
1
255
255
```

### Test: multi element append bug
```zong-program
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
	}
```
```execute
42
1
42
100
2
42
100
200
3
```

## Tests from string_execution_test.go

### Test: w a s m execution baseline
```zong-program
func main() { print(42); }
```
```execute
42
```

### Test: w a s m execution empty string
```zong-program
func main() { var s U8[] = ""; print(42); }
```
```execute
42
```

### Test: w a s m execution string assignment
```zong-program
func main() { var s U8[] = "hello"; print(42); }
```
```execute
42
```

### Test: w a s m execution string declaration
```zong-program
func main() { var s U8[]; print(42); }
```
```execute
42
```

## Tests from string_integration_test.go

### Test: string literal integration
```zong-program
func main() { print(42); }
```
```execute
42
```

### Test: multiple string literals
```zong-program
func main() { var s1 U8[] = "hello"; var s2 U8[] = "world"; print(5); }
```
```execute
5
```

### Test: string literal assignment
```zong-program
func main() { var s U8[] = "test"; print(4); }
```
```execute
4
```

### Test: string literal compilation
```zong-program
func main() { var msg U8[] = "hello world"; print(11); }
```
```execute
11
```

### Test: string literal deduplication
```zong-program
func main() { var s1 U8[] = "same"; var s2 U8[] = "same"; print(42); }
```
```execute
42
```
