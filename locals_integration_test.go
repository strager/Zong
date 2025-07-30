package main

// Test that variables work in expressions

// Check locals collection

// Compile and execute WASM

// 10 + 20 = 30

// Test nested blocks with variables (WebAssembly has function-level scope)

// Both variables should be available at function level

// Compile and execute WASM - should print the value of y (which was assigned from x)

// y = x = 42

// Test that non-I64 types are ignored (as per the plan)

// Only I64 variable should be collected

// Compile and execute WASM - should print the value of x

// x = 42

// Test complex calculations with multiple variables

// Compile and execute WASM - should calculate 15 * 3 + 5 = 50

// Test variable reassignment

// Compile and execute WASM - should calculate 5 + 10 = 15

// Comprehensive test showing all local variable features working together

// Verify all variables are collected

// Execute and verify the complex calculation

// (8 * 3) + 8 - 3 = 24 + 8 - 3 = 29
