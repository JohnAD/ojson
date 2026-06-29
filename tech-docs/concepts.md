# OJSON Concepts

`ojson` is built around one practical requirement: JSON text sometimes needs a stable field order. That need usually appears when JSON files are read and edited by people, committed to Git, reviewed in diffs, or processed by text-based tools that expect fields to appear in a canonical order.

Standard JSON does not assign meaning to object field order. Standard Go maps also do not preserve a caller-controlled order. `ojson` keeps order as document metadata so a JSON-compatible document can still be written back in a predictable textual form.

## Ordered Objects

An object is stored as an ordered list of key/value pairs rather than as a Go map. That means:

- fields can be written in the same order they were read
- new fields can be appended predictably
- schema-defined fields can be emitted in schema order
- unknown fields can be retained after known fields

This is most useful for pretty-printed JSON where each field starts on its own line. Stable ordering prevents unchanged fields from showing up as delete/add noise in `git diff`.

## JSON Kinds

The library represents values with a small set of JSON kinds:

```go
type JSONKind uint8

const (
    KindVoid JSONKind = iota
    KindObject
    KindArray
    KindString
    KindNumber
    KindBoolean
    KindNull
)
```

These kinds represent the JSON value space plus `Void`, which is an ojson-specific absence marker.

## Void, Null, And nil

`Void`, `Null`, and `nil` are different concepts:

- `Void` means the value does not exist in the JSON document.
- `Null` means the JSON document explicitly contains `null`.
- `nil` is a Go runtime concept and is not a JSON value.

For this JSON document:

```json
{
  "a": 1,
  "c": null
}
```

The field `a` exists and is a number. The field `c` exists and is null. The field `b` is not present, so a lookup for `b` should produce `Void`.

`Void` values are never written to JSON text. They are useful while traversing or editing documents because they let code ask for missing paths without immediately panicking or manufacturing a JSON value that did not exist.

## Decimal Numbers

JSON numbers are decimal text. They are not inherently `float64`, `int`, or any other binary representation.

`ojson` stores numbers as strings so a document can round-trip without losing the exact decimal spelling. This matters when:

- the exact value matters
- the exact text representation matters
- large numbers exceed native integer ranges
- decimal values should not be rounded through binary floating-point conversion

Application code can still convert number strings into `int`, `float64`, decimal packages, or domain-specific numeric types. The conversion decision belongs outside the JSON storage layer.

## Arrays

JSON arrays are heterogeneous lists. They are not arrays in the stricter computer science sense where every item must have the same type.

This is valid JSON:

```json
[1, "Hello", null, true, { "name": "Whiffles" }]
```

`ojson` arrays should preserve item order exactly. Schema support may describe the array itself, but the current schema model does not define a random `any` kind and should be conservative about mixed array item validation.

## Schema Order

An ojson schema gives object fields a canonical order. When a document is read with a schema:

- fields defined by the schema are stored in schema order
- incoming order does not need to match the schema
- fields not defined by the schema are retained after schema-defined fields
- missing fields with defaults are inserted
- missing required fields without defaults are errors

When a document is written, the document should already have been normalized through the schema-aware manipulation process. Serialization can then focus on producing the ordered JSON text.

## Performance Tradeoff

Preserving order requires different data structures and more bookkeeping than ordinary Go JSON handling. `ojson` should be expected to be slower than `encoding/json` for common API and data-transfer work.

That tradeoff is intentional. Use `ojson` for ordered document workflows. Use the Go standard library for ordinary JSON exchange.
