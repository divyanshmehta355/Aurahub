package opensearch

import "time"

// VideoDocument represents the indexed video entity in OpenSearch.
type VideoDocument struct {
	ID               string    `json:"id"`
	FileID           string    `json:"fileId"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	Category         string    `json:"category"`
	Tags             []string  `json:"tags"`
	ThumbnailUrl     string    `json:"thumbnailUrl"`
	Views            int32     `json:"views"`
	IsShort          bool      `json:"isShort"`
	IsAdult          bool      `json:"isAdult"`
	Visibility       string    `json:"visibility"`
	UploaderID       string    `json:"uploaderId"`
	UploaderUsername string    `json:"uploaderUsername"`
	CreatedAt        time.Time `json:"createdAt"`
}

// SearchResult wraps matching video documents and the total hit count.
type SearchResult struct {
	Videos []VideoDocument `json:"videos"`
	Total  int64           `json:"total"`
}

// AutocompleteSuggestion represents a fast prefix completion match.
type AutocompleteSuggestion struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	ThumbnailUrl string `json:"thumbnailUrl"`
	Category     string `json:"category"`
}
