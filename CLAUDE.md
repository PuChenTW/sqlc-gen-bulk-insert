# CLAUDE.md — sqlc-gen-bulk-insert

## Project overview

`sqlc-gen-bulk-insert` is a **sqlc process plugin** written in Go.
It reads a `GenerateRequest` from sqlc via stdin (protobuf), finds INSERT queries,
and emits a single Go file containing `Bulk<QueryName>` helper functions that batch
multiple rows into one `INSERT … VALUES (?,?),(?,?)` statement.

Target database: **MySQL / MariaDB** (`?` positional placeholders).

## Repository layout

```
.
├── main.go                        # Entry point: codegen.Run(gen.Generate)
├── internal/
│   └── gen/
│       ├── generator.go           # Core logic: Generate(), isInsertQuery(), buildBulkFunc(), template
│       ├── generator_test.go      # All tests live here
│       ├── helpers.go             # toPascalCase, extractInsertPrefix, buildPlaceholder, lowerFirst
│       ├── options.go             # Options struct + parseOptions()
│       └── typemap.go             # MySQL SQL type → Go type mapping
├── example/
│   ├── sqlc.yaml                  # Example plugin configuration
│   ├── schema.sql                 # Example schema
│   └── query.sql                  # Example queries (eligible + ineligible)
├── go.mod                         # module github.com/puchentw/sqlc-gen-bulk-insert
└── README.md
```

## Common commands

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Build the plugin binary
go build -o sqlc-gen-bulk-insert .

# Vet
go vet ./...

# Tidy dependencies
go mod tidy
```

## Architecture

### Data flow

```
sqlc stdin (protobuf GenerateRequest)
  → codegen.Run (plugin-sdk-go)
    → gen.Generate(ctx, req)
      → parseOptions(req.PluginOptions)     # JSON options
      → filter req.Queries via isInsertQuery
      → buildBulkFunc(q) per eligible query → BulkFunc struct
      → renderTemplate(fileData)             # text/template
      → format.Source(buf)                   # go/format
  → stdout (protobuf GenerateResponse with one File)
```

### Key types

**`BulkFunc`** (`generator.go`) — template data for one generated function:
- `FuncName` / `ConstName` / `ParamsType`
- `InsertPrefix` — SQL up to "VALUES " extracted from query text
- `Placeholder` — per-row `(?, ?, ?)` string
- `ValueArgsLine` — pre-built append expression, e.g. `arg.Name, arg.Email`
- `NeedsTime` / `NeedsJSON` / `NeedsSQL` — import flags

**`Options`** (`options.go`):
- `Package` (required) — Go package name for the generated file
- `OutFilename` (default: `bulk_insert.go`) — used only when `SplitBy` is `"single"`
- `SplitBy` (default: `"single"`) — controls output file count:
  - `"single"` — one file (`out_filename`)
  - `"file"` — one file per source `.sql` file (`bulk_<sqlfile>.go`)
  - `"query"` — one file per generated function (`bulk_insert_user.go`)

### Query selection rules

A query is eligible if **all** hold:
1. `query.InsertIntoTable != nil` — sqlc only sets this for INSERT statements
2. `query.Cmd` ∈ `{":exec", ":execresult", ":execrows"}`
3. `len(query.Params) >= 1`

### Multi-param vs single-param

| Params | Generated slice type | Field access in append |
|--------|---------------------|----------------------|
| 2+ | `[]<QueryName>Params` (sqlc-generated struct) | `arg.Name, arg.Email, …` |
| 1  | `[]<GoType>` (from typemap) | `arg` |

### Template & imports

The template in `generator.go` uses `{{- if .NeedsTime}}` / `{{- if .NeedsSQL}}` /
`{{- if .NeedsJSON}}` guards so that `go/format` never sees an unused import.
`"context"` and `"strings"` are always imported.

`any` is used instead of `interface{}` throughout generated code (Go 1.18+).

## Plugin SDK

Dependency: `github.com/sqlc-dev/plugin-sdk-go v1.23.0`

Key types used:
- `plugin.GenerateRequest` — input from sqlc (`Settings`, `Queries`, `PluginOptions`)
- `plugin.GenerateResponse` — output (`Files []*plugin.File`)
- `plugin.Query` — `Text`, `Name`, `Cmd`, `Params`, `InsertIntoTable`
- `plugin.Parameter` — `Number`, `Column`
- `plugin.Column` — `Name`, `Type` (*Identifier), `NotNull`, `Unsigned`, `Length`
- `sdk.DataType(col.Type)` — returns the SQL type name string (handles schema-prefixed enums)

Note: `plugin.Settings` in v1.23.0 has **no Go-specific sub-field** — the package
name must always come from `PluginOptions`.

## Adding new features

### New SQL type mappings
Edit the `switch` in `internal/gen/typemap.go` → `goType()`.
Add a test case in `TestGoType_BasicTypes` in `generator_test.go`.

### New plugin option
1. Add a field to `Options` in `options.go` with a `json:"..."` tag.
2. Document it in `README.md`.
3. Wire it into `Generate()` or `buildBulkFunc()` in `generator.go`.

### Changing generated code shape
Edit the `bulkInsertTmpl` constant in `generator.go`.
Run `go test ./...` — the integration tests check specific substrings in the output.
