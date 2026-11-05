package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSmartSelectChunks_ShortContentUntouched(t *testing.T) {
	short := "Ini adalah teks pendek yang tidak perlu dipotong."
	res := SmartSelectChunks(short, "apakah ada teks?", 1000)
	assert.Equal(t, short, res)
}

func TestSmartSelectChunks_PrioritizesRelevantChunkAtEnd(t *testing.T) {
	// Build a 3000 character document where paragraph 1 is fluff, paragraph 2 is fluff,
	// and paragraph 3 (at the end) contains the crucial fact: "Kapolres mengonfirmasi tersangka ditangkap di Surabaya."
	p1 := "Paragraf pembuka artikel berita umum yang menceritakan latar belakang cuaca dan situasi lalu lintas pagi hari di berbagai ruas jalan ibu kota yang terpantau ramai lancar tanpa hambatan berarti sampai siang hari ini."
	p2 := "Paragraf kedua membahas tentang rencana pembangunan jembatan penyeberangan orang di kawasan stasiun yang dijadwalkan akan dimulai bulan depan dengan anggaran pemerintah daerah setempat."
	p3 := "Dalam sesi konferensi pers terpisah, Kapolres mengonfirmasi bahwa tersangka utama telah berhasil ditangkap di Surabaya setelah buron selama dua pekan oleh tim gabungan reserse kriminal kepolisian."

	content := p1 + "\n\n" + p2 + "\n\n" + p3
	question := "Dimana tersangka ditangkap?"

	// Set budget tight enough so not all paragraphs fit (e.g. 500 chars)
	res := SmartSelectChunks(content, question, 500)

	// Verify that paragraph 3 (containing tersangka and Surabaya) was selected!
	assert.True(t, strings.Contains(res, "Surabaya"), "Selected context must include the paragraph with Surabaya")
	assert.True(t, strings.Contains(res, "tersangka"), "Selected context must include the paragraph with tersangka")
}

func TestSmartSelectChunks_FallbackToLeadingWhenNoMatch(t *testing.T) {
	p1 := "Paragraf pertama informasi awal."
	p2 := "Paragraf kedua informasi lanjutan."
	content := p1 + "\n\n" + p2

	res := SmartSelectChunks(content, "apa itu?", 40)
	assert.Contains(t, res, "Paragraf pertama")
}

func TestSmartSelectChunks_MultiByteUTF8Safe(t *testing.T) {
	// “ = 3 bytes (\xe2\x80\x9c)
	// ” = 3 bytes (\xe2\x80\x9d)
	// 📰 = 4 bytes (\xf0\x9f\x93\xb0)
	// Single paragraph with multi-byte characters
	text := "📰 “Kabar Terhangat Hari Ini” — Pemerintah meluncurkan program subsidi energi terbarukan."
	// Total runes: 1 (📰) + 1 ( ) + 1 (“) + 23 + 1 (”) + 1 ( ) + 1 (—) + 1 ( ) + 59 = 89 runes.
	// Byte count is 98 bytes due to multi-byte characters.

	// If cut at 3 runes: 📰 (1), ' ' (2), “ (3)
	// In naive byte slice [:3], byte 3 would cut inside the 4-byte emoji \xf0\x9f\x93\xb0!
	res := SmartSelectChunks(text, "apa", 3)
	assert.True(t, utf8.ValidString(res), "Result must be valid UTF-8 and not break runes")
	assert.False(t, strings.ContainsRune(res, utf8.RuneError), "Must not produce RuneError (replacement character)")
	assert.Equal(t, "📰 “...", res)

	// Test cutting right after quote (28 runes)
	res2 := SmartSelectChunks(text, "apa", 28)
	assert.True(t, utf8.ValidString(res2))
	assert.False(t, strings.ContainsRune(res2, utf8.RuneError))
	assert.Equal(t, "📰 “Kabar Terhangat Hari Ini”...", res2)
}


