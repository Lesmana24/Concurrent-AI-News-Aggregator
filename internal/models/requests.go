package models

// SummarizeRequest defines the input for the summarize endpoint
type SummarizeRequest struct {
	URLs []string `json:"urls" binding:"required,min=1,dive,url"`
}

// NewsQARequest defines the input for the news Q&A endpoint
type NewsQARequest struct {
	URLs     []string `json:"urls" binding:"required,min=1,max=5,dive,url"`
	Question string   `json:"question" binding:"required,min=3,max=500"`
}

// NewsQAResponse defines the output for the news Q&A endpoint
type NewsQAResponse struct {
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Sources  []string `json:"sources"`
	Source   string   `json:"source"`
}

// SemanticSearchResult represents an article matched by vector similarity
type SemanticSearchResult struct {
	ArticleID  uint    `json:"article_id"`
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Summary    string  `json:"summary"`
	Source     string  `json:"source"`
	Similarity float64 `json:"similarity"`
}

// GlobalQARequest defines the payload for asking questions across all archived news
type GlobalQARequest struct {
	Question string `json:"question" binding:"required,min=3,max=500"`
}

// GlobalQAResponse defines the answer and cited sources for global news Q&A
type GlobalQAResponse struct {
	Question string                 `json:"question"`
	Answer   string                 `json:"answer"`
	Sources  []SemanticSearchResult `json:"sources"`
	Provider string                 `json:"provider"`
}
