package normalize

import "testing"

func TestEncodePortBitmask(t *testing.T) {
	got := EncodePortBitmask([]int{3, 1, 3, 5})
	if got != 21 {
		t.Fatalf("EncodePortBitmask() = %d, want 21", got)
	}
}
