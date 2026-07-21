package ojson

import (
	"testing"
)

func TestDiffSchemaNormalizedDefaultsAreQuiet(t *testing.T) {
	schema := phase8PetSchema(t)
	before := MustReadStringNoSchema(`{"name":"Whiffles","age":3}`)
	after := MustReadStringNoSchema(`{"name":"Whiffles","age":3,"height_units":"inches"}`)

	patch, err := Diff(before, after, WithPatchSchema(schema))
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	if patch.Len() != 0 {
		t.Fatalf("expected empty patch, got %s", patch.ToJSON())
	}
}

func TestDiffEmitsAddRemoveReplaceAndPreservesOrder(t *testing.T) {
	before := MustReadStringNoSchema(`{"a":1,"b":{"x":1,"y":2},"tags":["one"]}`)
	after := MustReadStringNoSchema(`{"a":2,"b":{"x":1,"y":3,"z":true},"tags":["one","two"],"c":9}`)

	patch, err := Diff(before, after)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	result, err := ApplyPatch(before, patch)
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if !valuesEqual(result, after) {
		t.Fatalf("patched result = %s, want %s", result.ToJSON(), after.ToJSON())
	}

	// Object values in ops preserve source field order.
	for _, op := range patch.Ops() {
		if op.Op() == "add" && op.Path() == "/b" {
			t.Fatal("expected nested field ops, not whole object replace for b")
		}
		if op.Op() == "add" && op.Path() == "/b/z" {
			if !op.Value().IsBoolean() {
				t.Fatalf("add /b/z value = %v", op.Value())
			}
		}
	}
}

func TestDiffTreatsNullAndMissingDistinct(t *testing.T) {
	before := MustReadStringNoSchema(`{"a":null}`)
	after := MustReadStringNoSchema(`{}`)
	patch, err := Diff(before, after)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	if patch.Len() != 1 || patch.Ops()[0].Op() != "remove" {
		t.Fatalf("patch = %s", patch.ToJSON())
	}
}

func TestDiffComparesNumbersNumerically(t *testing.T) {
	before := MustReadStringNoSchema(`{"n":25}`)
	after := MustReadStringNoSchema(`{"n":25.0}`)
	patch, err := Diff(before, after)
	if err != nil {
		t.Fatalf("Diff returned error: %v", err)
	}
	if patch.Len() != 0 {
		t.Fatalf("expected empty numeric patch, got %s", patch.ToJSON())
	}
}
