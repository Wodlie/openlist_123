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
	return ""
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
}

type CommonResp struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
}

var _ model.Obj = (*File)(nil)
