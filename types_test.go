package main

import (
	"testing"
)

func TestTypesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     *TypeNode
		expected bool
	}{
		{
			name:     "same builtin types",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: true,
		},
		{
			name:     "different builtin types",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypeBuiltin, String: "Bool"},
			expected: false,
		},
		{
			name:     "different kinds",
			a:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: false,
		},
		{
			name:     "same pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: true,
		},
		{
			name:     "different pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "Bool"}},
			expected: false,
		},
		{
			name:     "nested pointer types",
			a:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			b:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypesEqual(test.a, test.b)
			if result != test.expected {
				t.Errorf("TypesEqual(%v, %v) = %v, expected %v", test.a, test.b, result, test.expected)
			}
		})
	}
}

func TestIsWASMI64Type(t *testing.T) {
	tests := []struct {
		name     string
		t        *TypeNode
		expected bool
	}{
		{
			name:     "nil type",
			t:        nil,
			expected: false,
		},
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: true,
		},
		{
			name:     "Bool builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Bool"},
			expected: true,
		},
		{
			name:     "unsupported builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "String"},
			expected: false,
		},
		{
			name:     "pointer type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: true,
		},
		{
			name:     "pointer to unsupported type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "String"}},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isWASMI64Type(test.t)
			if result != test.expected {
				t.Errorf("isWASMI64Type(%v) = %v, expected %v", test.t, result, test.expected)
			}
		})
	}
}

func TestGetTypeSize(t *testing.T) {
	tests := []struct {
		name     string
		t        *TypeNode
		expected int
	}{
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: 8,
		},
		{
			name:     "Bool builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Bool"},
			expected: 1,
		},
		{
			name:     "unknown builtin defaults to 8",
			t:        &TypeNode{Kind: TypeBuiltin, String: "UnknownType"},
			expected: 8,
		},
		{
			name:     "pointer type",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: 8,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetTypeSize(test.t)
			if result != test.expected {
				t.Errorf("GetTypeSize(%v) = %v, expected %v", test.t, result, test.expected)
			}
		})
	}
}

func TestTypeToString(t *testing.T) {
	tests := []struct {
		name     string
		t        *TypeNode
		expected string
	}{
		{
			name:     "I64 builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "I64"},
			expected: "I64",
		},
		{
			name:     "Bool builtin",
			t:        &TypeNode{Kind: TypeBuiltin, String: "Bool"},
			expected: "Bool",
		},
		{
			name:     "pointer to I64",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}},
			expected: "I64*",
		},
		{
			name:     "pointer to Bool",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "Bool"}},
			expected: "Bool*",
		},
		{
			name:     "pointer to pointer",
			t:        &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypePointer, Child: &TypeNode{Kind: TypeBuiltin, String: "I64"}}},
			expected: "I64**",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := TypeToString(test.t)
			if result != test.expected {
				t.Errorf("TypeToString(%v) = %q, expected %q", test.t, result, test.expected)
			}
		})
	}
}
