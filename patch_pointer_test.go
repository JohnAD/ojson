package ojson

import (
	"strings"
	"testing"
)

func TestParseAndFormatJSONPointer(t *testing.T) {
	segments, err := ParseJSONPointer("/a~1b/c~0d")
	if err != nil {
		t.Fatalf("ParseJSONPointer returned error: %v", err)
	}
	if len(segments) != 2 || segments[0] != "a/b" || segments[1] != "c~d" {
		t.Fatalf("segments = %#v", segments)
	}
	if got := FormatJSONPointer(segments...); got != "/a~1b/c~0d" {
		t.Fatalf("FormatJSONPointer = %q", got)
	}

	root, err := ParseJSONPointer("")
	if err != nil || len(root) != 0 {
		t.Fatalf("root pointer = %#v err=%v", root, err)
	}
	if _, err := ParseJSONPointer("a/b"); err == nil {
		t.Fatal("expected invalid pointer error")
	}
}

func TestPointerBuilder(t *testing.T) {
	got, err := Pointer("pet", "contact/email", 2)
	if err != nil {
		t.Fatalf("Pointer returned error: %v", err)
	}
	if got != "/pet/contact~1email/2" {
		t.Fatalf("Pointer = %q", got)
	}
}

func TestResolvePointerTargetArrayAppend(t *testing.T) {
	doc := MustReadStringNoSchema(`{"tags":["a"]}`)
	target, err := resolvePointerTarget(doc, "/tags/-", true)
	if err != nil {
		t.Fatalf("resolvePointerTarget returned error: %v", err)
	}
	if !target.isAppend || target.index != 1 {
		t.Fatalf("target = %#v", target)
	}
}

func MustReadStringNoSchema(text string) JSONValue {
	value, err := ReadStringNoSchema(text)
	if err != nil {
		panic(err)
	}
	return value
}

func TestResolvePointerMissingParent(t *testing.T) {
	doc := MustReadStringNoSchema(`{}`)
	_, err := resolvePointerTarget(doc, "/missing/child", true)
	if err == nil {
		t.Fatal("expected missing parent error")
	}
	if !strings.Contains(err.Error(), "path does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}
