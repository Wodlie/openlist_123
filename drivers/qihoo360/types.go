package qihoo360

// AuthResponse 认证响应结构
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Qid         string `json:"qid"`
	Sign        string `json:"sign"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// AuthInfo 认证信息
type AuthInfo struct {
	AccessToken string `json:"access_token"`
	Qid         string `json:"qid"`
	Sign        string `json:"sign"`
	RequestUrl  string `json:"request_url"`
	ExpiresAt   int64  `json:"expires_at"`
}

// FileListResponse 文件列表响应
type FileListResponse struct {
	Errno int    `json:"errno"`
	Error string `json:"error"`
	Data  struct {
		NodeList []YunPanFile `json:"node_list"`
		Total    int          `json:"total"`
		Page     int          `json:"page"`
		PageSize int          `json:"page_size"`
	} `json:"data"`
}

// YunPanFile 云盘文件信息
type YunPanFile struct {
	IsDir      bool   `json:"is_dir"`
	Name       string `json:"name"`
	CountSize  string `json:"count_size"`
	CreateTime string `json:"create_time"`
	ModifyTime string `json:"modify_time"`
	Nid        string `json:"nid"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Category   int    `json:"category"`
}

// DownloadUrlResponse 下载链接响应
type DownloadUrlResponse struct {
	Errno int    `json:"errno"`
	Error string `json:"error"`
	Data  struct {
		DownloadUrl string `json:"download_url"`
		ExpireTime  int64  `json:"expire_time"`
	} `json:"data"`
}

// CommonResponse 通用响应
type CommonResponse struct {
	Errno int    `json:"errno"`
	Error string `json:"error"`
	Data  interface{} `json:"data"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	Errno int    `json:"errno"`
	Error string `json:"error"`
	Data  struct {
		Qid        string `json:"qid"`
		UserName   string `json:"user_name"`
		UserAvatar string `json:"user_avatar"`
		TotalSize  int64  `json:"total_size"`
		UsedSize   int64  `json:"used_size"`
	} `json:"data"`
}