# Parser and Lexer Error Handling Refactoring

## User Intent

- Replace panic() and Go's built-in error type with new Error and ErrorCollection types in the compiler
- Error struct should NOT satisfy Go's Error interface - it's a distinct type
- No backwards compatibility - changing both internal and external design
- Lexer should have an *ErrorCollection field instead of returning errors
- SkipToken should still panic (internal compiler error)
- NextToken should accumulate errors in the lexer and keep going
- Parse functions should use the lexer's ErrorCollection rather than taking separate parameters
- Tests need to catch panics, convert to Error, and add to ErrorCollection before asserting
- Add a method to lexer to create and append errors (not a standalone NewError function)

## Current State Analysis

The codebase currently has:
- New Error and ErrorCollection types defined in main.go:8-18
- ~50+ panic() calls throughout the compiler, primarily in parser/lexer
- Basic Error struct with message field
- ErrorCollection with Append method

## Implementation Plan

### Phase 1: Enhance Error Types and Lexer
1. **Add ErrorCollection methods**: 
   - `HasErrors() bool` - check if collection has errors
   - `Count() int` - number of errors
   - `String() string` - format all errors for display
2. **Add ErrorCollection field to Lexer struct**: `Errors *ErrorCollection`
3. **Add Lexer error method**: `(l *Lexer) AddError(message string)` - creates Error and appends to l.Errors

### Phase 2: Update Lexer Error Handling
1. **Keep Lexer.SkipToken() panicking**: It's for internal compiler errors
2. **Update lexer token reading methods** in NextToken(): Use l.AddError() instead of panic
   - `readString()` - handle unterminated strings, call l.AddError(), continue
   - `readNumber()` - handle invalid numeric formats, call l.AddError(), continue
   - `readCharLiteral()` - handle invalid character literals, call l.AddError(), continue
3. **Update NextToken()**: Accumulate lexical errors using l.AddError(), keep parsing

### Phase 3: Update Parser Functions to Use Lexer's ErrorCollection
1. **Update core parsing functions** to use lexer's error method:
   - `ParseExpression(l *Lexer)` - call l.AddError() for syntax errors
   - `ParseStatement(l *Lexer)` - call l.AddError() for syntax errors
   - `parsePrimary(l *Lexer)` - call l.AddError() for syntax errors
   - `parseTypeExpression(l *Lexer)` - call l.AddError() for syntax errors
   - `parseBlockStatements(l *Lexer)` - call l.AddError() for syntax errors
   - `parseFunctionDeclaration(l *Lexer)` - call l.AddError() for syntax errors
   - `ParseProgram(l *Lexer)` - call l.AddError() for syntax errors

2. **Replace parsing panic() calls** with l.AddError() calls
3. **Add error recovery**: Continue parsing after syntax errors when possible

### Phase 4: Update Test Functions
1. **Add panic recovery in tests**: Catch panics from SkipToken(), convert to Error, add to lexer.Errors
2. **Update test assertions**: Check lexer.Errors instead of expecting panics for parsing errors
3. **Keep panic expectations**: For internal compiler errors (SkipToken misuse)
4. **Remove panic recovery for parsing**: Only catch SkipToken panics

### Phase 5: Update Main Pipeline
1. **Update main compilation pipeline**: Check lexer.Errors after parsing
2. **Update semantic analysis integration**: Combine lexer errors with type checker errors
3. **Change main() function**: Check ErrorCollection and exit gracefully
4. **Update CLI error reporting**: Display ErrorCollection nicely

## Benefits

- **Better error recovery**: Parser continues after syntax errors
- **Multiple error reporting**: Show all syntax/lexical errors at once
- **Clear error boundaries**: SkipToken panics = internal bugs, other errors = user input issues
- **Consistent error handling**: Unified error system via lexer.AddError()
- **Better testing**: Predictable error behavior with appropriate panic handling