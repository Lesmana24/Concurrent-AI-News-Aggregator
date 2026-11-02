package crawler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseFeed_RSS20(t *testing.T) {
	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Antara News - Top News</title>
    <link>https://www.antaranews.com</link>
    <description>Berita Terkini</description>
    <item>
      <title><![CDATA[Pemerintah Luncurkan Program Transformasi Digital Nasional]]></title>
      <link>https://www.antaranews.com/berita/12345/program-digital</link>
      <description><![CDATA[<p>Jakarta (ANTARA) - Pemerintah secara resmi mengumumkan peta jalan <b>transformasi digital</b> nasional.</p>]]></description>
      <pubDate>Mon, 02 Nov 2026 08:30:00 +0700</pubDate>
    </item>
    <item>
      <title>Kemenkeu Catat Penerimaan Pajak Meningkat Signifikan</title>
      <link>https://www.antaranews.com/berita/12346/penerimaan-pajak</link>
      <description>Kementerian Keuangan melaporkan tren positif pendapatan negara kuartal ketiga.</description>
      <pubDate>Mon, 02 Nov 2026 09:00:00 +0700</pubDate>
    </item>
  </channel>
</rss>`)

	items, err := ParseFeed(xmlData, "Antara News", "ekonomi")
	assert.NoError(t, err)
	assert.Len(t, items, 2)

	assert.Equal(t, "Pemerintah Luncurkan Program Transformasi Digital Nasional", items[0].Title)
	assert.Equal(t, "https://www.antaranews.com/berita/12345/program-digital", items[0].URL)
	assert.Contains(t, items[0].Description, "Pemerintah secara resmi mengumumkan peta jalan transformasi digital nasional.")
	assert.NotContains(t, items[0].Description, "<p>")
	assert.NotContains(t, items[0].Description, "<b>")
	assert.Equal(t, "Antara News", items[0].Source)
	assert.Equal(t, "ekonomi", items[0].Category)

	assert.Equal(t, "Kemenkeu Catat Penerimaan Pajak Meningkat Signifikan", items[1].Title)
	assert.Equal(t, "https://www.antaranews.com/berita/12346/penerimaan-pajak", items[1].URL)
}

func TestParseFeed_Atom(t *testing.T) {
	atomData := []byte(`<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>CNN Indonesia Terkini</title>
  <entry>
    <title>Pembangunan Infrastruktur Transportasi Masal Dipercepat</title>
    <link href="https://www.cnnindonesia.com/nasional/20261102-transportasi" rel="alternate"/>
    <summary>Kementerian Perhubungan menegaskan komitmen penyelesaian proyek MRT tepat waktu.</summary>
    <published>2026-11-02T07:15:00Z</published>
  </entry>
</feed>`)

	items, err := ParseFeed(atomData, "CNN Indonesia", "nasional")
	assert.NoError(t, err)
	assert.Len(t, items, 1)

	assert.Equal(t, "Pembangunan Infrastruktur Transportasi Masal Dipercepat", items[0].Title)
	assert.Equal(t, "https://www.cnnindonesia.com/nasional/20261102-transportasi", items[0].URL)
	assert.Contains(t, items[0].Description, "Kementerian Perhubungan menegaskan")
	assert.Equal(t, "CNN Indonesia", items[0].Source)
	assert.Equal(t, "nasional", items[0].Category)
}

func TestParseFeedDate_Variations(t *testing.T) {
	now := time.Now()

	d1 := parseFeedDate("Mon, 02 Nov 2026 10:15:30 +0700")
	assert.Equal(t, 2026, d1.Year())
	assert.Equal(t, time.November, d1.Month())
	assert.Equal(t, 2, d1.Day())

	d2 := parseFeedDate("2026-11-02T10:15:30Z")
	assert.Equal(t, 2026, d2.Year())

	// Fallback for empty date
	d3 := parseFeedDate("")
	assert.WithinDuration(t, now, d3, 2*time.Second)
}

func TestParseFeed_InvalidXML(t *testing.T) {
	_, err := ParseFeed([]byte("bukan xml format"), "Source", "cat")
	assert.Error(t, err)
}

