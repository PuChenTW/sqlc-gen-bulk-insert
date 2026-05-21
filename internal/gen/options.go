package gen

import (
	"encoding/json"
	"fmt"
	"go/token"
	"path/filepath"
)

// Options holds the plugin-specific configuration parsed from the PluginOptions
// JSON bytes supplied via sqlc.yaml's codegen options block.
type Options struct {
	// Package is the Go package name to use in the generated file.
	// Required — there is no default because an incorrect package name
	// would cause silent compilation failures.
	Package string `json:"package"`

	// OutFilename is the name of the generated file placed in the codegen
	// output directory.  Defaults to "bulk_insert.go".
	// Only used when SplitBy is "single" (the default).
	OutFilename string `json:"out_filename"`

	// SplitBy controls how many output files are produced.
	//
	//   "single" (default) — one file for all bulk functions (out_filename).
	//   "file"             — one file per source .sql file.
	//                        e.g. users.sql → bulk_users.go
	//   "query"            — one file per generated function.
	//                        e.g. BulkInsertUser → bulk_insert_user.go
	SplitBy string `json:"split_by"`
}

// parseOptions deserialises JSON plugin options.
// Returns an error if the JSON is malformed, the required "package" key is
// absent / empty, or "split_by" is not a recognised value.
func parseOptions(data []byte) (*Options, error) {
	opts := &Options{
		OutFilename: "bulk_insert.go",
		SplitBy:     "single",
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, opts); err != nil {
			return nil, fmt.Errorf("sqlc-gen-bulk-insert: parsing plugin options: %w", err)
		}
	}
	if opts.Package == "" {
		return nil, fmt.Errorf("sqlc-gen-bulk-insert: plugin option \"package\" is required")
	}
	// token.IsIdentifier rejects keywords and non-identifiers.
	// The blank identifier "_" also passes IsIdentifier but is not a valid
	// package name per the Go spec ("must not be the blank identifier").
	if !token.IsIdentifier(opts.Package) || opts.Package == "_" {
		return nil, fmt.Errorf(
			"sqlc-gen-bulk-insert: plugin option \"package\" %q is not a valid Go identifier",
			opts.Package,
		)
	}
	if opts.OutFilename == "" {
		opts.OutFilename = "bulk_insert.go"
	}
	// Reject directory components and the special "." name.
	base := filepath.Base(opts.OutFilename)
	if base != opts.OutFilename || base == "." {
		return nil, fmt.Errorf(
			"sqlc-gen-bulk-insert: out_filename %q must be a plain filename, not a path",
			opts.OutFilename,
		)
	}
	switch opts.SplitBy {
	case "", "single":
		opts.SplitBy = "single"
	case "file", "query":
		// valid
	default:
		return nil, fmt.Errorf(
			"sqlc-gen-bulk-insert: unknown split_by value %q (want \"single\", \"file\", or \"query\")",
			opts.SplitBy,
		)
	}
	return opts, nil
}
