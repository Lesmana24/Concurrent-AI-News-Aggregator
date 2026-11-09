package telegram

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSplitTelegramMessage_ShortMessage(t *testing.T) {
	text := "Halo! Ini adalah pesan singkat Telegram yang tidak melebihi batas 4096 karakter."
	chunks := SplitTelegramMessage(text, 4000)

	assert.Len(t, chunks, 1)
	assert.Equal(t, text, chunks[0])
}

func TestSplitTelegramMessage_Empty(t *testing.T) {
	assert.Nil(t, SplitTelegramMessage("", 4000))
	assert.Nil(t, SplitTelegramMessage("   \n\t  ", 4000))
}

func TestSplitTelegramMessage_ParagraphSplit(t *testing.T) {
	p1 := strings.Repeat("Paragraf pertama memuat informasi berita penting nasional. ", 30) // ~1740 runes
	p2 := strings.Repeat("Paragraf kedua menjelaskan perkembangan ekonomi terkini. ", 30)    // ~1740 runes
	p3 := strings.Repeat("Paragraf ketiga merangkum tanggapan analis dan pengamat. ", 30)     // ~1740 runes

	fullText := p1 + "\n\n" + p2 + "\n\n" + p3
	assert.Greater(t, utf8.RuneCountInString(fullText), 4000)

	chunks := SplitTelegramMessage(fullText, 2500)
	assert.GreaterOrEqual(t, len(chunks), 2)

	for i, c := range chunks {
		count := utf8.RuneCountInString(c)
		assert.LessOrEqual(t, count, 2500, fmt.Sprintf("chunk %d exceeded limit: %d runes", i, count))
		assert.True(t, utf8.ValidString(c), "chunk must be valid UTF-8")
	}

	// Verify all paragraphs are present in output chunks
	combined := strings.Join(chunks, " ")
	assert.Contains(t, combined, "Paragraf pertama")
	assert.Contains(t, combined, "Paragraf kedua")
	assert.Contains(t, combined, "Paragraf ketiga")
}

func TestSplitTelegramMessage_LineSplit(t *testing.T) {
	var lines []string
	for i := 1; i <= 80; i++ {
		lines = append(lines, fmt.Sprintf("%d. Berita headline nomor %d mengulas kebijakan fiskal.", i, i))
	}
	fullText := strings.Join(lines, "\n")
	assert.Greater(t, utf8.RuneCountInString(fullText), 4000)

	chunks := SplitTelegramMessage(fullText, 2000)
	assert.GreaterOrEqual(t, len(chunks), 2)

	for i, c := range chunks {
		count := utf8.RuneCountInString(c)
		assert.LessOrEqual(t, count, 2000, fmt.Sprintf("chunk %d exceeded limit: %d runes", i, count))
	}

	// Verify first and last line exist
	combined := strings.Join(chunks, "\n")
	assert.Contains(t, combined, "1. Berita headline")
	assert.Contains(t, combined, "80. Berita headline")
}

func TestSplitTelegramMessage_WordSplit(t *testing.T) {
	// Single long paragraph with spaces but no newlines
	fullText := strings.Repeat("kata ", 800) // ~4000 runes
	chunks := SplitTelegramMessage(fullText, 1000)

	assert.GreaterOrEqual(t, len(chunks), 4)
	for i, c := range chunks {
		count := utf8.RuneCountInString(c)
		assert.LessOrEqual(t, count, 1000, fmt.Sprintf("chunk %d exceeded limit: %d runes", i, count))
		// Ensure words are not chopped mid-word
		words := strings.Fields(c)
		for _, w := range words {
			assert.Equal(t, "kata", w, "word must not be broken in half")
		}
	}
}

func TestSplitTelegramMessage_HTMLTagPreservation(t *testing.T) {
	// A long bold section that spans across chunks
	textInsideBold := strings.Repeat("Teks tebal berita penting dunia industri dan ekonomi. ", 50)
	fullText := "<b>" + textInsideBold + "</b>"

	chunks := SplitTelegramMessage(fullText, 1000)
	assert.GreaterOrEqual(t, len(chunks), 2)

	for i, c := range chunks {
		count := utf8.RuneCountInString(c)
		assert.LessOrEqual(t, count, 1050, fmt.Sprintf("chunk %d exceeded limit: %d runes", i, count))

		// Chunk 1 must be closed with </b>
		if i == 0 {
			assert.True(t, strings.HasPrefix(c, "<b>"))
			assert.True(t, strings.HasSuffix(c, "</b>"), "first chunk must close </b>")
		}
		// Subsequent chunks must be reopened with <b>
		if i > 0 && i < len(chunks)-1 {
			assert.True(t, strings.HasPrefix(c, "<b>"), "middle chunk must reopen <b>")
			assert.True(t, strings.HasSuffix(c, "</b>"), "middle chunk must close </b>")
		}
		// Last chunk must end with </b>
		if i == len(chunks)-1 {
			assert.True(t, strings.HasPrefix(c, "<b>"), "last chunk must reopen <b>")
			assert.True(t, strings.HasSuffix(c, "</b>"), "last chunk must end with </b>")
		}
	}
}

func TestSplitTelegramMessage_AnchorTagPreservation(t *testing.T) {
	linkText := strings.Repeat("Klik tautan ini untuk membaca artikel lengkap. ", 40)
	fullText := `<a href="https://nasional.kompas.com/read/2026/01/01/kebijakan-ekonomi">` + linkText + `</a>`

	chunks := SplitTelegramMessage(fullText, 1000)
	assert.GreaterOrEqual(t, len(chunks), 2)

	for i, c := range chunks {
		assert.True(t, strings.Contains(c, "https://nasional.kompas.com"), fmt.Sprintf("chunk %d must retain anchor url", i))
		assert.True(t, strings.HasSuffix(c, "</a>"), fmt.Sprintf("chunk %d must end with </a>", i))
	}
}

func TestSplitTelegramMessage_MultiByteUTF8AndEmojis(t *testing.T) {
	// 📰 = 4 bytes, “/” = 3 bytes, 🚀 = 4 bytes
	snippet := "📰 “Headline Berita Utama Terkini: Perkembangan Teknologi AI Indonesia 🚀”. "
	fullText := strings.Repeat(snippet, 60) // ~4560 runes

	chunks := SplitTelegramMessage(fullText, 2000)
	assert.GreaterOrEqual(t, len(chunks), 2)

	for i, c := range chunks {
		assert.True(t, utf8.ValidString(c), fmt.Sprintf("chunk %d must be valid UTF-8", i))
		assert.False(t, strings.ContainsRune(c, utf8.RuneError), fmt.Sprintf("chunk %d must not have RuneError", i))
	}
}

func TestSplitTelegramMessage_ContinuousStringNoSpaces(t *testing.T) {
	// Extreme case: 5000 characters without a single space or newline
	noSpaces := strings.Repeat("A", 5000)
	chunks := SplitTelegramMessage(noSpaces, 2000)

	assert.Equal(t, 3, len(chunks))
	assert.Equal(t, 2000, utf8.RuneCountInString(chunks[0]))
	assert.Equal(t, 2000, utf8.RuneCountInString(chunks[1]))
	assert.Equal(t, 1000, utf8.RuneCountInString(chunks[2]))
}

func BenchmarkSplitTelegramMessage(b *testing.B) {
	p1 := strings.Repeat("Paragraf pertama memuat berita teknologi terkini. ", 40)
	p2 := strings.Repeat("Paragraf kedua menjelaskan implikasi kebijakan regulasi digital. ", 40)
	fullText := "<b>" + p1 + "</b>\n\n" + p2

	b.ReportAllocs()
	for b.Loop() {
		_ = SplitTelegramMessage(fullText, 2000)
	}
}

