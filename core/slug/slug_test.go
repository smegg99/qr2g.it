package slug

import "testing"

func TestEncode(t *testing.T) {
	cases := []struct {
		id   int64
		want string
	}{
		{1, "1"},
		{35, "Z"},
		{36, "10"},
		{1295, "ZZ"},
		{1296, "100"},
		{623698779, "ABC123"},
	}
	for _, c := range cases {
		got, err := Encode(c.id)
		if err != nil {
			t.Errorf("Encode(%d) err=%v", c.id, err)
			continue
		}
		if got != c.want {
			t.Errorf("Encode(%d) = %q, want %q", c.id, got, c.want)
		}
	}
}

func TestEncodeRejectNonPositive(t *testing.T) {
	for _, id := range []int64{0, -1, -100} {
		if _, err := Encode(id); err == nil {
			t.Errorf("Encode(%d) expected error", id)
		}
	}
}

func TestDecode(t *testing.T) {
	cases := []struct {
		s    string
		want int64
	}{
		{"1", 1},
		{"Z", 35},
		{"10", 36},
		{"ZZ", 1295},
		{"100", 1296},
		{"ABC123", 623698779},
	}
	for _, c := range cases {
		got, err := Decode(c.s)
		if err != nil {
			t.Errorf("Decode(%q) err=%v", c.s, err)
			continue
		}
		if got != c.want {
			t.Errorf("Decode(%q) = %d, want %d", c.s, got, c.want)
		}
	}
}

func TestDecodeRejects(t *testing.T) {
	bad := []string{
		"",
		"abc",
		"a",
		"01",
		"0",
		"-1",
		"AB!",
		"ZZZZZZZZZZZZZ",
		" ABC",
		"ABC ",
		"AB.C",
	}
	for _, s := range bad {
		if _, err := Decode(s); err == nil {
			t.Errorf("Decode(%q) expected error", s)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	for _, id := range []int64{1, 2, 35, 36, 100, 1295, 1296, 1 << 31, 1 << 60} {
		s, err := Encode(id)
		if err != nil {
			t.Fatalf("Encode(%d) err=%v", id, err)
		}
		got, err := Decode(s)
		if err != nil {
			t.Fatalf("Decode(%q) err=%v", s, err)
		}
		if got != id {
			t.Errorf("round trip: Encode(%d)=%q, Decode=%d", id, s, got)
		}
	}
}
