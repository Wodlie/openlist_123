package qihoo360

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	APIKey string `json:"api_key" required:"true" help:"360 AI云盘 API密钥，格式为 yunpan_ 开头的字符串"`
	OrderBy string `json:"order_by" type:"select" options:"name,size,time,type" default:"name" help:"文件排序方式"`
	OrderDirection string `json:"order_direction" type:"select" options:"asc,desc" default:"asc" help:"排序方向"`
}

var config = driver.Config{
	Name:        "Qihoo360",
	LocalSort:   false,
	OnlyProxy:   false,
	NoCache:     false,
	NoUpload:    false,
	NeedMs:      false,
	DefaultRoot: "/",
	CheckStatus: false,
	Alert:       "360 AI云盘驱动，需要从 https://open.yunpan.360.cn/ 获取API密钥",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Qihoo360{}
	})
}