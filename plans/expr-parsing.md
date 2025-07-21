# Expression parsing

Create an expression parser which builds upon the existing lexer.

AST definition:

```
type ASTNode struct {
    Kind     NodeKind
    // NodeIdent, NodeString:
    String   string
    // NodeInteger:
    Integer  int64
    // NodeBinary:
    Op       string   // "+", "-", "==", "!"
    Children []*ASTNode
}
```

AST node types:

* Ident - identifier reference
* String - string literal
* Integer - integer literal
* Binary - binary operator like '+'

## S-Expression Representation

AST nodes will be represented as s-expressions for testing and debugging:

* `(ident "name")` - identifier
* `(string "value")` - string literal
* `(integer 42)` - integer literal
* `(binary "op" left right)` - binary operation

There is a function that pretty-prints an ASTNode to the s-expression syntax.
Use this function in tests liberally.

## Test Cases

### Literals
* `42` → `(integer 42)`
* `"hello"` → `(string "hello")`
* `myVar` → `(ident "myVar")`

### Binary Operations
* `1 + 2` → `(binary "+" (integer 1) (integer 2))`
* `x == y` → `(binary "==" (ident "x") (ident "y"))`
* `"a" + "b"` → `(binary "+" (string "a") (string "b"))`

### Operator Precedence
* `1 + 2 * 3` → `(binary "+" (integer 1) (binary "*" (integer 2) (integer 3)))`
* `(1 + 2) * 3` → `(binary "*" (binary "+" (integer 1) (integer 2)) (integer 3))`

### Complex Expressions
* `x + y * z` → `(binary "+" (ident "x") (binary "*" (ident "y") (ident "z")))`
* `a == b + c` → `(binary "==" (ident "a") (binary "+" (ident "b") (ident "c")))`
