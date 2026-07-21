package ojson

import "fmt"

// TypedPath is a compile-time-associated JSON Pointer for document type Doc
// and value type T. Generators emit these for struct fields.
type TypedPath[Doc any, T any] struct {
	pointer string
}

// NewTypedPath constructs a typed path for pointer.
func NewTypedPath[Doc any, T any](pointer string) TypedPath[Doc, T] {
	return TypedPath[Doc, T]{pointer: pointer}
}

// Pointer returns the RFC6901 pointer string.
func (p TypedPath[Doc, T]) Pointer() string {
	return p.pointer
}

// Index returns a typed path for an array element at index.
func Index[Doc any, T any](path TypedPath[Doc, []T], index int) (TypedPath[Doc, T], error) {
	if index < 0 {
		return TypedPath[Doc, T]{}, fmt.Errorf("array index must be non-negative")
	}
	return TypedPath[Doc, T]{pointer: joinPointer(path.pointer, fmt.Sprintf("%d", index))}, nil
}

// Append returns a typed path that targets array append ("-").
func Append[Doc any, T any](path TypedPath[Doc, []T]) TypedPath[Doc, T] {
	return TypedPath[Doc, T]{pointer: joinPointer(path.pointer, "-")}
}

// Child appends an object field segment to a typed path.
func Child[Doc any, Parent, ChildT any](path TypedPath[Doc, Parent], field string) TypedPath[Doc, ChildT] {
	return TypedPath[Doc, ChildT]{pointer: joinPointer(path.pointer, field)}
}

// ReplaceAt constructs a replace operation for a typed path value.
func ReplaceAt[Doc any, T any](path TypedPath[Doc, T], value T) (PatchOp, error) {
	jsonValue, err := nativeToJSON(value, RootPath())
	if err != nil {
		return PatchOp{}, err
	}
	return PatchReplace(path.pointer, jsonValue), nil
}

// AddAt constructs an add operation for a typed path value.
func AddAt[Doc any, T any](path TypedPath[Doc, T], value T) (PatchOp, error) {
	jsonValue, err := nativeToJSON(value, RootPath())
	if err != nil {
		return PatchOp{}, err
	}
	return PatchAdd(path.pointer, jsonValue), nil
}

// TestAt constructs a test operation for a typed path value.
func TestAt[Doc any, T any](path TypedPath[Doc, T], value T) (PatchOp, error) {
	jsonValue, err := nativeToJSON(value, RootPath())
	if err != nil {
		return PatchOp{}, err
	}
	return PatchTest(path.pointer, jsonValue), nil
}

// RemoveAt constructs a remove operation for a typed path.
func RemoveAt[Doc any, T any](path TypedPath[Doc, T]) PatchOp {
	return PatchRemove(path.pointer)
}

// MoveAt constructs a move operation between typed paths of the same value type.
func MoveAt[Doc any, T any](from, to TypedPath[Doc, T]) PatchOp {
	return PatchMove(from.pointer, to.pointer)
}

// CopyAt constructs a copy operation between typed paths of the same value type.
func CopyAt[Doc any, T any](from, to TypedPath[Doc, T]) PatchOp {
	return PatchCopy(from.pointer, to.pointer)
}
