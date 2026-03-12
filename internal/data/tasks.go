package db

import (
	"gitlab.com/distributed_lab/kit/pgdb"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
)

type TaskRecord struct {
	ID        int64               `db:"id"`
	TaskType  string              `db:"task_type"`
	Status    types.ProcessStatus `db:"status"`
	Data      string              `db:"data"` // JSON encoded task data
	Error     *string             `db:"error"`
	CreatedAt time.Time           `db:"created_at"`
	UpdatedAt time.Time           `db:"updated_at"`
}

type TasksQ interface {
	New() TasksQ
	Insert(record TaskRecord) (int64, error)
	UpdateStatus(id int64, status types.ProcessStatus) error
	UpdateStatusWithError(id int64, status types.ProcessStatus, errMsg string) error
	Page(pageParams pgdb.OffsetPageParams) TasksQ
	FilterByStatus(status types.ProcessStatus) TasksQ
	Get() ([]TaskRecord, error)
	GetIncomplete() ([]TaskRecord, error) //status != COMPLETED and status != FAILED
}
