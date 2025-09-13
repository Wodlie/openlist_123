package qihoo360

import (
	"time"
)

// FileItem represents a file or folder in 360 AI云盘
type FileItem struct {
	NID        string `json:"nid"`         // 文件唯一标识
	Name       string `json:"name"`        // 文件名
	Size       int64  `json:"size"`        // 文件大小
	Type       int    `json:"type"`        // 文件类型：1=文件夹，0=文件
	Ctime      int64  `json:"ctime"`       // 创建时间戳
	Mtime      int64  `json:"mtime"`       // 修改时间戳
	Path       string `json:"path"`        // 文件路径
	ParentNID  string `json:"parent_nid"`  // 父目录NID
	IsDir      bool   `json:"is_dir"`      // 是否为目录
}

// ListResponse represents the response from file list API
type ListResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List []FileItem `json:"list"`
		Page struct {
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
			Total    int `json:"total"`
		} `json:"page"`
	} `json:"data"`
}

// SearchResponse represents the response from file search API
type SearchResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		List []FileItem `json:"list"`
		Page struct {
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
			Total    int `json:"total"`
		} `json:"page"`
	} `json:"data"`
}

// UploadResponse represents the response from file upload API
type UploadResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		NID  string `json:"nid"`
		Name string `json:"name"`
		Path string `json:"path"`
		Size int64  `json:"size"`
	} `json:"data"`
}

// DownloadResponse represents the response from file download API
type DownloadResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		Size        int64  `json:"size"`
	} `json:"data"`
}

// UserInfoResponse represents the response from user info API
type UserInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserID   string `json:"user_id"`
		UserName string `json:"user_name"`
		Avatar   string `json:"avatar"`
		Space    struct {
			Total int64 `json:"total"`
			Used  int64 `json:"used"`
			Free  int64 `json:"free"`
		} `json:"space"`
	} `json:"data"`
}

// CommonResponse represents a common API response
type CommonResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *APIError) Error() string {
	return e.Msg
}

// toTime converts timestamp to time.Time
func toTime(timestamp int64) time.Time {
	if timestamp == 0 {
		return time.Time{}
	}
	return time.Unix(timestamp, 0)
}