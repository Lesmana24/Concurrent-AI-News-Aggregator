package crawler

// DefaultSources returns the curated directory of high-credibility Indonesian RSS feeds.
func DefaultSources() []FeedSource {
	return []FeedSource{
		{
			ID:       "antara-top",
			Name:     "Antara News",
			Category: "utama",
			URL:      "https://www.antaranews.com/rss/top-news.xml",
			Enabled:  true,
		},
		{
			ID:       "antara-terkini",
			Name:     "Antara News",
			Category: "nasional",
			URL:      "https://www.antaranews.com/rss/terkini.xml",
			Enabled:  true,
		},
		{
			ID:       "antara-ekonomi",
			Name:     "Antara News",
			Category: "ekonomi",
			URL:      "https://www.antaranews.com/rss/ekonomi.xml",
			Enabled:  true,
		},
		{
			ID:       "antara-politik",
			Name:     "Antara News",
			Category: "politik",
			URL:      "https://www.antaranews.com/rss/politik.xml",
			Enabled:  true,
		},
		{
			ID:       "cnn-nasional",
			Name:     "CNN Indonesia",
			Category: "nasional",
			URL:      "https://www.cnnindonesia.com/nasional/rss",
			Enabled:  true,
		},
		{
			ID:       "cnn-teknologi",
			Name:     "CNN Indonesia",
			Category: "teknologi",
			URL:      "https://www.cnnindonesia.com/teknologi/rss",
			Enabled:  true,
		},
		{
			ID:       "cnbc-news",
			Name:     "CNBC Indonesia",
			Category: "ekonomi",
			URL:      "https://www.cnbcindonesia.com/news/rss",
			Enabled:  true,
		},
		{
			ID:       "republika-terkini",
			Name:     "Republika",
			Category: "umum",
			URL:      "https://republika.co.id/rss",
			Enabled:  true,
		},
	}
}

