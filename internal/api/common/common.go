package common

import (
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	grpcTypes "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"gitlab.com/distributed_lab/kit/pgdb"
)

func ToGetTasksResponse(records []db.TaskRecord) *grpcTypes.GetTasksResponse {
	mapped := make([]*grpcTypes.TaskRecord, 0, len(records))
	for _, task := range records {
		record := ToResponseTaskRecord(task)
		mapped = append(mapped, &record)
	}
	return &grpcTypes.GetTasksResponse{Records: mapped}
}

func ToResponseTaskRecord(record db.TaskRecord) grpcTypes.TaskRecord {
	var errMessage string
	if record.Error != nil {
		errMessage = *record.Error
	}
	return grpcTypes.TaskRecord{
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

const (
	DefaultLimit  = 15
	DefaultOffset = 0
)
