package qihoo360

import (
	"context"
	"fmt"
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
	// 验证API密钥格式
	if !strings.HasPrefix(d.APIKey, "yunpan_") {
		return fmt.Errorf("invalid API key format, should start with 'yunpan_'")
	}
	
	// 测试API连接 - 获取根目录文件列表
	_, err := d.listFiles(ctx, "/", 0, 1)
	if err != nil {
		return fmt.Errorf("failed to connect to 360 AI云盘: %w", err)
	}
	
	// 保存驱动配置
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
	
	// 获取文件列表
	resp, err := d.listFiles(ctx, path, 0, 1000)
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
	
	// 获取文件的NID
	nid := file.GetID()
	if nid == "" {
		var err error
		nid, err = d.getNIDFromPath(ctx, file.GetPath())
		if err != nil {
			return nil, fmt.Errorf("get file NID: %w", err)
		}
	}
	
	// 获取下载链接
	resp, err := d.getDownloadURL(ctx, nid)
	if err != nil {
		return nil, fmt.Errorf("get download URL: %w", err)
	}
	
	return &model.Link{
		URL: resp.Data.DownloadURL,
	}, nil
}

func (d *Qihoo360) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	parentPath := parentDir.GetPath()
	if parentPath == "" {
		parentPath = "/"
	}
	
	// 确保父路径以 / 结尾
	if !strings.HasSuffix(parentPath, "/") {
		parentPath += "/"
	}
	
	folderPath := parentPath + dirName + "/"
	
	err := d.createFolder(ctx, folderPath)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	
	// 返回创建的文件夹对象
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
	
	// 确保目标路径以 / 结尾
	if !strings.HasSuffix(dstPath, "/") {
		dstPath += "/"
	}
	
	err := d.moveFile(ctx, []string{srcPath}, dstPath)
	if err != nil {
		return nil, fmt.Errorf("move file: %w", err)
	}
	
	// 返回移动后的对象，更新路径
	newPath := dstPath + srcObj.GetName()
	return &model.Object{
		Name:     srcObj.GetName(),
		Size:     srcObj.GetSize(),
		IsFolder: srcObj.IsDir(),
		Path:     newPath,
		Modified: srcObj.ModTime(),
		Ctime:    srcObj.CreateTime(),
	}, nil
}

func (d *Qihoo360) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	srcPath := srcObj.GetPath()
	
	err := d.renameFile(ctx, srcPath, newName)
	if err != nil {
		return nil, fmt.Errorf("rename file: %w", err)
	}
	
	// 更新对象名称
	return &model.Object{
		Name:     newName,
		Size:     srcObj.GetSize(),
		IsFolder: srcObj.IsDir(),
		Path:     srcPath, // 路径保持不变，只是名称改了
		Modified: srcObj.ModTime(),
		Ctime:    srcObj.CreateTime(),
	}, nil
}

func (d *Qihoo360) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	// 360 AI云盘 API 目前不支持直接复制操作
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
	// 360 AI云盘的上传功能比较复杂，需要特殊的SDK
	// 这里暂不实现，返回不支持
	return nil, errs.NotImplement
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

// Other 实现其他自定义操作
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
		
	case "share":
		// 支持文件分享功能
		argsData, ok := args.Data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid args.Data format")
		}
		
		paths, ok := argsData["paths"].(string)
		if !ok {
			return nil, fmt.Errorf("missing paths parameter")
		}
		
		resp, err := d.shareFiles(ctx, paths)
		if err != nil {
			return nil, err
		}
		
		return resp.Data.Share, nil
		
	default:
		return nil, errs.NotSupport
	}
}

var _ driver.Driver = (*Qihoo360)(nil)