package qihoo360

import (
	"strconv"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

type File struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "1" for directory, "0" for file
	Nid          string `json:"nid"`
	CountSize    string `json:"count_size"`
	CreateTimeTS string `json:"create_time"`
	ModifyTimeTS string `json:"modify_time"`
	Path         string `json:"-"` // Full path, not from API
}

func (f File) GetName() string {
	return f.Name
}

func (f File) GetSize() int64 {
	size, _ := strconv.ParseInt(f.CountSize, 10, 64)
	return size
}

func (f File) ModTime() time.Time {
	timestamp, _ := strconv.ParseInt(f.ModifyTimeTS, 10, 64)
	return time.Unix(timestamp, 0)
}

func (f File) CreateTime() time.Time {
	timestamp, _ := strconv.ParseInt(f.CreateTimeTS, 10, 64)
	return time.Unix(timestamp, 0)
}

func (f File) IsDir() bool {
	return f.Type == "1"
}

func (f File) GetID() string {
	return f.Nid
}

func (f File) GetPath() string {
	return f.Path
}

func (f File) GetHash() utils.HashInfo {
	return utils.HashInfo{}
}

type FileListResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		NodeList []File `json:"node_list"`
	} `json:"data"`
}

type AuthResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		Token             string `json:"token"`
		AccessToken       string `json:"access_token"`
		AccessTokenExpire int64  `json:"access_token_expire"`
		Qid               string `json:"qid"`
	} `json:"data"`
	TraceId string `json:"trace_id"`
}

type CommonResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
}

type UploadAddrResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		HTTP        interface{} `json:"http"`     // can be string or null
		IsHttps     int         `json:"is_https"` // 0 or 1, not boolean
		Tk          interface{} `json:"tk"`       // can be string or null
		Addr2       string      `json:"addr_2"`
		NodeInfo    []File      `json:"node_info"` // returned when file exists (instant upload)
		AutoCommit  int         `json:"autoCommit"`
		FileHash    string      `json:"fhash"`
		FileName    string      `json:"fname"`
		FileSize    string      `json:"fsize"`
		IsCreateDir bool        `json:"is_createdir"`
	} `json:"data"`
}

type BlockInfo struct {
	BHash   string `json:"bhash"`
	BIdx    int    `json:"bidx"`
	BOffset int64  `json:"boffset"`
	BSize   int64  `json:"bsize"`
	Q       string `json:"q,omitempty"`
	T       string `json:"t,omitempty"`
	Token   string `json:"token,omitempty"`
	Tid     string `json:"tid,omitempty"`
}

type PreloadResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		BlockInfo []BlockInfo `json:"block_info"`
		Tid       string      `json:"tid"`
		Tk        string      `json:"tk"`
		HTTP      string      `json:"http"`
		Addr2     string      `json:"addr_2"`
		IsHttps   bool        `json:"is_https"`
	} `json:"data"`
}

type DownloadUrlResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		DownloadUrl string `json:"downloadUrl"`
	} `json:"data"`
}

type CommitResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		Nid        string `json:"nid"`
		Name       string `json:"fname"`
		Size       int64  `json:"fsize"`
		CreateTime int64  `json:"fctime"`
		ModifyTime int64  `json:"fmtime"`
		Tk         string `json:"tk"`
		AutoCommit int    `json:"autoCommit"` // 0 or 1
	} `json:"data"`
}

type AddFileResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		File File `json:"file"`
	} `json:"data"`
}

type UserDetailResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		Name         string `json:"name"`
		TotalSize    string `json:"total_size"`
		UsedSize     string `json:"used_size"`
		AvailableSize int64 `json:"available_size"`
		IsVip        bool   `json:"is_vip"`
		VipDesc      string `json:"vip_desc"`
		ExpireDay    int    `json:"expire_day"`
		Expire       string `json:"expire"`
	} `json:"data"`
}

var _ model.Obj = (*File)(nil)
