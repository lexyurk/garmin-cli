package convert

import "testing"

func TestFloatFromAny(t *testing.T) {
	cases := []struct {
		in   any
		want float64
	}{
		{in: float64(1.25), want: 1.25},
		{in: int(3), want: 3},
		{in: int64(4), want: 4},
		{in: "nope", want: 0},
		{in: nil, want: 0},
	}

	for _, tc := range cases {
		if got := FloatFromAny(tc.in); got != tc.want {
			t.Fatalf("FloatFromAny(%T(%v))=%v want %v", tc.in, tc.in, got, tc.want)
		}
	}
}
