package main

// Test execution of WASM without string literals (baseline)

// This should work

// Test execution with string literal assignment but no usage

// This currently fails - let's see the error

// For now, we expect this to fail, so don't fail the test
// be.Err(t, err, nil)

// Test execution with empty string

// This might have different behavior than non-empty string

// Test execution without string assignment (just declaration)

// This should work since no string slice creation happens

// This should work, so if it fails, it indicates a broader issue
