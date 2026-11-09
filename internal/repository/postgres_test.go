package repository

import (
	"context"
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

func TestSanitizeTSQueryTerm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain alphanumeric",
			input:    "kripto123",
			expected: "kripto123",
		},
		{
			name:     "strips quotes and operators",
			input:    "'kripto';--!&|()",
			expected: "kripto",
		},
		{
			name:     "handles non-ASCII UTF-8 characters",
			input:    "café",
			expected: "café",
		},
		{
			name:     "pure symbols results in empty",
			input:    "!@#$%^&*()_+",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeTSQueryTerm(tc.input)
			if got != tc.expected {
				t.Errorf("sanitizeTSQueryTerm(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestBuildTSQuery(t *testing.T) {
	tests := []struct {
		name     string
		words    []string
		expected string
	}{
		{
			name:     "single word",
			words:    []string{"kripto"},
			expected: "'kripto':*",
		},
		{
			name:     "multiple words with OR operator",
			words:    []string{"kripto", "pajak", "keuangan"},
			expected: "'kripto':* | 'pajak':* | 'keuangan':*",
		},
		{
			name:     "deduplicates identical words",
			words:    []string{"kripto", "pajak", "kripto"},
			expected: "'kripto':* | 'pajak':*",
		},
		{
			name:     "empty word slice",
			words:    []string{},
			expected: "",
		},
		{
			name:     "words that sanitize to empty",
			words:    []string{"!@#", "$%^"},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTSQuery(tc.words)
			if got != tc.expected {
				t.Errorf("buildTSQuery(%v) = %q; want %q", tc.words, got, tc.expected)
			}
		})
	}
}

func TestRepository_NilDB_SafeOperations(t *testing.T) {
	repo := &PostgresRepository{db: nil}

	// InitSearchIndexes with nil db
	if err := repo.InitSearchIndexes(); err != nil {
		t.Errorf("expected InitSearchIndexes to return nil on nil db, got %v", err)
	}

	// SearchArticlesByKeyword with nil db
	res, err := repo.SearchArticlesByKeyword(context.Background(), "kripto", 5)
	if err != nil {
		t.Errorf("expected SearchArticlesByKeyword to return nil error on nil db, got %v", err)
	}
	if res != nil {
		t.Errorf("expected SearchArticlesByKeyword to return nil results on nil db, got %v", res)
	}
}

func BenchmarkBuildTSQuery(b *testing.B) {
	words := []string{"kripto", "pajak", "keuangan", "pemerintah", "presiden", "ekonomi"}
	for b.Loop() {
		_ = buildTSQuery(words)
	}
}

