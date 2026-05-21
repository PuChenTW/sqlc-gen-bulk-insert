package gen

import (
	"context"
	"strings"
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// ─── Options tests ────────────────────────────────────────────────────────────

func TestParseOptions_Default(t *testing.T) {
	opts, err := parseOptions([]byte(`{"package":"db"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Package != "db" {
		t.Errorf("Package = %q, want %q", opts.Package, "db")
	}
	if opts.OutFilename != "bulk_insert.go" {
		t.Errorf("OutFilename = %q, want %q", opts.OutFilename, "bulk_insert.go")
	}
}

func TestParseOptions_CustomFilename(t *testing.T) {
	opts, err := parseOptions([]byte(`{"package":"mydb","out_filename":"custom.go"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.OutFilename != "custom.go" {
		t.Errorf("OutFilename = %q, want %q", opts.OutFilename, "custom.go")
	}
}

func TestParseOptions_MissingPackage(t *testing.T) {
	_, err := parseOptions([]byte(`{"out_filename":"out.go"}`))
	if err == nil {
		t.Error("expected error for missing package, got nil")
	}
}

func TestParseOptions_EmptyBytes(t *testing.T) {
	_, err := parseOptions(nil)
	if err == nil {
		t.Error("expected error when no options given, got nil")
	}
}

func TestParseOptions_InvalidJSON(t *testing.T) {
	_, err := parseOptions([]byte(`{bad json}`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseOptions_SplitBy(t *testing.T) {
	for _, valid := range []string{"single", "file", "query"} {
		opts, err := parseOptions([]byte(`{"package":"db","split_by":"` + valid + `"}`))
		if err != nil {
			t.Errorf("split_by=%q: unexpected error: %v", valid, err)
			continue
		}
		if opts.SplitBy != valid {
			t.Errorf("split_by=%q: got %q", valid, opts.SplitBy)
		}
	}
}

func TestParseOptions_SplitByDefault(t *testing.T) {
	opts, err := parseOptions([]byte(`{"package":"db"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.SplitBy != "single" {
		t.Errorf("default SplitBy = %q, want %q", opts.SplitBy, "single")
	}
}

func TestParseOptions_SplitByInvalid(t *testing.T) {
	_, err := parseOptions([]byte(`{"package":"db","split_by":"per_table"}`))
	if err == nil {
		t.Error("expected error for unknown split_by, got nil")
	}
}

// ─── isInsertQuery tests ──────────────────────────────────────────────────────

func TestIsInsertQuery(t *testing.T) {
	insertTable := &plugin.Identifier{Name: "users"}

	tests := []struct {
		name string
		q    *plugin.Query
		want bool
	}{
		{
			name: "exec INSERT",
			q:    &plugin.Query{Cmd: ":exec", InsertIntoTable: insertTable},
			want: true,
		},
		{
			name: "execresult INSERT",
			q:    &plugin.Query{Cmd: ":execresult", InsertIntoTable: insertTable},
			want: true,
		},
		{
			name: "execrows INSERT",
			q:    &plugin.Query{Cmd: ":execrows", InsertIntoTable: insertTable},
			want: true,
		},
		{
			name: "many INSERT (not a no-output cmd)",
			q:    &plugin.Query{Cmd: ":many", InsertIntoTable: insertTable},
			want: false,
		},
		{
			name: "one INSERT (not a no-output cmd)",
			q:    &plugin.Query{Cmd: ":one", InsertIntoTable: insertTable},
			want: false,
		},
		{
			name: "exec SELECT (no InsertIntoTable)",
			q:    &plugin.Query{Cmd: ":exec"},
			want: false,
		},
		{
			name: "nil query cmd",
			q:    &plugin.Query{InsertIntoTable: insertTable},
			want: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isInsertQuery(tc.q); got != tc.want {
				t.Errorf("isInsertQuery = %v, want %v", got, tc.want)
			}
		})
	}
}

// ─── toPascalCase tests ───────────────────────────────────────────────────────

func TestToPascalCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"name", "Name"},
		{"user_id", "UserID"},
		{"created_at", "CreatedAt"},
		{"id", "ID"},
		{"email_address", "EmailAddress"},
		{"user_email", "UserEmail"},
		{"", "_"},
		{"_", "_"},
		{"a_b_c", "ABC"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got := toPascalCase(tc.in)
			if got != tc.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ─── extractInsertPrefix tests ────────────────────────────────────────────────

func TestExtractInsertPrefix(t *testing.T) {
	tests := []struct {
		input  string
		want   string
		wantOK bool
	}{
		{
			input:  "INSERT INTO users (name, email) VALUES (?, ?)",
			want:   "INSERT INTO users (name, email) VALUES ",
			wantOK: true,
		},
		{
			input:  "INSERT INTO users (name) values (?)",
			want:   "INSERT INTO users (name) values ",
			wantOK: true,
		},
		{
			input:  "insert into t (a) VALUES (?)",
			want:   "insert into t (a) VALUES ",
			wantOK: true,
		},
		{
			input:  "SELECT * FROM users WHERE id = ?",
			wantOK: false,
		},
		{
			input:  "",
			wantOK: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := extractInsertPrefix(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("prefix = %q, want %q", got, tc.want)
			}
		})
	}
}

// ─── buildPlaceholder tests ───────────────────────────────────────────────────

func TestBuildPlaceholder(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "(?)"},
		{2, "(?, ?)"},
		{3, "(?, ?, ?)"},
		{0, "()"},
	}
	for _, tc := range tests {
		got := buildPlaceholder(tc.n)
		if got != tc.want {
			t.Errorf("buildPlaceholder(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}

// ─── toSnakeCase / filename helpers tests ─────────────────────────────────────

func TestToSnakeCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"BulkInsertUser", "bulk_insert_user"},
		{"BulkInsertProduct", "bulk_insert_product"},
		{"BulkInsertID", "bulk_insert_id"},
		{"BulkInsert", "bulk_insert"},
	}
	for _, tc := range tests {
		got := toSnakeCase(tc.in)
		if got != tc.want {
			t.Errorf("toSnakeCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSourceFileToOutName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"users.sql", "bulk_users.go"},
		{"product.sql", "bulk_product.go"},
		{"queries/users.sql", "bulk_users.go"},
		{"path/to/order_items.sql", "bulk_order_items.go"},
		{"", "bulk_queries.go"},
	}
	for _, tc := range tests {
		got := sourceFileToOutName(tc.in)
		if got != tc.want {
			t.Errorf("sourceFileToOutName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestQueryFuncToOutName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"BulkInsertUser", "bulk_insert_user.go"},
		{"BulkInsertProduct", "bulk_insert_product.go"},
	}
	for _, tc := range tests {
		got := queryFuncToOutName(tc.in)
		if got != tc.want {
			t.Errorf("queryFuncToOutName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ─── Generate integration tests ───────────────────────────────────────────────

// makeColumn creates a *plugin.Column for test fixtures.
func makeColumn(name, typeName string, notNull bool) *plugin.Column {
	return &plugin.Column{
		Name:    name,
		NotNull: notNull,
		Type:    &plugin.Identifier{Name: typeName},
	}
}

func TestGenerate_Empty(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries:       []*plugin.Query{},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(resp.Files))
	}
}

func TestGenerate_SkipsNonInsert(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "ListUsers",
				Cmd:  ":many",
				// InsertIntoTable is nil → SELECT query
				Params: []*plugin.Parameter{
					{Column: makeColumn("id", "bigint", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(resp.Files))
	}
}

func TestGenerate_SkipsZeroParams(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name:            "InsertDefault",
				Cmd:             ":exec",
				Text:            "INSERT INTO events (created_at) VALUES (NOW())",
				InsertIntoTable: &plugin.Identifier{Name: "events"},
				Params:          nil, // no params
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(resp.Files))
	}
}

func TestGenerate_MultiParam(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertUser",
				Cmd:  ":exec",
				Text: "INSERT INTO users (name, email) VALUES (?, ?)",
				InsertIntoTable: &plugin.Identifier{Name: "users"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("name", "varchar", true)},
					{Column: makeColumn("email", "varchar", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	content := string(resp.Files[0].Contents)

	assertContains(t, content, "func (q *Queries) BulkInsertUser(")
	assertContains(t, content, "[]InsertUserParams")
	assertContains(t, content, `"(?, ?)"`)
	assertContains(t, content, "arg.Name, arg.Email")
	assertContains(t, content, "strings.Join")
	assertContains(t, content, "q.db.ExecContext")
}

func TestGenerate_SingleParam(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertAuditLog",
				Cmd:  ":exec",
				Text: "INSERT INTO audit_log (user_id) VALUES (?)",
				InsertIntoTable: &plugin.Identifier{Name: "audit_log"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("user_id", "bigint", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	content := string(resp.Files[0].Contents)

	assertContains(t, content, "func (q *Queries) BulkInsertAuditLog(")
	assertContains(t, content, "[]int64")
	assertContains(t, content, `"(?)"`)
	// Single param uses 'arg' directly, not 'arg.FieldName'
	assertContains(t, content, "valueArgs = append(valueArgs, arg)")
}

func TestGenerate_MultipleQueries(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertUser",
				Cmd:  ":exec",
				Text: "INSERT INTO users (name) VALUES (?)",
				InsertIntoTable: &plugin.Identifier{Name: "users"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("name", "varchar", true)},
				},
			},
			{
				Name: "InsertPost",
				Cmd:  ":execrows",
				Text: "INSERT INTO posts (title, body) VALUES (?, ?)",
				InsertIntoTable: &plugin.Identifier{Name: "posts"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("title", "varchar", true)},
					{Column: makeColumn("body", "text", true)},
				},
			},
			{
				Name: "ListUsers",
				Cmd:  ":many",
				// no InsertIntoTable → should be skipped
				Params: []*plugin.Parameter{
					{Column: makeColumn("id", "bigint", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	content := string(resp.Files[0].Contents)

	assertContains(t, content, "BulkInsertUser")
	assertContains(t, content, "BulkInsertPost")
	if strings.Contains(content, "ListUsers") {
		t.Error("ListUsers should have been skipped")
	}
	// Single param → direct type
	assertContains(t, content, "[]string")
	// Multi param → struct
	assertContains(t, content, "[]InsertPostParams")
}

func TestGenerate_TimeImport(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertEvent",
				Cmd:  ":exec",
				Text: "INSERT INTO events (created_at) VALUES (?)",
				InsertIntoTable: &plugin.Identifier{Name: "events"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("created_at", "datetime", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(resp.Files[0].Contents)

	assertContains(t, content, `"time"`)
	assertContains(t, content, "[]time.Time")
}

func TestGenerate_SQLNullImport(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertItem",
				Cmd:  ":exec",
				Text: "INSERT INTO items (value) VALUES (?)",
				InsertIntoTable: &plugin.Identifier{Name: "items"},
				Params: []*plugin.Parameter{
					// nullable varchar → sql.NullString
					{Column: makeColumn("value", "varchar", false)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(resp.Files[0].Contents)

	assertContains(t, content, `"database/sql"`)
	assertContains(t, content, "[]sql.NullString")
}

// TestGenerate_MultiParamNoSpuriousImports verifies that a multi-param query
// whose columns include datetime/json/nullable types does NOT cause "time",
// "encoding/json", or "database/sql" to appear in the generated file.
// Those types live inside the sqlc-generated params struct, not in our file.
func TestGenerate_MultiParamNoSpuriousImports(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertEvent",
				Cmd:  ":exec",
				Text: "INSERT INTO events (name, started_at, metadata, count) VALUES (?, ?, ?, ?)",
				InsertIntoTable: &plugin.Identifier{Name: "events"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("name", "varchar", true)},
					{Column: makeColumn("started_at", "datetime", true)},  // time.Time
					{Column: makeColumn("metadata", "json", true)},        // json.RawMessage
					{Column: makeColumn("count", "int", false)},           // sql.NullInt32
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(resp.Files[0].Contents)

	// The params struct (InsertEventParams) is defined by sqlc's codegen, not ours.
	// Our file must NOT import packages that are only needed inside that struct.
	assertNotContains(t, content, `"time"`)
	assertNotContains(t, content, `"encoding/json"`)
	assertNotContains(t, content, `"database/sql"`)

	// Sanity: the function and struct reference are still present.
	assertContains(t, content, "BulkInsertEvent")
	assertContains(t, content, "[]InsertEventParams")
}

func TestGenerate_CustomFilename(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db","out_filename":"custom_bulk.go"}`),
		Queries: []*plugin.Query{
			{
				Name: "InsertUser",
				Cmd:  ":exec",
				Text: "INSERT INTO users (name) VALUES (?)",
				InsertIntoTable: &plugin.Identifier{Name: "users"},
				Params: []*plugin.Parameter{
					{Column: makeColumn("name", "varchar", true)},
				},
			},
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Files[0].Name != "custom_bulk.go" {
		t.Errorf("filename = %q, want %q", resp.Files[0].Name, "custom_bulk.go")
	}
}

func TestGenerate_MissingPackageOption(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{}`),
	}
	_, err := Generate(context.Background(), req)
	if err == nil {
		t.Error("expected error for missing package option, got nil")
	}
}

// ─── split_by tests ───────────────────────────────────────────────────────────

// makeQuery is a shorthand for building a two-param insert query with a source file.
func makeQuery(name, filename string) *plugin.Query {
	return &plugin.Query{
		Name:            name,
		Cmd:             ":exec",
		Text:            "INSERT INTO t (a, b) VALUES (?, ?)",
		Filename:        filename,
		InsertIntoTable: &plugin.Identifier{Name: "t"},
		Params: []*plugin.Parameter{
			{Column: makeColumn("a", "varchar", true)},
			{Column: makeColumn("b", "varchar", true)},
		},
	}
}

func TestGenerate_SplitByFile(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db","split_by":"file"}`),
		Queries: []*plugin.Query{
			makeQuery("InsertUser", "users.sql"),
			makeQuery("InsertProfile", "users.sql"),    // same source file
			makeQuery("InsertProduct", "products.sql"), // different source file
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(resp.Files), fileNames(resp.Files))
	}

	byName := indexByName(resp.Files)

	usersFile, ok := byName["bulk_users.go"]
	if !ok {
		t.Fatalf("missing bulk_users.go; got %v", fileNames(resp.Files))
	}
	assertContains(t, string(usersFile), "BulkInsertUser")
	assertContains(t, string(usersFile), "BulkInsertProfile")
	assertNotContains(t, string(usersFile), "BulkInsertProduct")

	productsFile, ok := byName["bulk_products.go"]
	if !ok {
		t.Fatalf("missing bulk_products.go; got %v", fileNames(resp.Files))
	}
	assertContains(t, string(productsFile), "BulkInsertProduct")
	assertNotContains(t, string(productsFile), "BulkInsertUser")
}

func TestGenerate_SplitByQuery(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db","split_by":"query"}`),
		Queries: []*plugin.Query{
			makeQuery("InsertUser", "users.sql"),
			makeQuery("InsertProduct", "products.sql"),
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(resp.Files), fileNames(resp.Files))
	}

	byName := indexByName(resp.Files)

	userFile, ok := byName["bulk_insert_user.go"]
	if !ok {
		t.Fatalf("missing bulk_insert_user.go; got %v", fileNames(resp.Files))
	}
	assertContains(t, string(userFile), "BulkInsertUser")
	assertNotContains(t, string(userFile), "BulkInsertProduct")

	productFile, ok := byName["bulk_insert_product.go"]
	if !ok {
		t.Fatalf("missing bulk_insert_product.go; got %v", fileNames(resp.Files))
	}
	assertContains(t, string(productFile), "BulkInsertProduct")
	assertNotContains(t, string(productFile), "BulkInsertUser")
}

func TestGenerate_SplitBySingleExplicit(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db","split_by":"single","out_filename":"all_bulk.go"}`),
		Queries: []*plugin.Query{
			makeQuery("InsertUser", "users.sql"),
			makeQuery("InsertProduct", "products.sql"),
		},
	}
	resp, err := Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(resp.Files))
	}
	if resp.Files[0].Name != "all_bulk.go" {
		t.Errorf("filename = %q, want %q", resp.Files[0].Name, "all_bulk.go")
	}
	content := string(resp.Files[0].Contents)
	assertContains(t, content, "BulkInsertUser")
	assertContains(t, content, "BulkInsertProduct")
}

func TestGenerate_SplitByInvalidOption(t *testing.T) {
	req := &plugin.GenerateRequest{
		PluginOptions: []byte(`{"package":"db","split_by":"table"}`),
	}
	_, err := Generate(context.Background(), req)
	if err == nil {
		t.Error("expected error for unknown split_by, got nil")
	}
}

// ─── helpers for split tests ──────────────────────────────────────────────────

func fileNames(files []*plugin.File) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Name
	}
	return names
}

func indexByName(files []*plugin.File) map[string][]byte {
	m := make(map[string][]byte, len(files))
	for _, f := range files {
		m[f.Name] = f.Contents
	}
	return m
}

// ─── Type-map tests ───────────────────────────────────────────────────────────

func TestGoType_BasicTypes(t *testing.T) {
	tests := []struct {
		col        *plugin.Column
		wantType   string
		wantImport string
	}{
		{makeColumn("c", "bigint", true), "int64", ""},
		{makeColumn("c", "bigint", false), "sql.NullInt64", "database/sql"},
		{makeColumn("c", "int", true), "int32", ""},
		{makeColumn("c", "varchar", true), "string", ""},
		{makeColumn("c", "varchar", false), "sql.NullString", "database/sql"},
		{makeColumn("c", "bool", true), "bool", ""},
		{makeColumn("c", "datetime", true), "time.Time", "time"},
		{makeColumn("c", "datetime", false), "sql.NullTime", "database/sql"},
		{makeColumn("c", "json", true), "json.RawMessage", "encoding/json"},
		{makeColumn("c", "blob", true), "[]byte", ""},
		{makeColumn("c", "blob", false), "[]byte", ""},
		{makeColumn("c", "unknown_type", true), "any", ""},
	}
	for _, tc := range tests {
		gotType, gotImport := goType(tc.col)
		if gotType != tc.wantType {
			t.Errorf("goType(%q) type = %q, want %q", tc.col.Type.Name, gotType, tc.wantType)
		}
		if gotImport != tc.wantImport {
			t.Errorf("goType(%q) import = %q, want %q", tc.col.Type.Name, gotImport, tc.wantImport)
		}
	}
}

func TestGoType_Tinyint1_Bool(t *testing.T) {
	col := &plugin.Column{
		Name:    "active",
		NotNull: true,
		Type:    &plugin.Identifier{Name: "tinyint"},
		Length:  1,
	}
	gotType, gotImport := goType(col)
	if gotType != "bool" {
		t.Errorf("tinyint(1) type = %q, want %q", gotType, "bool")
	}
	if gotImport != "" {
		t.Errorf("tinyint(1) import = %q, want %q", gotImport, "")
	}
}

func TestGoType_UnsignedBigint(t *testing.T) {
	col := &plugin.Column{
		Name:     "id",
		NotNull:  true,
		Unsigned: true,
		Type:     &plugin.Identifier{Name: "bigint"},
	}
	gotType, _ := goType(col)
	if gotType != "uint64" {
		t.Errorf("unsigned bigint type = %q, want %q", gotType, "uint64")
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("generated code does not contain %q\n\n--- Generated ---\n%s", substr, content)
	}
}

func assertNotContains(t *testing.T, content, substr string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("generated code must not contain %q\n\n--- Generated ---\n%s", substr, content)
	}
}
