# Project-Specific Instructions

These instructions apply to the `kached` project and extend the general Go and Go documentation
instructions. Prefer these rules when they conflict with the general instructions.

## Formatting

- Format code with `gofumpt` (extra-rules enabled) and `gofmt`.
- Manage imports with `goimports` and `gci`. Import sections in order:
  1. Standard library
  2. Third-party
  3. Blank imports
  4. Dot imports
- Keep lines within 120 characters in non-test files. Test files have no line-length limit.

## Table-driven tests

- Use `test` as the loop variable name, never `tc` or `tt`.
- Use `tests` as the slice variable name.
- Do not define a named struct type for test cases unless it is genuinely unavoidable. Use an
  anonymous struct inline.
- Standard fields are `name string`, `want <T>`, `wantErr require.ErrorAssertionFunc`. Add
  input fields as needed without a naming convention beyond being descriptive.
- Always use `t.Run` with subtests for table-driven tests. Do not use `t.Run` for single
  one-off tests.

Example:
```go
tests := []struct {
    name    string
    input   string
    want    int
    wantErr require.ErrorAssertionFunc
}{
    {
        name:    "valid input",
        input:   "42",
        want:    42,
        wantErr: require.NoError,
    },
    {
        name:    "invalid input",
        input:   "bad",
        wantErr: require.Error,
    },
}

for _, test := range tests {
    t.Run(test.name, func(t *testing.T) {
		t.Parallel()
		
        got, err := parse(test.input)
		
        test.wantErr(t, err)
        assert.Equal(t, test.want, got)
    })
}
```

## Assertions

- Use `testify` (`github.com/stretchr/testify`) for assertions: `assert` for non-fatal checks,
  `require` for fatal checks that should stop the test immediately.
- Use `require` for setup steps and for the error check before asserting on the result.

## Mocks

- Use `testify/mock` (`github.com/stretchr/testify/mock`) for mocks.
- Never include `context.Context` in mock method arguments. Contexts are infrastructure, not
  behaviour — pass `mock.Anything` implicitly by omitting context from assertions, or if the
  method signature requires a match, use `mock.Anything` and do not assert on the value.

## Logging

- Use `github.com/hamba/logger` for all structured logging.
- Obtain loggers via `github.com/hamba/cmd/v3/observe` at the application entrypoint.
- Inject the logger through the constructor and store it as a struct field. Do not use package-
  level loggers or pass loggers via `context`.

## Error handling

Do not shadow `err` when it is the only new variable. When `err` is already declared in the outer scope, use assignment (`=`)
for subsequent error-producing statements — not short variable declaration (`:=`).

```go
// Bad — shadows the outer err:
conn, err := pool.Get(ctx)
if err != nil { ... }
if err := codec.Write(w, req); err != nil { ... } // new err in if-init; shadows

// Good — reuses the outer err:
conn, err := pool.Get(ctx)
if err != nil { ... }
err = codec.Write(w, req)
if err != nil { ... }
```

Choose the error form based on what information must be carried:

| Situation | Form                                      |
|---|-------------------------------------------|
| A fixed, checkable condition with no extra data | `var ErrFoo = errors.New("foo")` sentinel |
| A condition that must carry extra data (address, key, command) | custom type implementing `error`          |
| Wrapping a lower-level error with context | `fmt.Errorf("... : %w", err)`             |

Wrap errors at every layer. Error message conventions:
- The **top-most** exported function or method uses `"could not <verb> <subject>"` phrasing.
- **Inner layers** use the verb + object directly: `"creating connection to %s"`, `"parsing
  request header"`. Do not repeat `"could not"` at every level.
- Error strings are lowercase and do not end with punctuation.

```go
// Top layer:
return fmt.Errorf("could not dial backend: %w", err)

// Inner layer:
return fmt.Errorf("reading response header: %w", err)
```

## Linting

- Follow `.golangci.yml` as the source of truth.
- `//nolint` directives are acceptable when splitting or restructuring code would make it
  genuinely harder to read (e.g. a naturally long switch or a parser loop). Always include the
  linter name and a brief reason: `//nolint:cyclop // parser state machine is inherently complex`.
- Long functions are acceptable when the logic is sequential and splitting it would introduce
  more indirection than clarity. Do not split a function just to satisfy a length heuristic.
- Do not add `//nolint` liberally to silence warnings without consideration.

## Performance & allocation

- Do not reach for `sync.Pool` preemptively. Introduce it only after a benchmark demonstrates
  a meaningful allocation reduction on a hot path.
- `unsafe` is permitted in performance-critical paths, but only when profiling shows a
  measurable improvement that is significant relative to the surrounding I/O cost. A 4 ns
  saving next to a 100 ms network call is not a justification. Document the unsafe block with
  an inline comment explaining exactly what invariant makes the operation safe.

## Context usage

- Accept `context.Context` as the first parameter on all functions that do I/O or block.
- Write paths (sending data to a backend, writing a response) must respect context
  cancellation.
- Read paths should respect context cancellation where practical.
- Defers and cleanup paths may use `context.WithoutCancel` or `context.Background()` when the
  parent context may already be cancelled but the cleanup must complete (e.g. returning a
  connection to a pool, flushing a write buffer).

