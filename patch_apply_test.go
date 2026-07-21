package ojson

import (
	"strings"
	"testing"
)

func TestReadPatchRoundTrip(t *testing.T) {
	patch, err := NewPatch(
		PatchTest("/name", NewString("Whiffles")),
		PatchReplace("/age", NewNumberFromInt(4)),
		PatchAdd("/tags/-", NewString("fast")),
		PatchCopy("/name", "/nickname"),
		PatchMove("/nickname", "/alias"),
		PatchRemove("/alias"),
	)
	if err != nil {
		t.Fatalf("NewPatch returned error: %v", err)
	}

	parsed, err := ReadPatchJSON(patch.ToJSON())
	if err != nil {
		t.Fatalf("ReadPatchJSON returned error: %v", err)
	}
	if parsed.Len() != 6 {
		t.Fatalf("parsed.Len() = %d", parsed.Len())
	}
	if got := parsed.ToJSON(); got != patch.ToJSON() {
		t.Fatalf("round-trip JSON mismatch:\n got %s\nwant %s", got, patch.ToJSON())
	}
}

func TestApplyPatchAllOpsAndAtomicity(t *testing.T) {
	original := MustReadStringNoSchema(`{"name":"Whiffles","tags":["a"],"keep":1}`)
	before := original.ToJSON()

	patch := MustNewPatch(
		PatchReplace("/name", NewString("Whiff")),
		PatchAdd("/tags/-", NewString("b")),
		PatchCopy("/keep", "/copied"),
		PatchTest("/copied", NewNumberFromInt(1)),
		PatchMove("/copied", "/moved"),
		PatchRemove("/moved"),
	)

	result, err := ApplyPatch(original, patch)
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if original.ToJSON() != before {
		t.Fatalf("original mutated: %s", original.ToJSON())
	}
	want := `{"name":"Whiff","tags":["a","b"],"keep":1}`
	if got := result.ToJSON(); got != want {
		t.Fatalf("result = %s, want %s", got, want)
	}

	bad := MustNewPatch(
		PatchReplace("/name", NewString("X")),
		PatchTest("/missing", NewString("nope")),
	)
	if _, err := ApplyPatch(original, bad); err == nil {
		t.Fatal("expected test failure")
	}
	if original.ToJSON() != before {
		t.Fatal("original mutated after failed patch")
	}
}

func TestApplyPatchSchemaDefaultRestoreAndRequired(t *testing.T) {
	schema := phase8PetSchema(t)
	doc, err := ReadStringWithSchema(`{"name":"Whiffles","age":3}`, schema)
	if err != nil {
		t.Fatalf("ReadStringWithSchema returned error: %v", err)
	}

	// Remove defaulted field; final normalization restores it.
	patch := MustNewPatch(PatchRemove("/height_units"))
	result, err := ApplyPatch(doc, patch, WithPatchSchema(schema))
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if got := result.Get("height_units").String(); got != "inches" {
		t.Fatalf("height_units = %q, want inches", got)
	}

	// Remove required field without default fails at final normalization.
	bad := MustNewPatch(PatchRemove("/name"))
	if err := ValidatePatch(doc, bad, WithPatchSchema(schema)); err == nil {
		t.Fatal("expected required field error")
	} else if !strings.Contains(err.Error(), "required field is missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyPatchRejectsVoidValue(t *testing.T) {
	if _, err := NewPatch(PatchAdd("/x", NewVoid())); err == nil {
		t.Fatal("expected void value rejection")
	}
}

func TestApplyPatchPointerEscapingAndRoot(t *testing.T) {
	doc := MustReadStringNoSchema(`{"a/b":{"c~d":1}}`)
	patch := MustNewPatch(PatchReplace("/a~1b/c~0d", NewNumberFromInt(2)))
	result, err := ApplyPatch(doc, patch)
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if got := result.Get("a/b").Get("c~d").String(); got != "2" {
		t.Fatalf("escaped path value = %q", got)
	}

	rootPatch := MustNewPatch(PatchReplace("", MustReadStringNoSchema(`{"ok":true}`)))
	rootResult, err := ApplyPatch(doc, rootPatch)
	if err != nil {
		t.Fatalf("root replace returned error: %v", err)
	}
	if got := rootResult.ToJSON(); got != `{"ok":true}` {
		t.Fatalf("root result = %s", got)
	}
}

func TestReadPatchRejectsUnknownFields(t *testing.T) {
	_, err := ReadPatchJSON(`[{"op":"add","path":"/x","value":1,"extra":true}]`)
	if err == nil || !strings.Contains(err.Error(), "unsupported operation field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPatchSchemaPrecedenceUsesOptionOverAttached(t *testing.T) {
	schema := phase8PetSchema(t)
	doc, err := ReadStringWithSchema(`{"name":"Whiffles"}`, schema)
	if err != nil {
		t.Fatalf("ReadStringWithSchema returned error: %v", err)
	}
	other, err := CompileSchemaJSON(`{
		"kind": "object",
		"children": [
			{"name": "name", "kind": "string", "required": true},
			{"name": "label", "kind": "string", "default": "pet"}
		]
	}`)
	if err != nil {
		t.Fatalf("CompileSchemaJSON returned error: %v", err)
	}
	result, err := ApplyPatch(doc, MustNewPatch(), WithPatchSchema(other))
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if got := result.Get("label").String(); got != "pet" {
		t.Fatalf("label = %q, want pet from explicit schema", got)
	}
}
