package catalogue

import (
	"testing"
	"time"
)

func TestParseYearRepairsBCEntries(t *testing.T) {
	now := time.Now().UTC().Year()

	cases := map[string]*int{
		"2018":  ptr(2018),
		"1995":  ptr(1995),
		"-3000": ptr(-3000), // already correct
		"3500":  ptr(-3500), // Senet, minus sign lost upstream
		"3000":  ptr(-3000), // Backgammon
		"2200":  ptr(-2200), // Go
		"":      nil,
		"0":     nil,
		"abc":   nil,
	}

	for in, want := range cases {
		got := parseYear(in)
		switch {
		case want == nil && got != nil:
			t.Errorf("parseYear(%q) = %d, want nil", in, *got)
		case want != nil && got == nil:
			t.Errorf("parseYear(%q) = nil, want %d", in, *want)
		case want != nil && got != nil && *got != *want:
			t.Errorf("parseYear(%q) = %d, want %d", in, *got, *want)
		}
	}

	// A game published this year must survive untouched.
	if got := parseYear(itoa(now)); got == nil || *got != now {
		t.Errorf("parseYear(current year) = %v, want %d", got, now)
	}
}

func TestSplitList(t *testing.T) {
	got := splitList("City Building,Economic, Farming ,Economic,")
	want := []string{"City Building", "Economic", "Farming"}
	if len(got) != len(want) {
		t.Fatalf("splitList = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitList = %v, want %v", got, want)
		}
	}
	if splitList("  ") != nil {
		t.Error("blank input should give nil")
	}
}

func ptr(v int) *int { return &v }

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
