package main

import (
	"testing"

	"github.com/nalgeon/be"
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
			be.Equal(t, test.expected, result)
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
			be.Equal(t, test.expected, result)
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
			be.Equal(t, test.expected, result)
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
			be.Equal(t, test.expected, result)
		})
	}
}
