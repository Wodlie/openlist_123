package qihoo360

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	pathpkg "path"
	"sort"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	log "github.com/sirupsen/logrus"
)

const (
	ApiUrl       = "https://openapi.eyun.360.cn/intf.php"
	ClientID     = "e4757e933b6486c08ed206ecb6d5d9e684fcb4e2"
	ClientSecret = "885fd3231f1c1e37c9f462261a09b8c38cde0c2b"
	SecretKey    = "e7b24b112a44fdd9ee93bdf998c6ca0e"
)

func phpUrlEncode(str string) string {
	encoded := url.QueryEscape(str)
	replacer := strings.NewReplacer(
		"!", "%21",
		"'", "%27",
		"(", "%28",
		")", "%29",
		"*", "%2A",
		",", "%2C",
		"~", "%7E",
	)
	encoded = replacer.Replace(encoded)
	encoded = strings.ReplaceAll(encoded, "%20", "+")
	return encoded
}

func generateSign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		encodedValue := phpUrlEncode(params[k])
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, encodedValue))
	}
	str := strings.Join(pairs, "&")
	str += SecretKey
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

func (d *Qihoo360) getAuth() (*AuthResp, error) {
	if d.authInfo != nil && d.authExpire > 0 && time.Now().Unix() < d.authExpire-300 {
		return d.authInfo, nil
	}

	params := map[string]string{
		"method":        "Oauth.getAccessTokenByApiKey",
		"client_id":     ClientID,
		"client_secret": ClientSecret,
		"api_key":       d.APIKey,
		"grant_type":    "authorization_code",
	}

	var resp AuthResp
	req := base.RestyClient.R().SetResult(&resp)
	for k, v := range params {
		req.SetQueryParam(k, v)
	}

	res, err := req.Get(ApiUrl)
	if err != nil {
		d.authInfo = nil
		d.authExpire = 0
		return nil, err
	}

	log.Debugf("Auth Response: %s", res.String())

	if resp.Errno != 0 {
		d.authInfo = nil
		d.authExpire = 0
		return nil, fmt.Errorf("auth failed: %s", resp.Errmsg)
	}
	d.authInfo = &resp
	if resp.Data.AccessTokenExpire > 0 {
		d.authExpire = resp.Data.AccessTokenExpire
	} else {
		d.authExpire = time.Now().Unix() + 3600
	}

	return &resp, nil
}

func (d *Qihoo360) request(method string, params map[string]string, result interface{}, excluded ...string) ([]byte, error) {
	return d.requestWithRetry(method, params, result, 0, excluded...)
}

func (d *Qihoo360) requestWithRetry(method string, params map[string]string, result interface{}, retryCount int, excluded ...string) ([]byte, error) {
	const maxRetries = 2
	if retryCount >= maxRetries {
		return nil, fmt.Errorf("max retries (%d) exceeded for method %s", maxRetries, method)
	}
	if d.authInfo == nil || d.authExpire <= 0 || time.Now().Unix() >= d.authExpire-300 {
		_, err := d.getAuth()
		if err != nil {
			return nil, err
		}
	}
	if d.authInfo == nil {
		return nil, fmt.Errorf("authentication failed: no auth info")
	}
	excludedMap := make(map[string]bool)
	for _, key := range excluded {
		excludedMap[key] = true
	}
	signParams := map[string]string{
		"method":       method,
		"access_token": d.authInfo.Data.AccessToken,
		"qid":          d.authInfo.Data.Qid,
	}
	for k, v := range params {
		if !excludedMap[k] {
			signParams[k] = v
		}
	}
	sign := generateSign(signParams)

	log.Debugf("Request method: %s", method)

	var err error

	if method == "File.getList" || method == "Sync.getVerifiedDownLoadUrl" || method == "Sync.getUploadFileAddr" || method == "User.getUserDetail" {
		allParams := map[string]string{
			"method":       method,
			"access_token": d.authInfo.Data.AccessToken,
			"qid":          d.authInfo.Data.Qid,
			"sign":         sign,
		}
		for k, v := range params {
			if !excludedMap[k] {
				allParams[k] = v
			}
		}
		req := base.RestyClient.R().
			SetQueryParams(allParams).
			SetResult(result).
			SetHeader("Access-Token", d.authInfo.Data.AccessToken)
		if method == "Sync.getVerifiedDownLoadUrl" {
			req.SetHeader("User-Agent", "yunpan_mcp_server")
		}
		_, err = req.Get(ApiUrl)
		if err != nil {
			return nil, err
		}
	} else {
		queryParams := map[string]string{
			"method":       method,
			"access_token": d.authInfo.Data.AccessToken,
			"qid":          d.authInfo.Data.Qid,
			"sign":         sign,
		}

		formData := make(map[string]string)
		for k, v := range params {
			formData[k] = v
		}

		_, err = base.RestyClient.R().
			SetQueryParams(queryParams).
			SetFormData(formData).
			SetResult(result).
			SetHeader("Access-Token", d.authInfo.Data.AccessToken).
			SetHeader("Content-Type", "application/x-www-form-urlencoded").
			Post(ApiUrl)
		if err != nil {
			return nil, err
		}
	}

	log.Debugf("Response data received")
	if resp, ok := result.(*FileListResp); ok {
		if resp.Errno == -1 || resp.Errno == -2 {
			log.Debugf("Auth token expired (errno: %d), clearing cache and retrying (attempt %d)", resp.Errno, retryCount+1)
			d.authInfo = nil
			d.authExpire = 0
			return d.requestWithRetry(method, params, result, retryCount+1, excluded...)
		}
	} else if resp, ok := result.(*DownloadUrlResp); ok {
		if resp.Errno == -1 || resp.Errno == -2 {
			log.Debugf("Auth token expired (errno: %d), clearing cache and retrying (attempt %d)", resp.Errno, retryCount+1)
			d.authInfo = nil
			d.authExpire = 0
			return d.requestWithRetry(method, params, result, retryCount+1, excluded...)
		}
	} else if resp, ok := result.(*UserDetailResp); ok {
		if resp.Errno == -1 || resp.Errno == -2 {
			log.Debugf("Auth token expired (errno: %d), clearing cache and retrying (attempt %d)", resp.Errno, retryCount+1)
			d.authInfo = nil
			d.authExpire = 0
			return d.requestWithRetry(method, params, result, retryCount+1, excluded...)
		}
	} else if resp, ok := result.(*CommonResp); ok {
		if resp.Errno == -1 || resp.Errno == -2 {
			log.Debugf("Auth token expired (errno: %d), clearing cache and retrying (attempt %d)", resp.Errno, retryCount+1)
			d.authInfo = nil
			d.authExpire = 0
			return d.requestWithRetry(method, params, result, retryCount+1, excluded...)
		}
	}

	return nil, nil
}

func (d *Qihoo360) getFiles(path string, page int, pageSize int) ([]File, error) {
	params := map[string]string{
		"path":      path,
		"page":      fmt.Sprintf("%d", page),
		"page_size": fmt.Sprintf("%d", pageSize),
	}

	var resp FileListResp
	_, err := d.request("File.getList", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("get files failed: %s", resp.Errmsg)
	}

	for i := range resp.Data.NodeList {
		rawName := resp.Data.NodeList[i].Name
		trimmed := strings.TrimPrefix(rawName, "/")
		trimmed = strings.TrimSuffix(trimmed, "/")
		base := pathpkg.Base(trimmed)
		resp.Data.NodeList[i].Name = base
		var fullPath string
		if path == "/" {
			fullPath = "/" + base
		} else {
			fullPath = path + base
		}
		if resp.Data.NodeList[i].Type == "1" && !strings.HasSuffix(fullPath, "/") {
			fullPath += "/"
		}
		resp.Data.NodeList[i].Path = fullPath
	}

	return resp.Data.NodeList, nil
}

func (d *Qihoo360) getDownloadUrl(nid string) (string, error) {
	params := map[string]string{
		"nid": nid,
	}

	var resp DownloadUrlResp
	_, err := d.request("MCP.getDownLoadUrl", params, &resp)
	if err != nil {
		return "", err
	}

	if resp.Errno != 0 {
		return "", fmt.Errorf("get download url failed: %s", resp.Errmsg)
	}

	return resp.Data.DownloadUrl, nil
}

func (d *Qihoo360) getUploadAddr(fname string, fsize int64, fhash string, fctime, fmtime int64) (*UploadAddrResp, error) {
	// Build all query parameters
	params := map[string]string{
		"owner_qid": d.authInfo.Data.Qid,
		"fname":     fname,
		"fsize":     fmt.Sprintf("%d", fsize),
		"fctime":    fmt.Sprintf("%d", fctime),
		"fmtime":    fmt.Sprintf("%d", fmtime),
		"fhash":     fhash,
		"qid":       d.authInfo.Data.Qid,
		"fattr":     "0",
		"token":     d.authInfo.Data.Token,
		"tk":        "",
		"devtype":   "ecs_openapi",
	}

	// Calculate sign using only specific parameters (per SDK)
	signParams := map[string]string{
		"fhash":        fhash,
		"qid":          d.authInfo.Data.Qid,
		"method":       "Sync.getUploadFileAddr",
		"fname":        fname,
		"fsize":        fmt.Sprintf("%d", fsize),
		"access_token": d.authInfo.Data.AccessToken,
	}
	params["sign"] = generateSign(signParams)

	var resp UploadAddrResp
	_, err := d.request("Sync.getUploadFileAddr", params, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Errno != 0 {
		return nil, fmt.Errorf("get upload addr failed: %s", resp.Errmsg)
	}
	return &resp, nil
}

func (d *Qihoo360) getUserDetail() (*UserDetailResp, error) {
	params := map[string]string{}

	var resp UserDetailResp
	_, err := d.request("User.getUserDetail", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("get user detail failed: %s", resp.Errmsg)
	}

	return &resp, nil
}
