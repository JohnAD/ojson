# RFC6902 Patching

`ojson` provides schema-aware RFC6902 patch construction, parsing, validation, application, and diffing while preserving ordered object values.

## Goals

- Apply patches atomically to a deep clone; never mutate the input document
- Validate and normalize with optional schema context
- Diff documents into deterministic `add` / `remove` / `replace` operations
- Serialize and parse RFC6902 JSON arrays with stable field order in operation `value`s
- Offer runtime constructors and generated compile-time typed paths

## Patch Model

A `Patch` is an ordered list of operations. Supported operations:

- `add`
- `remove`
- `replace`
- `test`
- `move`
- `copy`

Operations keep `JSONValue` payloads directly. Do not route patch values through `map[string]any`.

### Runtime constructors

```go
patch, err := ojson.NewPatch(
    ojson.PatchTest("/name", ojson.NewString("Whiffles")),
    ojson.PatchReplace("/age", ojson.NewNumberFromInt(4)),
    ojson.PatchAdd("/tags/-", ojson.NewString("fast")),
    ojson.PatchCopy("/name", "/nickname"),
    ojson.PatchMove("/nickname", "/alias"),
    ojson.PatchRemove("/alias"),
)
```

Build pointers from segments:

```go
pointer, err := ojson.Pointer("pet", "contact/email", 0)
// "/pet/contact~1email/0"
```

### Wire format

```go
text := patch.ToJSON()
bytes := patch.ToJSONBytes()

parsed, err := ojson.ReadPatchJSON(text)
```

`ReadPatchJSON` / `ReadPatchBytes` are strict: unknown operation fields, missing required fields, void values, and invalid pointers are errors.

## JSON Pointers

Pointers follow RFC6901:

- `""` is the document root
- `~0` escapes `~`
- `~1` escapes `/`
- array indexes are decimal integers without leading zeros
- `-` means append and is valid only as the final token of an `add` path

Diagnostic errors include both the RFC6901 pointer and an ojson path such as `"tags".0`.

## Schema Context

`ApplyPatch`, `ValidatePatch`, and `Diff` accept variadic `PatchOption` values.

```go
result, err := ojson.ApplyPatch(doc, patch, ojson.WithPatchSchema(schema))
```

Precedence:

1. `WithPatchSchema(schema)` when provided
2. otherwise a schema already attached to the document

### Normalization timing

RFC6902 operations run first with raw document semantics. If schema context is present, `ojson` normalizes once after the full patch succeeds.

Consequences:

- later operations see fields as left by previous operations
- removing a required field that has a default succeeds during ops and is restored during final normalization
- removing a required field without a default fails during final normalization
- Void is rejected as an operation `value`; explicit JSON `null` follows ordinary nullable rules

## Apply And Validate

```go
result, err := ojson.ApplyPatch(doc, patch, ojson.WithPatchSchema(schema))
err = ojson.ValidatePatch(doc, patch, ojson.WithPatchSchema(schema))
```

Behavior:

- deep-clones the input
- applies operations in order
- discards the clone on the first failure (all-or-nothing)
- leaves the original document unchanged
- with schema, validates the final document through ordinary schema normalization

`ValidatePatch` is a dry-run apply.

## Diff

```go
patch, err := ojson.Diff(before, after, ojson.WithPatchSchema(schema))
```

Schema mode normalizes both sides first so default-equivalent documents produce an empty patch. Diff emits only `add`, `remove`, and `replace`. It does not infer `move` or `copy`.

Equality rules used by diff and `test`:

- Null and missing are distinct
- numbers compare by numeric value (`25` equals `25.0`)
- object field order does not affect equality
- emitted object values preserve declaration/source order

## Typed Struct Helpers

```go
patch, err := ojson.DiffStructs(beforeMovie, afterMovie, ojson.WithPatchSchema(schema))
updated, err := ojson.ApplyPatchToStruct(beforeMovie, patch, ojson.WithPatchSchema(schema))
```

These convert through `JSONValue`, reuse existing native/JSON/Text marshal interfaces, and return a changed copy without mutating the input.

### Typed paths

```go
title := ojson.NewTypedPath[Movie, string]("/title")
op, err := ojson.ReplaceAt(title, "Dune")
```

Helpers: `AddAt`, `ReplaceAt`, `TestAt`, `RemoveAt`, `MoveAt`, `CopyAt`, `Index`, `Append`, `Child`.

## Generated Paths

Use `cmd/ojson-paths` for compile-time field paths:

```go
//go:generate go run github.com/JohnAD/ojson/cmd/ojson-paths -type Movie
```

The generator requires explicit `json` tag names, supports nested structs, pointers, slices, and imported leaf types such as `time.Time`, and rejects maps and unsupported dynamic shapes.

Generated code looks like:

```go
var MoviePaths = moviePaths{
    Title: ojson.NewTypedPath[Movie, string]("/title"),
    Meta: movieMetaPaths{
        Score: ojson.NewTypedPath[Movie, int]("/meta/score"),
    },
}
```

Use those paths when building a patch:

```go
replaceTitle, err := ojson.ReplaceAt(MoviePaths.Title, "Dune")
if err != nil {
    return err
}
replaceScore, err := ojson.ReplaceAt(MoviePaths.Meta.Score, 9)
if err != nil {
    return err
}

patch, err := ojson.NewPatch(replaceTitle, replaceScore)
if err != nil {
    return err
}
```

Runtime pointer constructors remain available for schema-only or dynamic paths.

## Errors

Patch failures use `PatchError`:

```text
patch op 2 at /tags/0 ("tags".0): test failed
```

Schema normalization failures after a successful raw patch are reported as a patch error whose reason begins with `schema normalization failed:`.

## Out Of Scope

- RFC7396 merge-patch
- Inferring `move` / `copy` during diff
