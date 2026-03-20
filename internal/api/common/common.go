package common

import (
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	"github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"github.com/Bridgeless-Project/tss-wrapper-svc/resources/types"
	"gitlab.com/distributed_lab/kit/pgdb"
	"time"
)

const (
	DefaultLimit         = 15
	DefaultOffset        = 0
	DefaultExecutionTime = 10
)

func ToGetTasksResponse(records []db.TaskRecord) *resources.GetTasksResponse {
	mapped := make([]*types.TaskRecord, 0, len(records))
	for _, task := range records {
		record := ToResponseTaskRecord(task)
		mapped = append(mapped, &record)
	}
	return &resources.GetTasksResponse{Records: mapped}
}

func ToResponseTaskRecord(record db.TaskRecord) types.TaskRecord {
	var errMessage string
	if record.Error != nil {
		errMessage = *record.Error
	}
	return types.TaskRecord{
		Id:       record.ID,
		Tasktype: record.TaskType,
		Status:   record.Status,
		Data:     record.Data,
		Error:    errMessage,
	}
}

func GetPaginationSettings(limitPtr *uint64, offsetPtr *uint64) pgdb.OffsetPageParams {
	var limit, offset uint64
	if limitPtr == nil {
		limit = DefaultLimit
	} else {
		limit = *limitPtr
	}
	if offsetPtr == nil {
		offset = DefaultOffset
	} else {
		offset = *offsetPtr
	}
	return pgdb.OffsetPageParams{Limit: limit, PageNumber: offset}
}

func ToUpdateTimeResponse(status bool) *resources.UpdateTimeResponse {
	return &resources.UpdateTimeResponse{Result: status}
}

func ValidateTime(request *resources.UpdateTimeRequest) bool {
	timeStart := time.Unix(request.StartTime, 0).Local()
	timeTarget := time.Unix(request.TargetTime, 0).Local().Add(DefaultExecutionTime * time.Second)
	if timeStart.After(timeTarget) {
		return false
	}
	if timeStart.Before(time.Now().UTC()) {
		return false
	}
	return true
}
