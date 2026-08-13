package model

type Profile struct {
	ID       string `json:"id"`
	Browser  string `json:"browser"`
	Name     string `json:"name"`
	Database string `json:"database"`
	Engine   string `json:"engine"`
}

type Record struct {
	URL       string
	Title     string
	Timestamp int64
	Browser   string
	Source    string
}

type ExportResult struct {
	Output      string `json:"output"`
	RecordCount int    `json:"recordCount"`
	MinTimeUsec int64  `json:"minTimeUsec"`
	MaxTimeUsec int64  `json:"maxTimeUsec"`
}
