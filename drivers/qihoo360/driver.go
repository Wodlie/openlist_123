package qihoo360

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Qihoo360 struct {
	model.Storage
	Addition
}

func (d *Qihoo360) Config() driver.Config {
	return config
}

func (d *Qihoo360) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Qihoo360) Init(ctx context.Context) error {
	// Validate API key format
	if !strings.HasPrefix(d.APIKey, "yunpan_") {
		return fmt.Errorf("invalid API key format, should start with 'yunpan_'")
	}
	
	// Test API connection by getting user info
	var userInfo UserInfoResponse
	err := d.getJSON(ctx, "/user/info", nil, &userInfo)
	if err != nil {
		return fmt.Errorf("failed to connect to 360 AI云盘: %w", err)
	}
	
	if userInfo.Code != 0 {
		return fmt.Errorf("API authentication failed: %s", userInfo.Msg)
	}
	
	// Save driver storage
	op.MustSaveDriverStorage(d)
	return nil
}

func (d *Qihoo360) Drop(ctx context.Context) error {
	return nil
}

func (d *Qihoo360) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	path := dir.GetPath()
	if path == "" {
		path = "/"
	}
	
	// Get file list
	resp, err := d.listFiles(ctx, path, 0, 1000) // Start from page 0, get up to 1000 files
	if err != nil {
		return nil, fmt.Errorf("list files failed: %w", err)
	}
	
	var objects []model.Obj
	for _, item := range resp.Data.List {
		obj := d.fileItemToObj(item)
		objects = append(objects, obj)
	}
	
	return objects, nil
}

func (d *Qihoo360) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	if file.IsDir() {
		return nil, errs.NotFile
	}
	
	// Get NID from file path
	nid, err := d.getNIDFromPath(ctx, file.GetPath())
	if err != nil {
		return nil, fmt.Errorf("get file NID: %w", err)
	}
	
	// Get download URL
	resp, err := d.getDownloadURL(ctx, nid)
	if err != nil {
		return nil, fmt.Errorf("get download URL: %w", err)
	}
	
	return &model.Link{
		URL: resp.Data.DownloadURL,
		Header: http.Header{
			"User-Agent": []string{UserAgent},
		},
	}, nil
}

func (d *Qihoo360) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	parentPath := parentDir.GetPath()
	if parentPath == "" {
		parentPath = "/"
	}
	
	// Ensure parent path ends with /
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}
	
	folderPath := parentPath + dirName + "/"
	
	err := d.createFolder(ctx, folderPath)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	
	// Return the created folder object
	return &model.Object{
		Name:     dirName,
		Size:     0,
		IsFolder: true,
		Path:     folderPath,
	}, nil
}

func (d *Qihoo360) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	srcPath := srcObj.GetPath()
	dstPath := dstDir.GetPath()
	
	if dstPath == "" {
		dstPath = "/"
	}
	
	// Ensure destination path ends with /
	if !strings.HasSuffix(dstPath, "/") {
		dstPath += "/"
	}
	
	err := d.moveFile(ctx, []string{srcPath}, dstPath)
	if err != nil {
		return nil, fmt.Errorf("move file: %w", err)
	}
	
	// Return the moved object with new path
	newPath := dstPath + srcObj.GetName()
	return &model.Object{
		Name:     srcObj.GetName(),
		Size:     srcObj.GetSize(),
		IsFolder: srcObj.IsDir(),
		Path:     newPath,
		Modified: srcObj.ModTime(),
	}, nil
}

func (d *Qihoo360) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	srcPath := srcObj.GetPath()
	
	err := d.renameFile(ctx, srcPath, newName)
	if err != nil {
		return nil, fmt.Errorf("rename file: %w", err)
	}
	
	// Update the path with new name
	dir := filepath.Dir(srcPath)
	if dir == "." {
		dir = "/"
	}
	newPath := filepath.Join(dir, newName)
	
	return &model.Object{
		Name:     newName,
		Size:     srcObj.GetSize(),
		IsFolder: srcObj.IsDir(),
		Path:     newPath,
		Modified: srcObj.ModTime(),
	}, nil
}

func (d *Qihoo360) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	// 360 AI云盘 API 不直接支持复制操作
	// 可以通过下载再上传的方式实现，但这对大文件不高效
	return nil, errs.NotImplement
}

func (d *Qihoo360) Remove(ctx context.Context, obj model.Obj) error {
	objPath := obj.GetPath()
	
	err := d.deleteFile(ctx, objPath)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	
	return nil
}

func (d *Qihoo360) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	dstPath := dstDir.GetPath()
	if dstPath == "" {
		dstPath = "/"
	}
	
	// Ensure destination path ends with /
	if !strings.HasSuffix(dstPath, "/") {
		dstPath += "/"
	}
	
	fileName := file.GetName()
	
	// Create a reader that can report progress
	reader := io.TeeReader(file, &progressWriter{
		total:    file.GetSize(),
		uploaded: 0,
		update:   up,
	})
	
	resp, err := d.uploadFile(ctx, reader, fileName, dstPath)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}
	
	// Return the uploaded file object
	return &model.Object{
		Name:     resp.Data.Name,
		Size:     resp.Data.Size,
		IsFolder: false,
		Path:     resp.Data.Path,
	}, nil
}

func (d *Qihoo360) GetArchiveMeta(ctx context.Context, obj model.Obj, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	// 使用内部工具处理压缩文件
	return nil, errs.NotImplement
}

func (d *Qihoo360) ListArchive(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	// 使用内部工具处理压缩文件
	return nil, errs.NotImplement
}

func (d *Qihoo360) Extract(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (*model.Link, error) {
	// 使用内部工具处理压缩文件
	return nil, errs.NotImplement
}

func (d *Qihoo360) ArchiveDecompress(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) ([]model.Obj, error) {
	// 使用内部工具处理压缩文件
	return nil, errs.NotImplement
}

// Other method can be implemented for custom operations
func (d *Qihoo360) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	switch args.Method {
	case "search":
		// 支持搜索功能
		argsData, ok := args.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid args.Data format")
		}
		
		keyword, ok := argsData["keyword"].(string)
		if !ok {
			return nil, fmt.Errorf("missing keyword parameter")
		}
		
		category := FileCategoryAll
		if c, ok := argsData["category"].(int); ok {
			category = c
		}
		
		page := 1
		if p, ok := argsData["page"].(int); ok {
			page = p
		}
		
		pageSize := 50
		if ps, ok := argsData["page_size"].(int); ok {
			pageSize = ps
		}
		
		resp, err := d.searchFiles(ctx, keyword, category, page, pageSize)
		if err != nil {
			return nil, err
		}
		
		var objects []model.Obj
		for _, item := range resp.Data.List {
			obj := d.fileItemToObj(item)
			objects = append(objects, obj)
		}
		
		return map[string]interface{}{
			"files": objects,
			"total": resp.Data.Page.Total,
			"page":  resp.Data.Page.Page,
		}, nil
		
	case "user_info":
		// 获取用户信息
		var userInfo UserInfoResponse
		err := d.getJSON(ctx, "/user/info", nil, &userInfo)
		if err != nil {
			return nil, err
		}
		
		if userInfo.Code != 0 {
			return nil, &APIError{Code: userInfo.Code, Msg: userInfo.Msg}
		}
		
		return userInfo.Data, nil
		
	default:
		return nil, errs.NotSupport
	}
}

// progressWriter implements io.Writer to report upload progress
type progressWriter struct {
	total    int64
	uploaded int64
	update   driver.UpdateProgress
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	pw.uploaded += int64(n)
	
	if pw.update != nil {
		percentage := float64(pw.uploaded) / float64(pw.total) * 100
		pw.update(percentage)
	}
	
	return n, nil
}

var _ driver.Driver = (*Qihoo360)(nil)