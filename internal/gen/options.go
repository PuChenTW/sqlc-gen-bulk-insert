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

	// EmitInterface controls whether a dedicated interface file declaring every
	// generated bulk method is produced. Defaults to false.
	// When true, the interface is always written to its own file
	// (toSnakeCase(InterfaceName) + ".go") regardless of SplitBy, so that mock
	// libraries (gomock/mockgen, mockery, …) can be run against it.
	EmitInterface bool `json:"emit_interface"`

	// InterfaceName is the name of the generated interface.
	// Defaults to "BulkQuerier". Only used when EmitInterface is true.
	InterfaceName string `json:"interface_name"`

	// EmitCombinedInterface controls whether a combined interface file is
	// produced. The combined interface embeds sqlc's own Querier interface
	// (see BaseQuerierName) and inlines every generated bulk method, giving a
	// single interface that covers the whole data-access layer. Defaults to
	// false. It is independent of EmitInterface and does not require it.
	//
	// Prerequisite: sqlc-gen-go must be configured with emit_interface: true so
	// that the embedded Querier interface exists in the same output package.
	EmitCombinedInterface bool `json:"emit_combined_interface"`

	// CombinedInterfaceName is the name of the combined interface.
	// Defaults to "ExtQuerier". Only used when EmitCombinedInterface is true.
	CombinedInterfaceName string `json:"combined_interface_name"`

	// BaseQuerierName is the name of the sqlc-generated interface that the
	// combined interface embeds. Defaults to "Querier" (the name sqlc-gen-go
	// always uses). Only used when EmitCombinedInterface is true.
	BaseQuerierName string `json:"base_querier_name"`
}

// parseOptions deserialises JSON plugin options.
// Returns an error if the JSON is malformed, the required "package" key is
// absent / empty, or "split_by" is not a recognised value.
func parseOptions(data []byte) (*Options, error) {
	opts := &Options{
		OutFilename:           "bulk_insert.go",
		SplitBy:               "single",
		InterfaceName:         "BulkQuerier",
		CombinedInterfaceName: "ExtQuerier",
		BaseQuerierName:       "Querier",
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
	if opts.InterfaceName == "" {
		opts.InterfaceName = "BulkQuerier"
	}
	if opts.CombinedInterfaceName == "" {
		opts.CombinedInterfaceName = "ExtQuerier"
	}
	if opts.BaseQuerierName == "" {
		opts.BaseQuerierName = "Querier"
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
