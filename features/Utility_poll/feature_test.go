package poll

import (
	"strings"
	"testing"
)

func TestNativePollValidation(t *testing.T) {
	p, err := Build("กินอะไร", "ข้าว | เส้น", 24, true)
	if err != nil || len(p.Answers) != 2 || !p.AllowMultiselect || p.Duration != 24 {
		t.Fatal(p, err)
	}
	for _, v := range []struct {
		q, a string
		h    int
	}{{"", "a|b", 24}, {"q", "a|a", 24}, {"q", "a|b", 0}, {"q", "a|b", 769}, {"q", strings.Repeat("ก", 56) + "|b", 24}} {
		if _, err := Build(v.q, v.a, v.h, false); err == nil {
			t.Fatal("invalid poll accepted", v)
		}
	}
}
