package smapiweb

import (
	"fmt"
	"testing"
)

func TestCompatStatusMapping(t *testing.T) {
	tests := []struct {
		raw, want string
	}{
		{"ok", StatusOK},
		{"compatible", StatusOK},
		{"optional", StatusOptional},
		{"workaround", StatusBroken},
		{"unofficial", StatusUnofficial},
		{"broken", StatusBroken},
		{"obsolete", StatusObsolete},
		{"abandoned", StatusAbandoned},
		{"unsafe", StatusUnknown},
		{"todo", StatusUnknown},
		{"OK", StatusOK},
		{"Broken", StatusBroken},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			body := fmt.Sprintf(`[{"id":"A.B","status":%q}]`, tt.raw)
			idx, err := parseCompatJSON([]byte(body))
			if err != nil {
				t.Fatal(err)
			}
			e, ok := idx.Lookup("A.B", 0)
			if !ok {
				t.Fatal("missing entry")
			}
			if e.Status != tt.want {
				t.Fatalf("status %q: got %q, want %q", tt.raw, e.Status, tt.want)
			}
			if tt.want == StatusUnknown && e.RawStatus != tt.raw {
				t.Fatalf("raw status: got %q, want %q", e.RawStatus, tt.raw)
			}
		})
	}
}
