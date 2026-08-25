package store

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Wingspan":                   "wingspan",
		"  Brass: Birmingham  ":      "brass-birmingham",
		"Twilight Struggle (Deluxe)": "twilight-struggle-deluxe",
		"7 Wonders Duel":             "7-wonders-duel",
		"A---B":                      "a-b",
		"!!!":                        "post",
		"":                           "post",
		"Café & Crème":               "caf-cr-me",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}

	long := Slugify("a very long board game title that keeps going well past the sixty character limit")
	if len(long) > 60 {
		t.Errorf("slug not truncated: %d chars", len(long))
	}
	if long[len(long)-1] == '-' {
		t.Errorf("truncated slug should not end in a hyphen: %q", long)
	}
}

func TestNormalizeUsername(t *testing.T) {
	cases := map[string]string{
		"SamuelTan": "samueltan",
		" sam_tan ": "sam_tan",
		"sam.tan":   "sam-tan",
		"--sam--":   "sam",
		"":          "",
		"!!!":       "",
	}
	for in, want := range cases {
		if got := NormalizeUsername(in); got != want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", in, got, want)
		}
	}

	if got := NormalizeUsername("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); len(got) > 32 {
		t.Errorf("username not truncated: %d chars", len(got))
	}
}
