# Expression parsing 2

Implement more expression types:

- Call - function call with named parameter support
  - `f()` -> `(call (ident f))`
  - `print("hello")` -> `(call (ident "print") (string "hello"))` (ASTNode.ParameterNames will be `[]string{""}`)
  - `atan2(y, x)` -> `(call (ident "atan2") (ident "y") (ident "x"))`
  - `Point(x: 1, y: 2)` -> `(call (ident "Point") "x" (integer 1) "y" (integer 2))` (ASTNode.ParameterNames will be `[]string{"x", "y"}`)
  - `httpGet("http://example.com", headers: h) -> `(call (ident "httpGet") (string "http://example.com") "headers" (ident "h"))` (ASTNode.ParameterNames will be `[]string{"", "h"}`)
- Subscript
  - no slicing (like `x[y:z]`)
  - `x[y]` -> `(idx (ident "x") (ident "y"))`
- unary not
  - `!x` -> `(unary "!" (ident "x"))`
