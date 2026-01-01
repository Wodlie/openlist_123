package qihoo360

import (
	"context"
	"fmt"
	"net/http"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

type Qihoo360 struct {
	model.Storage
	Addition
	authInfo   *AuthResp
	authExpire int64
}

func (d *Qihoo360) Config() driver.Config {
	return config
}

func (d *Qihoo360) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Qihoo360) Init(ctx context.Context) error {
	// Test authentication
	_, err := d.getAuth()
	return err
}

func (d *Qihoo360) Drop(ctx context.Context) error {
	d.authInfo = nil
	return nil
}

func (d *Qihoo360) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	path := dir.GetPath()
	if path == "" {
		path = d.RootFolderPath
		if path == "" {
			path = "/"
		}
	}

	files, err := d.getFiles(path, 0, 100)
	if err != nil {
		return nil, err
	}

	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return src, nil
	})
}

func (d *Qihoo360) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	// Get file ID (nid)
	nid := file.GetID()
	if nid == "" {
		return nil, fmt.Errorf("file id is empty")
	}

	// Get download URL from API
	downloadUrl, err := d.getDownloadUrl(nid)
	if err != nil {
		return nil, err
	}

	if downloadUrl == "" {
		return nil, fmt.Errorf("download url is empty")
	}

	return &model.Link{
		URL: downloadUrl,
		Header: http.Header{
			"User-Agent": []string{"curl/8.5.0"},
		},
	}, nil
}

func (d *Qihoo360) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	path := parentDir.GetPath()
	if path == "" {
		path = d.RootFolderPath
	}
	if path == "" {
		path = "/"
	}

	// Ensure path ends with /
	if path[len(path)-1] != '/' {
		path += "/"
	}
	// Ensure dirName ends with /
	if dirName[len(dirName)-1] != '/' {
		dirName += "/"
	}

	fname := path + dirName

	params := map[string]string{
		"fname": fname,
	}

	var resp CommonResp
	_, err := d.request("File.makeDir", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("make dir failed: %s", resp.Errmsg)
	}

	return nil, nil
}

func (d *Qihoo360) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	srcPath := srcObj.GetPath()
	if srcPath == "" {
		// Try to construct path from name
		srcPath = d.RootFolderPath
		if srcPath == "" {
			srcPath = "/"
		}
		if srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
		srcPath += srcObj.GetName()
		if srcObj.IsDir() && srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
	}

	dstPath := dstDir.GetPath()
	if dstPath == "" {
		dstPath = d.RootFolderPath
	}
	if dstPath == "" {
		dstPath = "/"
	}
	if dstPath[len(dstPath)-1] != '/' {
		dstPath += "/"
	}

	params := map[string]string{
		"src_name": srcPath,
		"new_name": dstPath,
	}

	var resp CommonResp
	_, err := d.request("File.move", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("move failed: %s", resp.Errmsg)
	}

	return nil, nil
}

func (d *Qihoo360) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	srcPath := srcObj.GetPath()
	if srcPath == "" {
		// Try to construct path from name
		srcPath = d.RootFolderPath
		if srcPath == "" {
			srcPath = "/"
		}
		if srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
		srcPath += srcObj.GetName()
		if srcObj.IsDir() && srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
	}

	// new_name should be just the name, not full path
	if srcObj.IsDir() && newName[len(newName)-1] != '/' {
		newName += "/"
	}

	params := map[string]string{
		"src_name": srcPath,
		"new_name": newName,
	}

	var resp CommonResp
	_, err := d.request("File.rename", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("rename failed: %s", resp.Errmsg)
	}

	return nil, nil
}

func (d *Qihoo360) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	// Copy is not documented in ecs_mcp_server
	return nil, errs.NotSupport
}

func (d *Qihoo360) Remove(ctx context.Context, obj model.Obj) error {
	// File.del API exists in ecs_mcp_server source
	srcPath := obj.GetPath()
	if srcPath == "" {
		// Try to construct path from name
		srcPath = d.RootFolderPath
		if srcPath == "" {
			srcPath = "/"
		}
		if srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
		srcPath += obj.GetName()
		if obj.IsDir() && srcPath[len(srcPath)-1] != '/' {
			srcPath += "/"
		}
	}

	params := map[string]string{
		"path": srcPath,
	}

	var resp CommonResp
	_, err := d.request("File.del", params, &resp)
	if err != nil {
		return err
	}

	if resp.Errno != 0 {
		return fmt.Errorf("remove failed: %s", resp.Errmsg)
	}

	return nil
}

func (d *Qihoo360) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	// File upload (file-upload-stdio) is mentioned in ecs_mcp_server README
	// However, it's marked as "仅支持Stdio接入方式" (only stdio mode)
	// Since this exceeds the basic API capabilities for HTTP-based implementation
	return nil, fmt.Errorf("未实现")
}

var _ driver.Driver = (*Qihoo360)(nil)
