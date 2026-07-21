package ojson

import (
	"fmt"
	"strconv"
	"strings"
)

// JSONPointer is an RFC6901 JSON Pointer string.
type JSONPointer string

// ParseJSONPointer parses an RFC6901 pointer into unescaped segments.
// The empty string refers to the document root and yields zero segments.
func ParseJSONPointer(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("JSON Pointer must be empty or start with '/'")
	}
	raw := strings.Split(pointer[1:], "/")
	segments := make([]string, len(raw))
	for i, part := range raw {
		segments[i] = unescapeJSONPointerSegment(part)
	}
	return segments, nil
}

// FormatJSONPointer builds an RFC6901 pointer from unescaped segments.
func FormatJSONPointer(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteByte('/')
		builder.WriteString(escapeJSONPointerSegment(segment))
	}
	return builder.String()
}

// Pointer joins path segments into an RFC6901 pointer.
// Segments may be strings or non-negative ints (array indexes).
func Pointer(parts ...any) (string, error) {
	segments := make([]string, 0, len(parts))
	for i, part := range parts {
		switch typed := part.(type) {
		case string:
			segments = append(segments, typed)
		case int:
			if typed < 0 {
				return "", fmt.Errorf("pointer part %d: array index must be non-negative", i)
			}
			segments = append(segments, strconv.Itoa(typed))
		case JSONPointer:
			parsed, err := ParseJSONPointer(string(typed))
			if err != nil {
				return "", err
			}
			segments = append(segments, parsed...)
		default:
			return "", fmt.Errorf("pointer part %d: unsupported type %T", i, part)
		}
	}
	return FormatJSONPointer(segments...), nil
}

func escapeJSONPointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~", "~0")
	segment = strings.ReplaceAll(segment, "/", "~1")
	return segment
}

func unescapeJSONPointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~1", "/")
	segment = strings.ReplaceAll(segment, "~0", "~")
	return segment
}

func pathFromPointerSegments(segments []string) Path {
	path := RootPath()
	for _, segment := range segments {
		if index, ok := parseArrayIndex(segment); ok {
			path = path.Index(index)
			continue
		}
		path = path.Field(segment)
	}
	return path
}

func parseArrayIndex(segment string) (int, bool) {
	if segment == "-" {
		return -1, false
	}
	if segment == "" {
		return 0, false
	}
	for _, r := range segment {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	// Leading zeros are invalid for array indexes except "0" itself.
	if len(segment) > 1 && segment[0] == '0' {
		return 0, false
	}
	index, err := strconv.Atoi(segment)
	if err != nil {
		return 0, false
	}
	return index, true
}

type pointerTarget struct {
	parent   JSONValue
	path     Path
	pointer  string
	field    string
	index    int
	isArray  bool
	isAppend bool
	exists   bool
	value    JSONValue
}

func resolvePointerTarget(doc JSONValue, pointer string, allowAppend bool) (pointerTarget, error) {
	segments, err := ParseJSONPointer(pointer)
	if err != nil {
		return pointerTarget{}, patchError(0, pointer, RootPath(), "%s", err.Error())
	}
	if len(segments) == 0 {
		return pointerTarget{
			parent:  NewVoid(),
			path:    RootPath(),
			pointer: pointer,
			exists:  !doc.IsVoid(),
			value:   doc,
		}, nil
	}

	current := doc
	path := RootPath()
	for i, segment := range segments {
		isLast := i == len(segments)-1
		if current.IsObject() {
			nextPath := path.Field(segment)
			if isLast {
				value, exists := objectFieldValue(current, segment)
				return pointerTarget{
					parent:  current,
					path:    nextPath,
					pointer: pointer,
					field:   segment,
					exists:  exists,
					value:   value,
				}, nil
			}
			child, exists := objectFieldValue(current, segment)
			if !exists {
				return pointerTarget{}, patchError(0, FormatJSONPointer(segments[:i+1]...), nextPath, "path does not exist")
			}
			current = child
			path = nextPath
			continue
		}
		if current.IsArray() {
			if segment == "-" {
				if !isLast || !allowAppend {
					return pointerTarget{}, patchError(0, FormatJSONPointer(segments[:i+1]...), path, "array append '-' is only valid as the final add token")
				}
				return pointerTarget{
					parent:   current,
					path:     path.Index(current.Len()),
					pointer:  pointer,
					index:    current.Len(),
					isArray:  true,
					isAppend: true,
					exists:   false,
				}, nil
			}
			index, ok := parseArrayIndex(segment)
			if !ok {
				return pointerTarget{}, patchError(0, FormatJSONPointer(segments[:i+1]...), path, "invalid array index %q", segment)
			}
			nextPath := path.Index(index)
			if isLast {
				if index > current.Len() || (!allowAppend && index >= current.Len()) {
					return pointerTarget{}, patchError(0, pointer, nextPath, "array index %d out of range", index)
				}
				exists := index < current.Len()
				var value JSONValue
				if exists {
					value = current.node.arrayValue[index]
				}
				return pointerTarget{
					parent:  current,
					path:    nextPath,
					pointer: pointer,
					index:   index,
					isArray: true,
					exists:  exists,
					value:   value,
				}, nil
			}
			if index < 0 || index >= current.Len() {
				return pointerTarget{}, patchError(0, FormatJSONPointer(segments[:i+1]...), nextPath, "array index %d out of range", index)
			}
			current = current.node.arrayValue[index]
			path = nextPath
			continue
		}
		return pointerTarget{}, patchError(0, FormatJSONPointer(segments[:i]...), path, "cannot traverse %s with pointer segment %q", current.Kind(), segment)
	}
	return pointerTarget{}, patchError(0, pointer, RootPath(), "invalid pointer")
}

func objectFieldValue(object JSONValue, key string) (JSONValue, bool) {
	if !object.IsObject() {
		return NewVoid(), false
	}
	for _, field := range object.node.objectValue {
		if field.Key == key {
			if field.Value.IsVoid() {
				return NewVoid(), false
			}
			return field.Value, true
		}
	}
	return NewVoid(), false
}

func pointerIsPrefix(prefix, full string) bool {
	if prefix == "" {
		return full != ""
	}
	if prefix == full {
		return false
	}
	return strings.HasPrefix(full, prefix+"/")
}
