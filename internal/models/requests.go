package models

// SummarizeRequest defines the input for the summarize endpoint
type SummarizeRequest struct {
	URLs []string `json:"urls" binding:"required,dive,url"`
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
