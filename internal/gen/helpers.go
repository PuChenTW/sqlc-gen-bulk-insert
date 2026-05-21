package gen

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// toPascalCase converts a snake_case column name to PascalCase.
//
// Special handling:
//   - The segment "id" is uppercased to "ID" (Go acronym convention).
//   - Non-letter, non-digit runes are treated as word separators.
//   - A result that would start with a digit is prefixed with "_".
//   - An empty result falls back to "_".
func toPascalCase(s string) string {
	// Normalise separators: replace any non-letter, non-digit rune with '_'.
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, s)

	var out strings.Builder
	for _, word := range strings.Split(s, "_") {
		if word == "" {
			continue
		}
		if strings.EqualFold(word, "id") {
			out.WriteString("ID")
			continue
		}
		r, size := utf8.DecodeRuneInString(word)
		out.WriteRune(unicode.ToUpper(r))
		out.WriteString(word[size:])
	}

	result := out.String()
	if result == "" {
		return "_"
	}
	r, _ := utf8.DecodeRuneInString(result)
	if unicode.IsDigit(r) {
		return "_" + result
	}
	return result
}

// lowerFirst returns s with the first Unicode rune lowercased.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[size:]
}

// extractInsertPrefix returns the portion of queryText up to and including the
// "VALUES" keyword, with exactly one trailing space appended.
//
//	"INSERT INTO users (name, email) VALUES (?, ?)"
//	  → ("INSERT INTO users (name, email) VALUES ", true)
//
// The search is case-insensitive.  Returns ("", false) if VALUES is not found.
func extractInsertPrefix(queryText string) (string, bool) {
	upper := strings.ToUpper(queryText)
	idx := strings.Index(upper, "VALUES")
	if idx < 0 {
		return "", false
	}
	// Take the original text (preserving original casing / backtick quoting).
	prefix := strings.TrimRight(queryText[:idx+len("VALUES")], " \t\r\n") + " "
	return prefix, true
}

// buildPlaceholder returns the per-row VALUES placeholder for n parameters.
//
//	buildPlaceholder(1) → "(?)"
//	buildPlaceholder(2) → "(?, ?)"
//	buildPlaceholder(3) → "(?, ?, ?)"
func buildPlaceholder(n int) string {
	if n <= 0 {
		return "()"
	}
	marks := make([]string, n)
	for i := range marks {
		marks[i] = "?"
	}
	return "(" + strings.Join(marks, ", ") + ")"
}

// toSnakeCase converts a PascalCase or camelCase identifier to snake_case.
//
//	"BulkInsertUser"  → "bulk_insert_user"
//	"BulkInsertID"    → "bulk_insert_id"
func toSnakeCase(s string) string {
	var out strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			// Insert underscore before an uppercase letter when it is:
			//   • not the first character, AND
			//   • either preceded by a lowercase letter, OR
			//     followed by a lowercase letter (handles "ID" → "id" not "i_d")
			if i > 0 {
				prev := runes[i-1]
				next := rune(0)
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if unicode.IsLower(prev) || (unicode.IsLower(next) && unicode.IsUpper(prev)) {
					out.WriteByte('_')
				}
			}
			out.WriteRune(unicode.ToLower(r))
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// sourceFileToOutName derives an output filename from the source SQL filename.
//
//	"queries/users.sql"  → "bulk_users.go"
//	"product.sql"        → "bulk_product.go"
//
// The directory component and all extensions are stripped; "bulk_" is prepended
// and ".go" is appended.
func sourceFileToOutName(sqlFile string) string {
	// Strip directory.
	base := sqlFile
	if idx := strings.LastIndexAny(base, "/\\"); idx >= 0 {
		base = base[idx+1:]
	}
	// Strip all extensions (e.g. "users.sql" → "users").
	if dot := strings.Index(base, "."); dot >= 0 {
		base = base[:dot]
	}
	if base == "" {
		base = "queries"
	}
	return "bulk_" + base + ".go"
}

// queryFuncToOutName derives an output filename from a generated function name.
//
//	"BulkInsertUser"     → "bulk_insert_user.go"
//	"BulkInsertProduct"  → "bulk_insert_product.go"
func queryFuncToOutName(funcName string) string {
	return toSnakeCase(funcName) + ".go"
}
