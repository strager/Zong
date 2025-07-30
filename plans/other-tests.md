# Plan: Handling Hard-to-Port Tests

This document analyzes the 24 Go test files in the Zong compiler and provides specific recommendations for each category of tests that cannot be easily ported to Sexy tests.

## Executive Summary

Of ~500 total test cases:
- **~300 can be ported** to Sexy tests (parsing, execution, type checking)
- **~200 cannot be ported** and need different strategies
- **Framework extensions needed** to handle compilation errors and robustness testing

## Test Categories & Recommendations

### Category 1: Unit Tests for Internal Functions (KEEP AS-IS)

These test internal compiler implementation details not visible to users.

#### `wasmutil_test.go` - **KEEP ALL TESTS**
- Tests: `writeByte()`, `writeLEB128()`, `EmitWASMHeader()`, WASM section generation
- **Rationale**: Critical low-level functionality. No user-visible equivalent.
- **Action**: Keep unchanged. These are essential unit tests.

#### `locals_test.go` - **MIXED APPROACH**
- **Keep**: Storage allocation tests (`TestCollectSingleLocalVariable`, address calculation)
- **Port**: Higher-level integration tests that have execution equivalents
- **Rationale**: Storage logic is internal, but some behaviors are testable via execution

#### `symboltable_test.go` - **KEEP UNIT TESTS, PORT SCENARIOS**
- **Keep**: `NewSymbolTable()`, `DeclareVariable()`, lookup functions
- **Port**: Variable scoping scenarios, shadowing behavior (can test via compilation errors)
- **Rationale**: API tests stay, behavioral tests move to Sexy

### Category 2: Error Recovery & Parser Robustness (FRAMEWORK EXTENSION NEEDED)

#### `parseexpr_test.go`, `parsestmt_test.go` - **EXTEND SEXY FRAMEWORK**
- Tests: `TestParseExpressionMalformedFunctionCall()`, `TestParsePrimaryUnknownToken()`
- **Recommendation**: Add `parse-error` test type to Sexy framework
- **New Syntax**:
  ```markdown
  ## Test: malformed function call
  ```zong-expr
  func(arg1 arg2  // missing comma
  ```
  ```parse-error
  Expected comma between arguments
  ```
  ```
- **Priority**: Medium (nice-to-have for completeness)

#### `typechecker_test.go` - **EXTEND SEXY FRAMEWORK**
- Tests: Type mismatch detection, undefined variables
- **Recommendation**: Add `compile-error` test type
- **New Syntax**:
  ```markdown
  ## Test: type mismatch
  ```zong-program
  func main() {
      var x I64;
      var y Boolean;
      x = y;  // type error
  }
  ```
  ```compile-error
  error: cannot assign Boolean to I64
  ```
  ```
- **Priority**: High (many type checking scenarios need this)

### Category 3: Panic & Recovery Tests (FRAMEWORK EXTENSION)

#### `compiler_test.go`, `locals_test.go` - **ADD PANIC TEST TYPE**
- Tests: `TestEmitExpressionUndefinedVariable()`, undefined variable panics
- **Recommendation**: Add `panic` test type (lower priority)
- **New Syntax**:
  ```markdown
  ## Test: undefined variable panic
  ```zong-expr
  undefinedVar
  ```
  ```panic
  Undefined variable: undefinedVar
  ```
  ```
- **Priority**: Low (most can be converted to compile-error tests)

### Category 4: WASM Generation Verification (MIXED)

#### `compiler_test.go` WASM tests - **SIMPLIFY & PORT**
- **Delete**: Detailed bytecode verification tests
- **Port**: Basic "compiles successfully" tests using existing execution framework
- **Keep**: Critical integration tests that verify WASM structure
- **Rationale**: Execution tests already verify WASM works; detailed bytecode tests are overkill

### Category 5: Specific File Analysis

#### `boolean_test.go` - **PORT ALL**
- Simple parsing and execution tests
- **Action**: Move to `test/types_comprehensive_test.md`

#### `u8_test.go` - **PORT WITH COMPILE-ERROR EXTENSION**
- Tests U8 range validation (0-255)
- **Action**: Port basic tests, use `compile-error` for out-of-range values

#### `func_test.go` - **PORT MOST, KEEP COMPILATION UNIT TESTS**
- **Port**: End-to-end function execution tests
- **Keep**: `TestBasicFunctionCompilation()` for WASM generation verification

#### `struct_test.go` - **MIXED**
- **Port**: Field access, struct usage scenarios
- **Keep**: `TestStructSymbolTable()`, `TestStructTypeSize()` (internal APIs)

#### `slice_test.go` - **PORT ALL**
- All tests have execution equivalents
- **Action**: Move to `test/slices_comprehensive_test.md`

#### `loop_test.go` - **PORT WITH COMPILE-ERROR**
- **Port**: Basic loop execution
- **Use compile-error**: Break/continue outside loop tests

#### `phase3_test.go` - **PORT ALL**
- Parsing and execution tests
- **Action**: Move to `test/advanced_features_test.md`

#### `var_init_test.go` - **PORT ALL** 
- Simple execution equivalence tests
- **Action**: Move to `test/variable_init_test.md`

#### `string_*.go` - **PORT EXECUTION, KEEP INTERNAL TESTS**
- **Port**: String execution tests to Sexy
- **Keep**: String literal compilation details

#### `shadowing_test.go` - **PORT WITH COMPILE-ERROR**
- Variable shadowing behavior
- **Action**: Most can use `compile-error` test type

#### `lex_test.go` - **KEEP AS-IS**
- Lexer unit tests, no user-visible equivalent
- **Action**: Keep unchanged

#### `debug_pointer_test.go` - **PORT ALL**
- Pointer operation tests
- **Action**: Move to `test/pointers_comprehensive_test.md`

## Implementation Plan

### Phase 1: Port Easy Tests (Week 1)
- Port ~300 easily portable test cases to Sexy format
- Files: `boolean_test.go`, `slice_test.go`, `phase3_test.go`, `var_init_test.go`, `debug_pointer_test.go`
- Create 8-10 comprehensive Sexy test files

### Phase 2: Extend Sexy Framework (Week 2)
- Implement `compile-error` test type in Sexy framework
- Port type checking and error condition tests
- Files: `typechecker_test.go`, `loop_test.go` (error cases), `u8_test.go` (range errors)

### Phase 3: Framework Polish (Week 3)
- Consider implementing `parse-error` test type for robustness tests
- Port remaining applicable tests from `parseexpr_test.go`, `parsestmt_test.go`

### Phase 4: Cleanup (Week 4)
- Remove redundant Go tests that are now covered by Sexy tests
- Document which Go tests remain and why
- Update test documentation

## Framework Extensions Needed

### High Priority: `compile-error` Test Type
```go
// In sexy/testcase.go
type CompileErrorTest struct {
    ExpectedError string
}
```

### Medium Priority: `parse-error` Test Type  
```go
type ParseErrorTest struct {
    ExpectedError string
}
```

### Low Priority: `panic` Test Type
```go
type PanicTest struct {
    ExpectedPanicMessage string
}
```

## Expected Outcomes

After implementation:
- **~300 Sexy tests** covering all user-facing functionality
- **~150 Go unit tests** for internal APIs and low-level functionality  
- **Comprehensive error testing** via `compile-error` test type
- **Clear separation** between user-facing behavior tests (Sexy) and implementation tests (Go)
- **Maintainable test suite** with declarative tests for features, unit tests for internals

## Risks & Mitigations

**Risk**: Over-engineering the Sexy framework
**Mitigation**: Start with `compile-error` only, add others based on actual need

**Risk**: Losing test coverage during transition
**Mitigation**: Port incrementally, run both test suites in parallel during transition

**Risk**: Sexy tests becoming too complex
**Mitigation**: Keep extensions simple, focus on user-visible behavior only