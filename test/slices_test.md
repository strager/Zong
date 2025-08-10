# Slice parsing tests

## Test: slice variable declaration
```zong-program
var nums I64[];
```
```ast
[(var-decl "nums" "I64[]")]
```

## Test: slice subscript
```zong-expr
nums[0]
```
```ast
(idx (var "nums") 0)
```

## Test: slice assignment
```zong-expr
nums[1] = 42
```
```ast
(binary "=" (idx (var "nums") 1) 42)
```

## Test: address-of operator parsing
```zong-expr
nums&
```
```ast
(unary "&" (var "nums"))
```

## Test: slice append function call parsing
```zong-expr
append(nums&, 42)
```
```ast
(call (var "append") (unary "&" (var "nums")) 42)
```

## Slice Execution Tests

## Test: execute append practical example
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

## Test: execute append program
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

## Test: execute append with field access
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

## Test: multi element append bug
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

## Slice Subscript Tests

## Test: slice index parsing from expressions
```zong-expr
nums[0]
```
```ast
(idx (var "nums") 0)
```

## Test: slice assignment from expressions  
```zong-expr
nums[1] = 42
```
```ast
(binary "=" (idx (var "nums") 1) 42)
```

## Slice Length Tests

## Test: slice length field access parsing
```zong-expr
nums.length
```
```ast
(dot (var "nums") "length")
```

## Additional Array Subscript Tests (from expressions_test.md)

## Test: function call with subscript
```zong-expr
f(x)[0]
```
```ast
(idx (call (var "f") (var "x")) 0)
```

## Test: array subscript variable
```zong-expr
x[y]
```
```ast
(idx (var "x") (var "y"))
```

## Test: array subscript integer from expressions
```zong-expr
arr[0]
```
```ast
(idx (var "arr") 0)
```

## Test: nested array subscript
```zong-expr
matrix[i][j]
```
```ast
(idx (idx (var "matrix") (var "i")) (var "j"))
```

## Test: array subscript with expression
```zong-expr
items[x + 1]
```
```ast
(idx (var "items") (binary "+" (var "x") 1))
```

## Additional Slice Tests (from extracted_execution_test.md)

## Test: slice address of
```zong-program
func main() {
	var nums I64[];
	print(42);
}
```
```execute
42
```

## Test: length increment bug
```zong-program
func main() {
	var nums I64[];
	append(nums&, 10);
	print(nums.length); // Should be 1
	append(nums&, 20);
	print(nums.length); // Should be 2
}
```
```execute
1
2
```

## Test: slice basics
```zong-program
func main() {
	var nums I64[];
	append(nums&, 42);
	append(nums&, 100);
	print(nums[0]);
	print(nums[1]);
	print(nums.length);
}
```
```execute
42
100
2
```

## Test: slice empty length
```zong-program
func main() {
	var nums I64[];
	print(nums.length);
}
```
```execute
0
```

## Additional Slice Tests (from more_test.md)

## Test: execute append practical example
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

## Test: execute append program
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

## Test: execute append with field access
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

## Test: multi element append bug
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

## Test: slice indexing with 64-byte element size

Regression test. Ensure slice indexing works with items that are >= 64 bytes.

```zong-program
// BigStruct is >= 64 bytes.
struct BigStruct(
    a: I64, b: I64, c: I64, d: I64, e: I64, f: I64, g: I64, h: I64
);

func main() {
    var slice BigStruct[];
    append(slice&, BigStruct(a: 1, b: 2, c: 3, d: 4, e: 5, f: 6, g: 7, h: 8));
    append(slice&, BigStruct(a: 10, b: 20, c: 30, d: 40, e: 50, f: 60, g: 70, h: 80));

    var retrieved0 BigStruct = slice[0];
    print(retrieved0.a);
    print(retrieved0.h);
    var retrieved1 BigStruct = slice[1];
    print(retrieved1.a);
    print(retrieved1.h);
}
```
```execute
1
8
10
80
```

## Test: U8 slice zero initialization
```zong-program
func main() {
    var s U8[] = U8[]();
    print(s.length);
}
```
```execute
0
```

## Test: I64 slice zero initialization
```zong-program
func main() {
    var s I64[] = I64[]();
    print(s.length);
}
```
```execute
0
```

## Test: Slice field access after zero init
```zong-program
func main() {
    var s U8[] = U8[]();
    print(s.length);
    // Note: We can access the length field
    var len I64 = s.length;
    print(len);
}
```
```execute
0
0
```

## Test: Zero init followed by append
```zong-program
func main() {
    var s U8[] = U8[]();
    print(s.length);
    append(s&, 42);
    print(s.length);
}
```
```execute
0
1
```

## Test: Multiple slice types
```zong-program
func main() {
    var bytes U8[] = U8[]();
    var numbers I64[] = I64[]();
    print(bytes.length);
    print(numbers.length);
}
```
```execute
0
0
```

## Test: Slice zero init in expression
```zong-program
func main() {
    print(U8[]().length);
}
```
```execute
0
```

## Test: Nested slice field access
```zong-program
struct Container(slice: U8[]);

func main() {
    var c Container = Container(slice: U8[]());
    print(c.slice.length);
}
```
```execute
0
```
