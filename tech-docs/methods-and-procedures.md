# Methods And Procedures

This document is pre-written project documentation for the intended ojson API surface described by the README. It should be kept synchronized with implementation code as the project evolves.

## Value Constructors

Constructors create `JSONValue` instances with the correct kind and internal storage.

### `NewObject`

Creates an empty ordered object.

Use this when building a document or nested object programmatically. Fields are written in insertion order unless the object is associated with a schema.

```go
doc := ojson.NewObject()
doc.Set("name", ojson.NewString("Whiffles"))
```

### `NewArray`

Creates an empty array.

Use this when the JSON value should be an ordered list of values. Array item order is always meaningful and should be preserved.

```go
ratings := ojson.NewArray()
ratings.Append(ojson.NewNumberTry("3.2"))
ratings.Append(ojson.NewNull())
```

### `NewString`

Creates a JSON string value.

```go
name := ojson.NewString("Whiffles")
```

### `NewNumber`

Creates a JSON number from decimal text and returns an error if the text is not a valid JSON number.

Use `NewNumber` when invalid numeric input should stop the operation.

```go
age, err := ojson.NewNumber("3.2")
if err != nil {
    return err
}
doc.Set("age", age)
```

### `NewNumberTry`

Creates a JSON number from decimal text, using a safe fallback when the text is not valid.

Use `NewNumberTry` when input has already been cleaned or when a fallback value is acceptable. Documentation and call sites should make the fallback behavior obvious because silently replacing a bad number can hide data problems.

```go
height := ojson.NewNumberTry("21.5")
```

### `NewBoolean`

Creates a JSON boolean value.

```go
safe := ojson.NewBoolean(true)
```

### `NewNull`

Creates an explicit JSON `null` value.

Use `NewNull` when the document should contain the field with a null value. Do not use it to represent a missing field; missing fields are represented by `Void`.

```go
doc.Set("middle_name", ojson.NewNull())
```

### `NewVoid`

Creates or returns the ojson absence marker.

`Void` is useful internally and for traversal results. It should not be serialized into JSON text.

## Reading Documents

### `ReadStringNoSchema`

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

### `ReadBytesNoSchema`

Reads JSON bytes without applying a schema.

This is the byte-slice equivalent of `ReadStringNoSchema` and is useful when JSON is already loaded from a file, network response, or embedded asset.

### `ReadFileNoSchema`

Reads a JSON file without applying a schema.

This procedure should:

1. read the file as bytes
2. parse the bytes as JSON
3. preserve source object order
4. return the root `JSONValue`

## Reading Schemas

### `ReadSchemaString`

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

### `ReadSchemaBytes`

Reads an ojson schema document from bytes.

Use this when schema JSON is already loaded in memory.

### `ReadSchemaFile`

Reads an ojson schema document from a file.

Use this for project-managed schema files that define the canonical order for one or more JSON document types.

## Reading Documents With Schemas

### `ReadStringWithSchema`

Reads JSON text and normalizes it through an ojson schema.

Expected behavior:

- parse the schema
- parse the JSON document
- validate known fields against schema kinds
- reorder schema-defined object fields into schema order
- insert default values for missing fields with defaults
- error when required fields are missing and no default exists
- preserve unknown fields after schema-defined fields

### `ReadBytesWithSchema`

Reads JSON bytes and applies a schema.

Use this form when both JSON and schema documents are already available in memory.

### `ReadFileWithSchema`

Reads a JSON file and applies a schema loaded from a schema document.

Use this procedure for project data files that must conform to a canonical order before being edited or written back.

## Traversal

### `Get`

Returns the value for an object field.

If the receiver is not an object, or if the key does not exist, `Get` should return `Void`.

```go
name := doc.Get("pet").Get("name")
```

Returning `Void` keeps chained lookups readable:

```go
city := doc.Get("location").Get("postal_city").GetString("Location city is missing")
```

### `AtIndex`

Returns the value at an array index.

If the receiver is not an array, or if the index is out of range, `AtIndex` should return a fallback value or `Void`, depending on the final API signature.

```go
fourth := doc.Get("ratings").AtIndex(3, "There is no fourth rating.")
```

## Mutation

### `Set`

Sets an object field.

For unordered, no-schema objects, `Set` should preserve existing field position when replacing a value and append new fields when adding a value. For schema-backed objects, `Set` should place schema-defined fields in schema order.

```go
doc.Set("pet", ojson.NewObject())
doc.Get("pet").Set("name", ojson.NewString("Whiffles"))
```

Calling `Set` on a `Void` value should not create an implicit parent object. For example, `doc.Get("location").Set("city", ...)` should not mutate the document if `location` is missing.

### `Append`

Adds a value to the end of an array.

If the receiver is not an array, the method should fail or no-op according to the final implementation policy. Prefer explicit errors for new public APIs because failed mutations are otherwise hard to detect.

### `Remove`

Removes an object field or array item.

Removal should produce absence, not JSON `null`. Use `NewNull` when the field should remain present with a null value.

## Accessors

Accessors convert a `JSONValue` into a convenient Go value while allowing callers to provide fallbacks for missing or mismatched kinds.

### `GetString`

Returns the string content for `KindString`.

If the value is missing or not a string, return the supplied fallback.

```go
name := doc.Get("name").GetString("unknown")
```

### `GetNumber`

Returns the decimal text for `KindNumber`.

Do not automatically convert to binary floating-point unless the method name makes that conversion explicit.

```go
amountText := doc.Get("amount").GetNumber("0")
```

### `GetBoolean`

Returns the bool content for `KindBoolean`.

```go
enabled := doc.Get("enabled").GetBoolean(false)
```

### `IsMissing`

Reports whether a value is `KindVoid`.

```go
if doc.Get("location").IsMissing() {
    fmt.Println("There is no location.")
}
```

### `IsNull`

Reports whether a value is explicit JSON `null`.

This should be separate from `IsMissing` so callers can distinguish absent data from known unknown data.

## Serialization

### `String`

Returns compact JSON text.

Compact serialization should still preserve object order.

### `PrettyPrint`

Returns indented JSON text.

The indentation argument controls how many spaces are used for each nesting level.

```go
fmt.Println(doc.PrettyPrint(2))
```

Pretty output is the preferred form for Git-tracked JSON files because it makes line-oriented diffs easier to review.

### `WriteFile`

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
