### Test: Integer literals
```zong-expr
print(42)
```
```execute
42
```

### Test: Addition
```zong-expr
print(10 + 5)
```
```execute
15
```

### Test: Subtraction
```zong-expr
print(10 - 3)
```
```execute
7
```

### Test: Multiplication
```zong-expr
print(6 * 7)
```
```execute
42
```

### Test: Division
```zong-expr
print(20 / 4)
```
```execute
5
```

### Test: Modulo
```zong-expr
print(17 % 5)
```
```execute
2
```

### Test: Operator precedence (mult before add)
```zong-expr
print(2 + 3 * 4)
```
```execute
14
```

### Test: Parentheses override precedence
```zong-expr
print((2 + 3) * 4)
```
```execute
20
```

### Test: Equality (true)
```zong-expr
print(5 == 5)
```
```execute
1
```

### Test: Equality (false)
```zong-expr
print(5 == 3)
```
```execute
0
```

### Test: Not equal
```zong-expr
print(5 != 3)
```
```execute
1
```

### Test: Greater than
```zong-expr
print(5 > 3)
```
```execute
1
```

### Test: Less than
```zong-expr
print(3 < 5)
```
```execute
1
```

### Test: Complex nested expression
```zong-expr
print((10 + 5) * 2 - 3)
```
```execute
27
```

### Test: Mixed arithmetic with precedence
```zong-expr
print(20 / 4 + 3 * 2)
```
```execute
11
```

