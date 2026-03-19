package pg

import (
	"time"

	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/kit/pgdb"
)

const (
	tasksTable = "tasks"

	taskIDField        = "id"
	taskTypeField      = "task_type"
	taskStatusField    = "status"
	taskDataField      = "data"
	taskErrorField     = "error"
	taskCreatedAtField = "created_at"
	taskUpdatedAtField = "updated_at"
)

type tasksQ struct {
	db       *pgdb.DB
	selector squirrel.SelectBuilder
}

func NewTasksQ(database *pgdb.DB) db.TasksQ {
	return &tasksQ{
		db:       database.Clone(),
		selector: squirrel.Select("*").From(tasksTable),
	}
}

func (q *tasksQ) New() db.TasksQ {
	return NewTasksQ(q.db.Clone())
}

func (q *tasksQ) Insert(record db.TaskRecord) (int64, error) {
	now := time.Now().UTC()

	// Store ProcessStatus as int32
	stmt := squirrel.Insert(tasksTable).
		Columns(taskTypeField, taskStatusField, taskDataField, taskCreatedAtField, taskUpdatedAtField).
		Values(record.TaskType, int32(record.Status), record.Data, now, now).
		Suffix("RETURNING " + taskIDField)

	var id int64
	err := q.db.Get(&id, stmt)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert task")
	}

	return id, nil
}

func (q *tasksQ) UpdateStatus(id int64, status types.ProcessStatus) error {
	stmt := squirrel.Update(tasksTable).
		Set(taskStatusField, int32(status)).
		Set(taskUpdatedAtField, time.Now().UTC()).
		Where(squirrel.Eq{taskIDField: id})

	err := q.db.Exec(stmt)
	return errors.Wrap(err, "failed to update task status")
}

func (q *tasksQ) UpdateStatusWithError(id int64, status types.ProcessStatus, errMsg string) error {
	stmt := squirrel.Update(tasksTable).
		Set(taskStatusField, int32(status)).
		Set(taskErrorField, errMsg).
		Set(taskUpdatedAtField, time.Now().UTC()).
		Where(squirrel.Eq{taskIDField: id})

	err := q.db.Exec(stmt)
	return errors.Wrap(err, "failed to update task status with error")
}

func (q *tasksQ) Page(pageParams pgdb.OffsetPageParams) db.TasksQ {
	q.selector = pageParams.ApplyTo(q.selector, "id")
	return q
}

func (q *tasksQ) FilterByStatus(status types.ProcessStatus) db.TasksQ {
	q.selector = q.selector.Where(squirrel.Eq{taskStatusField: int32(status)})
	return q
}

func (q *tasksQ) OrderByCreatedAt() db.TasksQ {
	q.selector = q.selector.OrderBy(taskCreatedAtField + " ASC")
	return q
}

func (q *tasksQ) GetAll() ([]db.TaskRecord, error) {
	var records []db.TaskRecord
	err := q.db.Select(&records, q.selector)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tasks")
	}

	return records, nil
}

func (q *tasksQ) GetIncomplete() ([]db.TaskRecord, error) {
	// Get tasks that are not COMPLETED and not FAILED
	stmt := q.selector.Where(
		squirrel.And{
			squirrel.NotEq{taskStatusField: int32(types.ProcessStatus_PROCESS_STATUS_COMPLETED)},
			squirrel.NotEq{taskStatusField: int32(types.ProcessStatus_PROCESS_STATUS_FAILED)},
		},
	).OrderBy(taskCreatedAtField + " ASC")

	var records []db.TaskRecord
	err := q.db.Select(&records, stmt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get incomplete tasks")
	}

	return records, nil
}
