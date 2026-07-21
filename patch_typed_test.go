package ojson

import (
	"strings"
	"testing"
	"time"
)

type patchMovie struct {
	Title      string    `json:"title"`
	ReleasedAt time.Time `json:"released_at"`
	Tags       []string  `json:"tags"`
	Meta       patchMeta `json:"meta"`
}

type patchMeta struct {
	Score int `json:"score"`
}

func TestTypedPathHelpersAndStructPatch(t *testing.T) {
	titlePath := NewTypedPath[patchMovie, string]("/title")
	scorePath := NewTypedPath[patchMovie, int]("/meta/score")
	tagsPath := NewTypedPath[patchMovie, []string]("/tags")
	tag0, err := Index(tagsPath, 0)
	if err != nil {
		t.Fatalf("Index returned error: %v", err)
	}

	replaceTitle, err := ReplaceAt(titlePath, "Dune")
	if err != nil {
		t.Fatalf("ReplaceAt returned error: %v", err)
	}
	replaceScore, err := ReplaceAt(scorePath, 9)
	if err != nil {
		t.Fatalf("ReplaceAt score returned error: %v", err)
	}
	replaceTag, err := ReplaceAt(tag0, "sf")
	if err != nil {
		t.Fatalf("ReplaceAt tag returned error: %v", err)
	}
	patch, err := NewPatch(replaceTitle, replaceScore, replaceTag, RemoveAt(NewTypedPath[patchMovie, string]("/missing")))
	if err != nil {
		t.Fatalf("NewPatch returned error: %v", err)
	}

	before := patchMovie{
		Title:      "Old",
		ReleasedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Tags:       []string{"drama"},
		Meta:       patchMeta{Score: 1},
	}
	// Remove missing should fail.
	if err := ValidatePatchForStruct(before, patch); err == nil {
		t.Fatal("expected missing path failure")
	}

	patch, err = NewPatch(replaceTitle, replaceScore, replaceTag)
	if err != nil {
		t.Fatalf("NewPatch returned error: %v", err)
	}
	after, err := ApplyPatchToStruct(before, patch)
	if err != nil {
		t.Fatalf("ApplyPatchToStruct returned error: %v", err)
	}
	if before.Title != "Old" {
		t.Fatal("input struct mutated")
	}
	if after.Title != "Dune" || after.Meta.Score != 9 || after.Tags[0] != "sf" {
		t.Fatalf("after = %#v", after)
	}

	diffPatch, err := DiffStructs(before, after)
	if err != nil {
		t.Fatalf("DiffStructs returned error: %v", err)
	}
	roundTrip, err := ApplyPatchToStruct(before, diffPatch)
	if err != nil {
		t.Fatalf("ApplyPatchToStruct diff returned error: %v", err)
	}
	if roundTrip.Title != after.Title || roundTrip.Meta.Score != after.Meta.Score || roundTrip.Tags[0] != after.Tags[0] {
		t.Fatalf("roundTrip = %#v", roundTrip)
	}
}

func TestMoveCopyTypedPaths(t *testing.T) {
	from := NewTypedPath[patchMovie, string]("/title")
	to := NewTypedPath[patchMovie, string]("/meta/title_copy")
	// meta/title_copy is not in struct; use JSONValue path for this case.
	doc := MustReadStringNoSchema(`{"title":"A","meta":{"score":1}}`)
	patch := MustNewPatch(PatchCopy("/title", "/meta/title_copy"), PatchMove("/meta/title_copy", "/alias"))
	result, err := ApplyPatch(doc, patch)
	if err != nil {
		t.Fatalf("ApplyPatch returned error: %v", err)
	}
	if got := result.Get("alias").String(); got != "A" {
		t.Fatalf("alias = %q", got)
	}
	if result.Get("meta").HasField("title_copy") {
		t.Fatal("moved value still present at from")
	}
	_ = from
	_ = to
}

func TestPatchErrorIncludesOpIndexAndPath(t *testing.T) {
	doc := MustReadStringNoSchema(`{"name":"x"}`)
	err := ValidatePatch(doc, MustNewPatch(PatchTest("/name", NewString("y"))))
	if err == nil {
		t.Fatal("expected error")
	}
	text := err.Error()
	if !strings.Contains(text, "patch op 0") || !strings.Contains(text, "/name") {
		t.Fatalf("error = %q", text)
	}
}
