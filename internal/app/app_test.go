package app

import "testing"

func TestParseProblemID(t *testing.T) {
	id, index, err := parseProblemID("1795b")
	if err != nil || id != 1795 || index != "B" {
		t.Fatalf("got %d %q %v", id, index, err)
	}
	if _, _, err := parseProblemID("bad"); err == nil {
		t.Fatal("expected invalid id error")
	}
}

func TestDefaultProgramType(t *testing.T) {
	if got := defaultProgramType("cpp"); got != "91" {
		t.Fatalf("C++ program type = %s", got)
	}
	if got := defaultProgramType("python"); got != "70" {
		t.Fatalf("Python program type = %s", got)
	}
}
