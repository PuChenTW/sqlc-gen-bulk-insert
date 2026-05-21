package gen

import (
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"
)

// goType returns the Go type string for a MySQL/MariaDB column and an import
// hint indicating which additional package (if any) is needed:
//
//	""               – no extra import
//	"time"           – import "time"
//	"encoding/json"  – import "encoding/json"
//	"database/sql"   – import "database/sql"
//
// The mapping follows sqlc-gen-go's mysql_type.go conventions so that the
// types emitted by this plugin are consistent with sqlc's own output.
func goType(col *plugin.Column) (typeName string, importHint string) {
	if col == nil {
		return "any", ""
	}
	columnType := sdk.DataType(col.Type)
	notNull := col.NotNull
	unsigned := col.Unsigned

	switch columnType {
	// ── Boolean / tiny integer ─────────────────────────────────────────────
	case "bool", "boolean":
		if notNull {
			return "bool", ""
		}
		return "sql.NullBool", "database/sql"

	case "tinyint":
		// MySQL convention: tinyint(1) == boolean
		if col.Length == 1 {
			if notNull {
				return "bool", ""
			}
			return "sql.NullBool", "database/sql"
		}
		if notNull {
			if unsigned {
				return "uint8", ""
			}
			return "int8", ""
		}
		// nullable tinyint – use NullInt16 (smallest available)
		return "sql.NullInt16", "database/sql"

	// ── Small integers ─────────────────────────────────────────────────────
	case "smallint":
		if notNull {
			if unsigned {
				return "uint16", ""
			}
			return "int16", ""
		}
		return "sql.NullInt16", "database/sql"

	case "year":
		// YEAR is stored as a 2-byte integer
		if notNull {
			return "int16", ""
		}
		return "sql.NullInt16", "database/sql"

	// ── Medium / regular integers ──────────────────────────────────────────
	case "mediumint", "int", "integer":
		if notNull {
			if unsigned {
				return "uint32", ""
			}
			return "int32", ""
		}
		return "sql.NullInt32", "database/sql"

	// ── Big integers ───────────────────────────────────────────────────────
	case "bigint":
		if notNull {
			if unsigned {
				return "uint64", ""
			}
			return "int64", ""
		}
		return "sql.NullInt64", "database/sql"

	// ── Floating point ─────────────────────────────────────────────────────
	case "float":
		if notNull {
			return "float32", ""
		}
		// sql.NullFloat32 does not exist in database/sql; NullFloat64 is used
		// instead. The widening from float32 to float64 is safe and lossless.
		return "sql.NullFloat64", "database/sql"

	case "double", "double precision", "real":
		if notNull {
			return "float64", ""
		}
		return "sql.NullFloat64", "database/sql"

	// ── Fixed-point / decimal ──────────────────────────────────────────────
	case "decimal", "dec", "numeric", "fixed":
		if notNull {
			return "string", ""
		}
		return "sql.NullString", "database/sql"

	// ── Text / string types ────────────────────────────────────────────────
	case "char", "varchar",
		"tinytext", "text", "mediumtext", "longtext",
		"nchar", "nvarchar":
		if notNull {
			return "string", ""
		}
		return "sql.NullString", "database/sql"

	// ── Binary / blob types ────────────────────────────────────────────────
	// []byte is used for all binary types regardless of nullability;
	// use a nil slice to represent NULL.
	case "binary", "varbinary",
		"tinyblob", "blob", "mediumblob", "longblob",
		"bit":
		return "[]byte", ""

	// ── Date / time ────────────────────────────────────────────────────────
	case "date", "datetime", "timestamp":
		if notNull {
			return "time.Time", "time"
		}
		return "sql.NullTime", "database/sql"

	case "time":
		// MySQL TIME can represent duration or time-of-day; map to string.
		if notNull {
			return "string", ""
		}
		return "sql.NullString", "database/sql"

	// ── JSON ───────────────────────────────────────────────────────────────
	case "json":
		// Use json.RawMessage(nil) to represent NULL.
		return "json.RawMessage", "encoding/json"

	// ── Fallback ───────────────────────────────────────────────────────────
	default:
		return "any", ""
	}
}
