package testgen

import (
	"regexp"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// =============================================================================
// Unit Tests
// =============================================================================

func TestReverseHexToHex(t *testing.T) {
	tests := []struct {
		revHex   string
		expected string
	}{
		{"k", "0"},
		{"t", "9"},
		{"u", "a"},
		{"z", "f"},
		{"klmnopqrstuvwxyz", "0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.revHex, func(t *testing.T) {
			result := reverseHexToHex(tt.revHex)
			if result != tt.expected {
				t.Errorf("reverseHexToHex(%q) = %q, want %q", tt.revHex, result, tt.expected)
			}
		})
	}
}

func TestChangeID_DefaultFull(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID().Draw(t, "id")

		// Should be 32 chars
		if len(id) != 32 {
			t.Fatalf("expected 32 chars, got %d: %q", len(id), id)
		}

		// Should only contain [k-z]
		if !regexp.MustCompile(`^[k-z]{32}$`).MatchString(id) {
			t.Fatalf("expected [k-z]{32}, got %q", id)
		}
	})
}

func TestChangeID_WithShort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithShort).Draw(t, "id")

		// Should be 8-12 chars
		if len(id) < 8 || len(id) > 12 {
			t.Fatalf("expected 8-12 chars, got %d: %q", len(id), id)
		}

		// Should only contain [k-z]
		if !regexp.MustCompile(`^[k-z]+$`).MatchString(id) {
			t.Fatalf("expected [k-z]+, got %q", id)
		}
	})
}

func TestChangeID_WithVersion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithVersion).Draw(t, "id")

		// Should contain /
		if !strings.Contains(id, "/") {
			t.Fatalf("expected version suffix with /, got %q", id)
		}

		// Should match pattern: 32 chars + /N
		if !regexp.MustCompile(`^[k-z]{32}/\d{1,2}$`).MatchString(id) {
			t.Fatalf("expected [k-z]{32}/\\d{1,2}, got %q", id)
		}
	})
}

func TestChangeID_WithShortAndVersion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithShort, WithVersion).Draw(t, "id")

		// Should match pattern: 8-12 chars + /N
		if !regexp.MustCompile(`^[k-z]{8,12}/\d{1,2}$`).MatchString(id) {
			t.Fatalf("expected [k-z]{8,12}/\\d{1,2}, got %q", id)
		}
	})
}

func TestCommitID_DefaultFull(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := CommitID().Draw(t, "id")

		// Should be 40 chars
		if len(id) != 40 {
			t.Fatalf("expected 40 chars, got %d: %q", len(id), id)
		}

		// Should only contain [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(id) {
			t.Fatalf("expected [0-9a-f]{40}, got %q", id)
		}
	})
}

func TestCommitID_WithShort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := CommitID(WithShort).Draw(t, "id")

		// Should be 7-12 chars
		if len(id) < 7 || len(id) > 12 {
			t.Fatalf("expected 7-12 chars, got %d: %q", len(id), id)
		}

		// Should only contain [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(id) {
			t.Fatalf("expected [0-9a-f]+, got %q", id)
		}
	})
}

func TestOperationID_DefaultFull(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := OperationID().Draw(t, "id")

		// Should be 128 chars
		if len(id) != 128 {
			t.Fatalf("expected 128 chars, got %d: %q", len(id), id)
		}

		// Should only contain [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]{128}$`).MatchString(id) {
			t.Fatalf("expected [0-9a-f]{128}, got %q", id)
		}
	})
}

func TestOperationID_WithShort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := OperationID(WithShort).Draw(t, "id")

		// Should be exactly 12 chars
		if len(id) != 12 {
			t.Fatalf("expected 12 chars, got %d: %q", len(id), id)
		}

		// Should only contain [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(id) {
			t.Fatalf("expected [0-9a-f]+, got %q", id)
		}
	})
}

// =============================================================================
// Property Tests
// =============================================================================

// Property: reverseHexToHex produces valid hex output
func TestReverseHexToHex_ProducesValidHex(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		revHex := rapid.StringMatching(`[k-z]{1,32}`).Draw(t, "revHex")
		hex := reverseHexToHex(revHex)

		// Should be same length
		if len(hex) != len(revHex) {
			t.Fatalf("length mismatch: %q → %q", revHex, hex)
		}

		// Should only contain [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(hex) {
			t.Fatalf("expected [0-9a-f]+, got %q from %q", hex, revHex)
		}
	})
}

// Property: WithShort always produces valid length
func TestChangeID_WithShort_ValidLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithShort).Draw(t, "id")
		if len(id) < 8 || len(id) > 12 {
			t.Fatalf("WithShort produced invalid length %d: %q", len(id), id)
		}
	})
}

// Property: WithVersion always produces valid format
func TestChangeID_WithVersion_ValidFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithVersion).Draw(t, "id")
		parts := strings.Split(id, "/")
		if len(parts) != 2 {
			t.Fatalf("WithVersion should produce exactly one /: %q", id)
		}
		if len(parts[0]) != 32 {
			t.Fatalf("base ID should be 32 chars: %q", id)
		}
	})
}

// Property: Transformers compose correctly (Short then Version)
func TestChangeID_TransformersCompose(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := ChangeID(WithShort, WithVersion).Draw(t, "id")
		parts := strings.Split(id, "/")
		if len(parts) != 2 {
			t.Fatalf("composed ID should have exactly one /: %q", id)
		}
		baseLen := len(parts[0])
		if baseLen < 8 || baseLen > 12 {
			t.Fatalf("base should be 8-12 chars, got %d: %q", baseLen, id)
		}
	})
}

// Property: Transformer order doesn't matter - any permutation produces valid output
func TestChangeID_TransformerOrderIndependent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// All available transformers
		allTransformers := []ChangeIDOption{WithShort, WithVersion, WithNormal}

		// Get a random permutation
		perm := rapid.Permutation(allTransformers).Draw(t, "perm")

		// Apply in random order
		id := ChangeID(perm...).Draw(t, "id")

		// Should have version suffix
		if !strings.Contains(id, "/") {
			t.Fatalf("missing version suffix: %q", id)
		}

		parts := strings.Split(id, "/")
		base := parts[0]

		// Base should be short (8-12 chars)
		if len(base) < 8 || len(base) > 12 {
			t.Fatalf("base should be 8-12 chars, got %d: %q", len(base), id)
		}

		// Base should be normal hex [0-9a-f]
		if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(base) {
			t.Fatalf("base should be [0-9a-f], got: %q", id)
		}

		// Version should be 1-2 digits
		if !regexp.MustCompile(`^\d{1,2}$`).MatchString(parts[1]) {
			t.Fatalf("version should be 1-2 digits: %q", id)
		}
	})
}
