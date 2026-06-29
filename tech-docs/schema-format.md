# Schema Format

An ojson schema is a JSON document that describes the expected kind and canonical order of another JSON document. It is intentionally smaller than JSON Schema. Its purpose is ordered document handling, defaults, required fields, and basic kind checking.

## Root Schema

The root schema describes the entire JSON text. The root usually has `kind: "object"`, but an array or scalar root is possible when the document itself is not an object.

```json
{
  "kind": "object",
  "children": []
}
```

The root schema does not need a `name` because it is not a field inside a parent object.

## Schema Entry Fields

### `kind`

Required. Defines the JSON kind for the value.

Supported values:

- `object`
- `array`
- `string`
- `number`
- `boolean`
- `null`

There is no `any` kind. A value either has a supported kind or it is outside the schema model.

### `name`

Required for object children. Not used on the root schema.

The `name` is the JSON object field name:

```json
{
  "name": "height_units",
  "kind": "string"
}
```

### `children`

Optional. Used by object schemas to define ordered fields.

Each child is another schema entry. The order of entries in `children` is the canonical order for the object.

```json
{
  "name": "pet",
  "kind": "object",
  "children": [
    { "name": "name", "kind": "string" },
    { "name": "age", "kind": "number" }
  ]
}
```

### `default`

Optional. Provides a value to insert when a field is missing from the source JSON.

The default must match the schema entry's `kind`.

```json
{
  "name": "height_units",
  "kind": "string",
  "default": "inches"
}
```

Examples by kind:

```json
{ "name": "name", "kind": "string", "default": "unknown" }
```

```json
{ "name": "age", "kind": "number", "default": 0 }
```

```json
{ "name": "safe", "kind": "boolean", "default": true }
```

```json
{ "name": "middle_name", "kind": "null", "default": null }
```

Object defaults should be used carefully. Prefer child defaults when only specific fields need canonical values.

### `required`

Optional. When set to `true`, the field must exist in the source JSON unless a valid default is available.

```json
{
  "name": "name",
  "kind": "string",
  "required": true
}
```

If a required field is missing and has no default, reading with the schema should fail.

## Complete Example

```json
{
  "kind": "object",
  "children": [
    {
      "name": "pet",
      "kind": "object",
      "children": [
        {
          "name": "name",
          "kind": "string",
          "required": true
        },
        {
          "name": "age",
          "kind": "number"
        },
        {
          "name": "height",
          "kind": "number",
          "default": 0
        },
        {
          "name": "height_units",
          "kind": "string",
          "default": "inches"
        },
        {
          "name": "safe",
          "kind": "boolean",
          "default": true
        }
      ]
    }
  ]
}
```

## Read Behavior

When JSON is read with a schema:

1. the source JSON is parsed
2. the schema is parsed
3. known fields are matched by name
4. known fields are validated against their schema kind
5. missing fields with defaults are inserted
6. missing required fields without defaults cause an error
7. known fields are stored in schema order
8. unknown fields are preserved after known fields

Source field order is accepted even when it differs from schema order. The stored document uses schema order after normalization.

## Write Behavior

When JSON is written after schema-backed editing, serialization should preserve the normalized order already present in memory. The writer should not need to rediscover schema ordering if document mutation already enforces it.

Pretty printing is recommended for schema-backed files because it makes field ordering visible and reviewable.

## Unknown Fields

Fields that appear in the JSON document but not in the schema should be retained after schema-defined fields.

For example, with schema fields `name` and `age`, this source:

```json
{
  "nickname": "Whiff",
  "age": 3.2,
  "name": "Whiffles"
}
```

should normalize to:

```json
{
  "name": "Whiffles",
  "age": 3.2,
  "nickname": "Whiff"
}
```

This keeps the schema authoritative without discarding data the application may not yet understand.

## Unsupported Validation

The ojson schema format does not attempt to cover every validation feature from JSON Schema.

Keep these concerns in application validation unless the schema format is intentionally expanded:

- string formats
- regular expressions
- numeric minimums or maximums
- array item schemas
- union types
- arbitrary `any` values
- object property patterns
- enum values
- cross-field validation

## Schema Evolution

Schema changes can create large diffs if they reorder existing fields. To keep Git history readable:

- append new fields when possible
- avoid reordering established fields without a migration reason
- add defaults for fields that should appear in every normalized document
- avoid marking newly added fields as required until existing documents can satisfy them
- normalize documents in a dedicated change so schema migration diffs are easy to review
