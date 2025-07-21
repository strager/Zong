# Statement parsing

Create a statement parser which builds upon the existing lexer and expression
parser.

Reuse ASTNode, adding new node types:

- If - `if cond { body; }`
- Var - `var x Type;`
  - The type is a child node (just an Ident node for now)
  - Only one variable per statement
  - Semicolon is required
- Block - `{ s1; s2; }`
  - Zero or more children
  - Example: `{ expr; }` is a Block node with the expression node as a child
- Return - `return;` or `return foo;`
  - Semicolon presence after `return` decides whether there's an expression
- Loop - `loop { body; }`
- Break - `break;` (for loops)
- Continue - `continue;` (for loops)

Expression statements are terminated by a semicolon.

There is no separate "Expr" node type; just use the underlying node (Binary, Integer, etc.).

Loop and If nodes contain their body statements directly as children, not wrapped in a Block node. For example, `loop {}` is a Loop with 0 children, and `loop { { } }` is a Loop with one Block child.
