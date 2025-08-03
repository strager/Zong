# Literals Tests

Tests for literal values (integers, booleans, strings, U8).

## Integer Literals

## Test: integer literal
```zong-expr
42
```
```ast
42
```

## Test: integer literals execution
```zong-expr
print(42)
```
```execute
42
```

## Boolean Literals

## Test: boolean literal true
```zong-expr
true
```
```ast
true
```

## Test: boolean literal false
```zong-expr
false
```
```ast
false
```

## String Literals

## Test: string literal
```zong-expr
"hello"
```
```ast
(string "hello")
```

## Test: string literal parsing
```zong-expr
"hello"
```
```ast
(string "hello")
```

## U8 Literals

## Test: U8 value in valid range
```zong-program
func main() {
    var b U8;
    b = 255;
    print(b);
}
```
```execute
255
```

## Test: U8 value zero
```zong-program
func main() {
    var b U8;
    b = 0;
    print(b);
}
```
```execute
0
```

## Additional Literal Tests (from expressions_test.md)

## Test: integer literal from expressions
```zong-expr
42
```
```ast
42
```

## Test: string literal from expressions
```zong-expr
"hello"
```
```ast
(string "hello")
```

## Boolean Execution Tests (from extracted_execution_test.md)

## Test: boolean comparisons
```zong-program
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
```
```execute
1
0
1
0
```

## Test: boolean literals
```zong-program
func main() {
		var t Boolean;
		var f Boolean;
		t = true;
		f = false;
		print(t);
		print(f);
	}
```
```execute
1
0
```

## Test: boolean return type
```zong-program
func getTrue(): Boolean {
		return true;
	}
	
	func main() {
		print(getTrue());
	}
```
```execute
1
```

## Basic Literal Execution Tests (from extracted_execution_test.md)

## Test: basic print expression
```zong-expr
print(42)
```
```execute
42
```

## Additional Basic Execution Tests (from execution_test.md)

## Test: simple expression execution
```zong-expr
print(42)
```
```execute
42
```

## Test: multiple print statements
```zong-program
func main() {
    print(1);
    print(2);
    print(3);
}
```
```execute
1
2
3
```

## String Literal Tests (from more_test.md)

## Test: wasm execution baseline
```zong-program
func main() { print(42); }
```
```execute
42
```

## Test: wasm execution empty string
```zong-program
func main() { var s U8[] = ""; print(42); }
```
```execute
42
```

## Test: wasm execution string assignment
```zong-program
func main() { var s U8[] = "hello"; print(42); }
```
```execute
42
```

## Test: wasm execution string declaration
```zong-program
func main() { var s U8[]; print(42); }
```
```execute
42
```

## Test: string literal integration
```zong-program
func main() { print(42); }
```
```execute
42
```

## Test: multiple string literals
```zong-program
func main() { var s1 U8[] = "hello"; var s2 U8[] = "world"; print(5); }
```
```execute
5
```

## Test: string literal assignment
```zong-program
func main() { var s U8[] = "test"; print(4); }
```
```execute
4
```

## Test: string literal compilation
```zong-program
func main() { var msg U8[] = "hello world"; print(11); }
```
```execute
11
```

## Test: string literal deduplication
```zong-program
func main() { var s1 U8[] = "same"; var s2 U8[] = "same"; print(42); }
```
```execute
42
```

## Type and Literal Compile Error Tests (from compile_error_test.md)

## Test: U8 out of range value
```zong-program
func main() {
    var slice U8[];
    append(slice&, 256);
}
```
```compile-error
error: cannot convert integer 256 to U8
```

## Test: type mismatch Boolean to I64
```zong-program
func main() {
    var x Boolean;
    x = true;
    var y I64;
    y = x;
}
```
```compile-error
error: cannot assign Boolean to I64
```

## Test: non-ASCII characters in string literals should be rejected
```zong-program
func main() { var s U8[] = "héllo"; }
```
```compile-error
error: non-ASCII characters are not supported in string literals
```

## Test: unterminated string literal should be rejected
```zong-program
func main() { print("unterminated; }
```
```compile-error
error: unterminated string literal
```

## Test: unexpected character should be rejected
```zong-expr
print(@invalid)
```
```compile-error
error: unexpected character '@'
```

## Test: variable not declared in expression
```zong-expr
undefined
```
```compile-error
undefined symbol 'undefined'
```

## Comprehensive Literal Expression Tests (from parsing_comprehensive_test.md)

## Test: expression statement with integer
```zong-program
42;
```
```ast
[42]
```

## Integer Type Verification Tests (from type_verification_test.md)

## Test: integer literal type verification
```zong-expr
42
```
```ast
42
```
