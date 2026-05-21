package gen

import (
	"encoding/json"
	"fmt"
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
	if opts.OutFilename == "" {
		opts.OutFilename = "bulk_insert.go"
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
