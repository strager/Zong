# Extracted execution tests

Generated from existing Go test files.

## Tests from boolean_test.go

### Test: boolean comparisons
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

### Test: boolean function parameters
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

### Test: boolean in if statements
```zong-program
func main() {
			var flag Boolean;
			flag = true;
			
			if flag {
				print(1);
			}
			
			flag = false;
			if flag {
				print(2);
			} else {
				print(3);
			}
		}
```
```execute
1
3
```

### Test: boolean literals
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

### Test: boolean loops
```zong-program
func main() {
			var i I64;
			var keepGoing Boolean;
			i = 0;
			keepGoing = true;
			
			loop {
				if i >= 3 {
					keepGoing = false;
				}
				
				if keepGoing {
					print(i);
					i = i + 1;
				} else {
					break;
				}
			}
		}
```
```execute
0
1
2
```

### Test: boolean return type
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

## Tests from compiler_test.go

### Test: arithmetic print
```zong-expr
print(42 + 8)
```
```execute
50
```

### Test: basic print expression
```zong-expr
print(42)
```
```execute
42
```

### Test: complex arithmetic
```zong-expr
print((10 + 5) * 2 - 3)
```
```execute
27
```

### Test: else if chain
```zong-program
func main() {
		var score I64;
		score = 85;
		if score >= 90 {
			print(4);
		} else if score >= 80 {
			print(3);
		} else if score >= 70 {
			print(2);
		} else {
			print(1);
		}
	}
```
```execute
3
```

### Test: if else statement
```zong-program
func main() {
		var x I64;
		x = 10;
		if x > 20 {
			print(1);
		} else {
			print(0);
		}
	}
```
```execute
0
```

### Test: if statement
```zong-program
func main() {
		var x I64;
		x = 42;
		if x == 42 {
			print(1);
		}
	}
```
```execute
1
```

### Test: nested expressions
```zong-expr
print(((2 + 3) * 4 - 8) / 2 + 1)
```
```execute
7
```

### Test: nested if statements
```zong-program
func main() {
		var x I64;
		var y I64;
		x = 5;
		y = 10;
		if x > 0 {
			if y > 0 {
				print(x + y);
			}
		}
	}
```
```execute
15
```

### Test: operator precedence
```zong-expr
print(1 + 2 * 3)
```
```execute
7
```

## Tests from func_test.go

### Test: end to end function execution
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

### Test: function return field access
```zong-program
struct Point { var x I64; var y I64; }
	
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

### Test: function returning struct
```zong-program
struct Point { var x I64; var y I64; }
	
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

### Test: function struct param copies
```zong-program
struct S { var i I64; }

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

### Test: function with complex expressions
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

### Test: mixed field access
```zong-program
struct Point { var x I64; var y I64; }
	
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

### Test: mixed parameters
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

### Test: multiple functions
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

### Test: named parameter calls
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

### Test: nested function calls
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

### Test: nested struct function return
```zong-program
struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
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

### Test: nested struct initialization
```zong-program
struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
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

### Test: nested structs
```zong-program
struct Address { var state I64; var zipCode I64; }
	struct Person { var name I64; var address Address; var age I64; }
	
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

### Test: simple main
```zong-program
func main() {
		print(42);
	}
```
```execute
42
```

### Test: void function
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

## Tests from loop_test.go

### Test: basic loop
```zong-program
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
```
```execute
0
1
2
```

### Test: break continue in nested loops
```zong-program
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
```
```execute
1
3
1
3
```

### Test: continue statement
```zong-program
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
```
```execute
1
3
```

### Test: empty loop
```zong-program
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
```
```execute
1
```

### Test: loop with variable modification
```zong-program
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
```
```execute
15
```

### Test: nested loop break bug
```zong-program
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
```
```execute
3
4
5
```

### Test: nested loops
```zong-program
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
```
```execute
0
1
0
1
```

## Tests from phase3_test.go

### Test: i64 pointer returns
```zong-program
func getPointer(): I64* {
		var x I64;
		x = 42;
		return x&;
	}
	
	func main() {
		var ptr I64*;
		ptr = getPointer();
		print(ptr*);
	}
```
```execute
42
```

### Test: struct parameter passing
```zong-program
struct Point { var x I64; var y I64; }
	
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

## Tests from shadowing_test.go

### Test: deep nested shadowing end to end
```zong-program
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
```
```execute
1
2
3
4
3
2
1
```

### Test: function parameter shadowing end to end
```zong-program
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
```
```execute
42
99
42
```

### Test: shadowing with different types
```zong-program
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
```
```execute
42
1
42
```

### Test: variable shadowing end to end
```zong-program
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
```
```execute
10
20
10
```

## Tests from slice_test.go

### Test: slice address of
```zong-program
func main() {
		var nums I64[];
		print(42);
	}
```
```execute
42
```

### Test: length increment bug
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

### Test: slice basics
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

### Test: slice basics current implementation
```zong-program
func main() {
		var nums I64[];
		append(nums&, 42);
		print(nums[0]);
		print(nums.length);
	}
```
```execute
42
1
```

### Test: slice basics just declaration
```zong-program
func main() {
		var nums I64[];
		print(42);
	}
```
```execute
42
```

### Test: slice basics minimal
```zong-program
func main() {
		var nums I64[];
		append(nums&, 42);
		print(42);
	}
```
```execute
42
```

### Test: slice empty length
```zong-program
func main() {
		var nums I64[];
		print(nums.length);
	}
```
```execute
0
```

### Test: slice field access
```zong-program
func main() {
		var nums I64[];
		var flags Boolean[];
		print(nums.length);
		print(flags.length);
	}
```
```execute
0
0
```

### Test: slice function parameter
```zong-program
func len(_ xs: I64[]): I64 {
		return xs.length
	}
	func first(_ ys: I64[]): I64 {
		return ys[0];
	}
	func edit_first(_ zs: I64[]): I64 {
		zs[0] = 30;
		return 0;
	}
	func main() {
		var nums I64[];
		append(nums&, 10);
		append(nums&, 20);
		print(len(nums));        // 2
		print(first(nums));      // 10
		print(edit_first(nums)); // 0
		print(nums[0]);          // 30
	}
```
```execute
2
10
0
30
```

## Tests from struct_slice_test.go

### Test: struct slice append
```zong-program
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	var p Point;
	p.x = 10;
	p.y = 20;
	append(points&, p);
	print(points.length);
}
```
```execute
1
```

### Test: struct slice append multiple
```zong-program
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
```
```execute
2
```

### Test: struct slice basics
```zong-program
struct Point { var x I64; var y I64; }

func main() {
	var points Point[];
	print(points.length);
}
```
```execute
0
```

### Test: struct slice complex operations
```zong-program
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
```
```execute
0
0
999
888
20
40
```

### Test: struct slice field access at index
```zong-program
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
```
```execute
10
20
```

### Test: struct slice field assignment at index
```zong-program
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
```
```execute
100
200
```

### Test: struct slice indexing
```zong-program
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
```
```execute
10
20
```

### Test: struct slice multiple elements field access
```zong-program
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
```
```execute
10
40
50
```

### Test: struct slice nested field access
```zong-program
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
```
```execute
126
184
```

### Test: struct slice whole struct assignment at index
```zong-program
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
```
```execute
100
200
```

### Test: struct slice with different struct size
```zong-program
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
```
```execute
1
10
20
100
200
```

## Tests from u8_test.go

### Test: i64 slice multiple append
```zong-program
func main() {
	var slice I64[];
	append(slice&, 10);
	append(slice&, 20);
	append(slice&, 30);
	
	print(slice[0]);
	print(slice[1]);
	print(slice[2]);
}
```
```execute
10
20
30
```

### Test: i64 slice with append
```zong-program
func main() {
	var slice I64[];
	append(slice&, 10);
	print(slice[0]);
}
```
```execute
10
```

### Test: u8 arithmetic
```zong-program
func main() {
	var a U8 = 10;
	var b U8 = 5;
	print(a + b);
	print(a - b);
	print(a * b);
	print(a / b);
	print(a % b);
}
```
```execute
15
5
50
2
0
```

### Test: u8 assignment
```zong-program
func main() {
	var x U8;
	x = 123;
	print(x);
	x = 200;
	print(x);
}
```
```execute
123
200
```

### Test: u8 basic variable declaration
```zong-program
func main() {
	var x U8 = 42;
	print(x);
}
```
```execute
42
```

### Test: u8 comparisons
```zong-program
func main() {
	var a U8 = 10;
	var b U8 = 5;
	var c U8 = 10;
	
	if (a == c) {
		print(1);
	}
	if (a != b) {
		print(2);
	}
	if (a > b) {
		print(3);
	}
	if (b < a) {
		print(4);
	}
	if (a >= c) {
		print(5);
	}
	if (b <= a) {
		print(6);
	}
}
```
```execute
1
2
3
4
5
6
```

### Test: u8 max value
```zong-program
func main() {
	var x U8 = 255;
	print(x);
}
```
```execute
255
```

### Test: u8 min value
```zong-program
func main() {
	var x U8 = 0;
	print(x);
}
```
```execute
0
```

### Test: u8 slice declaration only
```zong-program
func main() {
	var slice U8[];
	print(123);
}
```
```execute
123
```

### Test: u8 slice min max values
```zong-program
func main() {
	var slice U8[];
	append(slice&, 0);
	append(slice&, 255);
	
	print(slice[0]);
	print(slice[1]);
}
```
```execute
0
255
```

### Test: u8 slice multiple append
```zong-program
func main() {
	var slice U8[];
	append(slice&, 10);
	append(slice&, 20);
	append(slice&, 30);
	
	print(slice[0]);
	print(slice[1]);
	print(slice[2]);
}
```
```execute
10
20
30
```

### Test: u8 slice simple
```zong-program
func main() {
	var slice U8[];
	print(42);
}
```
```execute
42
```

### Test: u8 slice with append
```zong-program
func main() {
	var slice U8[];
	append(slice&, 10);
	print(slice[0]);
}
```
```execute
10
```

## Tests from var_init_test.go

### Test: basic variable initialization
```zong-program
func main() {
	var x I64 = 42;
	print(x);
}
```
```execute
42
```

### Test: boolean variable initialization
```zong-program
func main() {
	var flag Boolean = true;
	print(flag);
	var flag2 Boolean = false;
	print(flag2);
}
```
```execute
1
0
```

### Test: equivalence with separate assignment
```zong-program
func main() {
	var x I64 = 5;
	var y I64 = x * 2;
	print(y);
}
```
```execute
10
```

### Test: mixed initialized and uninitialized vars
```zong-program
func main() {
	var x I64 = 5;
	var y I64;
	y = x * 2;
	print(y);
}
```
```execute
10
```

### Test: multiple variable initialization
```zong-program
func main() {
	var x I64 = 10;
	var y I64 = 20;
	var z I64 = x + y;
	print(z);
}
```
```execute
30
```

### Test: pointer variable initialization
```zong-program
func main() {
	var x I64 = 42;
	var ptr I64* = x&;
	print(ptr*);
}
```
```execute
42
```

### Test: variable initialization with expressions
```zong-program
func main() {
	var a I64 = 3;
	var b I64 = 4;
	var hypotenuse I64 = a * a + b * b;
	print(hypotenuse);
}
```
```execute
25
```

### Test: slice simple declaration
```zong-program
func main() {
		var nums I64[];
		print(42);
	}
```
```execute
42
```
