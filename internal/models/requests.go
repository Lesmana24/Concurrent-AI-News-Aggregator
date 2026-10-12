package models

// SummarizeRequest defines the input for the summarize endpoint
type SummarizeRequest struct {
	URLs []string `json:"urls" binding:"required,dive,url"`
}
