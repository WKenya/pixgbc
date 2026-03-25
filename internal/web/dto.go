package web

type RenderResponse struct {
	ID         string `json:"id"`
	ReviewURL  string `json:"review_url"`
	RecordURL  string `json:"record_url"`
	PreviewURL string `json:"preview_url"`
	FinalURL   string `json:"final_url"`
	DebugURL   string `json:"debug_url,omitempty"`
}
