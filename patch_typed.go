package ojson

import "reflect"

// DiffStructs diffs two typed struct values after converting them to JSONValue.
// Optional WithPatchSchema normalizes both sides before comparison.
func DiffStructs[T any](before, after T, opts ...PatchOption) (Patch, error) {
	beforeValue, err := NewObjectFromStructTry(before)
	if err != nil {
		return Patch{}, err
	}
	afterValue, err := NewObjectFromStructTry(after)
	if err != nil {
		return Patch{}, err
	}
	return Diff(beforeValue, afterValue, opts...)
}

// ApplyPatchToStruct applies a patch to a typed struct and returns a changed copy.
// The input value is never mutated.
func ApplyPatchToStruct[T any](doc T, patch Patch, opts ...PatchOption) (T, error) {
	var zero T
	beforeValue, err := NewObjectFromStructTry(doc)
	if err != nil {
		return zero, err
	}
	afterValue, err := ApplyPatch(beforeValue, patch, opts...)
	if err != nil {
		return zero, err
	}

	resultPtr := reflect.New(reflect.TypeOf(zero))
	if err := afterValue.ToStructTry(resultPtr.Interface()); err != nil {
		return zero, err
	}
	return resultPtr.Elem().Interface().(T), nil
}

// ValidatePatchForStruct validates a patch against a typed struct document.
func ValidatePatchForStruct[T any](doc T, patch Patch, opts ...PatchOption) error {
	_, err := ApplyPatchToStruct(doc, patch, opts...)
	return err
}
