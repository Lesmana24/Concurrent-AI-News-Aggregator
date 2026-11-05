package strutil

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateRunes_BasicASCII(t *testing.T) {
	s := "Hello World"
	assert.Equal(t, "", TruncateRunes(s, 0))
	assert.Equal(t, "", TruncateRunes(s, -5))
	assert.Equal(t, "Hello", TruncateRunes(s, 5))
	assert.Equal(t, "Hello World", TruncateRunes(s, 11))
	assert.Equal(t, "Hello World", TruncateRunes(s, 20))
}

func TestTruncateRunes_MultiByteIndonesianAndUnicode(t *testing.T) {
	// “ (3 bytes: \xe2\x80\x9c)
	// ” (3 bytes: \xe2\x80\x9d)
	// — (3 bytes: \xe2\x80\x94)
	text := "“Kebijakan Baru” — Pemerintah Indonesia"
	// Total runes: 1 + 14 + 1 + 1 + 1 + 1 + 20 = 39 runes
	// Byte count is larger because of 3-byte characters: 3 + 14 + 3 + 1 + 3 + 1 + 20 = 45 bytes

	for r := 1; r <= utf8.RuneCountInString(text); r++ {
		truncated := TruncateRunes(text, r)
		assert.True(t, utf8.ValidString(truncated), "Truncated string must be valid UTF-8 for rune count %d", r)
		assert.Equal(t, r, utf8.RuneCountInString(truncated), "Rune count must match requested length")
	}

	// Specifically test slicing right after multi-byte opening quote
	assert.Equal(t, "“", TruncateRunes(text, 1))
	assert.Equal(t, "“K", TruncateRunes(text, 2))
	assert.Equal(t, "“Kebijakan Baru”", TruncateRunes(text, 16))
	assert.Equal(t, "“Kebijakan Baru” —", TruncateRunes(text, 18))
}

func TestTruncateRunes_Emojis(t *testing.T) {
	// 📰 = 4 bytes (\xf0\x9f\x93\xb0)
	// 🔥 = 4 bytes (\xf0\x9f\x94\xa5)
	// 🚀 = 4 bytes (\xf0\x9f\x9a\x80)
	s := "📰 Berita Hangat 🔥 Hari Ini 🚀"
	truncated := TruncateRunes(s, 1)
	assert.Equal(t, "📰", truncated)
	assert.True(t, utf8.ValidString(truncated))

	truncated2 := TruncateRunes(s, 17)
	assert.Equal(t, "📰 Berita Hangat 🔥", truncated2)
	assert.True(t, utf8.ValidString(truncated2))
}

func TestTruncateRunesWithEllipsis(t *testing.T) {
	s := "Berita Terkini Mengenai Banjir di Wilayah Jakarta"
	assert.Equal(t, "Berita Terkini...", TruncateRunesWithEllipsis(s, 17, "..."))
	assert.Equal(t, s, TruncateRunesWithEllipsis(s, 100, "..."))
	assert.Equal(t, "...", TruncateRunesWithEllipsis(s, 3, "..."))
	assert.Equal(t, "..", TruncateRunesWithEllipsis(s, 2, "..."))
	assert.Equal(t, "", TruncateRunesWithEllipsis(s, 0, "..."))

	// Multi-byte test with ellipsis
	unicodeStr := "“Laporan Utama” — Redaksi Nasional"
	res := TruncateRunesWithEllipsis(unicodeStr, 19, "...")
	assert.True(t, utf8.ValidString(res))
	assert.Equal(t, 19, utf8.RuneCountInString(res))
	assert.True(t, strings.HasSuffix(res, "..."))
}

func TestTruncateWords(t *testing.T) {
	text := "  Presiden Joko Widodo memimpin rapat terbatas mengenai penanganan bencana alam di Istana Negara  "
	res := TruncateWords(text, 45, 20, "...")
	assert.True(t, utf8.ValidString(res))
	assert.True(t, strings.HasSuffix(res, "..."))
	// Should not cut words mid-character, snaps to the previous space
	assert.Equal(t, "Presiden Joko Widodo memimpin rapat terbatas...", res)

	// Short text remains intact
	short := "Teks singkat saja."
	assert.Equal(t, short, TruncateWords(short, 50, 10, "..."))

	// Multi-byte text
	multi := "📰 “Kabar Terkini” liputan mendalam dari ibu kota negara"
	resMulti := TruncateWords(multi, 25, 10, "...")
	assert.True(t, utf8.ValidString(resMulti))
	assert.True(t, strings.HasSuffix(resMulti, "..."))
}

func TestCapitalizeFirst(t *testing.T) {
	assert.Equal(t, "", CapitalizeFirst(""))
	assert.Equal(t, "Kompas", CapitalizeFirst("kompas"))
	assert.Equal(t, "Detik", CapitalizeFirst("Detik"))
	// Unicode character: 'ü' (2 bytes) -> 'Ü'
	assert.Equal(t, "Über", CapitalizeFirst("über"))
	// Unicode character: 'é' -> 'É'
	assert.Equal(t, "Édition", CapitalizeFirst("édition"))
}

func BenchmarkTruncateRunes(b *testing.B) {
	longText := strings.Repeat("“Berita Terkini” — Liputan Khusus 📰 🔥 ", 100)
	for b.Loop() {
		_ = TruncateRunes(longText, 120)
	}
}

func BenchmarkRuneSliceAlloc(b *testing.B) {
	longText := strings.Repeat("“Berita Terkini” — Liputan Khusus 📰 🔥 ", 100)
	for b.Loop() {
		runes := []rune(longText)
		if len(runes) > 120 {
			_ = string(runes[:120])
		}
	}
}

