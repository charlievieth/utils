package types

import (
	"math"
	"testing"
)

func TestAddUint16(t *testing.T) {
	tests := []struct {
		A, B, Exp uint16
	}{
		{0, 0, 0},
		{0, math.MaxUint16, MaxUint16},
		{1, MaxUint16, MaxUint16},
	}
	for _, x := range tests {
		n := AddUint16(x.A, x.B)
		if n != x.Exp {
			t.Fatalf("a: %d b: %d: got: %d want: %d", x.A, x.B, n, x.Exp)
		}
	}
}
