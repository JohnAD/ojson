package ojson

// valuesEqual reports whether two JSON values are equal for patch test/diff.
// Numbers compare by numeric value; Null and Void are distinct; object field
// order does not affect equality.
func valuesEqual(a, b JSONValue) bool {
	if a.Kind() != b.Kind() {
		return false
	}
	switch a.Kind() {
	case KindVoid, KindNull:
		return true
	case KindBoolean:
		return a.node.boolValue == b.node.boolValue
	case KindString:
		return a.node.stringValue == b.node.stringValue
	case KindNumber:
		left, errLeft := parseJSONNumberRat(a.node.stringValue)
		right, errRight := parseJSONNumberRat(b.node.stringValue)
		if errLeft != nil || errRight != nil {
			return a.node.stringValue == b.node.stringValue
		}
		return left.Cmp(right) == 0
	case KindArray:
		if len(a.node.arrayValue) != len(b.node.arrayValue) {
			return false
		}
		for i := range a.node.arrayValue {
			if !valuesEqual(a.node.arrayValue[i], b.node.arrayValue[i]) {
				return false
			}
		}
		return true
	case KindObject:
		leftFields := nonVoidObjectFields(a)
		rightFields := nonVoidObjectFields(b)
		if len(leftFields) != len(rightFields) {
			return false
		}
		rightByKey := make(map[string]JSONValue, len(rightFields))
		for _, field := range rightFields {
			rightByKey[field.Key] = field.Value
		}
		for _, field := range leftFields {
			other, ok := rightByKey[field.Key]
			if !ok || !valuesEqual(field.Value, other) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func nonVoidObjectFields(value JSONValue) []JSONKeyValue {
	if !value.IsObject() {
		return nil
	}
	fields := make([]JSONKeyValue, 0, len(value.node.objectValue))
	for _, field := range value.node.objectValue {
		if field.Value.IsVoid() {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}
