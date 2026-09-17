package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Kind
	}{
		{"invalid", Invalid("bad"), KindInvalid},
		{"not found", NotFound("gone"), KindNotFound},
		{"conflict", Conflict("taken"), KindConflict},
		{"too large", TooLarge("big"), KindTooLarge},
		{"unavailable", Unavailable("off"), KindUnavailable},
		{"internal", Internal("boom", errors.New("cause")), KindInternal},
		{"foreign error defaults to internal", errors.New("whoops"), KindInternal},
		{"wrapped domain error keeps its kind", fmt.Errorf("ctx: %w", NotFound("gone")), KindNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KindOf(tt.err); got != tt.want {
				t.Errorf("KindOf(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorIsMatchesOnKind(t *testing.T) {
	if !errors.Is(NotFound("item not found"), ErrNotFound) {
		t.Error("NotFound should match ErrNotFound")
	}
	if errors.Is(NotFound("item not found"), ErrConflict) {
		t.Error("NotFound should not match ErrConflict")
	}
	if !errors.Is(fmt.Errorf("wrapped: %w", Conflict("dupe")), ErrConflict) {
		t.Error("wrapping should not hide the kind")
	}
}

func TestErrorMessage(t *testing.T) {
	// The message is the API contract, so it must survive verbatim.
	if got := Invalid("measurement_unit must be inch or cm").Error(); got != "measurement_unit must be inch or cm" {
		t.Errorf("Error() = %q", got)
	}
	cause := errors.New("pq: timeout")
	wrapped := Internal("loading item", cause)
	if got := wrapped.Error(); got != "loading item: pq: timeout" {
		t.Errorf("Error() = %q", got)
	}
	if !errors.Is(wrapped, cause) {
		t.Error("Internal should unwrap to its cause")
	}
}
