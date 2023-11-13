package models

type RequestURLJson struct {
	URLAddres string `json:"url"`
}
type ResponseURLJson struct {
	URLAddres string `json:"result"`
}
type RequestBatchURLModel struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"original_url"`
}
type ResponseBatchURLModel struct {
	CorrID      string `json:"correlation_id"`
	OriginalURL string `json:"short_url"`
}
type URLModel struct {
	ShortID    string `json:"short_url"`
	OriginalID string `json:"original_url"`
}
type BantchURL struct {
	OriginalURL string
	ShortURL    string
	UserID      string
}
