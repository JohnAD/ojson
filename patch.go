package ojson

import "fmt"

// Patch is an ordered RFC6902 patch document.
type Patch struct {
	ops []PatchOp
}

// NewPatch constructs a patch from operations.
func NewPatch(ops ...PatchOp) (Patch, error) {
	copied := make([]PatchOp, 0, len(ops))
	for i, op := range ops {
		if op.buildErr != nil {
			return Patch{}, fmt.Errorf("patch op %d: %w", i, op.buildErr)
		}
		if op.op == "" {
			return Patch{}, fmt.Errorf("patch op %d: empty operation", i)
		}
		copied = append(copied, clonePatchOp(op))
	}
	return Patch{ops: copied}, nil
}

// MustNewPatch constructs a patch and panics on error.
func MustNewPatch(ops ...PatchOp) Patch {
	patch, err := NewPatch(ops...)
	if err != nil {
		panic(err)
	}
	return patch
}

// Len returns the number of operations.
func (p Patch) Len() int {
	return len(p.ops)
}

// Ops returns a copy of the patch operations.
func (p Patch) Ops() []PatchOp {
	out := make([]PatchOp, len(p.ops))
	for i, op := range p.ops {
		out[i] = clonePatchOp(op)
	}
	return out
}

// ToJSON serializes the patch as an RFC6902 JSON array string.
func (p Patch) ToJSON() string {
	return p.ToJSONValue().ToJSON()
}

// ToJSONBytes serializes the patch as RFC6902 JSON bytes.
func (p Patch) ToJSONBytes() []byte {
	return []byte(p.ToJSON())
}

// ToJSONValue returns the patch as an ordered JSON array of operation objects.
func (p Patch) ToJSONValue() JSONValue {
	arr := NewArray()
	for _, op := range p.ops {
		value, err := op.toJSONValue()
		if err != nil {
			continue
		}
		arr.appendRaw(value)
	}
	return arr
}

// ReadPatchJSON parses a strict RFC6902 patch document from a JSON string.
func ReadPatchJSON(text string) (Patch, error) {
	return ReadPatchBytes([]byte(text))
}

// ReadPatchBytes parses a strict RFC6902 patch document from JSON bytes.
func ReadPatchBytes(data []byte) (Patch, error) {
	value, err := ReadBytesNoSchema(data)
	if err != nil {
		return Patch{}, err
	}
	return PatchFromJSONValue(value)
}

// PatchFromJSONValue parses a patch from an ordered JSON array value.
func PatchFromJSONValue(value JSONValue) (Patch, error) {
	if !value.IsArray() {
		return Patch{}, pathError(RootPath(), "patch document must be an array")
	}
	ops := make([]PatchOp, 0, value.Len())
	for i, item := range value.node.arrayValue {
		if item.IsVoid() {
			return Patch{}, patchError(i, "", RootPath().Index(i), "patch operation must not be void")
		}
		op, err := patchOpFromJSONValue(item, i)
		if err != nil {
			return Patch{}, err
		}
		ops = append(ops, op)
	}
	return Patch{ops: ops}, nil
}

func clonePatchOp(op PatchOp) PatchOp {
	cloned := op
	if op.hasValue {
		cloned.value = cloneJSONValue(op.value)
	}
	return cloned
}

// PatchError reports a patch validation or application failure.
type PatchError struct {
	OpIndex int
	Pointer string
	Path    Path
	Reason  string
}

func (e PatchError) Error() string {
	prefix := fmt.Sprintf("patch op %d", e.OpIndex)
	if e.Pointer != "" {
		prefix += fmt.Sprintf(" at %s", e.Pointer)
	}
	pathText := e.Path.visible()
	if pathText != "$" && pathText != "" {
		return fmt.Sprintf("%s (%s): %s", prefix, pathText, e.Reason)
	}
	return fmt.Sprintf("%s: %s", prefix, e.Reason)
}

func patchError(opIndex int, pointer string, path Path, format string, args ...any) error {
	return PatchError{
		OpIndex: opIndex,
		Pointer: pointer,
		Path:    path,
		Reason:  fmt.Sprintf(format, args...),
	}
}
