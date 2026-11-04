package market

import (
	"strings"
	"testing"
)

func TestNewBusinessID(t *testing.T) {
	for _, prefix := range []string{PrefixProduct, PrefixCart, PrefixOrder} {
		id := NewBusinessID(prefix)

		if !strings.HasPrefix(id, prefix+"-") {
			t.Errorf("%q ne commence pas par %q", id, prefix+"-")
		}
		if want := len(prefix) + 1 + idLength; len(id) != want {
			t.Errorf("len(%q) = %d, attendu %d", id, len(id), want)
		}
		if !IsBusinessID(id, prefix) {
			t.Errorf("IsBusinessID(%q, %q) = false", id, prefix)
		}
	}
}

func TestNewBusinessIDUsesWholeAlphabet(t *testing.T) {
	// 10 000 tirages donnent 60 000 caractères : chacun des 36 devrait sortir environ 1 667 fois.
	seen := map[rune]int{}
	for i := 0; i < 10000; i++ {
		code := strings.TrimPrefix(NewBusinessID(PrefixProduct), PrefixProduct+"-")
		for _, r := range code {
			seen[r]++
		}
	}
	for _, r := range idAlphabet {
		if seen[r] == 0 {
			t.Errorf("le caractère %q n'est jamais sorti", r)
		}
	}
	if len(seen) != len(idAlphabet) {
		t.Errorf("%d caractères différents, attendu %d", len(seen), len(idAlphabet))
	}
}

func TestIsBusinessID(t *testing.T) {
	tests := []struct {
		id     string
		prefix string
		want   bool
	}{
		{"PDT-7D2K8N", PrefixProduct, true},
		{"BSK-000000", PrefixCart, true},
		{"PDT-7D2K8N", PrefixOrder, false},    // mauvais préfixe
		{"PDT-7d2k8n", PrefixProduct, false},  // minuscules
		{"PDT-7D2K8", PrefixProduct, false},   // trop court
		{"PDT-7D2K8NX", PrefixProduct, false}, // trop long
		{"PDT7D2K8N", PrefixProduct, false},   // sans tiret
		{"PDT-7D2K8!", PrefixProduct, false},  // caractère interdit
		{"PDT-7D2K8É", PrefixProduct, false},  // lettre accentuée
		{"", PrefixProduct, false},
	}

	for _, tt := range tests {
		if got := IsBusinessID(tt.id, tt.prefix); got != tt.want {
			t.Errorf("IsBusinessID(%q, %q) = %v, attendu %v", tt.id, tt.prefix, got, tt.want)
		}
	}
}
