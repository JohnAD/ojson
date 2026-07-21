# Adding RFC6902 patch support

We are going to add RFC6902-style patching support.

I say "style" because I'm not expecting this library to read actual RFC6902 json-list string. Instead I want to the patch to be verified at compile-time as much as possible. Strings provided by humans are too prone to typos and problem are only detected at run-time. That is not appropriate.

It should, however, be able to write a RFC6902 json-list to `string` or `[]byte`.

## building a patch

Something akin to:

```go
patch := ojson.Patch{
  [
    PatchRemove{ path: "/a/b/c" },
    PatchAdd( path: "/a/b/c", value: [ "foo", "bar" ] },
  ]
}
```

```go
patch := ojson.Patch[FooBar]{
  [
    PatchRemove[FooBar]{ t.a.b.c },
    PatchAdd[FooBar]( t.a.b.c, "bling" },
  ]
}
```

I've not vetted ANY of the above code; it's just quick idea of form.

A user should be able to create a Patch in one of three ways:

1. Creating a Patch via "diff" of two typed objects.
2. Creating a Patch via "diff" of two ojson.JsonNode objects.
3. Manually building a Patch manually like seen in the examples.

In all three cases, we want it to be possible to vett the patch against the schema. The code should return an error on the first schema violation found. For example if the a.b.c path is required, but the patch removes the a.b.c path.

## applying a patch

The library should be able to apply a compliant patch to JsonNode or Typed object variable and returned a changed copy. It should leave the original alone. Again, if a schema is given as a parameter, it should verify the patch meets schema specs or return an error.

## Expressing a patch

The library should be able to express a Patch as a JSON List `string` or `[]byte` for compatiblity with external systems.

## Reading a patch

I'm not expecting the library to read a RFC6902 string or []byte, but if it turns out to be easy to do so, then we could have it support that as well.

## else

The next page has notes from the client library using ojson:

-----

# TODO: Schema-aware RFC6902 in ojson

Handoff from `datorium-client-go` patching design. DatoriumDB’s Go client will depend on this; it will **not** reimplement schema-aware patch algebra.

Context: [`datorium-client-go/tech-docs/PATCHING.md`](../datorium-client-go/tech-docs/PATCHING.md) (sibling repo).

## Why here

Existing Go RFC6902 libraries (evanphx/json-patch, wI2L/jsondiff, etc.) are **not** schema-aware. They ignore:

- required fields and defaults
- Void vs Null vs missing
- canonical object field order
- kind validation under schema

ojson already owns those concerns for read/apply/mutate. Patch support should extend that, not fight it.

## Scope (all three)

Implement native RFC6902 support with optional schema context:

1. **Apply** — apply a patch to a `JSONValue`, normalizing with schema when attached/applied (defaults, required, order, Void/Null).
2. **Diff** — produce a patch from before/after documents (schema-aware so defaults/missing do not spam noisy ops; object `value`s stay ordered).
3. **Validate** — check ops against schema before/during apply; on reject, **document why** (stable error paths / messages suitable for app authors).

### Out of scope for ojson

- Datorium system fields (`!`, `$`, `#`) as special cases — callers (e.g. datorium-client-go) enforce those policies.
- Access-language envelopes or HTTP transport.
- Becoming a general Kubernetes merge-patch (RFC7396) product unless it falls out naturally later.

## Consumer paths (for API shape)

**Path 1 — diff from edited document** (datorium: mutate `TypedRead.Doc` then patch):

- Mostly: schema-aware **Diff** (+ Validate) → ordered RFC6902 op list / patch value.
- Client wraps with `$` / `#` and posts.

**Path 2 — hand-crafted ops** (datorium: first focus):

- App builds ops (may use ojson types/helpers directly).
- **Validate** (+ optional dry-run **Apply**) against schema.
- Client sends vetted ops.

App authors using path 2 may need a little ojson; that is acceptable. Document constructors and errors clearly in ojson tech-docs.

## Suggested deliverables

### Types / wire

- [ ] Represent RFC6902 operations as ordered ojson values (or a small typed op API that lowers to ordered JSON).
- [ ] Support ops: at least `add`, `remove`, `replace`, `test`; then `move`, `copy` as needed.
- [ ] JSON Pointer (RFC6901) resolution with `~0` / `~1`, array `-` append.
- [ ] Round-trip: patch document ↔ apply ↔ serialize with stable field order in object `value`s.

### Validate

- [ ] `ValidatePatch(doc, patch, schema?)` (exact names TBD).
- [ ] Reject illegal paths, kind mismatches, removing required fields without default recovery policy (define policy), bad pointers, failed structural rules.
- [ ] Errors include **why** (path + rule + expected/actual where useful), consistent with existing ojson path-error style.

### Apply

- [ ] `ApplyPatch` / schema-backed apply on `JSONValue`.
- [ ] With schema: normalize after ops (or incrementally) — defaults, required, field order, unknown-field policy as today.
- [ ] Atomicity: define whether failed mid-patch rolls back (prefer RFC6902-style all-or-nothing).
- [ ] Without schema: still ordered apply; fewer semantic checks.

### Diff

- [ ] `Diff(before, after, schema?)` → patch.
- [ ] Schema mode: compare after normalization so default-equivalent documents yield empty/minimal patches.
- [ ] Prefer `replace`/`add`/`remove` clarity over clever `move`/`copy` unless factorization is opt-in.
- [ ] Emitted `value` objects preserve schema/declaration order.

### Docs & tests

- [ ] tech-docs page for patch API (apply / diff / validate, schema vs no-schema).
- [ ] Examples for path-1 style (diff) and path-2 style (hand-built ops + validate).
- [ ] Tests: required/default interactions, Void vs Null, array append `-`, pointer escaping, order stability of nested objects in `value`, validation error messages.

## Non-goals / caution

- Do not route patch `value` objects through `map[string]any` (destroys order).
- Do not special-case Datorium metadata keys in ojson.
- Keep the API about JSON + schema; let clients add domain rules on top.

## Done when

- datorium-client-go can depend on a published ojson release and:
  - validate hand-built ops against a collection schema (path 2),
  - diff schema-normalized before/after for typed-read editing (path 1),
  - serialize RFC6902 arrays with ordered object values for the access-language `RFC6902` field.
