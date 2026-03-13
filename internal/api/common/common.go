package common

import (
	types2 "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/types"
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
)

func ToGetTasksResponse(records []db.TaskRecord) *types2.GetTasksResponse {
	mapped := make([]*types2.TaskRecord, len(records))
	for i := range records {
		mapped[i] = ToResponseTaskRecord(records[i])
	}
	return &types2.GetTasksResponse{Records: mapped}
}

func ToResponseTaskRecord(record db.TaskRecord) *types2.TaskRecord {
	var errMessage string
	if record.Error != nil {
		errMessage = *record.Error
	}
	return &types2.TaskRecord{
		Id:       record.ID,
		Tasktype: record.TaskType,
		Status:   record.Status,
		Data:     record.Data,
		Error:    errMessage,
	}
}
