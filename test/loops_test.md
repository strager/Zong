# Loop control flow tests

## Test: basic loop
```zong-program
loop { print(42); }
```
```ast
[(loop (call (var "print") 42))]
```

## Test: loop with break
```zong-program
loop { break; }
```
```ast
[(loop break)]
```

## Test: loop with continue
```zong-program
loop { continue; }
```
```ast
[(loop continue)]
```

## Test: loop with break and semicolon
```zong-program
loop { break; print(1); }
```
```ast
[(loop break (call (var "print") 1))]
```

## Test: loop with continue and semicolon
```zong-program
loop { continue; print(1); }
```
```ast
[(loop continue (call (var "print") 1))]
```

## Test: nested loop with break
```zong-program
loop { loop { break; } }
```
```ast
[(loop (loop break))]
```

## Test: loop with multiple statements
```zong-program
loop { print(1); print(2); break; }
```
```ast
[(loop (call (var "print") 1) (call (var "print") 2) break)]
```