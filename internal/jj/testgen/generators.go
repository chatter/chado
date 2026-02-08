// Package testgen provides rapid generators for jj CLI output strings.
package testgen

import (
	"fmt"
	"strings"

	"pgregory.net/rapid"
)

// ChangeIDOption transforms a ChangeID generator.
type ChangeIDOption func(*rapid.Generator[string]) *rapid.Generator[string]

// ChangeID generates a jj change ID string.
//
// By default, generates a full 32-character reverse-hex ID using [k-z].
// Options are transformers that modify the generator.
//
// Examples:
//
//	ChangeID()                       // "mllvplstlztypzupmnsyoxsmnsozzpuz"
//	ChangeID(WithShort)              // "mllvplst" (8-12 chars)
//	ChangeID(WithVersion)            // "mllvplstlztypzupmnsyoxsmnsozzpuz/42"
//	ChangeID(WithShort, WithVersion) // "mllvplst/7"
func ChangeID(opts ...ChangeIDOption) *rapid.Generator[string] {
	gen := rapid.StringMatching(`[k-z]{32}`)
	for _, opt := range opts {
		gen = opt(gen)
	}
	return gen
}

// CommitID generates a Git commit hash (40-character hex).
// Accepts the same options as ChangeID (e.g., WithShort).
func CommitID(opts ...ChangeIDOption) *rapid.Generator[string] {
	gen := rapid.StringMatching(`[0-9a-f]{40}`)
	for _, opt := range opts {
		gen = opt(gen)
	}
	return gen
}

// OperationID generates a jj operation ID (128-character hex).
// Accepts the same options as ChangeID (e.g., WithShort).
func OperationID(opts ...ChangeIDOption) *rapid.Generator[string] {
	gen := rapid.StringMatching(`[0-9a-f]{128}`)
	for _, opt := range opts {
		gen = opt(gen)
	}
	return gen
}

// WithNormal converts the change ID from reverse-hex [k-z] to normal hex [0-9a-f].
// Preserves any /N version suffix if present, so order of transformers doesn't matter.
func WithNormal(gen *rapid.Generator[string]) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		id, suffix := preserveVersion(gen.Draw(t, "id"))
		return reverseHexToHex(id) + suffix
	})
}

// WithShort truncates the ID to a short form.
// For ChangeID (32 chars): 8-12 characters.
// For CommitID (40 chars): 7-12 characters.
// For OperationID (128 chars): exactly 12 characters.
// Preserves any /N version suffix if present, so order of transformers doesn't matter.
func WithShort(gen *rapid.Generator[string]) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		id, suffix := preserveVersion(gen.Draw(t, "id"))

		var length int
		switch len(id) {
		case 128: // OperationID - always 12
			length = 12
		case 40: // CommitID - 7-12
			length = rapid.IntRange(7, 12).Draw(t, "length")
		default: // ChangeID (32) - 8-12
			length = rapid.IntRange(8, 12).Draw(t, "length")
		}

		if length > len(id) {
			length = len(id)
		}
		return id[:length] + suffix
	})
}

// WithVersion appends a /N version suffix (e.g., "xsssnyux/2").
func WithVersion(gen *rapid.Generator[string]) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		id := gen.Draw(t, "id")
		v := rapid.IntRange(1, 99).Draw(t, "v")
		return fmt.Sprintf("%s/%d", id, v)
	})
}

// preserveVersion splits a change ID into its base and optional /N version suffix.
// This allows transformers to be order-independent by preserving the suffix through transformations.
func preserveVersion(id string) (base, suffix string) {
	if idx := strings.LastIndex(id, "/"); idx != -1 {
		return id[:idx], id[idx:]
	}
	return id, ""
}

// reverseHexToHex converts a reverse hex string [k-z] to hex [0-9a-f].
// Mapping: k→0, l→1, m→2, ..., t→9, u→a, v→b, w→c, x→d, y→e, z→f
func reverseHexToHex(revHex string) string {
	result := make([]byte, len(revHex))
	for i, c := range revHex {
		switch {
		case c >= 'k' && c <= 't':
			// k-t → 0-9
			result[i] = byte(c) - 59
		case c >= 'u' && c <= 'z':
			// u-z → a-f
			result[i] = byte(c) - 20
		default:
			panic(fmt.Sprintf("reverseHexToHex: invalid reverse-hex character %q at index %d", c, i))
		}
	}
	return string(result)
}

// PathComponent generates a valid path component (filename or directory name).
// Uses ASCII alphanumeric plus common safe characters: _ - .
func PathComponent() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_.-]{0,254}`)
}

// FilePath generates a relative POSIX file path.
// Depth 1-16 components, joined by '/'. Max path length ~4096 bytes.
func FilePath() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 16).Draw(t, "depth")
		components := make([]string, depth)
		for i := range components {
			components[i] = PathComponent().Draw(t, "component")
		}
		return strings.Join(components, "/")
	})
}

// FileStatus generates a file status string as it appears in jj diff output.
// One of "Added", "Modified", "Removed", "Copied", or "Renamed".
func FileStatus() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"Added", "Modified", "Removed", "Copied", "Renamed"})
}

// FileStatusChar generates a file status character.
// One of "M" (modified), "A" (added), "D" (removed), "C" (copied), or "R" (renamed).
func FileStatusChar() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"M", "A", "D", "C", "R"})
}

// Email generates an email-like string as jj represents them.
// Per jj docs: may be empty, may not contain @, or may contain multiple @s.
func Email() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just(""),                      // empty
		rapid.StringMatching(`[a-z]{3,10}`), // no @
		rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,10}\.[a-z]{2,4}`), // typical: user@host.com
		rapid.StringMatching(`[a-z]+@[a-z]+@[a-z]+`),                // multiple @
	)
}

// Timestamp generates an absolute timestamp "YYYY-MM-DD HH:MM:SS".
// Year range 1970-2037 (Unix 32-bit timestamp safe range).
func Timestamp() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		year := rapid.IntRange(1970, 2037).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		min := rapid.IntRange(0, 59).Draw(t, "min")
		sec := rapid.IntRange(0, 59).Draw(t, "sec")
		return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", year, month, day, hour, min, sec)
	})
}

// RelativeTimestamp generates a relative timestamp like "4 minutes ago".
func RelativeTimestamp() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		n := rapid.IntRange(1, 59).Draw(t, "n")
		unit := rapid.SampledFrom([]string{"second", "minute", "hour", "day", "week", "month", "year"}).Draw(t, "unit")
		if n > 1 {
			unit += "s"
		}
		return fmt.Sprintf("%d %s ago", n, unit)
	})
}

// GraphSymbol generates a jj log graph symbol.
// Symbols: @ (working copy), ○ (normal), ◆ (immutable), ◇ (empty), ● (hidden), × (conflict)
func GraphSymbol() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"@", "○", "◆", "◇", "●", "×"})
}
