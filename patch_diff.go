package ojson

import (
	"sort"
	"strconv"
)

// Diff produces an RFC6902 patch that transforms before into after.
//
// When a schema is available via WithPatchSchema or an attached schema on
// before/after, both documents are normalized first so default-equivalent
// documents produce an empty patch. Diff emits only add, remove, and replace.
func Diff(before, after JSONValue, opts ...PatchOption) (Patch, error) {
	cfg := patchConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt.applyPatchOption(&cfg)
		}
	}
	if !cfg.hasSchema {
		if attached := before.Schema(); attached != nil {
			cfg.schema = attached
			cfg.hasSchema = true
		} else if attached := after.Schema(); attached != nil {
			cfg.schema = attached
			cfg.hasSchema = true
		}
	}

	left := cloneJSONValue(before)
	right := cloneJSONValue(after)
	if cfg.hasSchema && cfg.schema != nil {
		var err error
		left, err = before.ApplySchema(*cfg.schema)
		if err != nil {
			return Patch{}, err
		}
		right, err = after.ApplySchema(*cfg.schema)
		if err != nil {
			return Patch{}, err
		}
		left = left.WithoutSchema()
		right = right.WithoutSchema()
	}

	ops := make([]PatchOp, 0)
	ops = appendDiffOps(ops, "", left, right)
	return NewPatch(ops...)
}

func appendDiffOps(ops []PatchOp, pointer string, before, after JSONValue) []PatchOp {
	if valuesEqual(before, after) {
		return ops
	}
	if before.IsVoid() {
		return append(ops, PatchAdd(pointer, after))
	}
	if after.IsVoid() {
		return append(ops, PatchRemove(pointer))
	}
	if before.Kind() != after.Kind() {
		return append(ops, PatchReplace(pointer, after))
	}

	switch before.Kind() {
	case KindObject:
		return appendObjectDiffOps(ops, pointer, before, after)
	case KindArray:
		return appendArrayDiffOps(ops, pointer, before, after)
	default:
		return append(ops, PatchReplace(pointer, after))
	}
}

func appendObjectDiffOps(ops []PatchOp, pointer string, before, after JSONValue) []PatchOp {
	beforeFields := nonVoidObjectFields(before)
	afterFields := nonVoidObjectFields(after)
	beforeByKey := make(map[string]JSONValue, len(beforeFields))
	for _, field := range beforeFields {
		beforeByKey[field.Key] = field.Value
	}

	seen := map[string]bool{}
	for _, field := range afterFields {
		seen[field.Key] = true
		childPointer := joinPointer(pointer, field.Key)
		beforeValue, ok := beforeByKey[field.Key]
		if !ok {
			ops = append(ops, PatchAdd(childPointer, field.Value))
			continue
		}
		ops = appendDiffOps(ops, childPointer, beforeValue, field.Value)
	}
	removedKeys := make([]string, 0)
	for _, field := range beforeFields {
		if seen[field.Key] {
			continue
		}
		removedKeys = append(removedKeys, field.Key)
	}
	sort.Strings(removedKeys)
	for _, key := range removedKeys {
		ops = append(ops, PatchRemove(joinPointer(pointer, key)))
	}
	return ops
}

func appendArrayDiffOps(ops []PatchOp, pointer string, before, after JSONValue) []PatchOp {
	beforeLen := before.Len()
	afterLen := after.Len()
	shared := beforeLen
	if afterLen < shared {
		shared = afterLen
	}
	for i := 0; i < shared; i++ {
		childPointer := joinPointer(pointer, strconv.Itoa(i))
		ops = appendDiffOps(ops, childPointer, before.node.arrayValue[i], after.node.arrayValue[i])
	}
	for i := beforeLen - 1; i >= shared; i-- {
		ops = append(ops, PatchRemove(joinPointer(pointer, strconv.Itoa(i))))
	}
	for i := shared; i < afterLen; i++ {
		ops = append(ops, PatchAdd(joinPointer(pointer, strconv.Itoa(i)), after.node.arrayValue[i]))
	}
	return ops
}

func joinPointer(base, segment string) string {
	return base + "/" + escapeJSONPointerSegment(segment)
}
