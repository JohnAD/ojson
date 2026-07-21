package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMoviePaths(t *testing.T) {
	dir := filepath.Join("testdata", "movie")
	code, err := generate(dir, "Movie")
	if err != nil {
		t.Fatalf("generate returned error: %v", err)
	}
	text := string(code)
	for _, want := range []string{
		"package movie",
		`ojson.NewTypedPath[Movie, string]("/title")`,
		`ojson.NewTypedPath[Movie, time.Time]("/released_at")`,
		`ojson.NewTypedPath[Movie, []string]("/tags")`,
		`ojson.NewTypedPath[Movie, int]("/meta/score")`,
		`"time"`,
		`"github.com/JohnAD/ojson"`,
		"var MoviePaths",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated code missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateRejectsMapField(t *testing.T) {
	dir := t.TempDir()
	src := `package bad
type Doc struct {
	Values map[string]string ` + "`json:\"values\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "doc.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := generate(dir, "Doc"); err == nil || !strings.Contains(err.Error(), "map fields are not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}
