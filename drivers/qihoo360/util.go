package qihoo360

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
)

// formatTime 格式化时间戳
func formatTime(timestamp string) time.Time {
	if timestamp == "" {
		return time.Now()
	}
	
	// 尝试解析时间戳（秒）
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		return time.Unix(ts, 0)
	}
	
	// 尝试解析时间戳（毫秒）
	if ts, err := strconv.ParseInt(timestamp, 10, 64); err == nil && ts > 1000000000000 {
		return time.Unix(ts/1000, 0)
	}
	
	return time.Now()
}

// fileToObj 将云盘文件转换为model.Obj
func fileToObj(file YunPanFile) model.Obj {
	return &model.Object{
		Name:     file.Name,
		Size:     file.Size,
		Modified: formatTime(file.ModifyTime),
		IsFolder: file.IsDir,
		ID:       file.Nid,
		Path:     file.Path,
	}
}

// generateSign 生成签名
func generateSign(params map[string]string, clientSecret string) string {
	// 对参数按key排序
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" { // 排除sign参数本身
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	
	// 构建签名字符串
	var signStr strings.Builder
	for _, k := range keys {
		if signStr.Len() > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(k)
		signStr.WriteString("=")
		signStr.WriteString(params[k])
	}
	
	// 添加client_secret
	signStr.WriteString("&client_secret=")
	signStr.WriteString(clientSecret)
	
	// MD5签名
	return fmt.Sprintf("%x", md5.Sum([]byte(signStr.String())))
}

// buildFormData 构建表单数据
func buildFormData(params map[string]string) url.Values {
	formData := url.Values{}
	for k, v := range params {
		formData.Set(k, v)
	}
	return formData
}

// parseFileSize 解析文件大小字符串
func parseFileSize(sizeStr string) int64 {
	if sizeStr == "" {
		return 0
	}
	
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size
	}
	
	return 0
}

// cleanPath 清理路径
func cleanPath(path string) string {
	if path == "" {
		return "/"
	}
	
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// 移除末尾的斜杠（除非是根目录）
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	
	return path
}

// joinPath 连接路径
func joinPath(parent, child string) string {
	parent = cleanPath(parent)
	if parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}