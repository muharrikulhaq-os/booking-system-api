package service

import "testing"

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"":                  "",
		"   ":               "",
		" 081234567890 ":    "081234567890",
		"+62 812-3456-7890": "+62 812-3456-7890",
		"(021) 555.1234":    "(021) 555.1234",
		"+628123456789012":  "+628123456789012",
	}
	for in, want := range ok {
		got, err := normalizePhone(in)
		if err != nil || got != want {
			t.Errorf("normalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{
		"0812",                // terlalu pendek
		"0812345678901234567", // > 15 digit
		"0812-abc-7890",       // huruf
		"62+8123456789",       // + bukan di depan
		"081234567890; DROP",  // karakter lain
	} {
		if _, err := normalizePhone(bad); err == nil {
			t.Errorf("normalizePhone(%q) seharusnya ditolak", bad)
		}
	}
}
