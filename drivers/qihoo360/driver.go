package qihoo360

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

type Qihoo360 struct {
	model.Storage
	Addition
	
	// 缓存认证信息
	authInfo *AuthInfo
}

func (d *Qihoo360) Config() driver.Config {
	return config
}

func (d *Qihoo360) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Qihoo360) Init(ctx context.Context) error {
	// 初始化时获取认证信息
	return d.refreshAuth(ctx)
}

func (d *Qihoo360) Drop(ctx context.Context) error {
	d.authInfo = nil
	return nil
}

// refreshAuth 刷新认证信息
func (d *Qihoo360) refreshAuth(ctx context.Context) error {
	// 如果认证信息仍然有效，直接返回
	if d.authInfo != nil && time.Now().Unix() < d.authInfo.ExpiresAt-300 { // 提前5分钟刷新
		return nil
	}
	
	// 构建认证请求参数
	params := map[string]string{
		"method":        "Auth.authorize",
		"client_id":     d.ClientId,
		"grant_type":    "client_credentials",
		"scope":         "basic",
	}
	
	// 生成签名
	params["sign"] = generateSign(params, d.ClientSecret)
	
	// 构建请求URL
	reqUrl := d.RequestUrl
	if reqUrl == "" {
		reqUrl = "https://pcs.yun.360.cn/api"
	}
	
	// 构建表单数据
	formData := buildFormData(params)
	
	// 发送请求
	resp, err := http.PostForm(reqUrl, formData)
	if err != nil {
		return fmt.Errorf("认证请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取认证响应失败: %w", err)
	}
	
	// 解析响应
	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("解析认证响应失败: %w", err)
	}
	
	// 保存认证信息
	d.authInfo = &AuthInfo{
		AccessToken: authResp.AccessToken,
		Qid:         authResp.Qid,
		Sign:        authResp.Sign,
		RequestUrl:  reqUrl,
		ExpiresAt:   time.Now().Unix() + int64(authResp.ExpiresIn),
	}
	
	return nil
}

// makeRequest 发送API请求
func (d *Qihoo360) makeRequest(ctx context.Context, method string, params map[string]string, formParams map[string]string) ([]byte, error) {
	// 确保认证信息有效
	if err := d.refreshAuth(ctx); err != nil {
		return nil, err
	}
	
	// 构建基础参数
	baseParams := map[string]string{
		"method":       method,
		"access_token": d.authInfo.AccessToken,
		"qid":          d.authInfo.Qid,
		"sign":         d.authInfo.Sign,
	}
	
	// 添加额外参数
	for k, v := range params {
		baseParams[k] = v
	}
	
	// 构建请求URL
	reqUrl, err := url.Parse(d.authInfo.RequestUrl)
	if err != nil {
		return nil, fmt.Errorf("解析请求URL失败: %w", err)
	}
	
	// 添加URL参数
	query := reqUrl.Query()
	for k, v := range baseParams {
		query.Set(k, v)
	}
	reqUrl.RawQuery = query.Encode()
	
	var resp *http.Response
	
	if len(formParams) > 0 {
		// POST请求，使用表单数据
		formData := buildFormData(formParams)
		resp, err = http.PostForm(reqUrl.String(), formData)
	} else {
		// GET请求
		resp, err = http.Get(reqUrl.String())
	}
	
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取API响应失败: %w", err)
	}
	
	return body, nil
}

func (d *Qihoo360) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	path := cleanPath(dir.GetPath())
	
	// 构建请求参数
	params := map[string]string{
		"path":      path,
		"page":      "0",
		"page_size": "100",
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.getList", params, nil)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp FileListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析文件列表响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("获取文件列表失败: %s", resp.Error)
	}
	
	// 转换为model.Obj
	var objects []model.Obj
	for _, file := range resp.Data.NodeList {
		// 解析文件大小
		file.Size = parseFileSize(file.CountSize)
		objects = append(objects, fileToObj(file))
	}
	
	return objects, nil
}

func (d *Qihoo360) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	// 获取文件下载链接
	params := map[string]string{
		"nid": file.GetID(),
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "Sync.getVerifiedDownLoadUrl", params, nil)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp DownloadUrlResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析下载链接响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("获取下载链接失败: %s", resp.Error)
	}
	
	return &model.Link{
		URL: resp.Data.DownloadUrl,
		Header: http.Header{
			"User-Agent": []string{"yunpan_alist"},
		},
	}, nil
}

func (d *Qihoo360) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	// 构建文件夹路径
	parentPath := cleanPath(parentDir.GetPath())
	folderPath := joinPath(parentPath, dirName)
	
	// 构建请求参数
	formParams := map[string]string{
		"fname": folderPath,
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.mkdir", nil, formParams)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp CommonResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析创建文件夹响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("创建文件夹失败: %s", resp.Error)
	}
	
	// 返回新创建的文件夹对象
	return &model.Object{
		Name:     dirName,
		Path:     folderPath,
		IsFolder: true,
		Modified: time.Now(),
	}, nil
}

func (d *Qihoo360) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	srcPath := cleanPath(srcObj.GetPath())
	dstPath := cleanPath(dstDir.GetPath())
	
	// 构建目标路径
	newPath := joinPath(dstPath, srcObj.GetName())
	
	// 构建请求参数
	formParams := map[string]string{
		"src_name": srcPath,
		"new_name": newPath,
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.move", nil, formParams)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp CommonResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析移动文件响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("移动文件失败: %s", resp.Error)
	}
	
	// 返回移动后的对象
	newObj := &model.Object{
		Name:     srcObj.GetName(),
		Path:     newPath,
		IsFolder: srcObj.IsDir(),
		Size:     srcObj.GetSize(),
		Modified: srcObj.ModTime(),
		ID:       srcObj.GetID(),
	}
	
	return newObj, nil
}

func (d *Qihoo360) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	srcPath := cleanPath(srcObj.GetPath())
	
	// 构建新路径
	parentPath := strings.TrimSuffix(srcPath, "/"+srcObj.GetName())
	if parentPath == "" {
		parentPath = "/"
	}
	newPath := joinPath(parentPath, newName)
	
	// 构建请求参数
	formParams := map[string]string{
		"src_name": srcPath,
		"new_name": newName, // 注意：重命名只需要新文件名，不是完整路径
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.rename", nil, formParams)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp CommonResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析重命名文件响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("重命名文件失败: %s", resp.Error)
	}
	
	// 返回重命名后的对象
	newObj := &model.Object{
		Name:     newName,
		Path:     newPath,
		IsFolder: srcObj.IsDir(),
		Size:     srcObj.GetSize(),
		Modified: srcObj.ModTime(),
		ID:       srcObj.GetID(),
	}
	
	return newObj, nil
}

func (d *Qihoo360) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	// 360云盘API没有直接的复制接口，返回不支持
	return nil, errs.NotImplement
}

func (d *Qihoo360) Remove(ctx context.Context, obj model.Obj) error {
	objPath := cleanPath(obj.GetPath())
	
	// 构建请求参数
	formParams := map[string]string{
		"fname": objPath,
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.delete", nil, formParams)
	if err != nil {
		return err
	}
	
	// 解析响应
	var resp CommonResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("解析删除文件响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return fmt.Errorf("删除文件失败: %s", resp.Error)
	}
	
	return nil
}

func (d *Qihoo360) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	// 360云盘的文件上传需要使用专门的SDK，这里返回不支持
	// 在实际实现中，可以参考 @aicloud360/sec-sdk-node 的实现
	return nil, errs.NotImplement
}

func (d *Qihoo360) GetArchiveMeta(ctx context.Context, obj model.Obj, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	return nil, errs.NotImplement
}

func (d *Qihoo360) ListArchive(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Qihoo360) Extract(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (*model.Link, error) {
	return nil, errs.NotImplement
}

func (d *Qihoo360) ArchiveDecompress(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

// Other 自定义方法，可以实现搜索等功能
func (d *Qihoo360) Other(ctx context.Context, args model.OtherArgs) (interface{}, error) {
	switch args.Method {
	case "search":
		return d.search(ctx, args.Data)
	case "user_info":
		return d.getUserInfo(ctx)
	default:
		return nil, errs.NotSupport
	}
}

// search 搜索文件
func (d *Qihoo360) search(ctx context.Context, data interface{}) (interface{}, error) {
	searchArgs, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("搜索参数格式错误")
	}
	
	key, ok := searchArgs["key"].(string)
	if !ok || key == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	
	// 构建搜索参数
	formParams := map[string]string{
		"key":           key,
		"file_category": "-1", // 默认搜索所有类型
		"page":          "1",
		"page_size":     "50",
	}
	
	// 添加可选参数
	if category, ok := searchArgs["file_category"].(string); ok {
		formParams["file_category"] = category
	}
	if page, ok := searchArgs["page"].(string); ok {
		formParams["page"] = page
	}
	if pageSize, ok := searchArgs["page_size"].(string); ok {
		formParams["page_size"] = pageSize
	}
	
	// 发送请求
	body, err := d.makeRequest(ctx, "File.searchList", nil, formParams)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp FileListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("搜索失败: %s", resp.Error)
	}
	
	// 转换为model.Obj
	var objects []model.Obj
	for _, file := range resp.Data.NodeList {
		// 解析文件大小
		file.Size = parseFileSize(file.CountSize)
		objects = append(objects, fileToObj(file))
	}
	
	return objects, nil
}

// getUserInfo 获取用户信息
func (d *Qihoo360) getUserInfo(ctx context.Context) (interface{}, error) {
	// 发送请求
	body, err := d.makeRequest(ctx, "User.getUserDetail", nil, nil)
	if err != nil {
		return nil, err
	}
	
	// 解析响应
	var resp UserInfoResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析用户信息响应失败: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %s", resp.Error)
	}
	
	return resp.Data, nil
}

var _ driver.Driver = (*Qihoo360)(nil)