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

// phpUrlEncode encodes a string in PHP/JS style used by SDK
// JavaScript's encodeURIComponent keeps - _ . ! ~ * ' ( ) unencoded,
// but the sign function encodes them again
func phpUrlEncode(str string) string {
	// First, do standard encoding but keep certain chars
	encoded := url.QueryEscape(str)
	// url.QueryEscape already encodes most things, but we need to ensure
	// these specific characters are encoded as the JS does
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
	// %20 should be + (last step)
	encoded = strings.ReplaceAll(encoded, "%20", "+")
	return encoded
}

// generateSign generates MD5 signature for API request
func generateSign(params map[string]string) string {
	// Sort keys alphabetically
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build key=encodedValue string
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		encodedValue := phpUrlEncode(params[k])
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, encodedValue))
	}
	str := strings.Join(pairs, "&")

	// Append secret key
	str += SecretKey

	// Calculate MD5
	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

func (d *Qihoo360) getAuth() (*AuthResp, error) {
	// Check if we have cached auth and it's not expired
	if d.authInfo != nil && time.Now().Unix() < d.authExpire-300 {
		return d.authInfo, nil
	}

	params := map[string]string{
		"method":        "Oauth.getAccessTokenByApiKey",
		"client_id":     ClientID,
		"client_secret": ClientSecret,
		"api_key":       d.APIKey,
		"grant_type":    "authorization_code",
	}

	// Build URL with query parameters (no sign needed for auth request)
	var resp AuthResp
	req := base.RestyClient.R().SetResult(&resp)
	for k, v := range params {
		req.SetQueryParam(k, v)
	}

	res, err := req.Get(ApiUrl)
	if err != nil {
		return nil, err
	}

	log.Debugf("Auth Response: %s", res.String())

	if resp.Errno != 0 {
		return nil, fmt.Errorf("auth failed: %s", resp.Errmsg)
	}

	// Cache auth info
	d.authInfo = &resp
	d.authExpire = time.Now().Unix() + resp.Data.AccessTokenExpire

	return &resp, nil
}

func (d *Qihoo360) request(method string, params map[string]string, result interface{}, excluded ...string) ([]byte, error) {
	// Get auth if not already authenticated
	if d.authInfo == nil || time.Now().Unix() >= d.authExpire-300 {
		_, err := d.getAuth()
		if err != nil {
			return nil, err
		}
	}

	// Build excluded params map
	excludedMap := make(map[string]bool)
	if len(excluded) > 0 {
		for _, key := range excluded {
			excludedMap[key] = true
		}
	}

	// Build params for sign (excluding specified params)
	signParams := map[string]string{
		"method":       method,
		"access_token": d.authInfo.Data.AccessToken,
		"qid":          d.authInfo.Data.Qid,
	}

	// Add params to sign if not excluded
	for k, v := range params {
		if !excludedMap[k] {
			signParams[k] = v
		}
	}

	// Generate sign
	sign := generateSign(signParams)

	log.Debugf("Request method: %s", method)

	// File.getList, Sync.getVerifiedDownLoadUrl, and Sync.getUploadFileAddr use GET
	var err error

	if method == "File.getList" || method == "Sync.getVerifiedDownLoadUrl" || method == "Sync.getUploadFileAddr" {
		// GET request: params in query string
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
		_, err = base.RestyClient.R().
			SetQueryParams(allParams).
			SetResult(result).
			SetHeader("Access-Token", d.authInfo.Data.AccessToken).
			Get(ApiUrl)
		if err != nil {
			return nil, err
		}
	} else {
		// POST request: basic params in query, all params in form
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

	if err != nil {
		return nil, err
	}

	log.Debugf("Response data received")

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

	// Normalize name display and full path for each file/dir
	for i := range resp.Data.NodeList {
		rawName := resp.Data.NodeList[i].Name
		// Trim leading slash and trailing slash for dir name
		trimmed := strings.TrimPrefix(rawName, "/")
		trimmed = strings.TrimSuffix(trimmed, "/")
		base := pathpkg.Base(trimmed)
		resp.Data.NodeList[i].Name = base

		// Construct full path
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
	_, err := d.request("Sync.getVerifiedDownLoadUrl", params, &resp)
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
