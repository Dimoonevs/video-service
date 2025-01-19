package models

type StatusErrorResp struct {
	Id       int    `json:"id"`
	FileName string `json:"file_name"`
}

type InfoVideosResp struct {
	Id       int    `json:"id"`
	FileName string `json:"file_name"`
	Status   string `json:"status"`
	IsStream bool   `json:"is_stream"`
	FilePath string `json:"file_path,omitempty"`
}
