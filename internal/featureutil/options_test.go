package featureutil

import (
	"testing"
)

func TestTimeAndChoices(t *testing.T) {
	a, err := Time("2026-10-01T20:00:00+07:00")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Time("2026-10-01 20:00 +07:00")
	if err != nil || !a.Equal(b) {
		t.Fatal(b, err)
	}
	if a.UTC().Hour() != 13 {
		t.Fatal("timezone lost")
	}
	for _, v := range []string{"2026-10-01 20:00", "2026-02-30T12:00:00Z", "99999999999999999"} {
		if _, err := Time(v); err == nil {
			t.Fatal("accepted bad time", v)
		}
	}
	if _, err := Choices("A | a", 2, 10, 55); err == nil {
		t.Fatal("duplicate allowed")
	}
	if _, err := Choices("A | ", 2, 10, 55); err == nil {
		t.Fatal("empty choice allowed")
	}
	if _, err := Choices("ก | ข", 2, 10, 1); err != nil {
		t.Fatal(err)
	}
}
