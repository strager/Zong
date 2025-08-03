# Function parsing tests

## Test: void function no parameters
```zong-program
func test() {}
```
```ast
[(func "test" [] nil [])]
```

## Test: function with I64 return type
```zong-program
func add(): I64 {}
```
```ast
[(func "add" [] "I64" [])]
```

## Test: function with positional parameters
```zong-program
func add(_ addA: I64, _ addB: I64): I64 {}
```
```ast
[(func "add"
  [(param "addA" "I64" positional)
   (param "addB" "I64" positional)]
  "I64"
  [])]
```

## Test: function with named parameters
```zong-program
func test(testX: I64, testY: I64) {}
```
```ast
[(func "test"
  [(param "testX" "I64" named)
   (param "testY" "I64" named)]
  nil
  [])]
```

## Test: function with body
```zong-program
func test() { var x I64; }
```
```ast
[(func "test"
  []
  nil
  [(var-decl "x" "I64")])]
```

## Function Execution Tests

## Test: simple function call
```zong-program
func add(_ addA5: I64, _ addB5: I64): I64 { return addA5 + addB5; }
					 func main() { print(add(5, 3)); }
```
```execute
8
```

## Test: void function
```zong-program
func printTwice(_ printTwiceX2: I64) { print(printTwiceX2); print(printTwiceX2); }
					 func main() { printTwice(42); }
```
```execute
42
42
```

## Test: multiple function calls
```zong-program
func double(_ doubleX2: I64): I64 { return doubleX2 * 2; }
					 func triple(_ tripleX2: I64): I64 { return tripleX2 * 3; }
					 func main() { print(double(5)); print(triple(4)); }
```
```execute
10
12
```

## Test: nested function calls
```zong-program
func add(_ addA6: I64, _ addB6: I64): I64 { return addA6 + addB6; }
					 func multiply(_ multiplyA2: I64, _ multiplyB2: I64): I64 { return multiplyA2 * multiplyB2; }
					 func main() { print(add(multiply(2, 3), multiply(4, 5))); }
```
```execute
26
```

## Test: function with complex expression
```zong-program
func compute(_ computeA2: I64, _ computeB2: I64, _ computeC2: I64): I64 { return (computeA2 + computeB2) * computeC2 - 10; }
					 func main() { print(compute(3, 4, 5)); }
```
```execute
25
```

## Advanced Function Features

## Test: I64 pointer return type parsing
```zong-program
func getPointer(): I64* {
    return null;
}
```
```ast
[(func "getPointer" [] "I64*" [(return (var "null"))])]
```

## Test: struct parameter parsing
```zong-program
func test(_ testP: Point): I64 { return 42; }
```
```ast
[(func "test" [(param "testP" "Point" positional)] "I64" [(return 42)])]
```

## Function Call Tests (from expressions_test.md)

## Test: function call no args
```zong-expr
f()
```
```ast
(call (var "f"))
```

## Test: function call with string arg
```zong-expr
print("hello")
```
```ast
(call (var "print") (string "hello"))
```

## Test: function call multiple args
```zong-expr
atan2(y, x)
```
```ast
(call (var "atan2") (var "y") (var "x"))
```

## Test: function call with named parameters
```zong-expr
Point(x: 1, y: 2)
```
```ast
(call (var "Point") "x" 1 "y" 2)
```

## Test: function call mixed parameters
```zong-expr
httpGet("http://example.com", headers: h)
```
```ast
(call (var "httpGet") (string "http://example.com") "headers" (var "h"))
```

## Test: parenthesized function call
```zong-expr
(foo)()
```
```ast
(call (var "foo"))
```

## Test: chained function call
```zong-expr
arr[0](x)
```
```ast
(call (idx (var "arr") 0) (var "x"))
```

## Boolean Function Tests (from extracted_execution_test.md)

## Test: boolean function parameters
```zong-program
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
```
```execute
1
0
```

## Function Execution Tests (from extracted_execution_test.md)

## Test: end to end function execution
```zong-program
func add(_ addA3: I64, _ addB3: I64): I64 {
	return addA3 + addB3;
}

func main() {
	print(add(5, 3));
}
```
```execute
8
```

## Test: function return field access
```zong-program
struct Point(x: I64, y: I64);

func makePoint(pointX: I64, pointY: I64): Point {
	var p Point;
	p.x = pointX;
	p.y = pointY;
	return p;
}

func main() {
	print(makePoint(pointX: 10, pointY: 20).x);
	print(makePoint(pointX: 30, pointY: 40).y);
}
```
```execute
10
40
```

## Test: function returning struct
```zong-program
struct Point(x: I64, y: I64);

func createPoint(_ createPointXVal: I64, _ createPointYVal: I64): Point {
	var p Point;
	p.x = createPointXVal;
	p.y = createPointYVal;
	return p;
}

func main() {
	var result Point;
	result = createPoint(10, 20);
	print(result.x);
	print(result.y);
}
```
```execute
10
20
```

## Test: function struct param copies
```zong-program
struct S(i: I64);

func f(_ fS: S) {
	fS.i = 3;
	print(fS.i);
}

func main() {
	var ss S;
	ss.i = 2;
	print(ss.i);
	f(ss);
	print(ss.i);
}
```
```execute
2
3
2
```

## Test: function with complex expressions
```zong-program
func compute(_ computeA: I64, _ computeB: I64, _ computeC: I64): I64 {
	return (computeA + computeB) * computeC - 10;
}

func main() {
	print(compute(3, 4, 5));
}
```
```execute
25
```

## Test: mixed field access
```zong-program
struct Point(x: I64, y: I64);

func makePoint(pointX: I64, pointY: I64): Point {
	var newP Point;
	newP.x = pointX;
	newP.y = pointY;
	return newP;
}

func main() {
	// Test variable field access
	var mainP Point;
	mainP.x = 100;
	mainP.y = 200;
	print(mainP.x);
	print(mainP.y);
	
	// Test function return field access
	print(makePoint(pointX: 300, pointY: 400).x);
	print(makePoint(pointX: 500, pointY: 600).y);
}
```
```execute
100
200
300
600
```

## Test: mixed parameters
```zong-program
func compute(_ computeBase: I64, computeMultiplier: I64, computeOffset: I64): I64 {
	return computeBase * computeMultiplier + computeOffset;
}

func main() {
	print(compute(5, computeMultiplier: 3, computeOffset: 10));
}
```
```execute
25
```

## Test: multiple functions
```zong-program
func double(_ doubleX: I64): I64 {
	return doubleX * 2;
}

func triple(_ tripleX: I64): I64 {
	return tripleX * 3;
}

func main() {
	print(double(5));
	print(triple(4));
}
```
```execute
10
12
```

## Test: named parameter calls
```zong-program
func greet(greetName: I64, greetAge: I64) {
	print(greetName);
	print(greetAge);
}

func main() {
	greet(greetName: 42, greetAge: 25);
	greet(greetAge: 30, greetName: 50); 
}
```
```execute
42
25
50
30
```

## Test: function arguments execute in source code order
```zong-program
func print_and_return(_ x: I64): I64 {
	print(x);
	return x;
}
func greet(greetName: I64, greetAge: I64) {
	print(greetName);
	print(greetAge);
}

func main() {
	greet(greetAge: print_and_return(30), greetName: print_and_return(50));
}
```
```execute
30
50
50
30
```

## Test: nested function calls
```zong-program
func add(_ addA4: I64, _ addB4: I64): I64 {
	return addA4 + addB4;
}

func multiply(_ multiplyA: I64, _ multiplyB: I64): I64 {
	return multiplyA * multiplyB;
}

func main() {
	print(add(multiply(2, 3), multiply(4, 5)));
}
```
```execute
26
```

## Test: nested struct function return
```zong-program
struct Address(state: I64, zipCode: I64);
struct Person(name: I64, address: Address, age: I64);

func createAddress(addrState: I64, addrZip: I64): Address {
	var addr Address;
	addr.state = addrState;
	addr.zipCode = addrZip;
	return addr;
}

func createPerson(personName: I64, personAge: I64): Person {
	var p Person;
	p.name = personName;
	p.age = personAge;
	p.address = createAddress(addrState: 77, addrZip: 98765);
	return p;
}

func main() {
	// Test function return field access with nested structs
	print(createPerson(personName: 300, personAge: 35).name);
	print(createPerson(personName: 400, personAge: 40).address.state);
	print(createPerson(personName: 500, personAge: 45).address.zipCode);
	print(createPerson(personName: 600, personAge: 50).age);
}
```
```execute
300
77
98765
50
```

## Test: simple main
```zong-program
func main() {
	print(42);
}
```
```execute
42
```

## Test: void function
```zong-program
func printTwice(_ printTwiceX: I64) {
	print(printTwiceX);
	print(printTwiceX);
}

func main() {
	printTwice(7);
}
```
```execute
7
7
```

## Test: struct parameter passing
```zong-program
struct Point(x: I64, y: I64);

func processPoint(_ processPointP: Point) {
	print(processPointP.x);
	print(processPointP.y);
}

func main() {
	var p Point;
	p.x = 10;
	p.y = 20;
	processPoint(p);
}
```
```execute
10
20
```

## Function Compile Error Tests (from compile_error_test.md)

## Test: undefined function call
```zong-expr
nonExistentFunction()
```
```compile-error
undefined symbol 'nonExistentFunction'
```

## Test: unknown function in call
```zong-expr
unknown(42)
```
```compile-error
undefined symbol 'unknown'
```

## Test: print with no arguments
```zong-expr
print()
```
```compile-error
error: print() function expects 1 argument
```

## Function Parser Robustness Tests (from parser_robustness_test.md)

## Test: malformed function call missing comma
```zong-expr
func(arg1 arg2)
```
```compile-error

```

## Test: function declaration missing parentheses
```zong-program
func test { print(42); }
```
```compile-error
expected '(' after function name
```

## Function Type Verification Tests (from type_verification_test.md)

## Test: function call type verification
```zong-program
func main() {
    print(42);
}
```
```execute
42
```
