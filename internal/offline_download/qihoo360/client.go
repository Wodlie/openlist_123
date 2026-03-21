package qihoo360

import (
	"context"
	"fmt"

	_360 "github.com/OpenListTeam/OpenList/v4/drivers/qihoo360"
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/offline_download/tool"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
)

type Qihoo360 struct{}

func (*Qihoo360) Name() string {
	return "Qihoo360"
}

func (*Qihoo360) Items() []model.SettingItem {
	return []model.SettingItem{
		{Key: conf.Qihoo360TempDir, Value: "", Type: conf.TypeString, Group: model.OFFLINE_DOWNLOAD, Flag: model.PRIVATE},
	}
}

func (*Qihoo360) Run(_ *tool.DownloadTask) error {
	return errs.NotSupport
}

func (*Qihoo360) Init() (string, error) {
	return "ok", nil
}

func (*Qihoo360) IsReady() bool {
	tempDir := setting.GetStr(conf.Qihoo360TempDir)
	if tempDir == "" {
		return false
	}
	storage, _, err := op.GetStorageAndActualPath(tempDir)
	if err != nil {
		return false
	}
	if _, ok := storage.(*_360.Qihoo360); !ok {
		return false
	}
	return true
}

func (*Qihoo360) AddURL(args *tool.AddUrlArgs) (string, error) {
	storage, actualPath, err := op.GetStorageAndActualPath(args.TempDir)
	if err != nil {
		return "", err
	}
	driver360, ok := storage.(*_360.Qihoo360)
	if !ok {
		return "", fmt.Errorf("unsupported storage driver for offline download, only Qihoo360 is supported")
	}
	ctx := context.Background()
	if err := op.MakeDir(ctx, storage, actualPath); err != nil {
		return "", err
	}
	taskID, err := driver360.OfflineDownload(ctx, args.Url, actualPath)
	if err != nil {
		return "", fmt.Errorf("failed to add offline download task: %w", err)
	}
	return taskID, nil
}

func (*Qihoo360) Remove(task *tool.DownloadTask) error {
	storage, _, err := op.GetStorageAndActualPath(task.TempDir)
	if err != nil {
		return err
	}
	driver360, ok := storage.(*_360.Qihoo360)
	if !ok {
		return fmt.Errorf("unsupported storage driver for offline download, only Qihoo360 is supported")
	}
	return driver360.DeleteOfflineTasks(context.Background(), []string{task.GID})
}

func (*Qihoo360) Status(task *tool.DownloadTask) (*tool.Status, error) {
	storage, _, err := op.GetStorageAndActualPath(task.TempDir)
	if err != nil {
		return nil, err
	}
	driver360, ok := storage.(*_360.Qihoo360)
	if !ok {
		return nil, fmt.Errorf("unsupported storage driver for offline download, only Qihoo360 is supported")
	}
	t, err := driver360.QueryOfflineTask(context.Background(), task.GID)
	if err != nil {
		return nil, err
	}
	statusStr := "downloading"
	completed := false
	var taskErr error
	switch t.Status {
	case 0:
		statusStr = "downloading"
	case 2:
		statusStr = "succeed"
		completed = true
	case 1, 3:
		statusStr = "failed"
		if t.Error != "" {
			taskErr = fmt.Errorf("offline download failed: %s", t.Error)
		} else {
			taskErr = fmt.Errorf("offline download failed")
		}
	default:
		statusStr = fmt.Sprintf("status_%d", t.Status)
	}
	return &tool.Status{
		TotalBytes: t.FileSize,
		Progress:   t.Progress,
		Completed:  completed,
		Status:     statusStr,
		Err:        taskErr,
	}, nil
}

var _ tool.Tool = (*Qihoo360)(nil)

func init() {
	tool.Tools.Add(&Qihoo360{})
}
