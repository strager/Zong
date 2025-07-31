# Control Flow Tests

Tests for if statements, loops, break/continue, and return.

## If Statements

## Test: if statement simple
```zong-program
if x { y; }
```
```ast
[(if (var "x") [(var "y")])]
```

## Test: if statement with expression
```zong-program
if 1 + 2 { 3; }
```
```ast
[(if (binary "+" 1 2) [3])]
```

## Test: if statement with equality
```zong-program
if foo == bar { return 42; }
```
```ast
[(if (binary "==" (var "foo") (var "bar"))
 [(return 42)])]
```

## Test: if else statement
```zong-program
if x { y; } else { z; }
```
```ast
[(if (var "x")
  [(var "y")]
  nil
  [(var "z")])]
```

## Test: if else with expressions
```zong-program
if x == 1 { print(1); } else { print(0); }
```
```ast
[(if (binary "==" (var "x") 1)
  [(call (var "print") 1)]
  nil
  [(call (var "print") 0)])]
```

## Test: if else if else chain
```zong-program
if x > 0 { print(1); } else if x < 0 { print(2); } else { print(0); }
```
```ast
[(if (binary ">" (var "x") 0)
  [(call (var "print") 1)]
  (binary "<" (var "x") 0)
  [(call (var "print") 2)]
  nil
  [(call (var "print") 0)])]
```

## Loops

## Test: basic loop
```zong-program
loop { print(42); }
```
```ast
[(loop
  [(call (var "print") 42)])]
```

## Test: loop with break
```zong-program
loop { break; }
```
```ast
[(loop [break])]
```

## Test: loop with continue
```zong-program
loop { continue; }
```
```ast
[(loop [continue])]
```

## Test: loop with break and semicolon
```zong-program
loop { break; print(1); }
```
```ast
[(loop
  [break
   (call (var "print") 1)])]
```

## Test: loop with continue and semicolon
```zong-program
loop { continue; print(1); }
```
```ast
[(loop
  [continue
   (call (var "print") 1)])]
```

## Test: nested loop with break
```zong-program
loop { loop { break; } }
```
```ast
[(loop
  [(loop [break])])]
```

## Test: loop with multiple statements
```zong-program
loop { print(1); print(2); break; }
```
```ast
[(loop
  [(call (var "print") 1)
   (call (var "print") 2)
   break])]
```

## Test: loop with conditional break
```zong-program
loop {
    var i I64;
    if i == 10 {
        break;
    }
}
```
```ast
[(loop
  [(var-decl "i" "I64")
   (if (binary "==" (var "i") 10)
    [break])])]
```

## Break and Continue

## Test: break statement
```zong-program
break;
```
```ast
[break]
```

## Test: continue statement
```zong-program
continue;
```
```ast
[continue]
```

## Return Statements

## Test: return statement void
```zong-program
return;
```
```ast
[(return)]
```

## Test: return statement with integer
```zong-program
return 42;
```
```ast
[(return 42)]
```

## Test: return statement with binary expression
```zong-program
return x + y;
```
```ast
[(return (binary "+" (var "x") (var "y")))]
```

## Test: return statement with equality
```zong-program
return foo == bar;
```
```ast
[(return (binary "==" (var "foo") (var "bar")))]
```

## Additional If Statement Tests (from statements_test.md)

## Test: if statement simple from statements
```zong-program
if x { y; }
```
```ast
[(if (var "x") [(var "y")])]
```

## Test: if statement with expression from statements
```zong-program
if 1 + 2 { 3; }
```
```ast
[(if (binary "+" 1 2) [3])]
```

## Test: if statement with equality from statements
```zong-program
if foo == bar { return 42; }
```
```ast
[(if (binary "==" (var "foo") (var "bar"))
 [(return 42)])]
```

## Test: if else statement from statements
```zong-program
if x { y; } else { z; }
```
```ast
[(if (var "x")
  [(var "y")]
  nil
  [(var "z")])]
```

## Test: if else with expressions from statements
```zong-program
if x == 1 { print(1); } else { print(0); }
```
```ast
[(if (binary "==" (var "x") 1)
  [(call (var "print") 1)]
  nil
  [(call (var "print") 0)])]
```

## Test: block statement from statements
```zong-program
{
    {
        {}
    }
}
```
```ast
[(block
  [(block
    [(block [])])])]
```

## Test: if statement with false condition execution
```zong-program
func main() {
    var x I64;
    x = 420;
    if x == 42 {
        print(1);
    }
}
```
```execute

```

## Boolean Control Flow Tests (from extracted_execution_test.md)

## Test: boolean in if statements
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

## Test: boolean loops
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

## Test: else if chain
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

## Test: if else statement
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

## Test: if statement
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

## Test: nested if statements
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

## Loop Tests (from extracted_execution_test.md)

## Test: basic loop
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

## Test: break continue in nested loops
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

## Test: continue statement
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

## Test: empty loop
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

## Test: loop with variable modification
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

## Test: nested loop break bug
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

## Test: nested loops
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

## Control Flow Compile Error Tests (from compile_error_test.md)

## Test: break outside of loop
```zong-program
func main() {
    break;
}
```
```compile-error
error: break statement outside of loop
```

## Test: continue outside of loop
```zong-program
func main() {
    continue;
}
```
```compile-error
error: continue statement outside of loop
```

## Control Flow Parser Robustness Tests (from parser_robustness_test.md)

## Test: if statement without brace
```zong-expr
if x == 1 ;
```
```compile-error

```

## Test: loop statement without brace
```zong-expr
loop ;
```
```compile-error

```

## Comprehensive Control Flow Tests (from parsing_comprehensive_test.md)

## Test: empty block statement
```zong-program
{ }
```
```ast
[(block [])]
```

## Test: block with single expression
```zong-program
{ x; }
```
```ast
[(block [(var "x")])]
```

## Test: block with multiple expressions
```zong-program
{ 1; 2; }
```
```ast
[(block [1 2])]
```

## Test: block with variable and return
```zong-program
{ var x int; return x; }
```
```ast
[(block [(var-decl "x" "int") (return (var "x"))])]
```

## Test: nested empty blocks
```zong-program
{ { } }
```
```ast
[(block [(block [])])]
```

## Test: complex if statement with nested blocks
```zong-program
if x > 0 { var y int; return y + 1; }
```
```ast
[(if (binary ">" (var "x") 0) [(var-decl "y" "int") (return (binary "+" (var "y") 1))])]
```

## Test: loop with break and continue
```zong-program
loop { if done { break; } continue; }
```
```ast
[(loop [(if (var "done") [break]) continue])]
```

## Test: deeply nested blocks
```zong-program
{ if a { { b; } } }
```
```ast
[(block [(if (var "a") [(block [(var "b")])])])]
```