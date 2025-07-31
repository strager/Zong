# Sexy Test Reorganization Plan

## Overview

This document provides a methodical plan to reorganize the Zong test suite from 30+ scattered files into 8 well-organized files. We will preserve all existing test content exactly, moving tests one at a time to ensure nothing is lost.

## Current State Analysis

### File Inventory (30 files)
1. **TestAddressOfOperations_test.md** - Address-of operator tests (8 execution tests)
2. **TestAdvancedPointerScenarios_test.md** - Complex pointer scenarios (0 tests - empty file)
3. **TestComparisons_test.md** - Comparison operators execution tests (5 tests)
4. **TestDivisionAndModulo_test.md** - Division/modulo execution tests (unknown content)
5. **TestFullCapabilitiesDemo_test.md** - Full language demo (unknown content)
6. **TestPhase1Functions_test.md** - Function tests (unknown content)
7. **TestPointerOperations_test.md** - Basic pointer operations (8 execution tests)
8. **TestTypeASTInCompilation_test.md** - Type AST tests (unknown content)
9. **advanced_features_test.md** - Advanced features (unknown content)
10. **arithmetic_ops_test.md** - Arithmetic operators AST tests (8 tests)
11. **binary_expr_test.md** - Binary expressions AST tests (10 tests)
12. **comparisons_test.md** - Comparison operators AST tests (8 tests)
13. **compile_error_test.md** - Compilation error tests (unknown content)
14. **execution_test.md** - Basic execution tests (5 tests)
15. **expressions_test.md** - Comprehensive expression tests (60+ tests)
16. **extracted_execution_test.md** - Auto-generated execution tests (100+ tests)
17. **functions_test.md** - Function parsing tests (5 tests)
18. **locals_test.md** - Local variable tests (unknown content)
19. **loops_test.md** - Loop control flow tests (8 tests)
20. **more_test.md** - Miscellaneous tests (unknown content)
21. **parser_robustness_test.md** - Parser robustness (unknown content)
22. **parsing_comprehensive_test.md** - Comprehensive parsing (unknown content)
23. **slices_test.md** - Slice operations tests (5 tests)
24. **statements_test.md** - Statement parsing tests (15+ tests)
25. **strings_execution_test.md** - String execution tests (unknown content)
26. **structs_test.md** - Struct parsing tests (5 tests)
27. **type_verification_test.md** - Type verification (unknown content)
28. **types_comprehensive_test.md** - Comprehensive type tests (unknown content)
29. **u8_comprehensive_test.md** - U8 type tests (4 tests)
30. **variable_init_test.md** - Variable initialization tests (7 tests)
31. **variable_shadowing_test.md** - Variable shadowing tests (unknown content)

## Target Organization (8 files)

### 1. `test/literals_test.md`
**Purpose**: All literal value tests (integers, booleans, strings, U8)
**Sources**: 
- `expressions_test.md` (literal tests: lines 3-25, 456-469)
- `u8_comprehensive_test.md` (all 4 tests)
- Any literal tests from other files

### 2. `test/operators_test.md`
**Purpose**: All operator tests (arithmetic, comparison, precedence)
**Sources**:
- `binary_expr_test.md` (all 10 tests)
- `arithmetic_ops_test.md` (all 8 tests) 
- `comparisons_test.md` (all 8 tests)
- `TestComparisons_test.md` (all 5 execution tests)
- `expressions_test.md` (operator tests: lines 27-181, 471-477)
- `TestDivisionAndModulo_test.md` (division/modulo tests)

### 3. `test/variables_test.md`
**Purpose**: Variable declarations, initialization, scoping, shadowing
**Sources**:
- `variable_init_test.md` (all 7 tests)
- `variable_shadowing_test.md` (all tests)
- `statements_test.md` (var-decl tests: lines 77-115)
- `locals_test.md` (all tests)
- Any variable-related tests from `extracted_execution_test.md`

### 4. `test/control_flow_test.md`
**Purpose**: If statements, loops, break/continue, return
**Sources**:
- `statements_test.md` (if/return/break/continue: lines 3-76, 131-177)
- `loops_test.md` (all 8 tests)
- Control flow tests from other files

### 5. `test/functions_test.md`
**Purpose**: Function declarations, parameters, calls
**Sources**:
- `functions_test.md` (keep existing 5 tests)
- `TestPhase1Functions_test.md` (all tests)
- `expressions_test.md` (function call tests: lines 183-237)
- Function-related tests from `extracted_execution_test.md`

### 6. `test/structs_test.md`
**Purpose**: Struct definitions, field access, operations
**Sources**:
- `structs_test.md` (keep existing 5 tests)
- `expressions_test.md` (field access if any)
- Struct tests from `extracted_execution_test.md` (lines 7-183)

### 7. `test/pointers_test.md`
**Purpose**: Address-of, dereference, pointer operations
**Sources**:
- `TestPointerOperations_test.md` (all 8 tests)
- `TestAddressOfOperations_test.md` (all 8 tests)
- `TestAdvancedPointerScenarios_test.md` (if any content)
- `expressions_test.md` (pointer tests: lines 335-453)

### 8. `test/slices_test.md`
**Purpose**: Slice declarations, indexing, append operations
**Sources**:
- `slices_test.md` (keep existing 5 tests)
- Slice tests from `extracted_execution_test.md` (lines 245-368)
- `expressions_test.md` (array subscript tests: lines 239-269)

## Migration Process

### Phase 1: Inventory and Analysis
1. Read all remaining unknown files to catalog their content
2. Create detailed mapping of every test case to target file

### Phase 2: Create New Target Files
1. Create each new target file with proper header and organization
2. Start with empty files, add content methodically

### Phase 3: Migrate Tests One by One
For each source file:
1. Copy each test case exactly as written
2. Rename test if needed for clarity (but preserve all content)
3. Add to appropriate target file
4. Preserve all duplicate tests
5. Delete source test to mark it as migrated
6. Verify test still passes

### Phase 4: Quality Assurance
1. Run full test suite after each file migration
2. Ensure no tests are lost (count before/after)

### Phase 5: Cleanup
1. Delete empty test files only after all tests migrated
2. Final test suite run to ensure everything works

## Detailed Migration Steps

### Step 1: Complete Inventory
Before moving any tests, read and catalog the content of these unknown files:
- TestDivisionAndModulo_test.md
- TestFullCapabilitiesDemo_test.md  
- TestPhase1Functions_test.md
- TestTypeASTInCompilation_test.md
- advanced_features_test.md
- compile_error_test.md
- locals_test.md
- more_test.md
- parser_robustness_test.md
- parsing_comprehensive_test.md
- strings_execution_test.md
- type_verification_test.md
- types_comprehensive_test.md
- variable_shadowing_test.md

### Step 2: Create Target Files
Create these 8 files with proper headers:

```markdown
# [Category] Tests

Tests for [description of what this file covers].

## [Section Name]

[Tests organized by sub-functionality]
```

### Step 3: Migration Order
Migrate in this order to minimize dependencies:
1. literals_test.md (no dependencies)
2. operators_test.md (may reference literals)
3. variables_test.md (may reference literals/operators)
4. control_flow_test.md (may reference variables)
5. functions_test.md (may reference all above)
6. structs_test.md (may reference variables/operators)
7. pointers_test.md (may reference variables/operators)
8. slices_test.md (may reference variables/operators)

### Step 4: Test Migration Template
For each test being moved:

1. **Copy exactly**: Preserve test name, zong code, ast/execute blocks
2. **Rename if needed**: Make test names descriptive and consistent
3. **Group logically**: Place related tests together
4. **Add comments**: Brief section headers for organization
5. **Verify**: Ensure test still passes in new location

Example migration:

Original location: expressions_test.md line 27

    ## Test: binary addition
    ```zong-expr
    1 + 2
    ```
    ```ast
    (binary "+" 1 2)
    ```

New location: operators_test.md

    ## Test: addition operator
    ```zong-expr
    1 + 2
    ```
    ```ast
    (binary "+" 1 2)
    ```

### Step 5: Quality Checks
After each file migration:
- [ ] Run `go test -run TestSexyAllTests`
- [ ] Verify tests pass, as before
- [ ] Verify test count matches expected
- [ ] Check no duplicate test names (but duplicate tests are okay)
- [ ] Ensure proper markdown formatting

## Success Criteria
- All existing tests preserved with exact same content
- Tests organized into 8 logical files
- No tests lost - exact same test count
- Full test suite passes
- Improved maintainability and discoverability

## Risk Mitigation
- Migrate one test at a time
- Run tests after each migration
- Never delete source files until target verified

This plan ensures a careful, methodical reorganization that preserves all existing test coverage while dramatically improving organization and maintainability.
