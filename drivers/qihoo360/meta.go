package qihoo360

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	APIKey string `json:"api_key" required:"true" help:"360 AI Cloud API Key (yunpan_ prefix)"`
	driver.RootPath
}

var config = driver.Config{
	Name:        "Qihoo360",
	LocalSort:   true,
	OnlyProxy:   true,
	DefaultRoot: "/",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Qihoo360{}
	})
}
