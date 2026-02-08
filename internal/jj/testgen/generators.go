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

// WithNormal converts the change ID from reverse-hex [k-z] to normal hex [0-9a-f].
// Preserves any /N version suffix if present, so order of transformers doesn't matter.
func WithNormal(gen *rapid.Generator[string]) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		id, suffix := preserveVersion(gen.Draw(t, "id"))
		return reverseHexToHex(id) + suffix
	})
}

// WithShort truncates the change ID to 8-12 characters.
// Preserves any /N version suffix if present, so order of transformers doesn't matter.
func WithShort(gen *rapid.Generator[string]) *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		id, suffix := preserveVersion(gen.Draw(t, "id"))

		// Truncate the base ID
		length := min(rapid.IntRange(8, 12).Draw(t, "length"), len(id))

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
