package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/types"
	"github.com/pkg/errors"
)

const TaskType = "update"

const (
	AttributeUpdateLink      = "update_link"
	AttributeUpdateStartTime = "update_start_time"
)

const checksumParam = "checksum"

const (
	newBinarySuffix = ".new"
	backupSuffix    = ".bak"
	binaryMode      = 0o755

	downloadTimeout = 10 * time.Minute
	verifyTimeout   = 30 * time.Second
)

type taskData struct {
	Link      string `json:"link"`
	StartTime int64  `json:"start_time"`
}

type Task struct {
	id int64

	BinaryPath string
	Link       string
	StartTime  time.Time

	backupPath string
}

func NewTask(tssconfig *config.TSSConfig) *Task {
	return &Task{BinaryPath: tssconfig.BinaryPath}
}

func (t *Task) Execute(ctx context.Context) (bool, error) {
	if t.BinaryPath == "" {
		return true, errors.New("binary path is not set")
	}
	if t.Link == "" {
		return true, errors.New("update link is not set")
	}

	link, checksum, err := splitChecksum(t.Link)
	if err != nil {
		return true, errors.Wrap(err, "invalid update link")
	}
	if checksum == "" {
		return true, errors.New("update link has no checksum")
	}

	newPath := t.BinaryPath + newBinarySuffix
	defer os.Remove(newPath)

	if err = t.downloadBinary(ctx, link, checksum, newPath); err != nil {
		return true, errors.Wrap(err, "failed to download TSS binary")
	}

	if err := verifyBinary(ctx, newPath); err != nil {
		return true, errors.Wrap(err, "downloaded TSS binary is not runnable")
	}

	return true, errors.Wrap(t.replaceBinary(newPath), "failed to replace TSS binary")
}

func (t *Task) GetTime() time.Time {
	return t.StartTime
}

func (t *Task) Parse(attributes []types.Attribute) (types.Task, error) {
	task := &Task{BinaryPath: t.BinaryPath}

	for _, attribute := range attributes {
		switch attribute.Key {
		case AttributeUpdateLink:
			task.Link = attribute.Value
		case AttributeUpdateStartTime:
			startTime, err := strconv.ParseInt(attribute.Value, 10, 64)
			if err != nil {
				return nil, errors.Wrap(err, "failed to parse update start time")
			}
			task.StartTime = time.Unix(startTime, 0)
		default:
			continue
		}
	}

	if task.Link == "" {
		return nil, errors.New("update event has no binary link")
	}

	return task, nil
}

func (t *Task) StartScheduling(ctx context.Context, taskChan chan<- types.Task) {
	delay := time.Until(t.StartTime)
	if delay <= 0 {
		taskChan <- t
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		taskChan <- t
	}
}

func (t *Task) GetName() string {
	return "UpdateTask"
}

func (t *Task) GetID() int64 {
	return t.id
}

func (t *Task) SetID(id int64) {
	t.id = id
}

func (t *Task) GetTaskType() string {
	return TaskType
}

func (t *Task) MarshalData() (string, error) {
	data := taskData{
		Link:      t.Link,
		StartTime: t.StartTime.Unix(),
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal task data")
	}
	return string(bytes), nil
}

func (t *Task) UnmarshalData(data string) error {
	var td taskData
	if err := json.Unmarshal([]byte(data), &td); err != nil {
		return errors.Wrap(err, "failed to unmarshal task data")
	}
	t.Link = td.Link
	t.StartTime = time.Unix(td.StartTime, 0)
	return nil
}

func (t *Task) downloadBinary(ctx context.Context, url string, checksum string, filepath string) error {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to build download request")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute download request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "failed to create local file")
	}
	defer out.Close()

	digest := sha256.New()
	if _, err = io.Copy(out, io.TeeReader(resp.Body, digest)); err != nil {
		return errors.Wrap(err, "failed to write and hash downloaded file")
	}

	if err = compareChecksum(checksum, digest.Sum(nil)); err != nil {
		return errors.Wrap(err, "checksum validation failed")
	}

	if err = out.Chmod(binaryMode); err != nil {
		return errors.Wrap(err, "failed to make downloaded binary executable")
	}

	return out.Sync()
}

func verifyBinary(ctx context.Context, filepath string) error {
	ctx, cancel := context.WithTimeout(ctx, verifyTimeout)
	defer cancel()

	if err := exec.CommandContext(ctx, filepath, "--help").Run(); err != nil {
		return errors.Wrap(err, "binary failed to execute --help")
	}
	return nil
}

func (t *Task) replaceBinary(newPath string) error {
	backupPath := t.BinaryPath + backupSuffix

	backed := true
	if err := os.Rename(t.BinaryPath, backupPath); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to back up current binary")
		}
		backed = false
	}

	if err := os.Rename(newPath, t.BinaryPath); err != nil {
		if backed {
			if restoreErr := os.Rename(backupPath, t.BinaryPath); restoreErr != nil {
				return errors.Wrap(restoreErr, "failed to restore current binary")
			}
		}
		return errors.Wrap(err, "failed to move new binary into place")
	}

	if backed {
		t.backupPath = backupPath
	}

	return nil
}

func splitChecksum(link string) (string, string, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to parse URL")
	}

	query := parsed.Query()
	checksum := query.Get(checksumParam)
	if checksum == "" {
		return link, "", nil
	}

	query.Del(checksumParam)
	parsed.RawQuery = query.Encode()

	return parsed.String(), checksum, nil
}

func compareChecksum(expected string, sum []byte) error {
	algorithm, want, ok := strings.Cut(expected, ":")
	if !ok {
		return errors.Errorf("malformed checksum %q, want <algorithm>:<hex>", expected)
	}
	if algorithm != "sha256" {
		return errors.Errorf("unsupported checksum algorithm %q", algorithm)
	}

	got := hex.EncodeToString(sum)
	if !strings.EqualFold(got, want) {
		return errors.Errorf("checksum mismatch: got %s, want %s", got, want)
	}

	return nil
}

func (t *Task) Revert() error {
	if t.backupPath == "" {
		return nil
	}

	if err := os.Rename(t.backupPath, t.BinaryPath); err != nil {
		return errors.Wrap(err, "failed to restore previous binary")
	}
	t.backupPath = ""

	return nil
}
