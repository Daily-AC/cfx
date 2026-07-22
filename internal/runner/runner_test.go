package runner

import "testing"

func TestNormalizeIgnoresTrailingWhitespace(t *testing.T) {
	left := normalize([]byte("YES  \n42\n\n"))
	right := normalize([]byte("YES\n42"))
	if left != right {
		t.Fatalf("%q != %q", left, right)
	}
}
