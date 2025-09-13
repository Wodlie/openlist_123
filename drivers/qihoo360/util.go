package qihoo360

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

const (
	// 360 AI云盘开放平台 API 地址
	BaseURL = "https://openapi.yunpan.360.cn/v2/openapi/entry"
	UserAgent = "yunpan_mcp_server"
	
	// 文件类别定义
	FileCategoryAll    = -1  // 全部
	FileCategoryOther  = 0   // 其他
	FileCategoryImage  = 1   // 图片
	FileCategoryDoc    = 2   // 文档 
	FileCategoryMusic  = 3   // 音乐
	FileCategoryVideo  = 4   // 视频
)

// generateSign 生成API请求签名
func (d *Qihoo360) generateSign(params map[string]string) string {
	// 按照官方文档要求生成签名
	// 1. 将参数按key排序
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// 2. 构建签名字符串
	var signStr strings.Builder
	for _, k := range keys {
		if k != "sign" && params[k] != "" {
			signStr.WriteString(k)
			signStr.WriteString("=")
			signStr.WriteString(params[k])
			signStr.WriteString("&")
		}
	}
	
	// 3. 添加API密钥
	signStr.WriteString("secret=" + d.APIKey)
	
	// 4. 计算MD5
	h := md5.New()
	h.Write([]byte(signStr.String()))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// buildClient creates HTTP client with proper headers
func (d *Qihoo360) buildClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return client
}

// request 发起API请求
func (d *Qihoo360) request(ctx context.Context, method string, params map[string]string) (*http.Response, error) {
	client := d.buildClient()
	
	// 添加基本参数
	params["method"] = method
	params["access_token"] = d.APIKey
	
	// 生成签名
	params["sign"] = d.generateSign(params)
	
	// 构建请求体
	formData := url.Values{}
	for k, v := range params {
		formData.Add(k, v)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)
	
	return client.Do(req)
}

// apiCall 调用API并解析响应
func (d *Qihoo360) apiCall(ctx context.Context, method string, params map[string]string, result interface{}) error {
	resp, err := d.request(ctx, method, params)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	
	return nil
}

// listFiles 获取文件列表
func (d *Qihoo360) listFiles(ctx context.Context, path string, page, pageSize int) (*ListResponse, error) {
	params := map[string]string{
		"path":      path,
		"page":      strconv.Itoa(page),
		"page_size": strconv.Itoa(pageSize),
	}
	
	var resp ListResponse
	err := d.apiCall(ctx, "File.getList", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return &resp, nil
}

// searchFiles 搜索文件
func (d *Qihoo360) searchFiles(ctx context.Context, keyword string, fileCategory, page, pageSize int) (*SearchResponse, error) {
	params := map[string]string{
		"key":           keyword,
		"file_category": strconv.Itoa(fileCategory),
		"page":          strconv.Itoa(page),
		"page_size":     strconv.Itoa(pageSize),
	}
	
	var resp SearchResponse
	err := d.apiCall(ctx, "File.searchList", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("search files: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return &resp, nil
}

// getDownloadURL 获取文件下载链接
func (d *Qihoo360) getDownloadURL(ctx context.Context, nid string) (*DownloadResponse, error) {
	params := map[string]string{
		"nid": nid,
	}
	
	var resp DownloadResponse
	err := d.apiCall(ctx, "Sync.getVerifiedDownLoadUrl", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("get download URL: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return &resp, nil
}

// createFolder 创建文件夹
func (d *Qihoo360) createFolder(ctx context.Context, folderPath string) error {
	params := map[string]string{
		"fname": folderPath,
	}
	
	var resp CommonResponse
	err := d.apiCall(ctx, "File.mkdir", params, &resp)
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}
	
	if resp.Errno != 0 {
		return &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return nil
}

// renameFile 重命名文件或文件夹
func (d *Qihoo360) renameFile(ctx context.Context, srcPath, newName string) error {
	params := map[string]string{
		"src_name": srcPath,
		"new_name": newName,
	}
	
	var resp CommonResponse
	err := d.apiCall(ctx, "File.rename", params, &resp)
	if err != nil {
		return fmt.Errorf("rename file: %w", err)
	}
	
	if resp.Errno != 0 {
		return &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return nil
}

// moveFile 移动文件
func (d *Qihoo360) moveFile(ctx context.Context, srcPaths []string, dstPath string) error {
	params := map[string]string{
		"src_name": strings.Join(srcPaths, "|"),
		"new_name": dstPath,
	}
	
	var resp CommonResponse
	err := d.apiCall(ctx, "File.move", params, &resp)
	if err != nil {
		return fmt.Errorf("move file: %w", err)
	}
	
	if resp.Errno != 0 {
		return &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return nil
}

// deleteFile 删除文件或文件夹
func (d *Qihoo360) deleteFile(ctx context.Context, filePath string) error {
	params := map[string]string{
		"fname": filePath,
	}
	
	var resp CommonResponse
	err := d.apiCall(ctx, "File.delete", params, &resp)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	
	if resp.Errno != 0 {
		return &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return nil
}

// shareFiles 生成分享链接
func (d *Qihoo360) shareFiles(ctx context.Context, paths string) (*ShareResponse, error) {
	params := map[string]string{
		"paths": paths,
	}
	
	var resp ShareResponse
	err := d.apiCall(ctx, "Share.preShare", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("share files: %w", err)
	}
	
	if resp.Errno != 0 {
		return nil, &APIError{Code: resp.Errno, Msg: resp.Errmsg}
	}
	
	return &resp, nil
}

// fileItemToObj converts FileItem to model.Obj
func (d *Qihoo360) fileItemToObj(item FileItem) model.Obj {
	obj := &model.Object{
		ID:       item.NID,
		Name:     item.Name,
		Size:     item.Size,
		Modified: toTime(item.Mtime),
		Ctime:    toTime(item.Ctime),
		IsFolder: item.Type == 1 || item.IsDir,
		Path:     item.Path,
	}
	
	return obj
}

// getNIDFromPath extracts NID from path or searches for it
func (d *Qihoo360) getNIDFromPath(ctx context.Context, path string) (string, error) {
	if path == "" || path == "/" {
		return "root", nil
	}
	
	// 尝试从路径中解析文件名进行搜索
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "root", nil
	}
	
	fileName := parts[len(parts)-1]
	resp, err := d.searchFiles(ctx, fileName, FileCategoryAll, 1, 50)
	if err != nil {
		return "", fmt.Errorf("search for file NID: %w", err)
	}
	
	// 查找精确匹配
	for _, item := range resp.Data.List {
		if item.Path == path || item.Name == fileName {
			return item.NID, nil
		}
	}
	
	return "", fmt.Errorf("file not found: %s", path)
}