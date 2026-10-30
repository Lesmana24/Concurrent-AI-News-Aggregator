package models

// NewsCompareRequest defines the input payload for comparing multiple news articles
type NewsCompareRequest struct {
	URLs []string `json:"urls" binding:"required,min=2,max=5,dive,url"`
}

// MediaFraming describes the framing, sentiment, and unique aspects of an article from a specific outlet
type MediaFraming struct {
	SourceURL        string   `json:"source_url"`
	OutletName       string   `json:"outlet_name"`
	Sentiment        string   `json:"sentiment"`        // Positif / Netral / Kritis
	SentimentScore   float64  `json:"sentiment_score"`  // -1.0 to 1.0
	DominantFraming  string   `json:"dominant_framing"` // Sudut pandang atau penekanan utama
	HighlightedFacts []string `json:"highlighted_facts"`
}

// NewsCompareResponse is the comparative journalism report across multiple news sources
type NewsCompareResponse struct {
	Topic            string         `json:"topic"`
	ConsensusFacts   []string       `json:"consensus_facts"`
	MediaComparisons []MediaFraming `json:"media_comparisons"`
	Divergences      []string       `json:"divergences"`
	ObjectiveSummary string         `json:"objective_summary"`
	Provider         string         `json:"provider"`
}

