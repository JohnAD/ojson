# Struct Tags And Schema JSON

This guide documents how to convert and compare Go `json` struct tags to and from an ojson schema JSON document. It is intended for tooling that keeps Go data structures and ordered JSON schema files aligned.

The conversion is intentionally conservative. Go types contain information that an ojson schema does not, and an ojson schema contains ordering and default information that ordinary Go struct tags do not. Tooling should report uncertainty instead of guessing silently.

## Goals

Struct-tag/schema tooling should help answer four questions:

- Do my Go struct fields appear in the same order as the schema?
- Do my Go `json` field names match the schema field names?
- Do my Go field types map to the schema kinds?
- Can a schema JSON document be used to suggest Go field declarations and tags?

## Struct Field Selection

When converting a Go struct to an ojson schema:

1. inspect fields in declaration order
2. include exported fields
3. skip unexported fields unless they are embedded fields that expose exported fields through normal Go JSON behavior
4. skip fields tagged `json:"-"`
5. use the `json` tag name when present
6. otherwise use the Go field name according to the project's naming policy

Recommended default naming policy: require explicit `json` tags for schema generation. This avoids ambiguous conversions such as `CustomerNumber` to `CustomerNumber` versus `customer_number`.

## Parsing `json` Tags

A Go JSON tag has a field name followed by comma-separated options:

```go
Name string `json:"name,omitempty"`
```

The field name is `name`. The option is `omitempty`.

Rules:

- `json:"name"` maps the field to schema name `name`.
- `json:"name,omitempty"` maps the field to schema name `name` and marks it as optional for comparison purposes.
- `json:",omitempty"` uses the default Go JSON field name and marks it optional.
- `json:"-"` excludes the field.
- tag options do not affect schema `kind`.

## Mapping Go Types To Schema Kinds

Use this mapping as the default conversion policy:

| Go type shape | Schema kind | Notes |
| --- | --- | --- |
| `string` | `string` | Direct mapping. |
| `bool` | `boolean` | Direct mapping. |
| signed or unsigned integers | `number` | JSON has one numeric kind. |
| floating-point numbers | `number` | Decimal round-trip concerns still apply. |
| `json.Number` | `number` | Good fit for decimal text preservation. |
| decimal package types | `number` | Require project-specific type allowlist. |
| struct | `object` | Recurse through exported fields. |
| pointer to struct | `object` | Optionality differs from kind. |
| slice or array | `array` | Item validation is limited by current schema model. |
| pointer to scalar | scalar kind | Optionality differs from kind. |
| `interface{}` or `any` | unsupported | No ojson `any` kind. |
| map | unsupported or `object` by policy | Maps do not preserve field order. |

Because JSON has only one `number` kind, converting from Go to schema loses whether the source was `int`, `float64`, `json.Number`, or a decimal type.

## Required And Optional Fields

`omitempty` should not automatically produce `required: true`.

Recommended policy:

- fields without `omitempty` may be treated as required by strict tooling
- fields with `omitempty` should be treated as optional
- pointer fields should be treated as optional unless project rules say otherwise
- defaults should come from schema metadata or explicit tooling configuration, not from struct tags

Example:

```go
type Pet struct {
    Name        string  `json:"name"`
    Age         string  `json:"age,omitempty"`
    Height      string  `json:"height,omitempty"`
    HeightUnits string  `json:"height_units,omitempty"`
    Safe        bool    `json:"safe"`
}
```

Conservative generated schema:

```json
{
  "kind": "object",
  "children": [
    { "name": "name", "kind": "string" },
    { "name": "age", "kind": "string" },
    { "name": "height", "kind": "string" },
    { "name": "height_units", "kind": "string" },
    { "name": "safe", "kind": "boolean" }
  ]
}
```

A stricter project policy could mark `name` and `safe` as required, but that should be explicit.

## Converting Struct Tags To Schema

Procedure:

1. Parse the Go package with `go/parser` and `go/ast`.
2. Find the target struct type.
3. Walk fields in declaration order.
4. Skip fields excluded by visibility or `json:"-"`.
5. Resolve the JSON field name from the tag.
6. Map the Go type to an ojson schema kind.
7. Recurse into nested structs for object children.
8. Emit schema children in struct declaration order.
9. Record warnings for unsupported fields.
10. Attach defaults only from explicit configuration.

Example input:

```go
type Pet struct {
    Name        string `json:"name"`
    Age         string `json:"age,omitempty"`
    Height      string `json:"height,omitempty"`
    HeightUnits string `json:"height_units,omitempty"`
    Safe        bool   `json:"safe"`
}
```

Example schema output:

```json
{
  "kind": "object",
  "children": [
    { "name": "name", "kind": "string" },
    { "name": "age", "kind": "string" },
    { "name": "height", "kind": "string" },
    { "name": "height_units", "kind": "string" },
    { "name": "safe", "kind": "boolean" }
  ]
}
```

If numeric fields are stored as strings in Go to preserve decimal text, the generated schema will say `string`. If the JSON document should contain JSON numbers, use a numeric Go type, `json.Number`, or a project decimal type that the converter maps to `number`.

## Comparing Struct Tags To Schema

Comparison should produce a structured report rather than a single boolean.

Procedure:

1. Convert the target struct into an intermediate ordered field list.
2. Parse the schema JSON into an intermediate ordered schema list.
3. Compare field names in order.
4. Compare field names as sets.
5. Compare schema kinds for matching fields.
6. Recurse into object fields.
7. Report unsupported Go types.
8. Report unsupported schema kinds.
9. Report required/default differences that cannot be represented in struct tags.

Recommended report categories:

- `missing_in_schema`: field appears in the struct but not the schema
- `missing_in_struct`: field appears in the schema but not the struct
- `order_mismatch`: same fields exist but order differs
- `kind_mismatch`: field names match but kinds differ
- `unsupported_go_type`: Go type cannot map to an ojson schema kind
- `unsupported_schema_feature`: schema metadata has no struct-tag equivalent
- `default_only_in_schema`: schema default exists and cannot be represented by `json` tags
- `required_policy_difference`: required status differs from the selected project policy

Example mismatch:

```go
type Pet struct {
    Name string `json:"name"`
    Safe bool   `json:"safe"`
    Age  string `json:"age,omitempty"`
}
```

```json
{
  "kind": "object",
  "children": [
    { "name": "name", "kind": "string" },
    { "name": "age", "kind": "number" },
    { "name": "safe", "kind": "boolean" }
  ]
}
```

Expected comparison findings:

- `order_mismatch`: `safe` appears before `age` in the struct, but after `age` in the schema.
- `kind_mismatch`: struct field `age` maps to `string`, but schema field `age` is `number`.

## Converting Schema To Struct Tags

Schema-to-struct conversion can generate a useful starting point, but it cannot know every desired Go type.

Procedure:

1. Parse the schema JSON.
2. Confirm the root schema kind is `object`.
3. Convert each child into an exported Go field name.
4. Preserve schema order as struct field order.
5. Add a `json` tag using the schema `name`.
6. Map schema kinds to default Go types.
7. Recurse into object children.
8. Add comments or warnings for defaults and required fields.

Default reverse mapping:

| Schema kind | Suggested Go type | Notes |
| --- | --- | --- |
| `string` | `string` | Direct mapping. |
| `number` | `json.Number` | Preserves decimal text better than `float64`. |
| `boolean` | `bool` | Direct mapping. |
| `object` | nested struct | Generate a named or anonymous struct by policy. |
| `array` | `[]ojson.JSONValue` or project type | Item kind is not fully described by the base schema model. |
| `null` | `*struct{}` or `any` by policy | Usually needs human review. |

Example schema:

```json
{
  "kind": "object",
  "children": [
    { "name": "name", "kind": "string", "required": true },
    { "name": "age", "kind": "number" },
    { "name": "safe", "kind": "boolean", "default": true }
  ]
}
```

Suggested Go struct:

```go
type Pet struct {
    Name string      `json:"name"`
    Age  json.Number `json:"age,omitempty"`
    Safe bool        `json:"safe,omitempty"`
}
```

The generated struct should include review notes:

- `name` is required by schema
- `safe` has schema default `true`
- `age` uses `json.Number` by default but may need a project-specific numeric type

## Field Name Generation

When generating Go names from schema field names:

- split on underscores, hyphens, and spaces
- capitalize each word
- preserve common initialisms by project policy
- avoid Go keywords
- make duplicate names unique with a clear suffix

Examples:

| Schema name | Go field name |
| --- | --- |
| `name` | `Name` |
| `customer_number` | `CustomerNumber` |
| `height_units` | `HeightUnits` |
| `type` | `TypeValue` |

## Limitations

Struct tags and ojson schemas do not contain the same information.

Struct tags cannot represent:

- schema field order independently of struct declaration order
- default values
- all required-field policies
- unknown field handling
- schema migration guidance

Ojson schemas cannot fully represent:

- exact Go numeric type choice
- custom marshal/unmarshal behavior
- embedded field promotion rules
- map key/value types
- interface implementations
- validation beyond basic JSON kind

Tooling should preserve these limitations in its output. A warning is better than a generated schema or struct that looks precise but is not.
