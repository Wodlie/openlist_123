package qihoo360

import (
	"time"
)

// FileItem represents a file or folder in 360 AI云盘
type FileItem struct {
	NID        string `json:"nid"`         // 文件唯一标识
	Name       string `json:"name"`        // 文件名
	Size       int64  `json:"count_size,string"` // 文件大小（字符串格式需要转换）
	Type       int    `json:"type,string"` // 文件类型：1=文件夹，0=文件
	Ctime      int64  `json:"create_time,string"` // 创建时间戳（字符串格式需要转换）
	Mtime      int64  `json:"modify_time,string"` // 修改时间戳（字符串格式需要转换）
	Path       string `json:"path"`        // 文件路径
	ParentNID  string `json:"parent_nid"`  // 父目录NID
	IsDir      bool   `json:"is_dir"`      // 是否为目录
}

// ListResponse represents the response from file list API
type ListResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		List []FileItem `json:"node_list"`
		Page struct {
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
			Total    int `json:"total"`
		} `json:"page"`
	} `json:"data"`
}

// SearchResponse represents the response from file search API
type SearchResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		List []FileItem `json:"node_list"`
		Page struct {
			Page     int `json:"page"`
			PageSize int `json:"page_size"`
			Total    int `json:"total"`
		} `json:"page"`
	} `json:"data"`
}

// UploadResponse represents the response from file upload API
type UploadResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		NID  string `json:"nid"`
		Name string `json:"name"`
		Path string `json:"path"`
		Size int64  `json:"size"`
	} `json:"data"`
}

// DownloadResponse represents the response from file download API
type DownloadResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		DownloadURL string `json:"download_url"`
		FileName    string `json:"file_name"`
		Size        int64  `json:"size"`
	} `json:"data"`
}

// UserInfoResponse represents the response from user info API
type UserInfoResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
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

// ShareResponse represents the response from share API
type ShareResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		Share struct {
			URL      string `json:"url"`
			Password string `json:"password"`
			ShortURL string `json:"shorturl"`
			QRCode   string `json:"qrcode"`
		} `json:"share"`
	} `json:"data"`
}

// CommonResponse represents a common API response
type CommonResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   interface{} `json:"data,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code int    `json:"errno"`
	Msg  string `json:"errmsg"`
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