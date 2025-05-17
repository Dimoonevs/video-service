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
	StatusAI string `json:"status_ai,omitempty"`
}

type VideoFormatLinksResp struct {
	VideoFormatId int           `json:"video_format_id"`
	FileId        int           `json:"file_id"`
	Filename      string        `json:"filename"`
	Formats       []VideoFormat `json:"formats"`
}
type VideoFormat struct {
	URL        string `json:"url"`
	Resolution string `json:"size"`
}

type FileStatus string

const (
	StatusNoConv    FileStatus = "no_conv"
	StatusConv      FileStatus = "conv"
	StatusProcess   FileStatus = "process"
	StatusDone      FileStatus = "done"
	StatusError     FileStatus = "error"
	StatusDeleted   FileStatus = "deleted"
	StatusLoading   FileStatus = "loading"
	StatusLoadError FileStatus = "loading_error"
)
