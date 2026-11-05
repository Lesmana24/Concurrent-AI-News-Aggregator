package ai

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestLocalFallbackProvider_ExtractSnippetMultiByte(t *testing.T) {
	// 📰 = 4 bytes (\xf0\x9f\x93\xb0)
	// “ = 3 bytes (\xe2\x80\x9c)
	// ” = 3 bytes (\xe2\x80\x9d)
	// — = 3 bytes (\xe2\x80\x94)
	text := "  📰 “Kabar Eksklusif Hari Ini” — Redaksi melaporkan bahwa perkembangan infrastruktur transportasi kereta cepat telah mencapai target pembangunan tahap pertama di Jawa Barat secara menyeluruh.  "

	snippet := extractSnippet(text, 80)
	assert.True(t, utf8.ValidString(snippet), "Snippet must be valid UTF-8")
	assert.False(t, strings.ContainsRune(snippet, utf8.RuneError), "Must not have RuneError")
	assert.True(t, strings.HasSuffix(snippet, "..."))
	assert.LessOrEqual(t, utf8.RuneCountInString(snippet), 83) // 80 runes + ellipsis
}

func TestLocalFallbackProvider_SummarizeAndAnalyzeBias(t *testing.T) {
	p := NewLocalFallbackProvider()
	assert.Equal(t, ProviderNameLocalFallback, p.Name())
	assert.True(t, p.IsConfigured())

	// Test Summarize with multi-byte title & content
	summary, err := p.Summarize(context.Background(), "Berita 📰 “Ekonomi” — Perkembangan kurs rupiah terhadap dolar.")
	assert.NoError(t, err)
	assert.NotEmpty(t, summary)
	assert.True(t, utf8.ValidString(summary))

	// Test AnalyzeBias with multi-byte content
	articles := []ArticleForAnalysis{
		{
			Title:   "📰 Berita Alpha",
			Content: "Isi berita alpha tentang kebijakan moneter dan inflasi nasional terkini dengan ulasan mendalam.",
			Outlet:  "Media Satu",
			URL:     "https://satu.com/1",
		},
		{
			Title:   "🔥 Berita Beta",
			Content: "Isi berita beta dengan fokus pada tanggapan pelaku usaha dan respon investor pasar modal.",
			Outlet:  "Media Dua",
			URL:     "https://dua.com/2",
		},
	}
	resp, err := p.AnalyzeBias(context.Background(), articles)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 2, len(resp.MediaComparisons))
	for _, comp := range resp.MediaComparisons {
		assert.True(t, utf8.ValidString(comp.HighlightedFacts[0]))
	}
}
