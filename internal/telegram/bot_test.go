package telegram

import (
	"context"
	"news-aggregator/internal/strutil"
	"strings"
	"testing"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func TestParseTelegramFloodWait(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expectedSec int
		expectedOk  bool
	}{
		{
			name: "tgbotapi.Error with RetryAfter",
			err: &tgbotapi.Error{
				Code:    429,
				Message: "Too Many Requests",
				ResponseParameters: tgbotapi.ResponseParameters{
					RetryAfter: 7,
				},
			},
			expectedSec: 7,
			expectedOk:  true,
		},
		{
			name: "tgbotapi.Error with 429 code and parsed message",
			err: &tgbotapi.Error{
				Code:    429,
				Message: "Too Many Requests: retry after 14",
			},
			expectedSec: 14,
			expectedOk:  true,
		},
		{
			name: "tgbotapi.Error with 429 code and no seconds defaults to 3",
			err: &tgbotapi.Error{
				Code:    429,
				Message: "Too Many Requests",
			},
			expectedSec: 3,
			expectedOk:  true,
		},
		{
			name:        "generic error string retry after",
			err:         assert.AnError,
			expectedSec: 0,
			expectedOk:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sec, ok := ParseTelegramFloodWait(tc.err)
			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedSec, sec)
		})
	}
}

func TestBotService_PushNotification_DisabledSafety(t *testing.T) {
	// Disabled bot returns nil without performing any action or panicking
	bot := &BotService{enabled: false}
	err := bot.PushNotification(context.Background(), "Judul Berita", "Isi Berita")
	assert.NoError(t, err)

	// Enabled bot with nil repo returns nil without panicking
	botEnabledNoRepo := &BotService{enabled: true, repo: nil}
	err = botEnabledNoRepo.PushNotification(context.Background(), "Judul Berita", "Isi Berita")
	assert.NoError(t, err)
}

func BenchmarkStripHTML(b *testing.B) {
	raw := "<b>Tebal</b> dan <i>Miring</i> serta <a href=\"https://example.com\">Link</a> artikel berita."
	b.ReportAllocs()
	for b.Loop() {
		_ = stripHTML(raw)
	}
}
