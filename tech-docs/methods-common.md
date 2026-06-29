# Common Methods And Procedures

This document covers ojson behavior shared across value kinds. Kind-specific methods live in the related method documents:

- [`methods-object.md`](methods-object.md)
- [`methods-array.md`](methods-array.md)
- [`methods-string.md`](methods-string.md)
- [`methods-number.md`](methods-number.md)
- [`methods-boolean.md`](methods-boolean.md)
- [`methods-null.md`](methods-null.md)
- [`methods-void.md`](methods-void.md)

## Failure Handling Convention

Method names should communicate how failure is handled:

- plain methods return a `Void` value on failure
- `*Try` methods return a result and an `error`
- `*OrDefault` methods return the caller-provided default on failure

Use plain methods for fluent document traversal, `*Try` methods when the caller needs the reason for failure, and `*OrDefault` methods when the caller has a meaningful replacement value.

## Reading Documents

### `ReadStringNoSchema(jsonText string) (JSONValue, error)`

Reads JSON text without applying a schema.

Expected behavior:

- parse JSON text into `JSONValue`
- preserve object field order from the source text
- preserve array item order
- store number values as decimal strings
- return an error for malformed JSON

```go
doc, err := ojson.ReadStringNoSchema(jsonText)
if err != nil {
    return err
}
```

### `ReadBytesNoSchema(jsonBytes []byte) (JSONValue, error)`

Reads JSON bytes without applying a schema.

This is the byte-slice equivalent of `ReadStringNoSchema` and is useful when JSON is already loaded from a file, network response, or embedded asset.

### `ReadFileNoSchema(path string) (JSONValue, error)`

Reads a JSON file without applying a schema.

This procedure should:

1. read the file as bytes
2. parse the bytes as JSON
3. preserve source object order
4. return the root `JSONValue`

## Reading Schemas

### `ReadSchemaString(schemaText string) (JSONSchema, error)`

Reads an ojson schema document from a string.

Expected behavior:

- parse the schema as JSON
- require supported schema fields and kinds
- preserve child order exactly
- return an error for malformed schema documents

```go
schema, err := ojson.ReadSchemaString(schemaText)
if err != nil {
    return err
}
```

### `ReadSchemaBytes(schemaBytes []byte) (JSONSchema, error)`

Reads an ojson schema document from bytes.

Use this when schema JSON is already loaded in memory.

### `ReadSchemaFile(path string) (JSONSchema, error)`

Reads an ojson schema document from a file.

Use this for project-managed schema files that define the canonical order for one or more JSON document types.

## Reading Documents With Schemas

### `ReadStringWithSchema(jsonText string, schema JSONSchema) (JSONValue, error)`

Reads JSON text and normalizes it through an ojson schema.

Expected behavior:

- parse the schema
- parse the JSON document
- validate known fields against schema kinds
- reorder schema-defined object fields into schema order
- insert default values for missing fields with defaults
- error when required fields are missing and no default exists
- preserve unknown fields after schema-defined fields

### `ReadBytesWithSchema(jsonBytes []byte, schema JSONSchema) (JSONValue, error)`

Reads JSON bytes and applies a schema.

Use this form when both JSON and schema documents are already available in memory.

### `ReadFileWithSchema(path string, schema JSONSchema) (JSONValue, error)`

Reads a JSON file and applies a schema loaded from a schema document.

Use this procedure for project data files that must conform to a canonical order before being edited or written back.

## State Methods

### `IsKnown() bool`

Reports whether a value is known.

`IsKnown` returns `true` for object, array, string, number, and boolean values. It returns `false` for `Void`, because `Void` is absence, and for `Null`, because `Null` means the value is unknown. It does not check whether a known value is empty for its kind.

```go
if doc.Get("pet").Get("name").IsKnown() {
    fmt.Println("name is known")
}
```

### `NotEmpty() bool`

Reports whether a value is known and is not the empty state for its kind.

`NotEmpty` should return `false` for `Void` and `Null`. For all other kinds, it should compare the value against the kind's empty state.

```go
if doc.Get("pet").Get("name").NotEmpty() {
    fmt.Println("name has a non-empty value")
}
```

### `IsEmpty() bool`

Reports whether a value is empty.

`IsEmpty` should be the inverse of `NotEmpty`. It should return `true` for `Void`, `Null`, and values that match the empty state for their kind.

```go
if doc.Get("pet").Get("name").IsEmpty() {
    fmt.Println("name is missing, null, or empty")
}
```

### Empty States

| Kind | Empty state | `IsKnown` | `NotEmpty` when empty | Notes |
| --- | --- | --- | --- | --- |
| `KindVoid` | missing value | `false` | `false` | `Void` is absence and is never serialized. |
| `KindNull` | `null` | `false` | `false` | `Null` is explicit JSON `null`, which means unknown. |
| `KindObject` | `{}` | `true` | `false` | An empty object is known, but has no fields. |
| `KindArray` | `[]` | `true` | `false` | An empty array is known, but has no items. |
| `KindString` | `""` | `true` | `false` | An empty string is known, but its content length is zero. |
| `KindNumber` | `0` | `true` | `false` | Zero is known; the canonical empty number is zero with no decimal part. |
| `KindBoolean` | `false` | `true` | `false` | False is known, but is treated as empty by common convention. |

## Serialization

### `String() string`

Returns the string-equivalent content of a value.

`String` is not a JSON serialization method. It is for caller-friendly value content. For a JSON string value, it should return the contained string without JSON quotes.

```go
value := ojson.NewString("foo")

fmt.Println(value.String()) // foo
```

For non-string values, `String` should return the closest useful content representation for that value. For example, a number can return its decimal text, a boolean can return `true` or `false`, and `null` can return `null`. Use `ToJSON` when the caller needs valid JSON text.

### `ToJSON() string`

Returns minified JSON text.

`ToJSON` should return the JSON representation of the value. For a JSON string, that includes JSON quotes and escaping.

```go
value := ojson.NewString("foo")

fmt.Println(value.ToJSON()) // "foo"
```

Minified serialization should still preserve object order.

### `ToJSONBytes() []byte`

Returns minified JSON as bytes.

This is the byte-slice equivalent of `ToJSON` and is useful for file writes, network responses, and APIs that accept `[]byte`.

```go
data := doc.ToJSONBytes()
```

### `ToPrettyJSON(indent int) string`

Returns indented JSON text.

The indentation argument controls how many spaces are used for each nesting level.

```go
fmt.Println(doc.ToPrettyJSON(2))
```

Pretty output is the preferred form for Git-tracked JSON files because it makes line-oriented diffs easier to review.

### `ToPrettyJSONBytes(indent int) []byte`

Returns indented JSON as bytes.

This is the byte-slice equivalent of `ToPrettyJSON`.

```go
data := doc.ToPrettyJSONBytes(2)
```

### `WriteFile(path string) error`

Writes JSON text to a file.

For project data files, prefer pretty output with stable indentation. File-writing procedures should avoid rewriting files when content has not changed if the surrounding application wants to minimize file modification timestamps.

## Schema Procedures

### Creating A Schema

1. Choose the root `kind`.
2. For objects, add ordered child schema entries.
3. Give each object child a `name`.
4. Add `required: true` for values that must be present.
5. Add `default` values for fields that should be inserted when missing.
6. Keep unsupported or application-specific validation outside the ojson schema document.

### Reading With A Schema

1. Load or parse the schema JSON.
2. Load or parse the target JSON.
3. Validate known fields by kind.
4. Insert defaults.
5. Reject missing required fields that do not have defaults.
6. Store known fields in schema order.
7. Store unknown fields after known fields.

### Writing With A Schema

1. Mutate the document through schema-aware methods.
2. Preserve the schema-defined order.
3. Pretty-print the document.
4. Write the result to disk only after successful validation.

### Updating A Schema

When changing a schema used by Git-tracked JSON files:

1. append new fields where possible to minimize diffs
2. use defaults for newly introduced optional fields when a canonical value is desired
3. mark fields as required only when every existing document can supply them
4. run a normalization pass over existing documents
5. review the resulting diffs for unexpected reordering or data loss
