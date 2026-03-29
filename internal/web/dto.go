package web

type RenderResponse struct {
	ID         string `json:"id"`
	ReviewURL  string `json:"review_url"`
	RecordURL  string `json:"record_url"`
	SourceURL  string `json:"source_url"`
	PreviewURL string `json:"preview_url"`
	FinalURL   string `json:"final_url"`
	CompareURL string `json:"compare_url"`
	DebugURL   string `json:"debug_url,omitempty"`
}

type RenderHistoryItem struct {
	ID         string `json:"id"`
	CreatedAt  string `json:"created_at"`
	Mode       string `json:"mode"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	SourceURL  string `json:"source_url"`
	PreviewURL string `json:"preview_url"`
	ReviewURL  string `json:"review_url"`
	RecordURL  string `json:"record_url"`
	FinalURL   string `json:"final_url"`
	CompareURL string `json:"compare_url"`
	DebugURL   string `json:"debug_url,omitempty"`
}

type SessionStatusResponse struct {
	AuthRequired  bool `json:"auth_required"`
	Authenticated bool `json:"authenticated"`
}
