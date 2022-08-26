package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var tt Tools

	s := tt.RandomString(10)
	if len(s) != 10 {
		t.Errorf("expected len of %d, got %d", 10, len(s))
	}
}
