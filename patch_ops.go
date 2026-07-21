package ojson

import "fmt"

const (
	opAdd     = "add"
	opRemove  = "remove"
	opReplace = "replace"
	opTest    = "test"
	opMove    = "move"
	opCopy    = "copy"
)

// PatchOp is one RFC6902 operation retained as structured data and ordered JSON.
type PatchOp struct {
	op       string
	path     string
	from     string
	value    JSONValue
	hasValue bool
	hasFrom  bool
	buildErr error
}

// Op returns the RFC6902 operation name.
func (op PatchOp) Op() string { return op.op }

// Path returns the target JSON Pointer.
func (op PatchOp) Path() string { return op.path }

// From returns the source JSON Pointer for move/copy.
func (op PatchOp) From() string { return op.from }

// Value returns the operation value for add/replace/test.
func (op PatchOp) Value() JSONValue {
	if !op.hasValue {
		return NewVoid()
	}
	return cloneJSONValue(op.value)
}

// PatchAdd constructs an add operation.
func PatchAdd(path string, value JSONValue) PatchOp {
	return newValueOp(opAdd, path, value)
}

// PatchRemove constructs a remove operation.
func PatchRemove(path string) PatchOp {
	return newPathOp(opRemove, path)
}

// PatchReplace constructs a replace operation.
func PatchReplace(path string, value JSONValue) PatchOp {
	return newValueOp(opReplace, path, value)
}

// PatchTest constructs a test operation.
func PatchTest(path string, value JSONValue) PatchOp {
	return newValueOp(opTest, path, value)
}

// PatchMove constructs a move operation.
func PatchMove(from, path string) PatchOp {
	return newFromOp(opMove, from, path)
}

// PatchCopy constructs a copy operation.
func PatchCopy(from, path string) PatchOp {
	return newFromOp(opCopy, from, path)
}

func newValueOp(opName, path string, value JSONValue) PatchOp {
	op := PatchOp{op: opName, path: path, hasValue: true, value: cloneJSONValue(value)}
	if err := validatePointerString(path); err != nil {
		op.buildErr = err
		return op
	}
	if value.IsVoid() {
		op.buildErr = fmt.Errorf("%s value must not be void", opName)
	}
	return op
}

func newPathOp(opName, path string) PatchOp {
	op := PatchOp{op: opName, path: path}
	if err := validatePointerString(path); err != nil {
		op.buildErr = err
	}
	return op
}

func newFromOp(opName, from, path string) PatchOp {
	op := PatchOp{op: opName, path: path, from: from, hasFrom: true}
	if err := validatePointerString(path); err != nil {
		op.buildErr = err
		return op
	}
	if err := validatePointerString(from); err != nil {
		op.buildErr = fmt.Errorf("from: %w", err)
	}
	return op
}

func validatePointerString(pointer string) error {
	_, err := ParseJSONPointer(pointer)
	if err != nil {
		return err
	}
	return nil
}

func (op PatchOp) toJSONValue() (JSONValue, error) {
	if op.buildErr != nil {
		return NewVoid(), op.buildErr
	}
	obj := NewObject()
	obj.setField("op", NewString(op.op))
	obj.setField("path", NewString(op.path))
	switch op.op {
	case opAdd, opReplace, opTest:
		if !op.hasValue || op.value.IsVoid() {
			return NewVoid(), fmt.Errorf("%s requires a value", op.op)
		}
		obj.setField("value", cloneJSONValue(op.value))
	case opMove, opCopy:
		if !op.hasFrom {
			return NewVoid(), fmt.Errorf("%s requires from", op.op)
		}
		obj.setField("from", NewString(op.from))
	case opRemove:
		// path only
	default:
		return NewVoid(), fmt.Errorf("unsupported op %q", op.op)
	}
	return obj, nil
}

func patchOpFromJSONValue(value JSONValue, index int) (PatchOp, error) {
	if !value.IsObject() {
		return PatchOp{}, patchError(index, "", RootPath(), "patch operation must be an object")
	}
	seen := map[string]bool{}
	for _, field := range value.node.objectValue {
		if field.Value.IsVoid() {
			continue
		}
		switch field.Key {
		case "op", "path", "from", "value":
			if seen[field.Key] {
				return PatchOp{}, patchError(index, "", RootPath(), "duplicate operation field %q", field.Key)
			}
			seen[field.Key] = true
		default:
			return PatchOp{}, patchError(index, "", RootPath(), "unsupported operation field %q", field.Key)
		}
	}

	opValue, ok := objectFieldValue(value, "op")
	if !ok || !opValue.IsString() {
		return PatchOp{}, patchError(index, "", RootPath(), "operation missing string op")
	}
	pathValue, ok := objectFieldValue(value, "path")
	if !ok || !pathValue.IsString() {
		return PatchOp{}, patchError(index, "", RootPath(), "operation missing string path")
	}
	op := PatchOp{
		op:   opValue.String(),
		path: pathValue.String(),
	}
	if err := validatePointerString(op.path); err != nil {
		return PatchOp{}, patchError(index, op.path, RootPath(), "%s", err.Error())
	}

	switch op.op {
	case opAdd, opReplace, opTest:
		valueField, ok := objectFieldValue(value, "value")
		if !ok {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "operation missing value")
		}
		if valueField.IsVoid() {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "operation value must not be void")
		}
		if _, hasFrom := objectFieldValue(value, "from"); hasFrom {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "%s must not include from", op.op)
		}
		op.hasValue = true
		op.value = cloneJSONValue(valueField)
	case opRemove:
		if _, hasValue := objectFieldValue(value, "value"); hasValue {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "remove must not include value")
		}
		if _, hasFrom := objectFieldValue(value, "from"); hasFrom {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "remove must not include from")
		}
	case opMove, opCopy:
		fromValue, ok := objectFieldValue(value, "from")
		if !ok || !fromValue.IsString() {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "%s missing string from", op.op)
		}
		if err := validatePointerString(fromValue.String()); err != nil {
			return PatchOp{}, patchError(index, fromValue.String(), RootPath(), "from: %s", err.Error())
		}
		if _, hasValue := objectFieldValue(value, "value"); hasValue {
			return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "%s must not include value", op.op)
		}
		op.hasFrom = true
		op.from = fromValue.String()
	default:
		return PatchOp{}, patchError(index, op.path, pathFromPointerSegments(mustPointerSegments(op.path)), "unsupported op %q", op.op)
	}
	return op, nil
}

func mustPointerSegments(pointer string) []string {
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return nil
	}
	return segments
}
