package qihoo360

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	ApiKey      string `json:"api_key" type:"string" required:"true" help:"360AI云盘 API 密钥，格式为 'yunpan_' 开头的字符串"`
	ClientId    string `json:"client_id" type:"string" required:"true" help:"360AI云盘 Client ID"`
	ClientSecret string `json:"client_secret" type:"string" required:"true" help:"360AI云盘 Client Secret"`
	RequestUrl  string `json:"request_url" type:"string" default:"https://pcs.yun.360.cn/api" help:"API请求地址"`
}

var config = driver.Config{
	Name:              "Qihoo360",
	LocalSort:         false,
	OnlyLinkMFile:     false,
	OnlyProxy:         false,
	NoCache:           false,
	NoUpload:          false,
	NeedMs:            false,
	DefaultRoot:       "/",
	CheckStatus:       false,
	Alert:             "",
	NoOverwriteUpload: false,
	NoLinkURL:         false,
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Qihoo360{}
	})
}