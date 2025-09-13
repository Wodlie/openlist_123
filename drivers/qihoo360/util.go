package qihoo360

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

const (
	BaseURL = "https://mcp.yunpan.com/api"
	UserAgent = "yunpan_mcp_server"
	
	// File categories for search
	FileCategoryAll    = -1
	FileCategoryOther  = 0
	FileCategoryImage  = 1
	FileCategoryDoc    = 2
	FileCategoryMusic  = 3
	FileCategoryVideo  = 4
)

// buildClient creates HTTP client with proper headers
func (d *Qihoo360) buildClient() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return client
}

// request makes HTTP request with authentication
func (d *Qihoo360) request(ctx context.Context, method, endpoint string, body interface{}, headers map[string]string) (*http.Response, error) {
	client := d.buildClient()
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, BaseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	
	// Set default headers
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+d.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Set custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	
	return client.Do(req)
}

// getJSON makes GET request and decode JSON response
func (d *Qihoo360) getJSON(ctx context.Context, endpoint string, params map[string]string, result interface{}) error {
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		endpoint += "?" + values.Encode()
	}
	
	resp, err := d.request(ctx, "GET", endpoint, nil, nil)
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

// postJSON makes POST request and decode JSON response
func (d *Qihoo360) postJSON(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	resp, err := d.request(ctx, "POST", endpoint, body, nil)
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

// listFiles gets file list from specified path
func (d *Qihoo360) listFiles(ctx context.Context, path string, page, pageSize int) (*ListResponse, error) {
	params := map[string]string{
		"path":      path,
		"page":      strconv.Itoa(page),
		"page_size": strconv.Itoa(pageSize),
	}
	
	var resp ListResponse
	err := d.getJSON(ctx, "/file/list", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return &resp, nil
}

// searchFiles searches files by keyword
func (d *Qihoo360) searchFiles(ctx context.Context, keyword string, fileCategory, page, pageSize int) (*SearchResponse, error) {
	params := map[string]string{
		"key":           keyword,
		"file_category": strconv.Itoa(fileCategory),
		"page":          strconv.Itoa(page),
		"page_size":     strconv.Itoa(pageSize),
	}
	
	var resp SearchResponse
	err := d.getJSON(ctx, "/file/search", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("search files: %w", err)
	}
	
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return &resp, nil
}

// getDownloadURL gets download URL for a file
func (d *Qihoo360) getDownloadURL(ctx context.Context, nid string) (*DownloadResponse, error) {
	params := map[string]string{
		"nid": nid,
	}
	
	var resp DownloadResponse
	err := d.getJSON(ctx, "/file/download", params, &resp)
	if err != nil {
		return nil, fmt.Errorf("get download URL: %w", err)
	}
	
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return &resp, nil
}

// uploadFile uploads a file to specified path
func (d *Qihoo360) uploadFile(ctx context.Context, reader io.Reader, fileName, uploadPath string) (*UploadResponse, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	
	// Add file field
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	
	_, err = io.Copy(part, reader)
	if err != nil {
		return nil, fmt.Errorf("copy file data: %w", err)
	}
	
	// Add upload path field
	writer.WriteField("upload_path", uploadPath)
	
	// Close writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}
	
	// Make request
	req, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"/file/upload", body)
	if err != nil {
		return nil, fmt.Errorf("create upload request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+d.APIKey)
	req.Header.Set("User-Agent", UserAgent)
	
	client := d.buildClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	var uploadResp UploadResponse
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read upload response: %w", err)
	}
	
	if err := json.Unmarshal(data, &uploadResp); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	
	if uploadResp.Code != 0 {
		return nil, &APIError{Code: uploadResp.Code, Msg: uploadResp.Msg}
	}
	
	return &uploadResp, nil
}

// createFolder creates a new folder
func (d *Qihoo360) createFolder(ctx context.Context, folderPath string) error {
	body := map[string]string{
		"fname": folderPath,
	}
	
	var resp CommonResponse
	err := d.postJSON(ctx, "/make-dir", body, &resp)
	if err != nil {
		return fmt.Errorf("create folder: %w", err)
	}
	
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return nil
}

// renameFile renames a file or folder
func (d *Qihoo360) renameFile(ctx context.Context, srcPath, newName string) error {
	body := map[string]string{
		"src_name": srcPath,
		"new_name": newName,
	}
	
	var resp CommonResponse
	err := d.postJSON(ctx, "/file/rename", body, &resp)
	if err != nil {
		return fmt.Errorf("rename file: %w", err)
	}
	
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return nil
}

// moveFile moves file(s) to a new location
func (d *Qihoo360) moveFile(ctx context.Context, srcPaths []string, dstPath string) error {
	body := map[string]string{
		"src_name": strings.Join(srcPaths, "|"),
		"new_name": dstPath,
	}
	
	var resp CommonResponse
	err := d.postJSON(ctx, "/file/move", body, &resp)
	if err != nil {
		return fmt.Errorf("move file: %w", err)
	}
	
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return nil
}

// deleteFile deletes a file or folder
func (d *Qihoo360) deleteFile(ctx context.Context, filePath string) error {
	body := map[string]string{
		"paths": filePath,
	}
	
	var resp CommonResponse
	err := d.postJSON(ctx, "/file/delete", body, &resp)
	if err != nil {
		return fmt.Errorf("delete file: %w", err)
	}
	
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Msg: resp.Msg}
	}
	
	return nil
}

// fileItemToObj converts FileItem to model.Obj
func (d *Qihoo360) fileItemToObj(item FileItem) model.Obj {
	obj := &model.Object{
		Name:     item.Name,
		Size:     item.Size,
		Modified: toTime(item.Mtime),
		IsFolder: item.Type == 1 || item.IsDir,
	}
	
	// Store NID in the path for later use
	if item.Path != "" {
		obj.Path = item.Path
	} else {
		obj.Path = item.NID
	}
	
	return obj
}

// getNIDFromPath extracts NID from path or searches for it
func (d *Qihoo360) getNIDFromPath(ctx context.Context, path string) (string, error) {
	if path == "" || path == "/" {
		return "root", nil
	}
	
	// Try to search for the file first
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "root", nil
	}
	
	fileName := parts[len(parts)-1]
	resp, err := d.searchFiles(ctx, fileName, FileCategoryAll, 1, 50)
	if err != nil {
		return "", fmt.Errorf("search for file NID: %w", err)
	}
	
	// Find the exact match
	for _, item := range resp.Data.List {
		if item.Path == path || item.Name == fileName {
			return item.NID, nil
		}
	}
	
	return "", fmt.Errorf("file not found: %s", path)
}