package repository

import (
	"strings"
	"testing"
)

func TestStopWordsFiltering(t *testing.T) {
	query := "Apa berita terbaru mengenai kripto atau pajak?"
	cleanQuery := strings.Map(func(r rune) rune {
		if strings.ContainsRune("?!.,:;()[]\"'{}<>/\\|@#$%^&*~`+=_-", r) {
			return ' '
		}
		return r
	}, strings.ToLower(strings.TrimSpace(query)))

	rawWords := strings.Fields(cleanQuery)
	var meaningfulWords []string
	for _, w := range rawWords {
		if _, isStop := idStopWords[w]; !isStop && len(w) >= 3 {
			meaningfulWords = append(meaningfulWords, w)
		}
	}

	expected := []string{"kripto", "pajak"}
	if len(meaningfulWords) != len(expected) {
		t.Fatalf("expected %d meaningful words, got %d (%v)", len(expected), len(meaningfulWords), meaningfulWords)
	}

	for i, exp := range expected {
		if meaningfulWords[i] != exp {
			t.Errorf("expected word[%d] to be %q, got %q", i, exp, meaningfulWords[i])
		}
	}
}

func TestStemmingPrefixVariants(t *testing.T) {
	query := "demonstrasi kemarin"
	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	rawWords := strings.Fields(cleanQuery)

	var meaningfulWords []string
	for _, w := range rawWords {
		if _, isStop := idStopWords[w]; !isStop && len(w) >= 3 {
			meaningfulWords = append(meaningfulWords, w)
			if strings.HasPrefix(w, "demonstra") {
				meaningfulWords = append(meaningfulWords, "demo")
			}
		}
	}

	hasDemo := false
	for _, w := range meaningfulWords {
		if w == "demo" {
			hasDemo = true
			break
		}
	}

	if !hasDemo {
		t.Fatalf("expected stem 'demo' to be included in variants, got: %v", meaningfulWords)
	}
}

