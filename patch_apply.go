package ojson

// ApplyPatch applies a patch to a deep clone of doc and returns the result.
// The original document is never mutated. On failure the clone is discarded.
//
// When a schema is supplied via WithPatchSchema or attached to doc, operations
// are validated during execution and the document is normalized once after all
// operations succeed. Required fields removed by the patch are restored from
// schema defaults during that final normalization when a default exists.
func ApplyPatch(doc JSONValue, patch Patch, opts ...PatchOption) (JSONValue, error) {
	cfg := newPatchConfig(doc, opts...)
	working := cloneJSONValue(doc)
	clearSchema(working)

	for i, op := range patch.ops {
		if err := applyPatchOp(&working, op, i); err != nil {
			return NewVoid(), err
		}
	}

	if cfg.hasSchema && cfg.schema != nil {
		normalized, err := working.ApplySchema(*cfg.schema)
		if err != nil {
			return NewVoid(), wrapSchemaPatchError(err, patch.Len())
		}
		return normalized, nil
	}
	return working, nil
}

// ValidatePatch checks whether a patch can be applied to doc.
// It performs a dry-run ApplyPatch and discards the result.
func ValidatePatch(doc JSONValue, patch Patch, opts ...PatchOption) error {
	_, err := ApplyPatch(doc, patch, opts...)
	return err
}

func wrapSchemaPatchError(err error, opCount int) error {
	if err == nil {
		return nil
	}
	if pe, ok := err.(PatchError); ok {
		return pe
	}
	if oe, ok := err.(OJSONError); ok {
		return PatchError{
			OpIndex: opCount,
			Pointer: "",
			Path:    oe.Path,
			Reason:  "schema normalization failed: " + oe.Reason,
		}
	}
	return PatchError{
		OpIndex: opCount,
		Path:    RootPath(),
		Reason:  "schema normalization failed: " + err.Error(),
	}
}

func applyPatchOp(doc *JSONValue, op PatchOp, index int) error {
	if op.buildErr != nil {
		return patchError(index, op.path, RootPath(), "%s", op.buildErr.Error())
	}
	switch op.op {
	case opAdd:
		return applyAdd(doc, op.path, op.value, index, true)
	case opRemove:
		_, err := applyRemove(doc, op.path, index)
		return err
	case opReplace:
		return applyReplace(doc, op.path, op.value, index)
	case opTest:
		return applyTest(*doc, op.path, op.value, index)
	case opMove:
		return applyMove(doc, op.from, op.path, index)
	case opCopy:
		return applyCopy(doc, op.from, op.path, index)
	default:
		return patchError(index, op.path, RootPath(), "unsupported op %q", op.op)
	}
}

func applyAdd(doc *JSONValue, pointer string, value JSONValue, index int, allowReplaceExisting bool) error {
	if value.IsVoid() {
		return patchError(index, pointer, RootPath(), "add value must not be void")
	}
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return patchError(index, pointer, RootPath(), "%s", err.Error())
	}
	if len(segments) == 0 {
		*doc = cloneJSONValue(value)
		return nil
	}

	target, err := resolvePointerTarget(*doc, pointer, true)
	if err != nil {
		if pe, ok := err.(PatchError); ok {
			pe.OpIndex = index
			return pe
		}
		return err
	}
	if target.isArray {
		if target.isAppend || target.index == target.parent.Len() {
			target.parent.appendRaw(cloneJSONValue(value))
			return nil
		}
		if target.index < 0 || target.index > target.parent.Len() {
			return patchError(index, pointer, target.path, "array index %d out of range", target.index)
		}
		return insertArrayRaw(target.parent, target.index, cloneJSONValue(value))
	}

	if !target.parent.IsObject() {
		return patchError(index, pointer, target.path, "add target parent must be object or array")
	}
	if target.exists && !allowReplaceExisting {
		return patchError(index, pointer, target.path, "target already exists")
	}
	target.parent.setField(target.field, cloneJSONValue(value))
	return nil
}

func applyRemove(doc *JSONValue, pointer string, index int) (JSONValue, error) {
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return NewVoid(), patchError(index, pointer, RootPath(), "%s", err.Error())
	}
	if len(segments) == 0 {
		removed := cloneJSONValue(*doc)
		*doc = NewVoid()
		return removed, nil
	}

	target, err := resolvePointerTarget(*doc, pointer, false)
	if err != nil {
		if pe, ok := err.(PatchError); ok {
			pe.OpIndex = index
			return NewVoid(), pe
		}
		return NewVoid(), err
	}
	if !target.exists {
		return NewVoid(), patchError(index, pointer, target.path, "path does not exist")
	}
	if target.isArray {
		return removeArrayRaw(target.parent, target.index)
	}
	if !target.parent.IsObject() {
		return NewVoid(), patchError(index, pointer, target.path, "remove target parent must be object or array")
	}
	removed := cloneJSONValue(target.value)
	target.parent.removeField(target.field)
	return removed, nil
}

func applyReplace(doc *JSONValue, pointer string, value JSONValue, index int) error {
	if value.IsVoid() {
		return patchError(index, pointer, RootPath(), "replace value must not be void")
	}
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return patchError(index, pointer, RootPath(), "%s", err.Error())
	}
	if len(segments) == 0 {
		*doc = cloneJSONValue(value)
		return nil
	}
	target, err := resolvePointerTarget(*doc, pointer, false)
	if err != nil {
		if pe, ok := err.(PatchError); ok {
			pe.OpIndex = index
			return pe
		}
		return err
	}
	if !target.exists {
		return patchError(index, pointer, target.path, "path does not exist")
	}
	if target.isArray {
		target.parent.node.arrayValue[target.index] = cloneJSONValue(value)
		return nil
	}
	target.parent.setField(target.field, cloneJSONValue(value))
	return nil
}

func applyTest(doc JSONValue, pointer string, value JSONValue, index int) error {
	if value.IsVoid() {
		return patchError(index, pointer, RootPath(), "test value must not be void")
	}
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return patchError(index, pointer, RootPath(), "%s", err.Error())
	}
	if len(segments) == 0 {
		if !valuesEqual(doc, value) {
			return patchError(index, pointer, RootPath(), "test failed")
		}
		return nil
	}
	target, err := resolvePointerTarget(doc, pointer, false)
	if err != nil {
		if pe, ok := err.(PatchError); ok {
			pe.OpIndex = index
			pe.Reason = "test failed: " + pe.Reason
			return pe
		}
		return err
	}
	if !target.exists || !valuesEqual(target.value, value) {
		return patchError(index, pointer, target.path, "test failed")
	}
	return nil
}

func applyMove(doc *JSONValue, from, path string, index int) error {
	if pointerIsPrefix(from, path) {
		return patchError(index, path, pathFromPointerSegments(mustPointerSegments(path)), "move cannot relocate a value into one of its descendants")
	}
	removed, err := applyRemove(doc, from, index)
	if err != nil {
		return err
	}
	if err := applyAdd(doc, path, removed, index, true); err != nil {
		return err
	}
	return nil
}

func applyCopy(doc *JSONValue, from, path string, index int) error {
	segments, err := ParseJSONPointer(from)
	if err != nil {
		return patchError(index, from, RootPath(), "from: %s", err.Error())
	}
	var source JSONValue
	if len(segments) == 0 {
		source = cloneJSONValue(*doc)
	} else {
		target, err := resolvePointerTarget(*doc, from, false)
		if err != nil {
			if pe, ok := err.(PatchError); ok {
				pe.OpIndex = index
				return pe
			}
			return err
		}
		if !target.exists {
			return patchError(index, from, target.path, "from path does not exist")
		}
		source = cloneJSONValue(target.value)
	}
	return applyAdd(doc, path, source, index, true)
}

func insertArrayRaw(array JSONValue, index int, value JSONValue) error {
	if !array.IsArray() {
		return pathError(RootPath(), "not an array")
	}
	if index < 0 || index > len(array.node.arrayValue) {
		return pathError(RootPath().Index(index), "array index %d out of range", index)
	}
	array.node.arrayValue = append(array.node.arrayValue, NewVoid())
	copy(array.node.arrayValue[index+1:], array.node.arrayValue[index:])
	array.node.arrayValue[index] = value
	return nil
}

func removeArrayRaw(array JSONValue, index int) (JSONValue, error) {
	if !array.IsArray() {
		return NewVoid(), pathError(RootPath(), "not an array")
	}
	if index < 0 || index >= len(array.node.arrayValue) {
		return NewVoid(), pathError(RootPath().Index(index), "array index %d out of range", index)
	}
	removed := array.node.arrayValue[index]
	array.node.arrayValue = append(array.node.arrayValue[:index], array.node.arrayValue[index+1:]...)
	return removed, nil
}
