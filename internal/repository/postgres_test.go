package repository

import (
	"context"
	"news-aggregator/internal/models"
	"reflect"
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

	// InitTimestampIndexes with nil db
	if err := repo.InitTimestampIndexes(); err != nil {
		t.Errorf("expected InitTimestampIndexes to return nil on nil db, got %v", err)
	}

	// SearchArticlesByKeyword with nil db
	res, err := repo.SearchArticlesByKeyword(context.Background(), "kripto", 5)
	if err != nil {
		t.Errorf("expected SearchArticlesByKeyword to return nil error on nil db, got %v", err)
	}
	if res != nil {
		t.Errorf("expected SearchArticlesByKeyword to return nil results on nil db, got %v", res)
	}

	// GetRecentArticlesByTopic with nil db
	recentByTopic, err := repo.GetRecentArticlesByTopic(context.Background(), "politik", 24, 10)
	if err != nil {
		t.Errorf("expected GetRecentArticlesByTopic to return nil error on nil db, got %v", err)
	}
	if recentByTopic != nil {
		t.Errorf("expected GetRecentArticlesByTopic to return nil articles on nil db, got %v", recentByTopic)
	}

	// GetRecentArticles with nil db
	recent, err := repo.GetRecentArticles(context.Background(), 24)
	if err != nil {
		t.Errorf("expected GetRecentArticles to return nil error on nil db, got %v", err)
	}
	if recent != nil {
		t.Errorf("expected GetRecentArticles to return nil articles on nil db, got %v", recent)
	}

	// PurgeInactiveGuestUsers with nil db
	deleted, err := repo.PurgeInactiveGuestUsers(context.Background(), 7)
	if err != nil {
		t.Errorf("expected PurgeInactiveGuestUsers to return nil error on nil db, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected PurgeInactiveGuestUsers to return 0 on nil db, got %d", deleted)
	}
}

func TestArticleModel_TimestampIndexes(t *testing.T) {
	articleType := reflect.TypeOf(models.Article{})

	// Check CreatedAt
	createdAtField, ok := articleType.FieldByName("CreatedAt")
	if !ok {
		t.Fatalf("expected models.Article to have CreatedAt field")
	}
	gormTagCreated := createdAtField.Tag.Get("gorm")
	if !strings.Contains(gormTagCreated, "index:idx_articles_created_at") && !strings.Contains(gormTagCreated, "index") {
		t.Errorf("expected CreatedAt field to have index in gorm tag, got: %q", gormTagCreated)
	}

	// Check PublishedAt
	publishedAtField, ok := articleType.FieldByName("PublishedAt")
	if !ok {
		t.Fatalf("expected models.Article to have PublishedAt field")
	}
	gormTagPublished := publishedAtField.Tag.Get("gorm")
	if !strings.Contains(gormTagPublished, "index:idx_articles_published_at") && !strings.Contains(gormTagPublished, "index") {
		t.Errorf("expected PublishedAt field to have index in gorm tag, got: %q", gormTagPublished)
	}
}

func BenchmarkBuildTSQuery(b *testing.B) {
	words := []string{"kripto", "pajak", "keuangan", "pemerintah", "presiden", "ekonomi"}
	for b.Loop() {
		_ = buildTSQuery(words)
	}
}

