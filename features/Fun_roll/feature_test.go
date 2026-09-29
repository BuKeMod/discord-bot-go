package roll

import "testing"

func TestDiceBounds(t *testing.T) {
	for _, v := range []string{"0d6", "101d6", "1d0", "1d1000001", "1d6+2", "999999999d6"} {
		if _, _, err := ParseDice(v); err == nil {
			t.Fatal("accepted", v)
		}
	}
	c, s, err := ParseDice("2d6")
	if err != nil || c != 2 || s != 6 {
		t.Fatal(c, s, err)
	}

	// Test human-friendly formats
	for _, tc := range []struct {
		input         string
		expectedCount int
		expectedSides int
	}{
		{"6", 1, 6},
		{"100", 1, 100},
		{"d20", 1, 20},
		{"D12", 1, 12},
		{"3d10", 3, 10},
	} {
		c, s, err := ParseDice(tc.input)
		if err != nil || c != tc.expectedCount || s != tc.expectedSides {
			t.Fatalf("ParseDice(%q) = (%d, %d, %v), want (%d, %d, nil)", tc.input, c, s, err, tc.expectedCount, tc.expectedSides)
		}
	}
}

