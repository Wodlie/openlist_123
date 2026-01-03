package qihoo360

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	}
	if path == "" {
		path = "/"
	}

	// Ensure directory paths end with / (required by API for non-root paths)
	if path != "/" && !strings.HasSuffix(path, "/") {
		path += "/"
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
			"User-Agent": []string{"yunpan_mcp_server"},
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
		"fname": srcPath,
	}

	var resp CommonResp
	// fname parameter is excluded from sign calculation
	_, err := d.request("File.delete", params, &resp, "fname")
	if err != nil {
		return err
	}

	if resp.Errno != 0 {
		return fmt.Errorf("remove failed: %s", resp.Errmsg)
	}

	return nil
}

func (d *Qihoo360) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
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

	fname := dstPath + file.GetName()
	fsize := file.GetSize()
	now := time.Now().Unix()

	// Calculate file hash
	const chunkSize = 524288 // 512KB per chunk
	numChunks := (fsize + chunkSize - 1) / chunkSize

	var blockHashes []string
	var blocks []struct {
		data   []byte
		offset int64
		size   int64
		hash   string
	}

	// Read file and calculate chunk hashes
	for i := int64(0); i < numChunks; i++ {
		size := chunkSize
		if i == numChunks-1 {
			size = int(fsize - i*chunkSize)
		}

		buf := make([]byte, size)
		n, err := io.ReadFull(file, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return nil, err
		}
		buf = buf[:n]

		// Calculate SHA1 hash for this block
		hash := sha1.Sum(buf)
		blockHash := hex.EncodeToString(hash[:])
		blockHashes = append(blockHashes, blockHash)

		blocks = append(blocks, struct {
			data   []byte
			offset int64
			size   int64
			hash   string
		}{
			data:   buf,
			offset: i * chunkSize,
			size:   int64(n),
			hash:   blockHash,
		})
	}

	// Calculate file hash (SHA1 of concatenated block hashes)
	fhashStr := strings.Join(blockHashes, "")
	fhash := sha1.Sum([]byte(fhashStr))
	fhashHex := hex.EncodeToString(fhash[:])

	// Get upload address
	uploadAddr, err := d.getUploadAddr(fname, fsize, fhashHex, now, now)
	if err != nil {
		return nil, err
	}

	// Check for instant upload (server already has the file)
	// When file exists, HTTP is null
	httpVal, httpOk := uploadAddr.Data.HTTP.(string)
	if !httpOk || httpVal == "" {
		return nil, nil // Instant upload success (file already exists)
	}

	// Build upload host
	uploadHost := httpVal
	if uploadAddr.Data.IsHttps == 1 {
		uploadHost = "https://" + uploadHost
	} else {
		uploadHost = "http://" + uploadHost
	}

	// Get upload token
	var tk string
	if tkVal, ok := uploadAddr.Data.Tk.(string); ok {
		tk = tkVal
	}

	// Prepare block info for preload
	blockInfoList := make([]BlockInfo, len(blocks))
	for i, block := range blocks {
		blockInfoList[i] = BlockInfo{
			BHash:   block.hash,
			BIdx:    i + 1,
			BOffset: block.offset,
			BSize:   block.size,
		}
	}

	// Preload - send block info
	preloadResp, err := d.preloadBlocks(ctx, uploadHost, fname, fsize, fhashHex, now, now, tk, blockInfoList)
	if err != nil {
		return nil, err
	}

	// Upload each block
	for i, block := range blocks {
		blockInfo := preloadResp.Data.BlockInfo[i]
		// Note: use user token (d.authInfo.Data.Token), not blockInfo.Token
		err = d.uploadBlock(ctx, uploadHost, block.data, block.hash, i+1, block.offset, block.size,
			fname, fsize, blockInfo.Q, blockInfo.T, d.authInfo.Data.Token, preloadResp.Data.Tid)
		if err != nil {
			return nil, fmt.Errorf("upload block %d failed: %w", i+1, err)
		}

		// Update progress
		if up != nil {
			up(float64(block.size))
		}
	}

	// Commit - merge blocks
	// Note: use user token (d.authInfo.Data.Token), not blockInfo.Token
	commitResp, err := d.commitUpload(ctx, uploadHost, preloadResp.Data.BlockInfo[0].Q, preloadResp.Data.BlockInfo[0].T,
		d.authInfo.Data.Token, preloadResp.Data.Tid)
	if err != nil {
		return nil, err
	}

	// If autoCommit is true (non-zero), file is already added (instant upload), use data from commit
	if commitResp.Data.AutoCommit != 0 {
		return &File{
			Name:         file.GetName(),
			Type:         "0",
			Nid:          commitResp.Data.Nid,
			CountSize:    fmt.Sprintf("%d", commitResp.Data.Size),
			CreateTimeTS: fmt.Sprintf("%d", commitResp.Data.CreateTime),
			ModifyTimeTS: fmt.Sprintf("%d", commitResp.Data.ModifyTime),
			Path:         fname,
		}, nil
	}

	// Call Sync.addFileToApi to finalize the upload and get file info
	addFileResp, err := d.addFileToApi(commitResp.Data.Tk)
	if err != nil {
		return nil, err
	}

	// Set the full path
	addFileResp.Data.File.Path = fname

	return &addFileResp.Data.File, nil
}

func (d *Qihoo360) preloadBlocks(ctx context.Context, uploadHost, fname string, fsize int64, fhash string, fctime, fmtime int64, tk string, blocks []BlockInfo) (*PreloadResp, error) {
	// Build query parameters
	queryParams := map[string]string{
		"method":    "Upload.request4Web",
		"owner_qid": d.authInfo.Data.Qid,
		"qid":       d.authInfo.Data.Qid,
		"devtype":   "ecs_openapi",
		"devid":     "node-sdk-v16.20.2", // device id
		"v":         "1.0.1",
		"ofmt":      "json",
		"devname":   "EYUN_WEB_UPLOAD",
		"rtick":     fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Build URL
	url := fmt.Sprintf("%s/intf.php", uploadHost)
	for k, v := range queryParams {
		if strings.Contains(url, "?") {
			url += "&"
		} else {
			url += "?"
		}
		url += k + "=" + v
	}

	// Prepare block_info JSON
	blockInfoMap := map[string]interface{}{
		"request": map[string]interface{}{
			"block_info": blocks,
		},
	}
	blockInfoJSON, _ := json.Marshal(blockInfoMap)

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	writer.WriteField("owner_qid", d.authInfo.Data.Qid)
	writer.WriteField("fname", fname)
	writer.WriteField("fsize", fmt.Sprintf("%d", fsize))
	writer.WriteField("fctime", fmt.Sprintf("%d", fctime))
	writer.WriteField("fmtime", fmt.Sprintf("%d", fmtime))
	writer.WriteField("fhash", fhash)
	writer.WriteField("qid", d.authInfo.Data.Qid)
	writer.WriteField("fattr", "0")
	writer.WriteField("token", d.authInfo.Data.Token)
	writer.WriteField("tk", tk)
	writer.WriteField("devtype", "ecs_openapi")

	// Add file part with block_info JSON
	part, _ := writer.CreateFormFile("file", "block_info.json")
	part.Write(blockInfoJSON)
	writer.Close()

	// Send request
	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Access-Token", d.authInfo.Data.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var preloadResp PreloadResp
	if err := json.Unmarshal(body, &preloadResp); err != nil {
		return nil, err
	}

	if preloadResp.Errno != 0 {
		return nil, fmt.Errorf("preload failed: %s", preloadResp.Errmsg)
	}

	return &preloadResp, nil
}

func (d *Qihoo360) uploadBlock(ctx context.Context, uploadHost string, data []byte, bhash string, bidx int, boffset, bsize int64, filename string, filesize int64, q, t, token, tid string) error {
	// Build query parameters
	queryParams := map[string]string{
		"method":    "Upload.block4Web",
		"owner_qid": d.authInfo.Data.Qid,
		"qid":       d.authInfo.Data.Qid,
		"devtype":   "ecs_openapi",
		"devid":     "node-sdk-v16.20.2",
		"v":         "1.0.1",
		"ofmt":      "json",
		"devname":   "EYUN_WEB_UPLOAD",
		"rtick":     fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Build URL
	url := fmt.Sprintf("%s/intf.php", uploadHost)
	for k, v := range queryParams {
		if strings.Contains(url, "?") {
			url += "&"
		} else {
			url += "?"
		}
		url += k + "=" + v
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add chunk data
	part, _ := writer.CreateFormFile("file", "chunk")
	part.Write(data)

	// Add form fields
	writer.WriteField("bhash", bhash)
	writer.WriteField("bidx", strconv.Itoa(bidx))
	writer.WriteField("boffset", fmt.Sprintf("%d", boffset))
	writer.WriteField("bsize", fmt.Sprintf("%d", bsize))
	writer.WriteField("filename", filename)
	writer.WriteField("filesize", fmt.Sprintf("%d", filesize))
	writer.WriteField("q", q)
	writer.WriteField("t", t)
	writer.WriteField("token", token)
	writer.WriteField("tid", tid)
	writer.Close()

	// Send request
	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Access-Token", d.authInfo.Data.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result CommonResp
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if result.Errno != 0 {
		return fmt.Errorf("upload block failed: %s", result.Errmsg)
	}

	return nil
}

func (d *Qihoo360) commitUpload(ctx context.Context, uploadHost, q, t, token, tid string) (*CommitResp, error) {
	// Build query parameters
	queryParams := map[string]string{
		"method":    "Upload.commit4Web",
		"owner_qid": d.authInfo.Data.Qid,
		"qid":       d.authInfo.Data.Qid,
		"devtype":   "ecs_openapi",
		"devid":     "node-sdk-v16.20.2",
		"v":         "1.0.1",
		"ofmt":      "json",
		"devname":   "EYUN_WEB_UPLOAD",
		"rtick":     fmt.Sprintf("%d", time.Now().Unix()),
	}

	// Build URL
	url := fmt.Sprintf("%s/intf.php", uploadHost)
	for k, v := range queryParams {
		if strings.Contains(url, "?") {
			url += "&"
		} else {
			url += "?"
		}
		url += k + "=" + v
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	writer.WriteField("q", q)
	writer.WriteField("t", t)
	writer.WriteField("token", token)
	writer.WriteField("tid", tid)
	writer.Close()

	// Send request
	req, _ := http.NewRequestWithContext(ctx, "POST", url, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Access-Token", d.authInfo.Data.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result CommitResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if result.Errno != 0 {
		return nil, fmt.Errorf("commit failed: %s", result.Errmsg)
	}

	return &result, nil
}

func (d *Qihoo360) addFileToApi(tk string) (*AddFileResp, error) {
	params := map[string]string{
		"qid": d.authInfo.Data.Qid,
		"tk":  tk,
	}

	var resp AddFileResp
	_, err := d.request("Sync.addFileToApi", params, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Errno != 0 {
		return nil, fmt.Errorf("add file to api failed: %s", resp.Errmsg)
	}

	return &resp, nil
}

var _ driver.Driver = (*Qihoo360)(nil)
