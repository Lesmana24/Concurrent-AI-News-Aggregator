package telegram

import (
	"testing"

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
