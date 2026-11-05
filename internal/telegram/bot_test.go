package telegram

import (
	"news-aggregator/internal/strutil"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestNewBotService_DisabledWhenTokenEmpty(t *testing.T) {
	bot, err := NewBotService("", nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, bot)
	assert.False(t, bot.IsEnabled())
}

func TestStripHTML(t *testing.T) {
	raw := "<b>Tebal</b> dan <i>Miring</i> serta <a href=\"https://example.com\">Link</a>"
	stripped := stripHTML(raw)

	assert.Equal(t, "Tebal dan Miring serta Link", stripped)
}

func TestURLRegex_Extraction(t *testing.T) {
	text := "Tolong baca ini https://nasional.kompas.com/read/123 dan juga http://detik.com/456 ya!"
	urls := urlRegex.FindAllString(text, -1)

	assert.Len(t, urls, 2)
	assert.Equal(t, "https://nasional.kompas.com/read/123", urls[0])
	assert.Equal(t, "http://detik.com/456", urls[1])
}

func TestTelegramSummary_TruncateRunesMultiByte(t *testing.T) {
	// 📰 = 4 bytes (\xf0\x9f\x93\xb0)
	// “ = 3 bytes (\xe2\x80\x9c)
	// ” = 3 bytes (\xe2\x80\x9d)
	longSummary := "📰 “Ringkasan Tren Utama Isu Nasional Hari Ini” yang disusun secara otomatis meliput dinamika kebijakan fiskal dan moneter pemerintah Indonesia bersama Bank Indonesia dalam rangka menjaga stabilitas nilai tukar rupiah dan inflasi pangan pokok masyarakat."
	assert.Greater(t, utf8.RuneCountInString(longSummary), 120)

	shortSummary := longSummary
	if utf8.RuneCountInString(shortSummary) > 120 {
		shortSummary = strutil.TruncateRunes(shortSummary, 117) + "..."
	}

	assert.True(t, utf8.ValidString(shortSummary), "Result must be valid UTF-8")
	assert.False(t, strings.ContainsRune(shortSummary, utf8.RuneError), "Must not have RuneError")
	assert.Equal(t, 120, utf8.RuneCountInString(shortSummary))
	assert.True(t, strings.HasSuffix(shortSummary, "..."))
}

func BenchmarkStripHTML(b *testing.B) {
	raw := "<b>Tebal</b> dan <i>Miring</i> serta <a href=\"https://example.com\">Link</a> artikel berita."
	b.ReportAllocs()
	for b.Loop() {
		_ = stripHTML(raw)
	}
}
